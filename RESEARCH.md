# Research: CLI-Based Minecraft Server Manager

Status: draft / living document
Last updated: 2026-09-16

## 1. Project vision (as understood)

A single CLI tool that is the *entire* interface for running a fleet of Minecraft
servers on one host. No web panel required (though one could be layered on later).
Each server:

- Runs in its own Docker container, image family:
  [`itzg/docker-minecraft-server`](https://github.com/itzg/docker-minecraft-server)
  (Java Edition) — Bedrock support via
  [`itzg/docker-minecraft-bedrock-server`](https://github.com/itzg/docker-minecraft-bedrock-server)
  could be a stretch goal.
- Has its own named volume/bind-mount on the host so world data, configs, mods,
  and plugins survive container recreation and are easy to back up / inspect
  directly with normal filesystem tools.
- Is fully controllable from the CLI: create/start/stop/destroy, update server
  version or server software (Vanilla/Paper/Spigot/Forge/Fabric/...), manage
  users (ops, whitelist, bans), run arbitrary console/RCON commands, monitor
  CPU/RAM/TPS/player count, and manage mods/plugins — including bulk modpack
  installs from Modrinth and CurseForge, and one-off individual mod/plugin
  installs.

This puts the project in the same space as Pterodactyl, Crafty Controller,
PufferPanel, MCSManager, AMP, and LinuxGSM, but deliberately narrower
(Minecraft-only, single-host, Docker-only) and opinionated (delegate
all the hard "download the right server jar / mod loader / modpack" work to
the itzg image rather than reimplementing it).

The CLI is the primary interface, but a **web UI is a first-class secondary
interface**, not a bolt-on: running `<minecraft-server-manager> web` starts a
local web server exposing the same management surface (create/start/stop
servers, users/whitelist, console/RCON, mods/plugins/modpacks, monitoring)
through a browser instead of a terminal. Both interfaces must be driven by
the same underlying logic and see the same fleet state — the web mode is not
a separate product, it's another front end bolted onto one core.

## 2. Survey of existing Minecraft server managers

| Tool | Interface | Containerization | Notes |
|---|---|---|---|
| **Pterodactyl Panel** | Web UI + API, "Wings" daemon | Docker-native, one container per server | Full RBAC, subusers, API keys, multi-node clustering. Strong at scale / multi-tenant hosting businesses. Heavier install (panel + database + Wings daemon + SSL). Uses "eggs" (JSON templates) to define how a game server is built and started. |
| **Crafty Controller** | Web UI (+ PWA) | Optional Docker, also runs bare-metal via bundled Java | Minecraft-focused: Vanilla/Paper/Forge/modded, multi-Java-version support, per-server scheduling (backups, restarts, commands), CPU/RAM/player graphs, built-in file manager, RBAC-lite (multiple admin accounts w/ permissions). Closest in *scope* to this project, but web-first, not CLI-first. |
| **PufferPanel** | Web UI + API, Go backend | Docker or native | Lightweight, template-driven ("packages"), fast to install. Less Minecraft-specific tooling (no built-in modpack installers) than Crafty. |
| **MCSManager** | Web UI, distributed daemon/panel split | Optional Docker | Multi-node, multi-user, i18n'd (popular in CN community). Similar daemon/panel split to Pterodactyl but lighter weight. |
| **AMP (Application Management Panel)** | Web UI, desktop-app-like | Native processes (Docker optional in newer versions) | Commercial (free tier limited), supports dozens of game types beyond Minecraft, polished UX, plugin/module system for the panel itself. |
| **LinuxGSM** | Pure CLI (bash scripts) | None (native processes, systemd) | Not Docker-based and not Minecraft-specific, but is the closest *existing* project in spirit: "the command-line tool for quick, simple deployment and management of Linux dedicated game servers." Good source of CLI UX ideas (config files per instance, `./server details`, `./server monitor`, systemd integration) even though its execution model (bare metal, its own downloader logic) differs from what we want.

### Feature checklist distilled from the above

Nearly every serious tool converges on the same feature set, which is a good
signal for our own MVP → v1 scope:

- Create/start/stop/restart/delete a server instance
- Console access + arbitrary command execution (send to stdin or via RCON)
- Player management: ops, whitelist, bans, kick
- Scheduled and on-demand backups, with pre/post hooks (flush + pause world
  saving before copying files)
- Resource monitoring: CPU/RAM per instance, disk usage, and Minecraft-level
  metrics (TPS, player count, world size)
- Version/server-type management: switch Vanilla ⇄ Paper ⇄ Forge ⇄ Fabric,
  upgrade/downgrade Minecraft version
- Mod/plugin management: install, update, remove, list installed, detect
  incompatible versions
- Modpack installs from CurseForge and Modrinth, including keeping them
  updated
- Multi-server / multi-instance awareness (list all servers, per-server
  status at a glance)
- Config file editing (server.properties and friends) without hand-editing
  raw files inside a container
- Log access / tailing
- Auto-restart on crash, with crash-loop detection so it doesn't restart
  forever into a broken state
- Permission/user model (only matters once this is more than single-operator)

Things Pterodactyl/AMP have that we can consciously **not** build (out of
scope for a single-operator tool): multi-node clusters, OAuth/subuser
accounts, billing/quota systems, generic non-Minecraft game support. A
basic web-based file manager and console, however, are in scope, since
`web` is meant to be a genuine alternative to the CLI, not just a status
dashboard.

One architectural pattern worth borrowing directly: Pterodactyl and
MCSManager both split into a "panel" (UI) and a "daemon"/"wings" (the thing
that actually talks to Docker), communicating over a local API. Our `web`
subcommand can follow the same shape *without* needing a second process —
the CLI binary itself starts an embedded HTTP server that calls straight
into the same core management library the CLI commands use, rather than
shelling out to itself or requiring a separately-run daemon.

## 3. Common challenges in Minecraft server operations

These recur across every hosting panel's docs/issue trackers and community
troubleshooting guides, and should directly inform design decisions:

1. **JVM memory tuning is not optional.** Under-provisioned heap or default
   GC settings cause slow memory-leak-like degradation (works fine, then
   crashes hours later). The community-standard fix is Aikar's flags
   (G1GC tuning) plus setting `-Xms`/`-Xmx` equal to avoid heap resizing
   pauses. The itzg image supports this out of the box (`MEMORY`,
   `USE_AIKAR_FLAGS`), so our CLI should expose/default to it rather than
   asking users to hand-roll `JVM_OPTS`.

2. **Plugin/mod ⇄ server-version compatibility lag.** New Minecraft versions
   ship before Spigot/Paper/Forge/Fabric and third-party plugins/mods catch
   up. A naive "always update to latest" feature will break servers. The CLI
   should support pinning versions, and warn (not silently fail) when
   upgrading a server whose installed plugins/mods haven't confirmed support
   for the target version.

3. **Backup correctness under a live world.** A hard crash or a backup taken
   mid-autosave can leave region files half-written / corrupt. Correct
   practice is: RCON `save-off` → `save-all flush` → copy/snapshot data →
   `save-on`, ideally while the world isn't being written to. This is
   exactly what `itzg/docker-mc-backup` automates as a sidecar; we should
   reuse that pattern (RCON-coordinated backups) rather than a naive
   `tar` of a live world directory.

4. **Crash-loop / bad-config death spirals.** A blind "always restart"
   policy can hide a config or mod error behind an infinite restart loop
   that never surfaces the real problem to the operator. Needs restart
   backoff + surfacing the last crash reason in `status`/logs.

5. **RCON/query port exposure.** RCON has a plaintext-ish, weak protocol
   and is a common attack surface if exposed publicly. Should default to
   binding RCON only on the Docker network / localhost, not published to
   the host's public interface, and require a generated (not
   user-typed-weak) password by default.

6. **CurseForge/Modrinth API friction.** CurseForge now requires a personal
   API key for some install paths (the itzg image ships a bundled key for
   Java 17+ images but users may need their own for rate-limit or ToS
   reasons); Modrinth's API is keyless but rate-limited. The CLI needs a
   place to store an optional `CF_API_KEY` per-user/config, and should
   handle rate-limit/HTTP errors from both APIs gracefully (retry/backoff,
   clear error messages) rather than surfacing raw HTTP failures.

7. **EULA acceptance.** Mojang's EULA must be explicitly accepted
   (`eula=true`) before a server will run at all — itzg's image supports
   this via `EULA=TRUE` env var, but the CLI's `create` flow should make
   this an explicit, informed step (show the EULA URL, require
   confirmation) rather than silently setting it.

8. **Host ⇄ container UID/permission mismatches.** Bind-mounting host
   directories into a container run as a non-root user (itzg's image runs
   as UID 1000 by default) can produce files the host user can't
   read/write, or vice versa. The CLI should either standardize on the
   image's default UID/GID for created data directories or explicitly pass
   `UID`/`GID` env vars to match the host operator's user.

9. **Port collisions across many servers on one host.** Each server needs a
   unique game port (and RCON/query port if exposed); a fleet manager needs
   to track port allocation itself rather than relying on the user to avoid
   collisions.

10. **"Which files are safe to touch by hand" ambiguity.** Since world data
    lives on the host (per the project's design goal), operators may edit
    files directly while the container runs, causing desync or corruption.
    Docs/CLI warnings should clarify that most edits are safe only while the
    server is stopped, and mutation of live worlds should go through
    RCON/console commands where possible.

## 4. `itzg/docker-minecraft-server` deep dive

This image is effectively the "engine" our CLI orchestrates — it already
solves most of the hard, version-churn-prone problems (downloading the
correct server jar, mod loader, and modpack files), so the CLI's job is
mostly translating user intent into the right environment variables / volume
layout / lifecycle commands, plus everything the image intentionally leaves
to the operator (multi-server orchestration, host-side backup scheduling,
monitoring dashboards, a CLI UX).

### Core capabilities

- **Server software / `TYPE`**: `VANILLA` (default), `PAPER`, `SPIGOT`,
  `BUKKIT`, `FOLIA`, `PURPUR`, `FABRIC`, `FORGE`, `NEOFORGE`, `QUILT`,
  `MOHIST`, `CATSERVER`, `SPONGEVANILLA`, `MAGMA`, and modpack-oriented types
  `AUTO_CURSEFORGE`, `CURSEFORGE` (manual pack), `MODRINTH`, `FTBA`. The
  image downloads and wires up the right jar/installer for whichever is
  chosen and handles upgrades between Minecraft versions automatically on
  container restart.
- **Version selection**: `VERSION` (e.g. `LATEST`, `SNAPSHOT`, or an exact
  version string); per-server-type variables for mod loader version
  (`FORGE_VERSION`, `FABRIC_LOADER_VERSION`, etc).
- **EULA**: `EULA=TRUE` required to boot.
- **Memory/JVM tuning**: `MEMORY` (simple heap size), `USE_AIKAR_FLAGS=true`
  for community-standard GC tuning, plus raw `JVM_OPTS`/`JVM_XX_OPTS` escape
  hatches.
- **World/data persistence**: everything lives under `/data` in the
  container, meant to be bind-mounted or volume-mounted from the host — this
  matches our "server files on host" requirement directly.
- **Ops/whitelist**: `OPS`, `WHITELIST`, `ENABLE_WHITELIST` env vars accept
  comma-separated usernames/UUIDs and are applied at startup; can also be
  managed at runtime via RCON commands.
- **RCON**: enabled by default (`ENABLE_RCON`, `RCON_PASSWORD`,
  `RCON_PORT`), which is the intended channel for our CLI to send arbitrary
  console commands and to safely coordinate backups.
- **Mods/plugins (individual)**: `MODS`/`PLUGINS` (space/newline-separated
  URLs) for ad hoc jar downloads, plus first-class integration with
  **Modrinth** (`MODRINTH_PROJECTS`, resolved by project slug/ID/URL, with
  version and loader/game-version matching) and **CurseForge**
  (`CURSEFORGE_FILES`, needs `CF_API_KEY` for some operations).
- **Modpacks**: `MODRINTH_MODPACK` (type `MODRINTH`) and `AUTO_CURSEFORGE`
  (`CF_SLUG`/`CF_FILE_ID` + `CF_API_KEY`) install a full modpack and its
  matching mod loader automatically, and handle upgrade/downgrade cleanup of
  old mod files when the pack version changes.
- **Health checks & lifecycle**: built-in Docker `HEALTHCHECK` support (via
  the bundled `mc-monitor` binary), graceful shutdown (sends `stop` via RCON
  and waits up to `STOP_DURATION` seconds before killing), and
  `SETUP_ONLY=true` mode to just materialize server files without booting —
  useful for a CLI "create" step that provisions a data directory before
  first start.
- **Auto-pause / auto-stop**: `ENABLE_AUTOPAUSE` (pause the process when idle
  to save CPU) and `ENABLE_AUTOSTOP` (stop the container entirely when no
  players are connected) — potentially attractive for a hobbyist fleet
  running many low-traffic servers on one host.

### Companion tools from the same author (reusable instead of reinventing)

- **`itzg/mc-monitor`** — CLI/lib for querying Minecraft server status (ping,
  player count, version) and exposing Prometheus metrics; also what backs
  the image's health check. Good fit for our "monitor resources" feature
  alongside `docker stats`.
- **`itzg/docker-mc-backup`** — sidecar container that performs scheduled,
  RCON-coordinated backups (`save-off`/`save-all flush`/copy/`save-on`),
  with tar, rsync, restic, or rclone backends, retention/pruning, and
  pre/post backup hooks. This is very likely what we want to shell out to
  or vendor logic from, rather than writing our own backup coordination.
- **`itzg/rcon-cli`** — minimal RCON client; useful as the primitive our CLI
  uses (or shells out to) for arbitrary command execution against a running
  server.
- **`itzg/mc-router`** — SNI/hostname-based reverse proxy that lets many
  Minecraft servers share port 25565 behind one host by routing based on the
  hostname the client connects with. Directly solves the "port collision
  across many servers" challenge from §3 for anyone running multiple
  public-facing servers, at the cost of needing hostnames instead of raw
  ports.
- **`itzg/mc-image-helper`** — the internal Java helper CLI baked into the
  server image that actually implements version resolution, mod/modpack
  fetching, etc. Not something we call directly, but useful to know it
  exists when debugging the image's behavior.

## 5. Implications for our design

- **One container per server, host-mounted data dir**: aligns exactly with
  itzg's `/data` convention — see the "Server data root" and "State
  storage" bullets below for the resolved host layout and metadata
  approach.
- **Editing an existing server (name, image/type)**: a server's name and its
  underlying image/software must both be editable after creation, not just
  fixed at `mcm create` time — e.g. `mcm rename <old> <new>` and `mcm edit
  <name> --type PAPER --version 1.21.1` (or an equivalent `--image` flag for
  pinning a specific itzg image tag/digest per §"risk" discussion). Renaming
  updates the server's directory name, `manager.json`, and the container's
  name/labels together as one operation (container recreated under the new
  name — Docker doesn't support renaming a container's identity-relevant
  labels in place). Changing type/version/image recreates the container
  with new env vars against the *same* `data/` directory, which is exactly
  the upgrade/downgrade path the itzg image already supports (§4) — the CLI
  just needs to surface it as an explicit edit rather than only as part of
  `create`, and should warn (per the plugin/mod version-lag challenge in
  §3) before applying a change that might break installed mods/plugins.
- **Docker orchestration approach**: use the Docker Engine API directly via
  the official Go SDK (`docker/docker/client`) rather than shelling out to
  `docker`/`docker compose`, so we can stream `docker stats`-equivalent data,
  logs, and health/container events without parsing CLI output.
- **Command execution**: use RCON (via a small embedded RCON client, same
  protocol as `rcon-cli`) rather than `docker exec` into the console, since
  RCON is what the backup/monitoring ecosystem already assumes and avoids
  fragile stdin attachment.
- **Backups**: either run `itzg/mc-backup` as a managed sidecar per server
  (simplest, reuses a battle-tested tool) or reimplement its
  pause/flush/copy/resume sequence directly in our CLI for tighter UX
  integration (e.g. `mcm backup <server>` without needing a second
  container running at all times). Leaning toward reusing the sidecar for
  scheduled backups, with our CLI also able to trigger one-off backups
  through the same mechanism.
- **Mods/plugins/modpacks**: mostly a matter of the CLI translating
  `mcm install-mod <server> <modrinth-or-curseforge-ref>` /
  `mcm install-modpack <server> --source modrinth <slug>` into the right
  env vars and restarting the container (since the itzg image resolves
  everything at container start). For plugins on Spigot/Paper specifically,
  same idea via `PLUGINS`/CurseForge/Modrinth "plugin" project types.
- **Users/whitelist/ops**: expose as CLI subcommands that either (a) update
  env vars + restart, or (b) issue live RCON commands (`whitelist add`, `op`,
  `ban`) for zero-downtime changes — prefer (b) when the server is already
  running, falling back to (a) for initial provisioning.
- **Monitoring**: combine Docker-level stats (CPU/RAM/net/disk from the
  Docker Engine API) with Minecraft-level stats (`mc-monitor`/RCON `list`,
  `tps` if a plugin exposes it) into one `mcm status`/`mcm top` view.
- **CLI/web sharing one core**: structure the codebase as a Go core package
  (fleet state, Docker orchestration, RCON, mod/modpack install logic) with
  two thin front ends on top — the CLI command parser (e.g. `cobra`), and
  an embedded HTTP server started by `mcm web` that serves a JSON/REST (or
  RPC) API. Neither front end should contain business logic; both just
  call the same Go functions/service layer. This is what keeps "same
  thing, two interfaces" true instead of the web UI drifting into a
  separate reimplementation.
- **`mcm web` specifics to work out later**: what it binds to by default
  (localhost-only vs. LAN-exposed — should default to localhost given no
  auth model yet), whether it needs its own auth/session layer before it's
  safe to expose beyond localhost, whether it's read/write from day one or
  starts read-only (status/monitoring) before console/lifecycle actions are
  added, and whether the web server can coexist with the CLI acting on the
  same fleet concurrently (both should just read/write the same on-disk
  state and Docker resources — no separate "web-only" state).
- **State storage, resolved**: Docker itself is the source of truth for
  runtime state (running/stopped/health/stats), queried live via the Go
  Docker SDK rather than mirrored into a file that can drift out of sync
  (e.g. after someone runs `docker stop` outside the CLI, or a host
  reboot). Manager-owned metadata that Docker can't hold is split by scope:
  - Per-server config (requested type/version, port, backup schedule) lives
    in a small `manager.json` sitting next to that server's `data/`
    directory — inside the server's own folder, not inside `data/` itself
    (so it's never swept into a world backup) — making each server folder
    self-contained and portable as a single unit. The same values are also
    set as Docker labels on the container at creation time where useful, so
    `docker inspect`/`docker ps --filter` can cross-check against the file.
  - Global manager config (CurseForge API key, default servers-root path,
    web auth token) lives separately in the standard per-OS config location
    (`~/.config/mcm/config.toml` on Linux via XDG, respectively `os.UserConfigDir()` in Go)
    since it's manager identity, not any one server's state.
- **Server data root, resolved**: a configurable root directory the
  operator can browse directly (default e.g. `~/mc-servers/`), with each
  server at `<root>/<name>/{data/, manager.json}`. `data/` is what's
  bind-mounted into the container's `/data`. Deliberately not tucked away
  under an XDG *data* directory — discoverability of the actual world/mod
  files was an explicit project goal.

## 6. Decisions made

- **Language/stack**: CLI and the embedded web *server* (API/backend) are
  both **Go** — one core module/binary, sharing the Docker SDK client, RCON
  client, and fleet/service logic between the `mcm` CLI commands and the
  `mcm web` HTTP server (see §5's "CLI/web sharing one core").
- **Web UI**: the frontend is **SvelteKit**, talking to the Go backend's API
  (REST/JSON, plus websockets/SSE for log/console streaming and live
  resource graphs — see open question below). Built and either embedded
  into the Go binary (via `embed.FS`) for single-binary distribution, or
  served separately in dev — final packaging approach still open.
- **State storage**: resolved as described in §5 — Docker is the runtime
  source of truth (queried live, no mirrored running/stopped flag to go
  stale), per-server metadata lives in `manager.json` next to each server's
  `data/` directory, and global manager config lives in the OS-standard
  config dir. No SQLite/central DB for v0.

## 7. Open questions / decisions still needed

1. **Multi-user/permissions** — is this strictly single-operator, or does it
   need any notion of "who can run what," even locally?
2. **Networking model** — plain per-server host port mapping vs. adopting
   `mc-router` for hostname-based routing when multiple public servers share
   a host.
3. **Backup strategy** — sidecar container (`itzg/mc-backup`) vs. built into
   the CLI itself.
4. **CurseForge API key handling** — where/how a user-supplied `CF_API_KEY`
   is stored (plain config vs. OS keychain) given it's a personal
   credential.
5. **Web UI auth** — the CLI has an implicit trust model (whoever has shell
   access on the host). `mcm web` breaks that as soon as it's reachable from
   anything but localhost — needs at least a single-operator password/token
   before it's safe to bind beyond `127.0.0.1`. Does v0 just hard-bind to
   localhost and defer real auth, or is basic auth/token required from day
   one?
6. **SvelteKit packaging** — embed the built static assets into the Go
   binary (`embed.FS`, one artifact to ship) vs. deploy the SvelteKit
   frontend as its own process/container hitting the Go API separately
   (more moving parts, but a normal Node dev loop for UI work).
7. **Realtime updates in the web UI** — console/log tailing and live
   resource graphs imply websockets or SSE from the Go backend; worth
   deciding early since it affects the core library's API shape (needs to
   support streaming, not just request/response), and SvelteKit's data
   loading model (stores fed by a websocket vs. polling a REST endpoint).

## 8. Suggested next steps

1. Decide the remaining open questions in §7 (at least #1 and #5 —
   everything else can evolve).
2. Define the v0 CLI command surface (`mcm create`, `mcm rename`, `mcm edit`
   [name/type/version/image], `mcm start/stop`, `mcm list`, `mcm logs`,
   `mcm exec`, `mcm backup`, `mcm mods add/remove/list`, `mcm modpack
   install`, `mcm whitelist/op/ban`, `mcm status`, `mcm web`) as a spec doc,
   phrased as calls into a core Go package/service layer rather than logic
   embedded in command handlers — so `mcm web` can call the exact same
   layer later without a rewrite.
3. Prototype: create one server end-to-end (create → EULA accept → start →
   RCON command → stop → destroy) against the itzg image via the Go Docker
   SDK, to validate the volume/env-var model and `manager.json` layout
   before building the rest of the CLI around it.
4. Layer in mods/modpacks (Modrinth first — no API key friction — then
   CurseForge).
5. Layer in monitoring and backups once basic lifecycle management is solid.
6. Once the core Go package and CLI are stable for basic lifecycle + status,
   scaffold the SvelteKit frontend against a first-cut Go API: start with a
   localhost-only, read-only dashboard (fleet list, per-server
   status/resource graphs, log tailing) before adding write actions
   (start/stop, console commands, mod installs) gated behind whatever auth
   answer comes out of open question §7.5.

## Sources

- [Pterodactyl vs Crafty vs MCSManager 2026 — MineGuard](https://mineguard.pro/en/blog/pterodactyl-vs-crafty-vs-mcsmanager-2026)
- [Pterodactyl vs AMP vs Crafty Controller — Big Iron](https://www.bigiron.cc/guides/pterodactyl-vs-amp-vs-crafty-controller)
- [Crafty Controller official site](https://craftycontrol.com/)
- [Crafty Controller guide — IBRACORP docs](https://docs.ibracorp.io/docs/gaming/crafty-minecraft/)
- [MCSManager — GitHub](https://github.com/mcsmanager/MCSManager)
- [LinuxGSM vs AMP — SaaSHub](https://www.saashub.com/compare-linux-game-server-managers-vs-application-management-panel-amp)
- [itzg/docker-minecraft-server — GitHub](https://github.com/itzg/docker-minecraft-server)
- [itzg/docker-minecraft-server — env variable list discussion](https://github.com/itzg/docker-minecraft-server/discussions/1847)
- [Auto CurseForge docs — docker-minecraft-server](https://docker-minecraft-server.readthedocs.io/en/latest/types-and-platforms/mod-platforms/auto-curseforge/)
- [Modrinth Modpacks docs — docker-minecraft-server](https://github.com/itzg/docker-minecraft-server/blob/master/docs/types-and-platforms/mod-platforms/modrinth-modpacks.md)
- [itzg/docker-mc-backup — GitHub](https://github.com/itzg/docker-mc-backup)
- [Administer a Docker Minecraft Server with RCON — setupmc.com](https://setupmc.com/guides/administer-docker-minecraft-server-rcon-console/)
- [Why Minecraft servers crash — GameTeam](https://gameteam.io/blog/why-minecraft-servers-crash-common-causes-solutions/)
- [Minecraft server backup & recovery guide — Host Havoc](https://hosthavoc.com/blog/minecraft-server-backup-recovery)
- [How to fix out-of-version plugin errors — Apex Hosting](https://apexminecrafthosting.com/how-to-fix-out-of-version-plugins-errors/)
