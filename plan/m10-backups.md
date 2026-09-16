# M10 — Backups

[← Plan overview](README.md) · Previous: [M9 — Modpack install](m09-modpack-install.md) · Next: [M11 — Crash handling](m11-crash-handling.md)

## Build

Integrate `itzg/mc-backup` as a managed sidecar (or the RCON-coordinated
save-off/flush/copy/save-on sequence directly, per the open decision in
`RESEARCH.md` §7.3) behind `mcm backup <name>` and `mcm restore <name>
<backup-id>`.

## Try it

Build something in-game, `mcm backup myworld`, break/delete something
significant, `mcm restore` to the prior backup, and confirm the world is
back to the pre-break state.
