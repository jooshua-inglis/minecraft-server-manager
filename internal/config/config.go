// Package config loads mcm's global, manager-wide configuration.
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const defaultServersRoot = "mc-servers"

type Config struct {
	ServersRoot string `toml:"servers_root"`
	CFAPIKey    string `toml:"cf_api_key"`
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

	if cfg.ServersRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		cfg.ServersRoot = filepath.Join(home, defaultServersRoot)
	}

	return cfg, nil
}
