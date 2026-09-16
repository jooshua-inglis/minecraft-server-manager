# M12 — Web backend (read-only API)

[← Plan overview](README.md) · Previous: [M11 — Crash handling](m11-crash-handling.md) · Next: [M13 — Web UI (read-only dashboard)](m13-web-ui-readonly.md)

## Build

`mcm web` starts the embedded Go HTTP server, binds to localhost by
default, and exposes read-only JSON endpoints mirroring `mcm
list`/`status`/`logs` (plus a websocket/SSE stream for logs and live
stats, per `RESEARCH.md` §7.7).

## Try it

`mcm web`, then `curl localhost:<port>/api/servers` and watch real data
come back for the servers created in earlier milestones. No UI yet — this
milestone is just proving the API works standalone.
