# M1 — Create/start/stop/destroy a single Vanilla server

[← Plan overview](README.md) · Previous: [M0 — Skeleton binary](m00-skeleton-binary.md) · Next: [M2 — List and inspect](m02-list-and-inspect.md)

## Build

Docker Go SDK integration. `mcm create <name>` provisions
`~/mc-servers/<name>/data/` + `manager.json`, pulls the itzg image, and
creates (but doesn't necessarily start) the container with `EULA=TRUE`, a
host port, and the data bind mount. `mcm start/stop/destroy <name>` drive
the container lifecycle.

## Try it

`mcm create myworld && mcm start myworld`, then connect to it from an
actual Minecraft Java client on `localhost:<port>` and play. `mcm stop
myworld` and confirm the world files are sitting in
`~/mc-servers/myworld/data/` on the host. `mcm destroy myworld` and confirm
the container is gone but (decide here, and confirm) the data directory is
kept unless a `--purge`/force flag is passed.

**This is the milestone that validates the whole volume/env-var model** —
worth not rushing past.
