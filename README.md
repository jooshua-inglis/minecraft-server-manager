# mcm

Manage a fleet of Minecraft servers running in Docker (via
[itzg/minecraft-server](https://github.com/itzg/docker-minecraft-server)) from
a CLI, a terminal dashboard, or a web UI — all three drive the same core.
Design notes are in [RESEARCH.md](RESEARCH.md); the build plan is in
[plan/](plan/README.md).

```sh
./install.sh        # builds and installs to ~/.local/bin/mcm (./uninstall.sh removes it)
mcm create myworld --accept-eula
mcm start myworld
mcm attach myworld      # live logs + stats + console in one screen
mcm web start           # dashboard in the background; `mcm web stop` to stop
```

`mcm --help` lists everything (mods, modpacks, backups, world export/import,
whitelist/ops/bans, ...).

## Downloading a prebuilt binary

CI publishes static Linux binaries (`amd64` and `arm64`) to
[Releases](https://github.com/jooshua-inglis/minecraft-server-manager/releases):
every push to `main` refreshes the rolling `latest` prerelease, and pushing a
`v*` tag creates a versioned release. On another machine:

```sh
arch=$(uname -m); case $arch in x86_64) arch=amd64;; aarch64) arch=arm64;; esac
mkdir -p ~/.local/bin
curl -fL "https://github.com/jooshua-inglis/minecraft-server-manager/releases/download/latest/mcm-linux-$arch" -o ~/.local/bin/mcm
chmod +x ~/.local/bin/mcm
```

`SHA256SUMS` is attached to each release. (Use `.../releases/download/v0.1.0/...`
for a tagged version.)

## Configuration

Config lives in `config.toml` under your OS config dir (`~/.config/mcm/`).
Environment variables override it:

| Variable | Meaning | Default |
|---|---|---|
| `MCM_SERVERS_ROOT` | Directory holding every server's `manager.json` + `data/` | `$XDG_STATE_HOME/mcm/servers` (`~/.local/state/mcm/servers`); an existing `~/mc-servers` is still used until the new dir exists |
| `MCM_CF_API_KEY` | CurseForge API key (modpack/mod installs) | unset |

## Web dashboard

- `mcm web` runs in the foreground; `mcm web start` / `stop` / `status`
  background it (PID + log in the config dir). Binds `127.0.0.1:8080` by
  default (`--addr` to change).
- Reads are open; every write needs the bearer token `mcm web` prints (it's
  generated once and kept in `config.toml`). Paste it into the dashboard's
  lock control to unlock write actions.
- The UI (`web/`, SvelteKit, built with Deno) is embedded in the binary. After
  changing it run `deno task build` in `web/` — the output in
  `internal/webui/dist` is committed so `go build` needs no JS toolchain.

## Running mcm in Docker

```sh
docker build -t mcm .
MCM_SERVERS_ROOT=/srv/minecraft docker compose up -d   # see docker-compose.yml
```

mcm doesn't run Minecraft inside its own container: it uses the host's Docker
daemon (via the mounted socket) to start sibling containers. Two consequences:

1. **Mount the socket:** `-v /var/run/docker.sock:/var/run/docker.sock`. The
   image runs as root because socket access is already root-equivalent on the
   host.
2. **Mount the servers directory at the same path on both sides.** mcm passes
   `data/` paths to the daemon as bind-mount sources, and the daemon resolves
   them on the *host*. So `MCM_SERVERS_ROOT` must be an absolute host path,
   mounted into the mcm container at that identical path
   (`-v /srv/minecraft:/srv/minecraft -e MCM_SERVERS_ROOT=/srv/minecraft`).

The container runs `mcm web` in the foreground bound to `0.0.0.0:8080`
(published to `127.0.0.1` in the compose file). Use `docker start/stop` for its
lifecycle; `mcm web start/stop` is for running directly on a host.
