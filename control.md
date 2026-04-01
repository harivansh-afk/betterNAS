# Control

This repo is the coordination and implementation ground for betterNAS.

Use it for:

- shared contracts
- architecture and planning docs
- runtime scripts
- stack verification
- implementation of the self-hosted stack

## Current product focus

The default betterNAS product is:

- self-hosted on the user's NAS
- WebDAV-first
- Finder-mountable
- managed through a web control plane

The main parts are:

- `node-service`
  - `apps/node-agent`
- `control-server`
  - `apps/control-plane`
- `web control plane`
  - `apps/web`
- `optional cloud adapter`
  - `apps/nextcloud-app`

## Rules

- shared interface changes land in `packages/contracts` first
- `docs/architecture.md` is the canonical architecture contract
- the self-hosted mount flow is the critical path
- optional Nextcloud work must not drive the main architecture

## Command surface

- `pnpm verify`
  - static verification
- `pnpm stack:up`
  - boot the self-hosted stack
- `pnpm stack:verify`
  - verify the working stack
- `pnpm stack:down --volumes`
  - tear the stack down cleanly
- `pnpm agent:verify`
  - bootstrap, verify, boot, and stack-verify in one loop
