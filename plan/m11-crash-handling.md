# M11 — Crash handling

[← Plan overview](README.md) · Previous: [M10 — Backups](m10-backups.md) · Next: [M12 — Web backend](m12-web-backend.md)

## Build

Container restart policy + crash-loop backoff, with the last crash reason
surfaced in `mcm status` (per the crash-loop challenge in `RESEARCH.md`
§3).

## Try it

Deliberately break a server (e.g. install an incompatible mod version or
corrupt a config), restart it, and confirm the CLI shows a clear
"crashing, last error: ..." status instead of silently restart-looping
forever.
