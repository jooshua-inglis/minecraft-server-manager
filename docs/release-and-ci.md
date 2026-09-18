# Release, CI, install

Verified end to end on 2026-09-18: CI passed, the rolling `latest` release was
published with all assets, `curl … install.sh | bash` installed a working binary
from it, and the release's `Dockerfile` + binary built a working image.

## Workflow (`.github/workflows/build.yml`)

Triggers: push to `main`, push of a `v*` tag, pull requests, manual dispatch.

1. **`test`** — `go vet ./...` and `go test ./...` (Go version from `go.mod`).
2. **`build`** (needs `test`, matrix `amd64`/`arm64`) — `CGO_ENABLED=0 GOOS=linux
   GOARCH=<arch> go build -trimpath -ldflags "-s -w -X …/internal/cliapp.version=<git describe>"`.
   Pure Go, so arm64 is cross-compiled on one runner (no QEMU). Uploads
   `mcm-linux-<arch>` as workflow artifacts (login-required, expire — for PR
   inspection only).
3. **`release`** (needs `build`, push events only, `contents: write`) — downloads
   the artifacts, copies `Dockerfile.release` → `Dockerfile` and
   `docker-compose.yml` into `dist/`, writes `SHA256SUMS`, then:
   - **tag `v*`** → `gh release create <tag>` with generated notes;
   - **`main`** → deletes and recreates the rolling **`latest` prerelease**
     (`--cleanup-tag`, brief window with no release).

Release assets: `mcm-linux-amd64`, `mcm-linux-arm64`, `Dockerfile`,
`docker-compose.yml`, `SHA256SUMS`. **No container image is published** — the
maintainer doesn't want to manage image uploads; the Dockerfile ships instead and
users build locally.

Action versions in use: `checkout@v7`, `setup-go@v7`, `upload-artifact@v7`,
`download-artifact@v8`. (Looked up via the GitHub API; re-check when editing.)

### Quirks

- **`/releases/latest/download/...` does NOT point at our `latest`.** GitHub's
  "latest" means the newest *non-prerelease*; ours is a prerelease tagged
  `latest`. Use `/releases/download/latest/<asset>` (that's what `install.sh`
  does).
- **`mcm version` on CI builds can read `latest-1-g<sha>`** instead of a bare sha,
  because the rolling `latest` tag is picked up by `git describe --tags`. Cosmetic
  bug; fix is `git describe --tags --match 'v*' --always --dirty` in the build step.
- CI does not rebuild the web UI or check that `internal/webui/dist` is fresh.
- Linux amd64/arm64 only.

## `install.sh` / `uninstall.sh`

`install.sh` is **standalone** (no checkout; runs via `curl … | bash`):

- Detects arch (`x86_64`→amd64, `aarch64`→arm64; anything else errors), Linux only.
- Downloads `mcm-linux-<arch>` **and `SHA256SUMS`** from
  `https://github.com/<repo>/releases/download/${MCM_VERSION:-latest}` and verifies
  the checksum before installing. A mismatch aborts and leaves any existing
  install untouched.
- Downloads next to the destination and `mv`s into `~/.local/bin/mcm`, so
  reinstalling works while `mcm web` is running the old binary. Warns if
  `~/.local/bin` isn't on `PATH`.
- Env: `MCM_VERSION` (default `latest`; e.g. `v0.1.0`), `MCM_DOWNLOAD_BASE`
  (override the whole download URL — use it to test against a local
  `python3 -m http.server` serving a directory laid out like a release, or for a
  mirror).

`uninstall.sh` (also standalone) runs `mcm web stop` if the binary exists, removes
`~/.local/bin/mcm`, and leaves servers/config alone.

Test both against a fake `HOME` (development.md).

## Docker files

Two Dockerfiles, **keep their runtime sections in sync**:

- `Dockerfile` — builds from source (multi-stage, `golang:1.26-alpine` →
  `alpine:3.20`, `CGO_ENABLED=0`). Uses the committed `internal/webui/dist`.
- `Dockerfile.release` — wraps a prebuilt binary
  (`COPY --chmod=0755 mcm-linux-${TARGETARCH}`), so curl-downloaded
  (non-executable) files work. CI ships it as plain `Dockerfile`.

Both: run as root, `EXPOSE 8080`, `ENTRYPOINT ["mcm"]`,
`CMD ["web","--addr","0.0.0.0:8080"]`. `docker-compose.yml` requires
`MCM_SERVERS_ROOT` (absolute host path, mounted at the identical path) and
publishes to `127.0.0.1:8080`. See gotchas.md for why.

The source `Dockerfile` does not currently cross-compile or stamp a version (a
`--platform=$BUILDPLATFORM` + `TARGETARCH` + `VERSION` build-arg variant was
prototyped and works with no QEMU, but wasn't adopted since no image is
published).

## Releasing a version

Push a tag: `git tag v0.1.0 && git push origin v0.1.0`. CI builds and creates a
normal release with the same five assets. The maintainer pushes; agents should
not push or tag unless asked.
