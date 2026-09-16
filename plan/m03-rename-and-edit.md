# M3 — Rename and edit (type/version)

[← Plan overview](README.md) · Previous: [M2 — List and inspect](m02-list-and-inspect.md) · Next: [M4 — Console access via RCON](m04-console-rcon.md)

## Build

`mcm rename <old> <new>` (directory + `manager.json` + container recreated
under new name/labels) and `mcm edit <name> --type PAPER --version <x>`
(recreates the container against the same `data/` dir with new env vars —
this is also where the "warn before a risky type/version change" behavior
from `RESEARCH.md` §5 lands).

## Try it

Create a Vanilla server, `mcm edit` it to `PAPER`, restart, and confirm a
`plugins/` folder now exists and Paper-specific behavior works. Rename it
and confirm the directory and `docker ps` name both changed together, and
the server still starts correctly afterward.
