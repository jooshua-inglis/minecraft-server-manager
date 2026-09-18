# Gotchas

Non-obvious things that have already cost time. Each is fixed or worked around in
the code; this is so you don't rediscover them.

## Docker / itzg image

- **Docker doesn't report a port binding for a container that has never run.**
  `ContainerList`'s `Ports` is empty for created-but-never-started containers, so
  port allocation based only on Docker let several fresh servers all get 25567.
  Fixed by also consulting every `manager.json` (`fleet.usedPorts`). Anything
  that needs "what does this server own" should read `manager.json`, not Docker.
- **`MaximumRetryCount` only works with the `on-failure` restart policy.** That's
  why crash handling uses `on-failure`+cap, not `unless-stopped`. `on-failure`
  doesn't restart after a clean exit (0), which matches a normal `mcm stop`.
  `Status` shows `running` transiently between restarts, so a crash-looping
  container can look healthy for a poll or two before it settles to `exited`.
- **`LEVEL` env var** controls the world directory name; mcm sets it from
  `Metadata.Level()` so world commands and the game agree.
- **RCON isn't ready when the container is "running"** (see development.md
  timing). Whitelist/op edits fall back to file edits only when the container
  isn't *running*, not when RCON isn't ready yet.
- **Invalid `VERSION`** makes the image exit 1 with "Minecraft version '…' is not
  valid" — handy for reproducing the crash-loop path (`mcm edit x --version bogus
  --confirm`).
- `mcm edit` changing type/version needs `--confirm`; `mcm destroy --purge` needs
  `--yes`.

## Bind mounts & filesystems

- **Never replace a bind-mounted directory's own inode.** On Docker Desktop's
  WSL2 backend, renaming a new directory over `data/` permanently breaks the
  mount for containers already created against it, even stopped ones. Restore and
  import therefore move *children* in and out (`moveChildren` /
  `replaceDirFromArchive`) and leave the directory itself alone. There's a test
  asserting the inode is unchanged.
- `data/` must be created by mcm (as the invoking host user) *before* Docker sees
  the bind mount, otherwise the daemon creates it as root and the host user can't
  write whitelist/backups. (`createContainer` does this.)
- Files created by a containerized mcm are owned by root on the host.

## mcm in Docker (Docker-outside-of-Docker)

- mcm creates **sibling** containers through the mounted socket, and passes bind
  sources to the daemon, which resolves them on the **host**. So the servers dir
  must be mounted into the mcm container at the **identical absolute path**, and
  `MCM_SERVERS_ROOT` must be that path. A path that only exists inside the mcm
  container silently creates empty dirs on the host instead.
- The image runs as **root**: a non-root user can't read the socket (its group
  GID differs per host), and socket access is root-equivalent anyway.
- `127.0.0.1` inside a container isn't reachable via published ports; the
  container CMD binds `0.0.0.0:8080` (compose publishes to host loopback only).
- Inside a container run `mcm web` (foreground). `mcm web start` daemonizes and
  would make PID 1 exit immediately.
- `docker compose` re-interpolates env on *every* invocation, so `docker compose
  logs/down` also need `MCM_SERVERS_ROOT` set (or use `docker logs <container>`).

## Go

- **`os.Process.Release()` sets `Pid` to -1 on Unix.** Read the pid before
  releasing (this shipped as a bug in `mcm web start`: it recorded pid -1).
- `signal.NotifyContext` is set up inside `runWebForeground`; cobra's
  `cmd.Context()` has no signal handling by default.
- `go get` for the TUI libs needed a follow-up `go mod tidy` for missing `go.sum`
  entries.
- Process management (`mcm web start/stop`) uses `syscall` (`Setsid`, signal 0) —
  Unix only, consistent with everything else here.

## Web / frontend tooling

- **Chrome's `pattern` attribute is compiled with the `v` flag.** A trailing
  literal `-` inside a character class (`[a-z0-9-]`) is invalid there; write
  `[-a-z0-9]`. (Caught via the page console, not visually.)
- **Don't double-wrap `.card`.** `PlayerListEditor` renders its own `.card`;
  wrapping it in another produced nested cards and broke test selectors.
- **Flex rows in narrow bento tiles overflow** unless the row wraps and the input
  has `min-width: 0` (the Add button was clipped once).
- `npm:svelte-kit` ≠ SvelteKit's CLI (see development.md); using it silently pulls
  an unrelated old package and pollutes `deno.lock`. Deno skips dependency build
  scripts by default (it warns; `deno approve-scripts` opts in). If Deno itself is
  installed via npm, npm blocks its postinstall — use `npm i -g --allow-scripts=deno deno`.
- Svelte's build prints "Run `npm run preview`" — cosmetic, ignore.
- The built favicon isn't in the static `index.html` (it's injected client-side in
  SPA mode); that's expected.

## Testing tooling

- **zsh: `status` is a read-only variable.** Don't name a shell variable `status`
  in scripts run by the harness.
- Each harness shell call is fresh: `export` doesn't persist; `tmux` doesn't
  inherit the harness env.
- tmux `capture-pane` returns exactly the pane height; if line 1 isn't the
  header, the TUI's layout arithmetic is off.
- Playwright: the npm package's expected browser revision can differ from the
  cached one; launch with `executablePath` and `--no-sandbox`.
- `curl -sf` hides the response body on HTTP errors; drop `-f` (use `-sS -w
  '%{http_code}'`) when debugging API failures.

## Process / repo

- **Commit messages can lie.** M16/M17's commits said "Adds …" but only added
  plan docs; the code came later. Verify with `git show --stat` / the code.
- Deleting a root-owned test dir from the host may fail; let the containerized
  mcm's `DELETE …?purge=true` (or `destroy --purge`) remove its own files.
