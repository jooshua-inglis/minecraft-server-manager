# Architecture

`mcm` manages Minecraft servers that run as Docker containers
(`itzg/minecraft-server`). It has **three front ends over one core**:

```
  cliapp (cobra)     tui (bubbletea)      webapi (net/http) + webui (SvelteKit SPA)
  `mcm <cmd>`        `mcm attach`         `mcm web`
        \                 |                        /
         +----------->  fleet  <-----------------+      <- ALL business logic
                          |
      +---------+---------+----------+----------+----------+
   dockerctl  serverstore   rcon   archiveutil  mojang   modpackapi
   (Docker    (manager.json (RCON   (tar.gz)    (UUIDs)  (Modrinth/
    SDK)       + player      client)                      CurseForge search)
               lists)
   config: global settings (config.toml + env), used by cliapp to build a Fleet
```

## The one rule

**No business logic in a front end.** `cliapp` handlers translate flags into a
`fleet` call and print the result; `webapi` handlers translate JSON into the
same call; `tui` composes the same calls. If you're adding behavior, put it in
`internal/fleet` and call it from every front end that needs it. This is what
keeps "the CLI and the web UI see identical state" true.

`fleet.Fleet` is `{Root, Docker, CFAPIKey}`. `cliapp.newFleet()` builds one from
config; every command closes `f.Docker` when done.

## Package map

| Package | Role |
|---|---|
| `cmd/mcm` | `main`: calls `cliapp.Execute()` |
| `internal/cliapp` | Cobra commands. One file per command group. `root.go` registers them |
| `internal/fleet` | The core. `fleet.go` (create/start/stop/destroy/list/status/rename/edit/exec/logs/ports/crash), `top.go` (stats), `players.go` (whitelist/op/ban), `mods.go`, `modpack.go`, `backup.go`, `world.go` |
| `internal/dockerctl` | Thin wrapper over the Docker Engine SDK: create/start/stop/inspect/logs/stats/list, restart policy, `UsedHostPorts` |
| `internal/serverstore` | `manager.json` (`Metadata`) and whitelist/ops/bans JSON files; `ServerDir`/`DataDir`; name validation |
| `internal/rcon` | Minimal RCON client (own implementation, unit-tested against a fake server) |
| `internal/archiveutil` | `CreateTarGz` / `ExtractTarGz` (with zip-slip guard) — used by backups and worlds |
| `internal/mojang` | Username→UUID lookup and the offline-mode UUID algorithm |
| `internal/modpackapi` | Modrinth/CurseForge modpack search |
| `internal/config` | `config.toml` + env overrides; `Save` used to persist the web token |
| `internal/webapi` | HTTP API (`webapi.go` routes + read handlers + SSE, `write.go` write handlers) |
| `internal/webui` | `go:embed` of `dist/` (the built SvelteKit app) + SPA-fallback file server |
| `internal/tui` | `mcm attach` bubbletea model |
| `web/` | SvelteKit source (Deno toolchain). Builds into `internal/webui/dist` |

## State model

- **Docker is the runtime source of truth.** Whether a server is running,
  crashed, etc. is always queried live (`Inspect`); there is no mirrored
  "running" flag to go stale.
- **`manager.json` is mcm's own record**, one per server, at
  `<servers_root>/<name>/manager.json`, *next to* (not inside) `<name>/data/`
  (the directory bind-mounted into the container as `/data`). It holds what
  Docker can't: requested type/version/memory, ports, RCON password, mod and
  modpack lists, `LevelName`. Legacy files missing newer fields must keep
  working (e.g. `Metadata.Level()` defaults to `"world"`).
- Container name is `mcm-<name>`; containers carry labels `mcm.managed=true` and
  `mcm.server=<name>`.
- Anything "env-shaped" (type, version, mods, modpack) is applied via
  `saveAndRecreate`: remove the container, save metadata, create a new one,
  restart it if it was running. Host data survives because it's a bind mount.
- Servers root (`config.ServersRoot`): `MCM_SERVERS_ROOT` env > `servers_root` in
  `config.toml` > `$XDG_STATE_HOME/mcm/servers` (`~/.local/state/mcm/servers`),
  except a pre-existing `~/mc-servers` is still used until the new default dir
  exists (so older installs don't lose their servers).
- Global config: `config.toml` in the OS config dir (`~/.config/mcm/`). Holds
  `servers_root`, `cf_api_key`, and the generated `web_token`. `mcm web start`
  also keeps `web.pid.json` and `web.log` there.

## Behaviors worth knowing before you touch them

- **Ports** (`fleet.Create`): host game port from 25565, RCON from 25575. A port
  is "used" if Docker reports it **or** any server's `manager.json` records it
  (`usedPorts`/`recordedPorts`) — Docker doesn't report a port for a container
  that has never run, so Docker alone lets two fresh servers collide.
  `isPortFree` additionally rejects ports something on the host is listening on.
- **Exposure:** game port published on `0.0.0.0`; RCON on `127.0.0.1` only. RCON
  password is generated per server.
- **Crash handling:** restart policy `on-failure` with a cap
  (`maxRestartRetries = 6`) so Docker's own backoff applies but a bad
  config/mod can't restart-loop forever. `fleet.Status` fills `Crash`
  (`CrashInfo`: restart count, exit code, OOM, gave-up flag, last 20 log lines)
  whenever the container isn't running and is restarting or exited non-zero.
- **Whitelist/op/ban:** if the server is running they go over RCON; if stopped,
  the JSON files are edited directly (`--offline` uses the offline-mode UUID
  instead of a Mojang lookup).
- **Backups** (`fleet.Backup`): whole `data/` dir → `<server>/backups/<UTC
  timestamp>.tar.gz`. If running: RCON `save-off`, `save-all flush`, sleep 2s,
  archive, `save-on` (deferred). **Restore requires the server stopped.**
- **Worlds** (`fleet/world.go`): scoped to `data/<level>/` →
  `<server>/world-backups/`. Export/import are portable tar.gz between servers.
  All world operations require the server stopped. The container gets `LEVEL=`
  from `Metadata.Level()` so mcm and the game agree which directory is the world.
- **Restore/import swap directory *contents*, never the directory** — see
  `replaceDirFromArchive` and gotchas.md (WSL2 bind-mount inode).

## Web API (`internal/webapi`)

- Reads (`GET`) are open; **every state-changing route is wrapped in
  `requireAuth`** (bearer token, constant-time compare). The token lives in
  `config.toml` (`web_token`), is generated on first `mcm web`, and is printed at
  startup. Adding a write route? Wrap it in `h.requireAuth`.
- Errors: read handlers use `writeError` (404 for `serverstore.ErrNotFound`, else
  500); write handlers use `writeActionError` (404 / else 400, since nearly all
  write failures are caller-caused: bad name, EULA, wrong state).
- Streaming is **SSE, not WebSockets**: `logs?follow=true` and
  `stats/stream`. `sseLineWriter` adapts `fleet.Logs`' byte stream to one event
  per line. Both validate the server exists *before* writing SSE headers so bad
  names still get a real 404.
- JSON field names are snake_case via struct tags on the `fleet`/`serverstore`
  types that get serialized. New serialized types need tags too.
- `mcm web` mounts the API at `/api/` and the embedded SPA at `/`. Bind address
  defaults to `127.0.0.1:8080`; a non-loopback bind prints a warning (reads are
  unauthenticated).

## Web UI (`web/`)

SvelteKit built as a **static SPA** (`adapter-static`, `ssr = false`, `fallback:
index.html`), output to `internal/webui/dist` and embedded in the binary. No JS
runtime at runtime. All data is client-side `fetch`/`EventSource` against
`/api/*` (`src/lib/api.ts`). The write token is stored in `localStorage`
(`src/lib/auth.svelte.ts`) and sent as `Authorization: Bearer`. **`dist/` is
committed** so `go build` (and the Dockerfile, CI) need no JS toolchain — which
means you must rebuild and commit it whenever `web/` changes.

## TUI (`internal/tui`)

`mcm attach <name>`: bubbletea model with a log viewport (fed by a goroutine
running `fleet.Logs(follow)` through a line-splitting `io.Writer` into a
channel), a stats header (polls `fleet.SingleStats` every 2s), and an RCON input
line (`fleet.Exec` in a `tea.Cmd`). Ctrl+C quits (only Ctrl+C — `q` must stay
typeable). The session context is cancelled on exit so the log goroutine stops.
Layout math has to subtract every border/header row or the top of the screen is
clipped (this already happened once).

## Deviations from the plan (`plan/`)

- **M10 backups:** built into the CLI (RCON coordination + tar.gz), not the
  `itzg/mc-backup` sidecar.
- **M13:** frontend is embedded in the Go binary; toolchain is **Deno**, not
  Node/npm (maintainer's choice).
- **M12/M13 realtime:** SSE rather than WebSockets.
- **M14 auth:** one shared bearer token, not per-user auth.
- **M15:** the real work was fixing the port-collision bug; `mc-router` skipped.
- **M16/M17:** the plan docs landed in git long before the code did (commit
  messages said "Adds mcm attach…" but the diffs were docs only). Code was added
  later. **Don't trust commit messages for "is X implemented" — read the diff.**
- **Beyond the plan:** `mcm web start|stop|status`, the Docker image, release
  binaries + `install.sh`, the XDG state default, env overrides.
