package config

import "testing"

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
