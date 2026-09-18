package archiveutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "top.txt"), []byte("top"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "nested.txt"), []byte("nested"), 0o644); err != nil {
		t.Fatal(err)
	}

	archive := filepath.Join(t.TempDir(), "out.tar.gz")
	if err := CreateTarGz(src, archive); err != nil {
		t.Fatalf("CreateTarGz: %v", err)
	}

	dest := t.TempDir()
	if err := ExtractTarGz(archive, dest); err != nil {
		t.Fatalf("ExtractTarGz: %v", err)
	}

	top, err := os.ReadFile(filepath.Join(dest, "top.txt"))
	if err != nil || string(top) != "top" {
		t.Errorf("top.txt = %q, %v", top, err)
	}
	nested, err := os.ReadFile(filepath.Join(dest, "sub", "nested.txt"))
	if err != nil || string(nested) != "nested" {
		t.Errorf("sub/nested.txt = %q, %v", nested, err)
	}
}

func TestExtractRejectsPathTraversal(t *testing.T) {
	if _, err := safeJoin("/data", "../../etc/passwd"); err == nil {
		t.Error("safeJoin allowed a path-traversal entry")
	}
	if _, err := safeJoin("/data", "sub/ok.txt"); err != nil {
		t.Errorf("safeJoin rejected a legitimate entry: %v", err)
	}
}
