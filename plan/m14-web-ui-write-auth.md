# M14 — Web UI write actions + auth

[← Plan overview](README.md) · Previous: [M13 — Web UI (read-only dashboard)](m13-web-ui-readonly.md) · Next: [M15 — Multi-server networking polish](m15-networking-polish.md)

## Build

Start/stop, console command execution, whitelist/op edits, and
mod/modpack install from the web UI, gated behind whatever minimal auth
(token/password) answer comes out of `RESEARCH.md` §7.5. Only after this
does binding beyond localhost become reasonable to even consider.

## Try it

Perform a full create → play → mod install → backup cycle using *only*
the browser, no CLI commands, and confirm the CLI (`mcm list`, etc.) sees
the exact same resulting state afterward — proving the "one core, two
interfaces" goal actually holds.
