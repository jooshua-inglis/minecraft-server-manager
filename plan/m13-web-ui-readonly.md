# M13 — Web UI (read-only dashboard)

[← Plan overview](README.md) · Previous: [M12 — Web backend](m12-web-backend.md) · Next: [M14 — Web UI write actions + auth](m14-web-ui-write-auth.md)

## Build

SvelteKit frontend against the [M12](m12-web-backend.md) API: fleet list,
per-server status/resource graphs, live log tail.

## Try it

Open the dashboard in a browser while a server is running and being played
on, and watch the resource graph and log tail update live — same thing you
validated with `mcm top`/`mcm logs -f`, now in the browser.
