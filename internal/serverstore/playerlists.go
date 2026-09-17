package serverstore

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// WhitelistEntry, OpEntry, and BanEntry mirror the shapes Minecraft
// itself reads/writes in data/whitelist.json, data/ops.json, and
// data/banned-players.json, so mcm can edit those files directly while a
// server is stopped instead of going through env vars and a restart.
type WhitelistEntry struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type OpEntry struct {
	UUID                string `json:"uuid"`
	Name                string `json:"name"`
	Level               int    `json:"level"`
	BypassesPlayerLimit bool   `json:"bypassesPlayerLimit"`
}

type BanEntry struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Created string `json:"created"`
	Source  string `json:"source"`
	Expires string `json:"expires"`
	Reason  string `json:"reason"`
}

func whitelistPath(root, name string) string {
	return filepath.Join(DataDir(root, name), "whitelist.json")
}

func opsPath(root, name string) string {
	return filepath.Join(DataDir(root, name), "ops.json")
}

func bannedPlayersPath(root, name string) string {
	return filepath.Join(DataDir(root, name), "banned-players.json")
}

func LoadWhitelist(root, name string) ([]WhitelistEntry, error) {
	var entries []WhitelistEntry
	err := loadJSONList(whitelistPath(root, name), &entries)
	return entries, err
}

func SaveWhitelist(root, name string, entries []WhitelistEntry) error {
	return saveJSONList(whitelistPath(root, name), entries)
}

func LoadOps(root, name string) ([]OpEntry, error) {
	var entries []OpEntry
	err := loadJSONList(opsPath(root, name), &entries)
	return entries, err
}

func SaveOps(root, name string, entries []OpEntry) error {
	return saveJSONList(opsPath(root, name), entries)
}

func LoadBans(root, name string) ([]BanEntry, error) {
	var entries []BanEntry
	err := loadJSONList(bannedPlayersPath(root, name), &entries)
	return entries, err
}

func SaveBans(root, name string, entries []BanEntry) error {
	return saveJSONList(bannedPlayersPath(root, name), entries)
}

// loadJSONList decodes a JSON array file into out, treating a missing
// file as an empty list (a server that's never been started yet won't
// have any of these files).
func loadJSONList(path string, out any) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func saveJSONList(path string, list any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
