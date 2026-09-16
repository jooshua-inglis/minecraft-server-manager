# Implementation Plan

Status: draft
Last updated: 2026-09-16

Companion to `RESEARCH.md` (background, survey of existing tools, and design
decisions — Go for CLI + web backend, SvelteKit for the web UI, Docker as
the live source of truth for server state, per-server `manager.json`,
itzg/docker-minecraft-server as the image family).

Each milestone below ends in something you can run yourself and see work —
no milestone depends on trusting an internal implementation detail you
can't observe. Milestones are meant to be done roughly in order; a couple
have obvious parallel sub-tracks noted where it matters. `mcm` is used as
the placeholder binary name throughout.

## M0 — Skeleton binary

**Build:** Go module + `cobra`-based CLI skeleton. No Docker interaction
yet — just `mcm version` and `mcm --help` wired up, plus the project's
config-loading (`~/.config/mcm/config.toml`) stubbed out.

**Try it:** run `mcm --help` and `mcm version` and get sane output.

## M1 — Create/start/stop/destroy a single Vanilla server

**Build:** Docker Go SDK integration. `mcm create <name>` provisions
`~/mc-servers/<name>/data/` + `manager.json`, pulls the itzg image, and
creates (but doesn't necessarily start) the container with `EULA=TRUE`,
a host port, and the data bind mount. `mcm start/stop/destroy <name>` drive
the container lifecycle.

**Try it:** `mcm create myworld && mcm start myworld`, then connect to it
from an actual Minecraft Java client on `localhost:<port>` and play. `mcm
stop myworld` and confirm the world files are sitting in
`~/mc-servers/myworld/data/` on the host. `mcm destroy myworld` and confirm
the container is gone but (decide here, and confirm) the data directory is
kept unless a `--purge`/force flag is passed.

**This is the milestone that validates the whole volume/env-var model** —
worth not rushing past.

## M2 — List and inspect the fleet

**Build:** `mcm list` (name, type, version, status, port — queried live
from Docker, per the resolved state-storage approach) and `mcm status
<name>` (richer single-server view: container health, uptime, port,
data dir path).

**Try it:** create 2-3 servers, `mcm list` shows all of them accurately,
including if you `docker stop` one manually outside the CLI — status should
reflect reality, not a stale flag, proving out the "Docker is the source of
truth" decision.

## M3 — Rename and edit (type/version)

**Build:** `mcm rename <old> <new>` (directory + `manager.json` + container
recreated under new name/labels) and `mcm edit <name> --type PAPER
--version <x>` (recreates the container against the same `data/` dir with
new env vars — this is also where the "warn before a risky
type/version change" behavior from `RESEARCH.md` §5 lands).

**Try it:** create a Vanilla server, `mcm edit` it to `PAPER`, restart, and
confirm a `plugins/` folder now exists and Paper-specific behavior works.
Rename it and confirm the directory and `docker ps` name both changed
together, and the server still starts correctly afterward.

## M4 — Console access via RCON

**Build:** embedded RCON client; `mcm exec <name> "<command>"` sends
arbitrary console commands. RCON enabled automatically on server creation
(bound to the Docker network, not published publicly, per the security note
in `RESEARCH.md` §3).

**Try it:** `mcm exec myworld "say hello from the CLI"` and see it appear
in-game/in the console log. Try `mcm exec myworld "list"` and see actual
connected players.

## M5 — Player management (ops/whitelist/bans)

**Build:** `mcm whitelist add/remove/list`, `mcm op add/remove`, `mcm ban
add/remove` — implemented as live RCON commands when the server is running
(no restart), falling back to env-var + restart only if the server is
stopped.

**Try it:** with a friend (or a second game account), `mcm whitelist add
<name>` while the server is running and have them join immediately without
a restart.

## M6 — Logs

**Build:** `mcm logs <name>` and `mcm logs <name> -f` streaming the
container's stdout/stderr via the Docker SDK.

**Try it:** `mcm logs myworld -f` in one terminal while playing/connecting
in-game in another, and watch join/leave/chat lines appear live.

## M7 — Resource monitoring

**Build:** `mcm top` (or `mcm stats`) combining Docker-level CPU/RAM stats
per container with Minecraft-level stats (player count via RCON `list`,
`mc-monitor`-style ping) into one live view.

**Try it:** run it while a couple of servers are up and idle vs. under load
(e.g. someone exploring/generating chunks) and see CPU/RAM move
accordingly, cross-checked against `docker stats` to make sure the numbers
agree.

## M8 — Individual mod/plugin install

**Build:** `mcm mods add <name> <modrinth-or-curseforge-ref>`, `mcm mods
list`, `mcm mods remove` — translates to `MODRINTH_PROJECTS` /
`CURSEFORGE_FILES` / `PLUGINS` env vars + restart.

**Try it:** on a Paper server from M3, `mcm mods add myworld
<some-plugin-slug>`, restart, and confirm the plugin loads (e.g. its
startup log line, or an in-game command it adds).

## M9 — Modpack install (Modrinth, then CurseForge)

**Build:** `mcm modpack install <name> --source modrinth <slug>` first
(no API key friction), then the CurseForge equivalent (`--source
curseforge`, reading `CF_API_KEY` from global config if the operator
supplied one).

**Try it:** spin up a real modpack end to end and connect with a matching
modded Minecraft launcher/client profile — this is the milestone that
proves the "install modpacks from Modrinth and CurseForge" requirement from
the original brief.

## M10 — Backups

**Build:** integrate `itzg/mc-backup` as a managed sidecar (or the
RCON-coordinated save-off/flush/copy/save-on sequence directly, per the
open decision in `RESEARCH.md` §7.3) behind `mcm backup <name>` and `mcm
restore <name> <backup-id>`.

**Try it:** build something in-game, `mcm backup myworld`, break/delete
something significant, `mcm restore` to the prior backup, and confirm the
world is back to the pre-break state.

## M11 — Crash handling

**Build:** container restart policy + crash-loop backoff, with the last
crash reason surfaced in `mcm status` (per the crash-loop challenge in
`RESEARCH.md` §3).

**Try it:** deliberately break a server (e.g. install an incompatible mod
version or corrupt a config), restart it, and confirm the CLI shows a clear
"crashing, last error: ..." status instead of silently restart-looping
forever.

## M12 — Web backend (read-only API)

**Build:** `mcm web` starts the embedded Go HTTP server, binds to
localhost by default, and exposes read-only JSON endpoints mirroring `mcm
list`/`status`/`logs` (plus a websocket/SSE stream for logs and live
stats, per `RESEARCH.md` §7.7).

**Try it:** `mcm web`, then `curl localhost:<port>/api/servers` and watch
real data come back for the servers created in earlier milestones. No UI
yet — this milestone is just proving the API works standalone.

## M13 — Web UI (read-only dashboard)

**Build:** SvelteKit frontend against the M12 API: fleet list, per-server
status/resource graphs, live log tail.

**Try it:** open the dashboard in a browser while a server is running and
being played on, and watch the resource graph and log tail update live —
same thing you validated with `mcm top`/`mcm logs -f`, now in the browser.

## M14 — Web UI write actions + auth

**Build:** start/stop, console command execution, whitelist/op edits, and
mod/modpack install from the web UI, gated behind whatever minimal
auth (token/password) answer comes out of `RESEARCH.md` §7.5. Only after
this does binding beyond localhost become reasonable to even consider.

**Try it:** perform a full create → play → mod install → backup cycle using
*only* the browser, no CLI commands, and confirm the CLI (`mcm list`, etc.)
sees the exact same resulting state afterward — proving the "one core, two
interfaces" goal actually holds.

## M15 — Multi-server networking polish

**Build:** automatic port allocation across the fleet (no manual port
bookkeeping), and optionally `itzg/mc-router` integration for
hostname-based routing if you want multiple servers reachable on the
default port 25565.

**Try it:** `mcm create` three servers back to back with no port flags,
confirm none collide, and connect to each independently.

## Suggested pacing

M0–M4 are the core loop and worth doing carefully and in order — everything
else builds on "create/start/stop a real server + run a command against
it" being solid. M5–M11 can be reordered based on what you personally want
to use first (e.g. if backups matter more to you than mod installs, swap
M8–M9 and M10). M12–M14 (the web UI) deliberately come after the CLI is
functionally complete, per the "CLI first, web is a second front end on the
same core" design — building them earlier risks the web UI accidentally
becoming the place business logic lives, which is exactly what
`RESEARCH.md` §5 warns against.
