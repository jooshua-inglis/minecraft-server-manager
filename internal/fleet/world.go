package fleet

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/archiveutil"
	"github.com/jooshua-inglis/minecraft-server-manager/internal/serverstore"
)

const worldBackupsDirName = "world-backups"

// worldDir is the path to name's world save — data/<level>/, where
// <level> is whatever meta.Level() says (mirrors the itzg image's
// LEVEL env var). Everything else in data/ (configs, mods, plugins,
// whitelist/ops/bans, logs) is a server-config concern (M10), not a
// world one.
func worldDir(root, name string, meta *serverstore.Metadata) string {
	return filepath.Join(serverstore.DataDir(root, name), meta.Level())
}

func worldBackupsDir(root, name string) string {
	return filepath.Join(serverstore.ServerDir(root, name), worldBackupsDirName)
}

// requireStopped errors out unless meta's server is currently stopped.
// v0 of every world operation needs this: no live RCON
// save-coordination yet (that's a possible follow-up once this and
// M10 both exist), so mutating the world directory underneath a
// running container would corrupt whatever it has open.
func (f *Fleet) requireStopped(ctx context.Context, meta *serverstore.Metadata) error {
	running, err := f.isRunning(ctx, meta)
	if err != nil {
		return err
	}
	if running {
		return fmt.Errorf("stop %q before running world operations on it", meta.Name)
	}
	return nil
}

// WorldBackup snapshots just a server's world save (data/<level>/) to
// a tar.gz under <server>/world-backups/ — unlike Backup (M10), it
// leaves mods/plugins/configs/whitelist untouched. The server must be
// stopped.
func (f *Fleet) WorldBackup(ctx context.Context, name string) (*BackupInfo, error) {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return nil, err
	}
	if err := f.requireStopped(ctx, meta); err != nil {
		return nil, err
	}

	dir := worldBackupsDir(f.Root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	id := time.Now().UTC().Format("20060102-150405")
	dest := filepath.Join(dir, id+".tar.gz")
	if err := archiveutil.CreateTarGz(worldDir(f.Root, name, meta), dest); err != nil {
		return nil, fmt.Errorf("archiving world: %w", err)
	}

	info, err := os.Stat(dest)
	if err != nil {
		return nil, err
	}
	return &BackupInfo{ID: id, SizeBytes: info.Size(), CreatedAt: info.ModTime().UTC()}, nil
}

// WorldBackupList returns name's world backups, oldest first.
func (f *Fleet) WorldBackupList(name string) ([]BackupInfo, error) {
	if !serverstore.Exists(f.Root, name) {
		return nil, serverstore.ErrNotFound
	}
	return listArchivesIn(worldBackupsDir(f.Root, name))
}

// WorldRestore replaces a server's world save with a previously taken
// world backup, leaving everything else in data/ untouched. The
// server must be stopped.
func (f *Fleet) WorldRestore(ctx context.Context, name, id string) error {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return err
	}
	if err := f.requireStopped(ctx, meta); err != nil {
		return err
	}

	src := filepath.Join(worldBackupsDir(f.Root, name), id+".tar.gz")
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("world backup %q not found for %q", id, name)
	}

	dir := worldDir(f.Root, name, meta)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := replaceDirFromArchive(dir, src); err != nil {
		return fmt.Errorf("restoring world backup %q: %w", id, err)
	}
	return nil
}

// WorldReset deletes a server's current world save so the itzg image
// generates a brand new one on next start, without touching
// type/version/mods/whitelist. The server must be stopped.
func (f *Fleet) WorldReset(ctx context.Context, name string) error {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return err
	}
	if err := f.requireStopped(ctx, meta); err != nil {
		return err
	}
	return os.RemoveAll(worldDir(f.Root, name, meta))
}

// WorldExport writes a server's world save out as a portable tar.gz at
// outputPath, independent of the server's type/version/mods — the file
// WorldImport reads back in, on this server or a different one
// entirely. The server must be stopped.
func (f *Fleet) WorldExport(ctx context.Context, name, outputPath string) error {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return err
	}
	if err := f.requireStopped(ctx, meta); err != nil {
		return err
	}
	if err := archiveutil.CreateTarGz(worldDir(f.Root, name, meta), outputPath); err != nil {
		return fmt.Errorf("exporting world: %w", err)
	}
	return nil
}

// WorldImport replaces a server's world save with the contents of a
// tar.gz previously produced by WorldExport (from this server or a
// different one, possibly running different server software
// entirely). The server must be stopped.
func (f *Fleet) WorldImport(ctx context.Context, name, inputPath string) error {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return err
	}
	if err := f.requireStopped(ctx, meta); err != nil {
		return err
	}
	if _, err := os.Stat(inputPath); err != nil {
		return fmt.Errorf("reading %q: %w", inputPath, err)
	}

	dir := worldDir(f.Root, name, meta)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := replaceDirFromArchive(dir, inputPath); err != nil {
		return fmt.Errorf("importing world: %w", err)
	}
	return nil
}
