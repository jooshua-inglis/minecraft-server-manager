# M16 — Per-server TUI dashboard

[← Plan overview](README.md) · Depends on: [M4 — Console access via RCON](m04-console-rcon.md), [M6 — Logs](m06-logs.md), [M7 — Resource monitoring](m07-resource-monitoring.md)

## Build

`mcm attach <name>` opens a full-screen interactive terminal dashboard for
a single server, built with `charmbracelet/bubbletea` (+ `bubbles` for the
scrolling viewport and text input, `lipgloss` for layout/styling) — the Go
ecosystem's standard TUI stack, so this doesn't reinvent terminal
rendering.

The dashboard is a live composition of work already proven out
individually in earlier milestones, calling the same `fleet` package
functions as every other command:

- a scrolling log pane (same source as `mcm logs -f`)
- a stats header (CPU/RAM/player count, same source as `mcm top`)
- a command input line at the bottom that sends whatever you type via
  RCON (same path as `mcm exec`) and echoes the result inline

No new business logic — this is strictly a new front end over `fleet`,
same as the CLI commands and (eventually) the web server.

## Try it

`mcm attach myworld` while playing, and:
- watch the log pane scroll live as players join/chat/leave
- watch the stats header update as load changes
- type a console command (e.g. `say hello`) directly into the input line
  and see it take effect in-game

Same three things you already validated separately in M4/M6/M7, now in
one interactive screen instead of three terminal windows.
