package fleet

import (
	"context"
	"fmt"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/modpackapi"
	"github.com/jooshua-inglis/minecraft-server-manager/internal/serverstore"
)

const (
	ModpackSourceModrinth   = "modrinth"
	ModpackSourceCurseForge = "curseforge"
)

// ModpackInstall points name at a Modrinth or CurseForge modpack,
// switching its TYPE to the matching itzg launcher (MODRINTH or
// AUTO_CURSEFORGE) so the pack is resolved and installed on next start,
// then recreates the container so the change takes effect. ref is a
// Modrinth project slug/ID/URL, or a CurseForge slug/page URL.
func (f *Fleet) ModpackInstall(ctx context.Context, name, source, ref string) error {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return err
	}

	switch source {
	case ModpackSourceModrinth:
		meta.Type = "MODRINTH"
	case ModpackSourceCurseForge:
		if f.CFAPIKey == "" {
			return fmt.Errorf("installing a CurseForge modpack requires cf_api_key to be set in mcm's config")
		}
		meta.Type = "AUTO_CURSEFORGE"
	default:
		return fmt.Errorf("unknown source %q (want %q or %q)", source, ModpackSourceModrinth, ModpackSourceCurseForge)
	}

	meta.ModpackSource = source
	meta.ModpackRef = ref

	return f.saveAndRecreate(ctx, meta)
}

// ModpackRemove clears name's modpack and reverts it to a plain VANILLA
// server, recreating the container so the change takes effect. Follow up
// with `mcm edit` to pick a different type if VANILLA isn't wanted.
func (f *Fleet) ModpackRemove(ctx context.Context, name string) error {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return err
	}
	if meta.ModpackRef == "" {
		return fmt.Errorf("server %q has no modpack installed", name)
	}

	meta.ModpackSource = ""
	meta.ModpackRef = ""
	meta.Type = "VANILLA"

	return f.saveAndRecreate(ctx, meta)
}

// ModpackSearch looks up candidate refs for `mcm modpack install` by
// querying the given source's search API.
func (f *Fleet) ModpackSearch(ctx context.Context, source, query string, limit int) ([]modpackapi.Result, error) {
	switch source {
	case ModpackSourceModrinth:
		return modpackapi.SearchModrinth(ctx, query, limit)
	case ModpackSourceCurseForge:
		return modpackapi.SearchCurseForge(ctx, f.CFAPIKey, query, limit)
	default:
		return nil, fmt.Errorf("unknown source %q (want %q or %q)", source, ModpackSourceModrinth, ModpackSourceCurseForge)
	}
}

type ModpackStatus struct {
	Source string
	Ref    string
	Type   string
}

func (f *Fleet) ModpackStatus(ctx context.Context, name string) (*ModpackStatus, error) {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return nil, err
	}
	return &ModpackStatus{Source: meta.ModpackSource, Ref: meta.ModpackRef, Type: meta.Type}, nil
}
