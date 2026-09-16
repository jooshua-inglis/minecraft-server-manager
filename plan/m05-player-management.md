# M5 — Player management (ops/whitelist/bans)

[← Plan overview](README.md) · Previous: [M4 — Console access via RCON](m04-console-rcon.md) · Next: [M6 — Logs](m06-logs.md)

## Build

`mcm whitelist add/remove/list`, `mcm op add/remove`, `mcm ban add/remove`
— implemented as live RCON commands when the server is running (no
restart), falling back to env-var + restart only if the server is stopped.

## Try it

With a friend (or a second game account), `mcm whitelist add <name>` while
the server is running and have them join immediately without a restart.
