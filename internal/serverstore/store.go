// Package serverstore manages each server's on-disk directory and its
// manager.json metadata file, which lives next to (not inside) the
// server's data directory that gets bind-mounted into its container.
package serverstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

const metadataFile = "manager.json"

var ErrNotFound = errors.New("server not found")
var ErrAlreadyExists = errors.New("server already exists")

var nameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)

func ValidateName(name string) error {
	if !nameRE.MatchString(name) {
		return fmt.Errorf("invalid server name %q: must be lowercase alphanumeric/hyphen, 1-32 chars, starting with a letter or digit", name)
	}
	return nil
}

// Metadata is the CLI-owned, per-server record that Docker itself can't
// hold: what was requested at create time, and identifiers needed to
// recreate the container against the same data directory.
type Metadata struct {
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Version       string    `json:"version"`
	Memory        string    `json:"memory"`
	Port          int       `json:"port"`
	RCONPort      int       `json:"rcon_port"`
	ContainerName string    `json:"container_name"`
	RCONPassword  string    `json:"rcon_password"`
	CreatedAt     time.Time `json:"created_at"`

	// Individual mod/plugin installs (modpacks are a separate, later
	// milestone). These map directly onto the itzg image's env vars of
	// the same shape: MODRINTH_PROJECTS, CURSEFORGE_FILES, MODS, PLUGINS.
	ModrinthProjects []string `json:"modrinth_projects,omitempty"`
	CurseForgeFiles  []string `json:"curseforge_files,omitempty"`
	ModURLs          []string `json:"mod_urls,omitempty"`
	PluginURLs       []string `json:"plugin_urls,omitempty"`

	// Modpack install. At most one of these is set at a time; installing
	// a modpack overrides Type to the matching itzg launcher type
	// (MODRINTH or AUTO_CURSEFORGE).
	ModpackSource string `json:"modpack_source,omitempty"`
	ModpackRef    string `json:"modpack_ref,omitempty"`

	// LevelName mirrors the itzg image's LEVEL env var: which
	// subdirectory of data/ is the world save, so `mcm world ...` (M17)
	// knows exactly what to snapshot/reset/export without touching the
	// rest of data/ (configs, mods, plugins, whitelist/ops/bans, logs).
	LevelName string `json:"level_name,omitempty"`
}

const defaultLevelName = "world"

// Level returns m's world save directory name, defaulting to "world"
// for metadata saved before LevelName existed.
func (m *Metadata) Level() string {
	if m.LevelName == "" {
		return defaultLevelName
	}
	return m.LevelName
}

func ServerDir(root, name string) string {
	return filepath.Join(root, name)
}

func DataDir(root, name string) string {
	return filepath.Join(ServerDir(root, name), "data")
}

func metadataPath(root, name string) string {
	return filepath.Join(ServerDir(root, name), metadataFile)
}

func Exists(root, name string) bool {
	_, err := os.Stat(metadataPath(root, name))
	return err == nil
}

func Load(root, name string) (*Metadata, error) {
	data, err := os.ReadFile(metadataPath(root, name))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func Save(root string, meta *Metadata) error {
	if err := os.MkdirAll(ServerDir(root, meta.Name), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metadataPath(root, meta.Name), data, 0o600)
}

// List returns the names of every server that has a manager.json under
// root, sorted by directory read order (not guaranteed alphabetical).
func List(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if Exists(root, e.Name()) {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// Remove deletes the server's directory entirely, including its data.
// Callers should confirm with the operator before calling this.
func Remove(root, name string) error {
	return os.RemoveAll(ServerDir(root, name))
}

// Rename moves a server's directory (data + manager.json) to a new name,
// updating the metadata's Name/ContainerName fields in the process.
func Rename(root string, meta *Metadata, newName string) error {
	oldDir := ServerDir(root, meta.Name)
	newDir := ServerDir(root, newName)
	if err := os.Rename(oldDir, newDir); err != nil {
		return err
	}
	meta.Name = newName
	meta.ContainerName = ContainerName(newName)
	return Save(root, meta)
}

func ContainerName(name string) string {
	return "mcm-" + name
}
