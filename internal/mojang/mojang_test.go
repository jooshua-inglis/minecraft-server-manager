package mojang

import (
	"regexp"
	"testing"
)

var uuidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-3[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestOfflineUUIDIsDeterministicAndWellFormed(t *testing.T) {
	got := OfflineUUID("Notch")
	if !uuidRE.MatchString(got) {
		t.Fatalf("OfflineUUID(%q) = %q, not a well-formed v3 UUID", "Notch", got)
	}
	if again := OfflineUUID("Notch"); again != got {
		t.Errorf("OfflineUUID not deterministic: %q != %q", got, again)
	}
	if other := OfflineUUID("Jeb_"); other == got {
		t.Errorf("OfflineUUID(%q) and OfflineUUID(%q) collided: %q", "Notch", "Jeb_", got)
	}
}

func TestDashUUID(t *testing.T) {
	got := dashUUID("0123456789abcdef0123456789abcdef")
	want := "01234567-89ab-cdef-0123-456789abcdef"
	if got != want {
		t.Errorf("dashUUID = %q, want %q", got, want)
	}
}
