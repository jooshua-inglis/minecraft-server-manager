# M0 — Skeleton binary

[← Plan overview](README.md) · Next: [M1 — Create/start/stop/destroy](m01-create-start-stop-destroy.md)

## Build

Go module + `cobra`-based CLI skeleton. No Docker interaction yet — just
`mcm version` and `mcm --help` wired up, plus the project's config-loading
(`~/.config/mcm/config.toml`) stubbed out.

## Try it

Run `mcm --help` and `mcm version` and get sane output.
