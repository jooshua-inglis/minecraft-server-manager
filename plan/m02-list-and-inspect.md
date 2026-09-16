# M2 — List and inspect the fleet

[← Plan overview](README.md) · Previous: [M1 — Create/start/stop/destroy](m01-create-start-stop-destroy.md) · Next: [M3 — Rename and edit](m03-rename-and-edit.md)

## Build

`mcm list` (name, type, version, status, port — queried live from Docker,
per the resolved state-storage approach) and `mcm status <name>` (richer
single-server view: container health, uptime, port, data dir path).

## Try it

Create 2-3 servers, `mcm list` shows all of them accurately, including if
you `docker stop` one manually outside the CLI — status should reflect
reality, not a stale flag, proving out the "Docker is the source of truth"
decision.
