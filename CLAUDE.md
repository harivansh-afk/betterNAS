# Project Constraints

## Delivery sequencing

- Start with `apps/control-plane` first.
- Deliver the core backend in 2 steps, not 3:
  1. `control-server` plus `node-service` contract and runtime loop
  2. web control plane on top of that stable backend seam
- Do not start web UI work until the `control-server` and `node-service` contract is stable.

## Architecture

- `control-server` is the clean backend contract that other parts consume.
- `apps/node-agent` reports into `apps/control-plane`.
- `apps/web` reads from `apps/control-plane`.
- Local mount UX is issued by `apps/control-plane`.

## Backend contract priorities

- The first backend seam must cover:
  - node enrollment
  - node heartbeats
  - node export reporting
  - control-server persistence of nodes and exports
  - mount profile issuance for one export
- `control-server` should own:
  - node auth
  - user auth
  - mount issuance

## Mount profile shape

- Prefer standard WebDAV username and password semantics for Finder compatibility.
- The consumer-facing mount profile should behave like:
  - export id
  - display name
  - mount URL
  - username
  - password
  - readonly
  - expires at

## Service boundary

- Keep `node-service` limited to the WebDAV mount surface.
- Route admin and control actions through `control-server`, not directly from browsers to `node-service`.
