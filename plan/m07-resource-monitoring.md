# M7 — Resource monitoring

[← Plan overview](README.md) · Previous: [M6 — Logs](m06-logs.md) · Next: [M8 — Individual mod/plugin install](m08-individual-mods.md)

## Build

`mcm top` (or `mcm stats`) combining Docker-level CPU/RAM stats per
container with Minecraft-level stats (player count via RCON `list`,
`mc-monitor`-style ping) into one live view.

## Try it

Run it while a couple of servers are up and idle vs. under load (e.g.
someone exploring/generating chunks) and see CPU/RAM move accordingly,
cross-checked against `docker stats` to make sure the numbers agree.
