# M15 — Multi-server networking polish

[← Plan overview](README.md) · Previous: [M14 — Web UI write actions + auth](m14-web-ui-write-auth.md)

## Build

Automatic port allocation across the fleet (no manual port bookkeeping),
and optionally `itzg/mc-router` integration for hostname-based routing if
you want multiple servers reachable on the default port 25565.

## Try it

`mcm create` three servers back to back with no port flags, confirm none
collide, and connect to each independently.
