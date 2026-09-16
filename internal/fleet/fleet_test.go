package fleet

import (
	"testing"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/serverstore"
)

func TestSelectPort(t *testing.T) {
	always := func(int) bool { return true }

	got, err := selectPort(map[int]bool{}, 25565, 10, always)
	if err != nil || got != 25565 {
		t.Fatalf("selectPort(empty used) = %d, %v; want 25565, nil", got, err)
	}

	used := map[int]bool{25565: true, 25566: true}
	got, err = selectPort(used, 25565, 10, always)
	if err != nil || got != 25567 {
		t.Fatalf("selectPort(skip used) = %d, %v; want 25567, nil", got, err)
	}

	onlyFree := func(p int) bool { return p == 25569 }
	got, err = selectPort(map[int]bool{}, 25565, 10, onlyFree)
	if err != nil || got != 25569 {
		t.Fatalf("selectPort(skip host-bound) = %d, %v; want 25569, nil", got, err)
	}

	never := func(int) bool { return false }
	if _, err := selectPort(map[int]bool{}, 25565, 5, never); err == nil {
		t.Fatalf("selectPort(none free) = nil error, want error")
	}
}

func TestEULAErrorMentionsFlag(t *testing.T) {
	err := EULAError{}
	if err.Error() == "" {
		t.Fatal("EULAError.Error() is empty")
	}
}

func TestTypeOrVersionChanging(t *testing.T) {
	meta := &serverstore.Metadata{Type: "VANILLA", Version: "LATEST"}
	opts := EditOptions{}
	if opts.TypeOrVersionChanging(meta) {
		t.Errorf("no-op edit reported as changing")
	}

	opts = EditOptions{Type: meta.Type}
	if opts.TypeOrVersionChanging(meta) {
		t.Errorf("same-value type reported as changing")
	}

	opts = EditOptions{Type: "PAPER"}
	if !opts.TypeOrVersionChanging(meta) {
		t.Errorf("different type not reported as changing")
	}

	opts = EditOptions{Version: "1.20.1"}
	if !opts.TypeOrVersionChanging(meta) {
		t.Errorf("different version not reported as changing")
	}
}
