# Builds mcm itself as a container image. mcm doesn't run Minecraft
# servers inside this image — it drives the HOST's Docker daemon over
# a mounted socket ("Docker-outside-of-Docker") to create sibling
# containers for them, the same way it would if run directly on the
# host. See README.md's "Running mcm in Docker" section for the
# socket/volume setup this requires.
#
# The SvelteKit dashboard (web/) is NOT built here — its output is
# committed to internal/webui/dist and embedded via go:embed, so this
# build needs only Go, not Node/Deno. Run `deno task build` in web/
# first if you've changed the frontend.

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/mcm ./cmd/mcm

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/mcm /usr/local/bin/mcm

# Runs as root deliberately: the mounted Docker socket is what actually
# matters for privilege here, and access to it is already
# root-equivalent on the host (a socket-holder can trivially run a
# container that mounts / and does anything). Dropping to a non-root
# user here wouldn't add real isolation, and would just break against
# hosts where the socket's group GID doesn't match one baked into this
# image at build time.
ENV HOME=/root
WORKDIR $HOME

EXPOSE 8080
ENTRYPOINT ["mcm"]
# Foreground by default, matching what a container runtime expects —
# `mcm web start`/`stop` are for host/bare-metal use, not here.
# Binds every interface, since 127.0.0.1 wouldn't be reachable through
# Docker's own port publishing.
CMD ["web", "--addr", "0.0.0.0:8080"]
