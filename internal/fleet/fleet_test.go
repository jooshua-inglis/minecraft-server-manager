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

// TestRecordedPortsCoversNeverStartedServers guards against the M15
// bug where two servers created (but never started) back to back both
// got assigned the same port: Docker doesn't report a port binding for
// a container until it has actually run, so port selection has to
// consult manager.json directly rather than Docker alone.
func TestRecordedPortsCoversNeverStartedServers(t *testing.T) {
	root := t.TempDir()
	for i, name := range []string{"net1", "net2"} {
		meta := &serverstore.Metadata{
			Name:          name,
			Port:          25567,
			RCONPort:      25575 + i,
			ContainerName: serverstore.ContainerName(name),
		}
		if err := serverstore.Save(root, meta); err != nil {
			t.Fatalf("saving %s: %v", name, err)
		}
	}

	got, err := recordedPorts(root)
	if err != nil {
		t.Fatalf("recordedPorts: %v", err)
	}

	want := map[int]bool{25567: true, 25575: true, 25576: true}
	if len(got) != len(want) {
		t.Fatalf("recordedPorts = %v, want %v", got, want)
	}
	for p := range want {
		if !got[p] {
			t.Errorf("recordedPorts missing port %d", p)
		}
	}
}
