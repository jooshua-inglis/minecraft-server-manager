package fleet

import (
	"context"
	"fmt"
	"strings"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/serverstore"
)

// pluginServerTypes are the itzg TYPE values that run Bukkit-API
// plugins rather than mod-loader mods, used to decide whether a raw
// "url" install belongs in PLUGINS or MODS.
var pluginServerTypes = map[string]bool{
	"PAPER":  true,
	"SPIGOT": true,
	"BUKKIT": true,
	"FOLIA":  true,
	"PURPUR": true,
}

func isPluginType(serverType string) bool {
	return pluginServerTypes[strings.ToUpper(serverType)]
}

type Mods struct {
	ModrinthProjects []string
	CurseForgeFiles  []string
	ModURLs          []string
	PluginURLs       []string
}

const (
	SourceModrinth   = "modrinth"
	SourceCurseForge = "curseforge"
	SourceURL        = "url"
)

func (f *Fleet) ModsAdd(ctx context.Context, name, source, ref string) error {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return err
	}

	switch source {
	case SourceModrinth:
		meta.ModrinthProjects = appendUnique(meta.ModrinthProjects, ref)
	case SourceCurseForge:
		meta.CurseForgeFiles = appendUnique(meta.CurseForgeFiles, ref)
	case SourceURL:
		if isPluginType(meta.Type) {
			meta.PluginURLs = appendUnique(meta.PluginURLs, ref)
		} else {
			meta.ModURLs = appendUnique(meta.ModURLs, ref)
		}
	default:
		return fmt.Errorf("unknown source %q (want %q, %q, or %q)", source, SourceModrinth, SourceCurseForge, SourceURL)
	}

	return f.saveAndRecreate(ctx, meta)
}

func (f *Fleet) ModsRemove(ctx context.Context, name, source, ref string) error {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return err
	}

	switch source {
	case SourceModrinth:
		meta.ModrinthProjects = removeString(meta.ModrinthProjects, ref)
	case SourceCurseForge:
		meta.CurseForgeFiles = removeString(meta.CurseForgeFiles, ref)
	case SourceURL:
		if isPluginType(meta.Type) {
			meta.PluginURLs = removeString(meta.PluginURLs, ref)
		} else {
			meta.ModURLs = removeString(meta.ModURLs, ref)
		}
	default:
		return fmt.Errorf("unknown source %q (want %q, %q, or %q)", source, SourceModrinth, SourceCurseForge, SourceURL)
	}

	return f.saveAndRecreate(ctx, meta)
}

func (f *Fleet) ModsList(ctx context.Context, name string) (Mods, error) {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return Mods{}, err
	}
	return Mods{
		ModrinthProjects: meta.ModrinthProjects,
		CurseForgeFiles:  meta.CurseForgeFiles,
		ModURLs:          meta.ModURLs,
		PluginURLs:       meta.PluginURLs,
	}, nil
}

func appendUnique(list []string, item string) []string {
	for _, existing := range list {
		if existing == item {
			return list
		}
	}
	return append(list, item)
}

func removeString(list []string, item string) []string {
	filtered := list[:0]
	for _, existing := range list {
		if existing != item {
			filtered = append(filtered, existing)
		}
	}
	return filtered
}
