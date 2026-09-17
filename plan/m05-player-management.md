# M5 — Player management (ops/whitelist/bans)

[← Plan overview](README.md) · Previous: [M4 — Console access via RCON](m04-console-rcon.md) · Next: [M6 — Logs](m06-logs.md)

## Build

`mcm whitelist add/remove/list`, `mcm op add/remove/list`, `mcm ban
add/remove/list` work in two modes depending on whether the server is
running:

- **Running**: sent as live RCON commands (`whitelist add`, `op`, `ban`,
  ...) via the `Exec` path from [M4](m04-console-rcon.md) — takes effect
  immediately, no restart.
- **Stopped**: edited directly on disk instead of going through env
  vars/restart — since the data directory is a plain host-mounted folder
  (per the project's core "files on host" design), `whitelist.json`,
  `ops.json`, `banned-players.json`, and `banned-ips.json` under
  `data/` can just be read, modified, and written back by the CLI. This
  also works for a server that's never been started yet, which live RCON
  obviously can't do.

Both modes should converge on the same `serverstore`-level read/list
behavior (`mcm whitelist list` etc. reads the on-disk JSON either way,
since that's the durable source of truth regardless of how an entry got
there) — only the *write* path branches on running vs. stopped.

## Try it

- With a friend (or a second game account), `mcm whitelist add <name>`
  while the server is running and have them join immediately without a
  restart.
- Stop the server, `mcm whitelist add <name> <other-friend>`, and confirm
  `data/whitelist.json` was updated directly with no container
  start/restart involved — then start the server and confirm that friend
  can join too.
- Try `mcm op add <name> <friend>` on a server that has never been
  started yet (right after `mcm create`, before the first `mcm start`),
  and confirm it works by editing `ops.json` up front.
