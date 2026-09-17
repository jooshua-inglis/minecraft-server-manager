package fleet

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/mojang"
	"github.com/jooshua-inglis/minecraft-server-manager/internal/serverstore"
)

const defaultOpLevel = 4

// resolvePlayer turns a username into the UUID/canonical-name pair that
// belongs in whitelist.json/ops.json/banned-players.json. RCON resolves
// this itself server-side, so it's only needed for the offline,
// direct-file-edit path.
func resolvePlayer(ctx context.Context, name string, offline bool) (uuid, canonicalName string, err error) {
	if offline {
		return mojang.OfflineUUID(name), name, nil
	}
	uuid, canonicalName, err = mojang.LookupUUID(ctx, name)
	if err != nil {
		return "", "", fmt.Errorf("%w (pass --offline if this server runs with online-mode=false)", err)
	}
	return uuid, canonicalName, nil
}

func (f *Fleet) WhitelistAdd(ctx context.Context, name, player string, offline bool) error {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return err
	}

	running, err := f.isRunning(ctx, meta)
	if err != nil {
		return err
	}
	if running {
		_, err := f.Exec(ctx, name, "whitelist add "+player)
		return err
	}

	uuid, canonicalName, err := resolvePlayer(ctx, player, offline)
	if err != nil {
		return err
	}

	entries, err := serverstore.LoadWhitelist(f.Root, name)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if strings.EqualFold(e.Name, canonicalName) {
			return nil
		}
	}
	entries = append(entries, serverstore.WhitelistEntry{UUID: uuid, Name: canonicalName})
	return serverstore.SaveWhitelist(f.Root, name, entries)
}

func (f *Fleet) WhitelistRemove(ctx context.Context, name, player string) error {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return err
	}

	running, err := f.isRunning(ctx, meta)
	if err != nil {
		return err
	}
	if running {
		_, err := f.Exec(ctx, name, "whitelist remove "+player)
		return err
	}

	entries, err := serverstore.LoadWhitelist(f.Root, name)
	if err != nil {
		return err
	}
	filtered := entries[:0]
	for _, e := range entries {
		if !strings.EqualFold(e.Name, player) {
			filtered = append(filtered, e)
		}
	}
	return serverstore.SaveWhitelist(f.Root, name, filtered)
}

func (f *Fleet) WhitelistList(ctx context.Context, name string) ([]serverstore.WhitelistEntry, error) {
	if _, err := serverstore.Load(f.Root, name); err != nil {
		return nil, err
	}
	return serverstore.LoadWhitelist(f.Root, name)
}

func (f *Fleet) OpAdd(ctx context.Context, name, player string, offline bool) error {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return err
	}

	running, err := f.isRunning(ctx, meta)
	if err != nil {
		return err
	}
	if running {
		_, err := f.Exec(ctx, name, "op "+player)
		return err
	}

	uuid, canonicalName, err := resolvePlayer(ctx, player, offline)
	if err != nil {
		return err
	}

	entries, err := serverstore.LoadOps(f.Root, name)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if strings.EqualFold(e.Name, canonicalName) {
			return nil
		}
	}
	entries = append(entries, serverstore.OpEntry{
		UUID:                uuid,
		Name:                canonicalName,
		Level:               defaultOpLevel,
		BypassesPlayerLimit: false,
	})
	return serverstore.SaveOps(f.Root, name, entries)
}

func (f *Fleet) OpRemove(ctx context.Context, name, player string) error {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return err
	}

	running, err := f.isRunning(ctx, meta)
	if err != nil {
		return err
	}
	if running {
		_, err := f.Exec(ctx, name, "deop "+player)
		return err
	}

	entries, err := serverstore.LoadOps(f.Root, name)
	if err != nil {
		return err
	}
	filtered := entries[:0]
	for _, e := range entries {
		if !strings.EqualFold(e.Name, player) {
			filtered = append(filtered, e)
		}
	}
	return serverstore.SaveOps(f.Root, name, filtered)
}

func (f *Fleet) OpList(ctx context.Context, name string) ([]serverstore.OpEntry, error) {
	if _, err := serverstore.Load(f.Root, name); err != nil {
		return nil, err
	}
	return serverstore.LoadOps(f.Root, name)
}

const defaultBanReason = "Banned by an operator."

func (f *Fleet) BanAdd(ctx context.Context, name, player, reason string, offline bool) error {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return err
	}
	if reason == "" {
		reason = defaultBanReason
	}

	running, err := f.isRunning(ctx, meta)
	if err != nil {
		return err
	}
	if running {
		_, err := f.Exec(ctx, name, "ban "+player+" "+reason)
		return err
	}

	uuid, canonicalName, err := resolvePlayer(ctx, player, offline)
	if err != nil {
		return err
	}

	entries, err := serverstore.LoadBans(f.Root, name)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if strings.EqualFold(e.Name, canonicalName) {
			return nil
		}
	}
	entries = append(entries, serverstore.BanEntry{
		UUID:    uuid,
		Name:    canonicalName,
		Created: time.Now().UTC().Format("2006-01-02 15:04:05 -0700"),
		Source:  "Server",
		Expires: "forever",
		Reason:  reason,
	})
	return serverstore.SaveBans(f.Root, name, entries)
}

func (f *Fleet) BanRemove(ctx context.Context, name, player string) error {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return err
	}

	running, err := f.isRunning(ctx, meta)
	if err != nil {
		return err
	}
	if running {
		_, err := f.Exec(ctx, name, "pardon "+player)
		return err
	}

	entries, err := serverstore.LoadBans(f.Root, name)
	if err != nil {
		return err
	}
	filtered := entries[:0]
	for _, e := range entries {
		if !strings.EqualFold(e.Name, player) {
			filtered = append(filtered, e)
		}
	}
	return serverstore.SaveBans(f.Root, name, filtered)
}

func (f *Fleet) BanList(ctx context.Context, name string) ([]serverstore.BanEntry, error) {
	if _, err := serverstore.Load(f.Root, name); err != nil {
		return nil, err
	}
	return serverstore.LoadBans(f.Root, name)
}
