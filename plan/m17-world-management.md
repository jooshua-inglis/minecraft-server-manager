# M17 — World management (independent of server config)

[← Plan overview](README.md) · Depends on: [M1 — Create/start/stop/destroy](m01-create-start-stop-destroy.md)

Complements [M10 — Backups](m10-backups.md) rather than replacing it: M10
snapshots a server's *entire* `data/` directory (mods, plugins, configs,
whitelist, and the world together). M17 adds operations scoped to just the
world save itself, so you can back up, reset, export, or swap a world
without touching — or needing to know anything about — the server's
type/version/mods configuration.

## Build

- Add `LevelName` to `manager.json` (mirrors the itzg image's `LEVEL` env
  var, default `world`), so the CLI knows exactly which subdirectory of
  `data/` is "the world" — everything else in `data/` (configs, mods,
  plugins, whitelist/ops/ban lists, logs) is left alone by every command
  below.
- `mcm world backup <name>` / `mcm world restore <name> <backup-id>` —
  snapshot or restore just `data/<level>/`.
- `mcm world reset <name>` — delete the current world directory so the
  server generates a brand new one on next start, without touching
  type/version/mods/whitelist.
- `mcm world export <name> <output.tar.gz>` / `mcm world import <name>
  <input.tar.gz>` — pull a world out as a portable archive, or drop one in
  (a downloaded map, a singleplayer world, or an export from a *different*
  mcm server, possibly running different server software entirely),
  replacing whatever's currently active.
- v0 requires the target server to be stopped for all of these — no live
  RCON save-coordination yet, which keeps the first cut simple. Once
  [M4](m04-console-rcon.md) and [M10](m10-backups.md) exist, this can be
  revisited to allow world operations against a running server the same
  way M10 does (RCON `save-off`/flush/copy/`save-on`).

## Try it

- Create and start a server, let it generate a world, then `mcm world
  export myworld ~/backups/myworld-day1.tar.gz`.
- `mcm world reset myworld` and restart it — confirm you get a fresh map
  while mods, config, and whitelist are untouched.
- `mcm world import myworld ~/backups/myworld-day1.tar.gz` and restart —
  confirm your original world is back exactly as exported.
- Import that same export into a *different* server (different type or
  version even) and confirm it works — proving world management really is
  independent of any one server's configuration.
