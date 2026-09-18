# mcm web dashboard

SvelteKit frontend for `mcm web`'s API (`internal/webapi`). Built as a
static SPA (`adapter-static`, `ssr = false`) and embedded into the `mcm`
Go binary via `go:embed` (see `internal/webui`) — there's no Node
runtime involved once it's built.

## Developing

```sh
npm install
npm run dev
```

The dev server proxies `/api/*` to `http://127.0.0.1:8080`, so run
`mcm web` alongside it (from the repo root) to develop against real
data.

## Building

```sh
npm run build
```

Writes the static site to `../internal/webui/dist`, which `go build`
picks up via `go:embed`. Run this before building `mcm` after changing
anything here — the committed `dist/` output is what ships.
