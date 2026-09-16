package serverstore

import (
	"path/filepath"
	"testing"
	"time"
)

func TestValidateName(t *testing.T) {
	valid := []string{"a", "myworld", "my-world-2"}
	for _, n := range valid {
		if err := ValidateName(n); err != nil {
			t.Errorf("ValidateName(%q) = %v, want nil", n, err)
		}
	}

	invalid := []string{"", "My-World", "-leading", "has_underscore", "has space", string(make([]byte, 33))}
	for _, n := range invalid {
		if err := ValidateName(n); err == nil {
			t.Errorf("ValidateName(%q) = nil, want error", n)
		}
	}
}

func TestSaveLoad(t *testing.T) {
	root := t.TempDir()
	meta := &Metadata{
		Name:          "myworld",
		Type:          "PAPER",
		Version:       "1.21.1",
		Memory:        "2G",
		Port:          25565,
		ContainerName: ContainerName("myworld"),
		RCONPassword:  "secret",
		CreatedAt:     time.Now().UTC().Truncate(time.Second),
	}

	if err := Save(root, meta); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !Exists(root, "myworld") {
		t.Fatalf("Exists = false after Save")
	}

	got, err := Load(root, "myworld")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if *got != *meta {
		t.Errorf("Load = %+v, want %+v", got, meta)
	}

	if _, err := Load(root, "nope"); err != ErrNotFound {
		t.Errorf("Load(missing) = %v, want ErrNotFound", err)
	}
}

func TestList(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"alpha", "beta"} {
		if err := Save(root, &Metadata{Name: name}); err != nil {
			t.Fatalf("Save(%s): %v", name, err)
		}
	}

	names, err := List(root)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("List = %v, want 2 entries", names)
	}
}

func TestRename(t *testing.T) {
	root := t.TempDir()
	meta := &Metadata{Name: "old", ContainerName: ContainerName("old")}
	if err := Save(root, meta); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := Rename(root, meta, "new"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if Exists(root, "old") {
		t.Errorf("old server dir still exists after rename")
	}
	if !Exists(root, "new") {
		t.Errorf("new server dir missing after rename")
	}
	if meta.Name != "new" || meta.ContainerName != "mcm-new" {
		t.Errorf("metadata not updated in place: %+v", meta)
	}

	got, err := Load(root, "new")
	if err != nil {
		t.Fatalf("Load(new): %v", err)
	}
	if got.Name != "new" {
		t.Errorf("persisted Name = %q, want %q", got.Name, "new")
	}
}

func TestDataDirUnderServerDir(t *testing.T) {
	root := "/srv/mc-servers"
	got := DataDir(root, "myworld")
	want := filepath.Join(root, "myworld", "data")
	if got != want {
		t.Errorf("DataDir = %q, want %q", got, want)
	}
}
