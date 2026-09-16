# M8 — Individual mod/plugin install

[← Plan overview](README.md) · Previous: [M7 — Resource monitoring](m07-resource-monitoring.md) · Next: [M9 — Modpack install](m09-modpack-install.md)

## Build

`mcm mods add <name> <modrinth-or-curseforge-ref>`, `mcm mods list`, `mcm
mods remove` — translates to `MODRINTH_PROJECTS` / `CURSEFORGE_FILES` /
`PLUGINS` env vars + restart.

## Try it

On a Paper server from [M3](m03-rename-and-edit.md), `mcm mods add
myworld <some-plugin-slug>`, restart, and confirm the plugin loads (e.g.
its startup log line, or an in-game command it adds).
