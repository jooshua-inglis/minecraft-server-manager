# Development

## Prerequisites

- **Go 1.26** (see `go.mod`) and **Docker** (a real daemon — most behavior can
  only be verified live).
- **Deno** only if you touch `web/`. Go builds need no JS toolchain because the
  built UI is committed.
- For manual UI/TUI verification: `tmux`, and Playwright with a Chromium (see
  below).

## Build, vet, test

```sh
go build ./... && go vet ./... && go test ./...
gofmt -l .            # must print nothing
```

Unit tests are **pure logic only** — no Docker, no network. That's deliberate,
and the pattern for keeping it that way is to split the testable part out of
anything that needs Docker. Existing examples to copy: `selectPort`,
`recordedPorts` (vs. `usedPorts`), `replaceDirFromArchive`, `listArchivesIn`,
`parsePlayerCount`, `webapi.sseLineWriter`, `webapi.requireAuth`,
`tui.lineChanWriter`, `cliapp.isLoopback`. When you fix a bug, add a regression
test for the pure part (see `TestRecordedPortsCoversNeverStartedServers`).

There are no tests for the cobra commands, the TUI screen, or the web UI; those
are verified live.

## Live testing — read this before running anything

The maintainer runs **real servers** on this machine (at time of writing
`my-server` and `backuptest` on ports 25565/25566, data under `~/mc-servers`).
Do not touch them, and never run `destroy`/`stop` on a server you didn't create.

Rules that have worked:

1. **Point mcm at a project-local servers root.** Set `MCM_SERVERS_ROOT` to
   `<repo>/tmp/mc-servers`. The maintainer's Claude Code setup does this
   permanently via `.claude/settings.local.json` (gitignored):
   ```json
   { "env": { "MCM_SERVERS_ROOT": "/absolute/path/to/repo/tmp/mc-servers" } }
   ```
   Shell state doesn't persist between tool calls, and `tmux` sessions don't
   inherit it — pass it explicitly (`MCM_SERVERS_ROOT=... ./tmp/mcm ...`).
2. **Work inside the repo.** Use `<repo>/tmp/` (gitignored) for scratch files,
   built binaries (`go build -o tmp/mcm ./cmd/mcm`), fake HOMEs, downloaded
   files. The maintainer asked for this so they don't have to approve
   out-of-tree access. Don't use `/tmp` or `$HOME` for scratch.
3. **Use unique server names and clean up:** `mcm stop x && mcm destroy x --purge
   --yes`. Note `destroy --purge` needs `--yes`.
4. **Ports** are auto-allocated from 25565 upward and skip the real servers'
   ports; don't hardcode.
5. **Test install/uninstall against a fake `HOME`** (e.g. `tmp/fakehome`), never
   the real `~/.local/bin`.
6. **Clean up Docker things you create** (containers, images, compose stacks).

### Timing

A server's Docker status is `running` immediately, but Minecraft takes ~20–30s
(longer on first run) before RCON works. Poll `mcm logs <name> --tail 30` for
`Done (` rather than sleeping a fixed time; `mcm exec` before that fails or
hangs.

### Verifying each surface

- **CLI:** run the command against a throwaway server and check the effect on
  disk / via `mcm status|list|logs`.
- **Web API/UI:** `mcm web --addr 127.0.0.1:<free port>` in the background;
  `curl` the API; the write token is printed at startup and stored in
  `config.toml`. For the UI, drive it with Playwright (a headless Chromium via
  `executablePath`; the npm `playwright` package may not match the cached
  browser version), screenshot, and **read the screenshot**, and check the page
  `console` for errors. Handle `confirm()`/`alert()` with a `page.on('dialog')`
  handler or the script hangs. Don't commit the driver scripts.
- **TUI:** run `mcm attach` inside `tmux new-session -d -x 160 -y 40`, use
  `send-keys` / `capture-pane -p`, and check the header row is visible (line 1)
  and the pane line count equals the tmux height.
- **Docker image:** `docker build`, run with the socket + same-path servers
  mount (README → "Running mcm in Docker"), create/start/destroy a server through
  it, confirm files land at the matching host path.
- Kill any `mcm web` you start (`mcm web stop`, or kill the port's listener).

## Frontend (`web/`)

```sh
cd web
deno install          # first time
deno task dev         # vite dev server; proxies /api to 127.0.0.1:8080 (run `mcm web` too)
deno task check       # svelte-check
deno task build       # writes ../internal/webui/dist  -> COMMIT this output
```

- After any `web/` change: `deno task check`, `deno task build`, then rebuild the
  Go binary (the UI is `go:embed`ed) and commit `internal/webui/dist` along with
  the source. CI does **not** rebuild the UI, so a stale `dist/` ships silently.
- `web/deno.json` tasks wrap npm packages via `npm:` specifiers;
  `package.json` is only the dependency manifest. To run a package's bin use the
  right package (`npm:@sveltejs/kit`, **not** `npm:svelte-kit`, which is an
  unrelated old package).
- SvelteKit config lives in `vite.config.ts` (`sveltekit({ adapter, ... })`);
  there is no `svelte.config.js`.
- Svelte 5 runes are forced on. Shared reactive state uses `.svelte.ts` modules.

### UI design preferences (from the maintainer)

- **Light theme** using exactly this palette: Rich Cerulean `#2274a5` (accent),
  Sand Dune `#e7dfc6` (alt tiles/topbar), Alice Blue `#e9f1f7` (page bg), Ink
  Black `#131b23` (text), Taupe `#816c61` (muted text). Status colors
  (good/warn/bad) are separate muted semantic hues. Tokens are CSS variables in
  `web/src/app.css`; buttons use `--accent-contrast` (white) on the cerulean.
- **Bento-box layout**: a grid of variously sized rounded cards (`.bento`,
  `.card`, `.card.alt`, `.span-2` in `app.css`).
- **Sans-serif.** The maintainer explicitly rejected serif after seeing it.
- **Deno, not Node.**

## Conventions

- **Commits:** one commit per milestone/feature; message explains *why* and what
  was verified live; end with the `Co-Authored-By` trailer the harness specifies.
  New commits, don't amend. The maintainer does the pushing — don't push unless
  asked (a push triggers CI and publishes the rolling `latest` release).
- **Don't overclaim.** Before saying something is done, confirm from the diff and
  a real run. Report what you couldn't verify.
- **Comments:** only for non-obvious *why* (hidden constraints, workarounds).
  Several exist for exactly that reason (WSL2 inode, `Process.Release`, Docker
  port reporting) — keep them.
- **Error handling / abstractions:** validate at boundaries only; don't add
  layers for hypothetical future needs.
- Plans/decision docs: update `RESEARCH.md`/`plan/` only when a decision is
  actually made; otherwise record it here in `docs/`.
