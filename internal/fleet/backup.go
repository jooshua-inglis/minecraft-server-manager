package fleet

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/archiveutil"
	"github.com/jooshua-inglis/minecraft-server-manager/internal/serverstore"
)

const backupsDirName = "backups"

func backupsDir(root, name string) string {
	return filepath.Join(serverstore.ServerDir(root, name), backupsDirName)
}

type BackupInfo struct {
	ID        string    `json:"id"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

// Backup snapshots a server's entire data/ directory (world, mods,
// plugins, configs, whitelist/ops/bans — everything) to a single
// tar.gz under <server>/backups/. If the server is running, it's
// coordinated over RCON (save-off, save-all flush, then save-on
// afterward) so the copy isn't taken mid-write; if it's stopped, the
// directory is just archived directly.
func (f *Fleet) Backup(ctx context.Context, name string) (*BackupInfo, error) {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return nil, err
	}

	running, err := f.isRunning(ctx, meta)
	if err != nil {
		return nil, err
	}

	if running {
		if _, err := f.Exec(ctx, name, "save-off"); err != nil {
			return nil, fmt.Errorf("pausing world saves: %w", err)
		}
		defer f.Exec(ctx, name, "save-on")

		if _, err := f.Exec(ctx, name, "save-all flush"); err != nil {
			return nil, fmt.Errorf("flushing world saves: %w", err)
		}
		// save-all flush's RCON response returns before the async disk
		// write it triggers is guaranteed to have finished; a short
		// pause avoids racing the archive read against it.
		time.Sleep(2 * time.Second)
	}

	dir := backupsDir(f.Root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	id := time.Now().UTC().Format("20060102-150405")
	dest := filepath.Join(dir, id+".tar.gz")
	if err := archiveutil.CreateTarGz(serverstore.DataDir(f.Root, name), dest); err != nil {
		return nil, fmt.Errorf("archiving data directory: %w", err)
	}

	info, err := os.Stat(dest)
	if err != nil {
		return nil, err
	}
	return &BackupInfo{ID: id, SizeBytes: info.Size(), CreatedAt: info.ModTime().UTC()}, nil
}

// ListBackups returns name's backups, oldest first (backup IDs are
// timestamps, so this is also chronological order).
func (f *Fleet) ListBackups(name string) ([]BackupInfo, error) {
	if !serverstore.Exists(f.Root, name) {
		return nil, serverstore.ErrNotFound
	}

	entries, err := os.ReadDir(backupsDir(f.Root, name))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var backups []BackupInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tar.gz") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			return nil, err
		}
		backups = append(backups, BackupInfo{
			ID:        strings.TrimSuffix(e.Name(), ".tar.gz"),
			SizeBytes: info.Size(),
			CreatedAt: info.ModTime().UTC(),
		})
	}
	sort.Slice(backups, func(i, j int) bool { return backups[i].ID < backups[j].ID })
	return backups, nil
}

// Restore replaces a server's entire data/ directory with the contents
// of a previously taken backup. The server must be stopped, since
// restoring underneath a running container's bind mount would corrupt
// whatever it has open.
func (f *Fleet) Restore(ctx context.Context, name, id string) (err error) {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return err
	}
	running, err := f.isRunning(ctx, meta)
	if err != nil {
		return err
	}
	if running {
		return fmt.Errorf("stop %q before restoring a backup", name)
	}

	src := filepath.Join(backupsDir(f.Root, name), id+".tar.gz")
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("backup %q not found for %q", id, name)
	}

	dataDir := serverstore.DataDir(f.Root, name)
	tmpDir := dataDir + ".restore-tmp"
	if err := os.RemoveAll(tmpDir); err != nil {
		return err
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}
	if err := archiveutil.ExtractTarGz(src, tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("extracting backup %q: %w", id, err)
	}

	// Swap dataDir's *contents*, not the directory itself: on Docker
	// Desktop's WSL2 backend, replacing a bind-mounted directory's own
	// inode (e.g. via rename) permanently breaks that mount for any
	// container already created against it, even while stopped. Moving
	// children in and out leaves dataDir's inode untouched.
	preRestoreDir := dataDir + ".pre-restore"
	if err := os.RemoveAll(preRestoreDir); err != nil {
		return err
	}
	if err := os.MkdirAll(preRestoreDir, 0o755); err != nil {
		return err
	}
	if err := moveChildren(dataDir, preRestoreDir); err != nil {
		return fmt.Errorf("moving aside current data before restore: %w", err)
	}
	if err := moveChildren(tmpDir, dataDir); err != nil {
		// Best-effort roll back so a failed restore doesn't leave the
		// server without its prior data.
		moveChildren(preRestoreDir, dataDir)
		return fmt.Errorf("moving restored data into place: %w", err)
	}
	os.Remove(tmpDir)
	return os.RemoveAll(preRestoreDir)
}

// moveChildren renames every entry directly inside srcDir into dstDir,
// leaving both directories' own inodes untouched.
func moveChildren(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := os.Rename(filepath.Join(srcDir, e.Name()), filepath.Join(dstDir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
