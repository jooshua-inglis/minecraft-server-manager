# Implementation Plan

Status: all milestones (M0–M17) implemented — see [../docs/README.md](../docs/README.md) for current state, deviations, and known gaps
Last updated: 2026-09-16

Companion to `../RESEARCH.md` (background, survey of existing tools, and
design decisions — Go for CLI + web backend, SvelteKit for the web UI,
Docker as the live source of truth for server state, per-server
`manager.json`, itzg/docker-minecraft-server as the image family).

Each milestone ends in something you can run yourself and see work — no
milestone depends on trusting an internal implementation detail you can't
observe. Milestones are meant to be done roughly in order; a couple have
obvious parallel sub-tracks noted where it matters. `mcm` is used as the
placeholder binary name throughout.

## Milestones

| # | Milestone | File |
|---|---|---|
| M0 | Skeleton binary | [m00-skeleton-binary.md](m00-skeleton-binary.md) |
| M1 | Create/start/stop/destroy a single Vanilla server | [m01-create-start-stop-destroy.md](m01-create-start-stop-destroy.md) |
| M2 | List and inspect the fleet | [m02-list-and-inspect.md](m02-list-and-inspect.md) |
| M3 | Rename and edit (type/version) | [m03-rename-and-edit.md](m03-rename-and-edit.md) |
| M4 | Console access via RCON | [m04-console-rcon.md](m04-console-rcon.md) |
| M5 | Player management (ops/whitelist/bans) | [m05-player-management.md](m05-player-management.md) |
| M6 | Logs | [m06-logs.md](m06-logs.md) |
| M7 | Resource monitoring | [m07-resource-monitoring.md](m07-resource-monitoring.md) |
| M8 | Individual mod/plugin install | [m08-individual-mods.md](m08-individual-mods.md) |
| M9 | Modpack install (Modrinth, then CurseForge) | [m09-modpack-install.md](m09-modpack-install.md) |
| M10 | Backups | [m10-backups.md](m10-backups.md) |
| M11 | Crash handling | [m11-crash-handling.md](m11-crash-handling.md) |
| M12 | Web backend (read-only API) | [m12-web-backend.md](m12-web-backend.md) |
| M13 | Web UI (read-only dashboard) | [m13-web-ui-readonly.md](m13-web-ui-readonly.md) |
| M14 | Web UI write actions + auth | [m14-web-ui-write-auth.md](m14-web-ui-write-auth.md) |
| M15 | Multi-server networking polish | [m15-networking-polish.md](m15-networking-polish.md) |
| M16 | Per-server TUI dashboard | [m16-tui-dashboard.md](m16-tui-dashboard.md) |
| M17 | World management (independent of server config) | [m17-world-management.md](m17-world-management.md) |

## Suggested pacing

M0–M4 are the core loop and worth doing carefully and in order —
everything else builds on "create/start/stop a real server + run a command
against it" being solid. M5–M11 can be reordered based on what you
personally want to use first (e.g. if backups matter more to you than mod
installs, swap M8–M9 and M10). M12–M14 (the web UI) deliberately come
after the CLI is functionally complete, per the "CLI first, web is a
second front end on the same core" design — building them earlier risks
the web UI accidentally becoming the place business logic lives, which is
exactly what `RESEARCH.md` §5 warns against.

M16 (the TUI dashboard) only needs M4, M6, and M7 done — it doesn't depend
on M8–M15 at all, so it can slot in right after M7 if a live per-server
dashboard is more valuable to you sooner than mods/modpacks/backups, or be
left until last as a polish item. It's numbered last here only because it
was added to the plan after M0–M15 already existed.

M17 (world management) only needs M1 — it's independent of M4–M16 and
could even come right after M3 if separating "manage the save data" from
"manage the server's config" matters to you before mods/monitoring/backups
do. It's also numbered last only because of when it was added, not because
of a real ordering dependency.
