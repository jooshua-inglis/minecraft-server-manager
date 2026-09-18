# Contributor & agent docs

Read these before changing anything. They record what the code and git history
don't say on their own: how the pieces fit, how to test without wrecking the
maintainer's real servers, and the non-obvious things that have already bitten
us.

| Doc | Read it for |
|---|---|
| [architecture.md](architecture.md) | Package map, the "one core, three front ends" rule, state model, and where the shipped code deviates from the plan |
| [development.md](development.md) | Build/test/live-test workflow, the safe-testing rules, frontend (Deno) workflow, UI design preferences, commit conventions |
| [gotchas.md](gotchas.md) | Hard-won, non-obvious lessons (Docker, WSL2, tooling). Skim before debugging something weird |
| [release-and-ci.md](release-and-ci.md) | GitHub Actions, releases, `install.sh`, the Docker files, and their known quirks |

Older, still-authoritative background:

- [../RESEARCH.md](../RESEARCH.md) — survey of existing tools and the design
  decisions. §7's open questions are all resolved; each answer is recorded
  inline.
- [../plan/](../plan/README.md) — the milestone plan (M0–M17). It's the
  original spec; see "Deviations from the plan" in architecture.md for where
  reality differs.

## Status (as of 2026-09-18)

All milestones M0–M17 are implemented, committed, pushed, and were verified
live against a real Docker daemon. Nothing is known to be half-built.

Known gaps / possible follow-ups (none are bugs; all were deliberate cuts):

- `itzg/mc-router` hostname routing (M15's optional half) — deferred; plain
  per-server host ports with automatic allocation shipped instead.
- The web UI doesn't expose ban, rename, or edit (type/version) — only the CLI
  does. Whitelist/ops, mods, modpack, backups/restore, console, and
  create/start/stop/destroy are in the UI.
- World operations (`mcm world ...`) require the server to be stopped; no live
  RCON save-coordination like whole-server backup has.
- `install.sh` only downloads release binaries; there's no build-from-source
  mode. Binaries are Linux amd64/arm64 only.
- Unit tests cover pure logic only. Docker-dependent behavior, the CLI
  commands, and the TUI/web UI are verified by hand (see development.md).
- One shared web token; no per-user accounts (RESEARCH.md §7.1, by design).
