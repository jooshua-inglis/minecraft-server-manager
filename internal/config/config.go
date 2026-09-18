// Package config loads mcm's global, manager-wide configuration.
package config

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const defaultServersRoot = "mc-servers"

// Environment variable overrides, applied on top of config.toml — the
// simplest way to configure mcm when it's running containerized
// (docker run -e ...) without baking a config file into an image or
// mounting one in.
const (
	envServersRoot = "MCM_SERVERS_ROOT"
	envCFAPIKey    = "MCM_CF_API_KEY"
)

type Config struct {
	ServersRoot string `toml:"servers_root"`
	CFAPIKey    string `toml:"cf_api_key"`
	// WebToken authenticates write requests to `mcm web`'s API (M14 /
	// RESEARCH.md §7.5). Generated on first use by `mcm web` and
	// persisted here so it survives restarts.
	WebToken string `toml:"web_token,omitempty"`
}

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mcm", "config.toml"), nil
}

func Load() (*Config, error) {
	cfg := &Config{}

	path, err := Path()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); err == nil {
		if _, err := toml.DecodeFile(path, cfg); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if v := os.Getenv(envServersRoot); v != "" {
		cfg.ServersRoot = v
	}
	if v := os.Getenv(envCFAPIKey); v != "" {
		cfg.CFAPIKey = v
	}

	if cfg.ServersRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		cfg.ServersRoot = filepath.Join(home, defaultServersRoot)
	}

	return cfg, nil
}

// Save persists cfg to disk, creating its parent directory if needed.
func Save(cfg *Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o600)
}
