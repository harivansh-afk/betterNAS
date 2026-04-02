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

## User-scoped auth requirements

- Remove the bootstrap token flow for v1.
- Use a single user-provided username and password across the entire stack:
  - `apps/node-agent` authenticates with the user's username and password from environment variables
  - web app sessions authenticate with the same username and password
  - WebDAV and Finder authentication use the same username and password
- Do not generate separate WebDAV credentials for users.
- Nodes and exports must be owned by users and scoped so authenticated users can only view and mount their own resources.
- Package the node binary for user download and distribution.

## V1 simplicity

- Keep the implementation as simple as possible.
- Do not over-engineer the auth or distribution model for v1.
- Prefer the smallest change set that makes the product usable and distributable.

## Live operations

- If modifying the live Netcup deployment, only stop the `betternas` node process unless the user explicitly asks to modify the deployed backend service.

## Node availability UX

- Prefer default UI behavior that does not present disconnected nodes as mountable.
- Surface connected and disconnected node state in the product when node availability is exposed.
