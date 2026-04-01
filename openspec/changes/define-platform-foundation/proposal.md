## Why

The scaffold phase is complete, but the repo still lacks a clear product-level plan for how betternas should evolve beyond a thin Nextcloud shell and a minimal backend stub. Before implementing more code, we need planning artifacts that define the system of record, the role Nextcloud should play, the control-plane domain we intend to own, and the delivery sequence for the first real product capabilities.

## What Changes

- Add a north-star planning layer for the betternas platform.
- Define the Nextcloud substrate that betternas will adopt instead of rebuilding from scratch.
- Define the first high-level control-plane domain model and service contracts.
- Define the intended standalone web control-plane direction outside Nextcloud.
- Define the future device access layer boundary, including the distinction between cloud-drive style access and true remote mounts.

## Capabilities

### New Capabilities
- `nextcloud-substrate`: A documented contract for which server, storage, sharing, and client primitives betternas will adopt from Nextcloud.
- `control-plane-core`: A documented contract for the first real betternas backend domain model and API surface.
- `standalone-control-plane-web`: A documented contract for the future Next.js web console that will become the betternas product surface outside Nextcloud.
- `device-access-layer`: A documented contract for how betternas will eventually support native device access, including mount and sync modes.

### Modified Capabilities
- None.

## Impact

- Affected code: planning artifacts only
- Affected systems: architecture, product sequencing, service boundaries, implementation roadmap
- Dependencies: Nextcloud server, Nextcloud desktop/iOS references, betternas control-plane service, future Postgres/Redis-backed product metadata
- APIs: establishes the intended betternas-owned API and adapter boundaries at a high level before implementation
