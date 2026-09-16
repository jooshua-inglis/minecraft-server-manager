# M4 — Console access via RCON

[← Plan overview](README.md) · Previous: [M3 — Rename and edit](m03-rename-and-edit.md) · Next: [M5 — Player management](m05-player-management.md)

## Build

Embedded RCON client; `mcm exec <name> "<command>"` sends arbitrary console
commands. RCON enabled automatically on server creation (bound to the
Docker network, not published publicly, per the security note in
`RESEARCH.md` §3).

## Try it

`mcm exec myworld "say hello from the CLI"` and see it appear in-game/in
the console log. Try `mcm exec myworld "list"` and see actual connected
players.
