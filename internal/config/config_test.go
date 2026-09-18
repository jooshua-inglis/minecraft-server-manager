package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvOverrides(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(envServersRoot, "/srv/mc")
	t.Setenv(envCFAPIKey, "key123")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServersRoot != "/srv/mc" || cfg.CFAPIKey != "key123" {
		t.Errorf("got root=%q key=%q, want env values", cfg.ServersRoot, cfg.CFAPIKey)
	}
}

func TestDefaultServersRoot(t *testing.T) {
	setup := func(t *testing.T) (home string) {
		home = t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
		t.Setenv("XDG_STATE_HOME", "")
		t.Setenv(envServersRoot, "")
		return home
	}
	load := func(t *testing.T) string {
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		return cfg.ServersRoot
	}

	t.Run("fresh install uses the state dir", func(t *testing.T) {
		home := setup(t)
		want := filepath.Join(home, ".local", "state", "mcm", "servers")
		if got := load(t); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("honors XDG_STATE_HOME", func(t *testing.T) {
		setup(t)
		state := t.TempDir()
		t.Setenv("XDG_STATE_HOME", state)
		want := filepath.Join(state, "mcm", "servers")
		if got := load(t); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("keeps using the legacy dir until the new one exists", func(t *testing.T) {
		home := setup(t)
		legacy := filepath.Join(home, "mc-servers")
		os.MkdirAll(legacy, 0o755)
		if got := load(t); got != legacy {
			t.Errorf("got %q, want legacy %q", got, legacy)
		}

		newRoot := filepath.Join(home, ".local", "state", "mcm", "servers")
		os.MkdirAll(newRoot, 0o755)
		if got := load(t); got != newRoot {
			t.Errorf("got %q, want new %q once it exists", got, newRoot)
		}
	})
}
