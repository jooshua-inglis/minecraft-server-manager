package fleet

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/archiveutil"
)

// replaceDirFromArchive underpins both whole-server restore (M10) and
// world restore/import (M17): it must swap in the archive's contents
// while leaving dir itself (a bind-mount target) in place, and drop
// anything that wasn't in the archive.
func TestReplaceDirFromArchive(t *testing.T) {
	tmp := t.TempDir()

	src := filepath.Join(tmp, "src")
	os.MkdirAll(filepath.Join(src, "sub"), 0o755)
	os.WriteFile(filepath.Join(src, "keep.txt"), []byte("from-archive"), 0o644)
	os.WriteFile(filepath.Join(src, "sub", "nested.txt"), []byte("nested"), 0o644)
	archive := filepath.Join(tmp, "a.tar.gz")
	if err := archiveutil.CreateTarGz(src, archive); err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(tmp, "dir")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "stale.txt"), []byte("old"), 0o644)
	before, _ := os.Stat(dir)

	if err := replaceDirFromArchive(dir, archive); err != nil {
		t.Fatal(err)
	}

	if b, _ := os.ReadFile(filepath.Join(dir, "keep.txt")); string(b) != "from-archive" {
		t.Errorf("keep.txt = %q", b)
	}
	if _, err := os.Stat(filepath.Join(dir, "sub", "nested.txt")); err != nil {
		t.Errorf("nested file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "stale.txt")); !os.IsNotExist(err) {
		t.Errorf("stale.txt should be gone, stat err = %v", err)
	}
	if after, _ := os.Stat(dir); !os.SameFile(before, after) {
		t.Errorf("dir's own inode changed; bind mounts would break")
	}
	for _, leftover := range []string{dir + ".restore-tmp", dir + ".pre-restore"} {
		if _, err := os.Stat(leftover); !os.IsNotExist(err) {
			t.Errorf("scratch dir %s left behind", leftover)
		}
	}
}

func TestListArchivesIn(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"20260102-000000.tar.gz", "20260101-000000.tar.gz", "notes.txt"} {
		os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644)
	}
	got, err := listArchivesIn(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "20260101-000000" || got[1].ID != "20260102-000000" {
		t.Errorf("got %+v, want two archives oldest first", got)
	}
	if got, err := listArchivesIn(filepath.Join(dir, "missing")); err != nil || got != nil {
		t.Errorf("missing dir = %v, %v; want nil, nil", got, err)
	}
}
