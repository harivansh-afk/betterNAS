# `@betternas/contracts`

This package is the machine-readable source of truth for shared interfaces in
betterNAS.

Use it to keep the four product parts aligned:

- NAS node
- control plane
- local device
- cloud/web layer

## What belongs here

- OpenAPI source documents
- shared TypeScript types
- route constants
- JSON schemas for payloads we want to validate outside TypeScript

## What does not belong here

- business logic
- per-service config
- implementation-specific helpers

## Current contract layers

- [`src/control-plane.ts`](./src/control-plane.ts)
  - current runtime scaffold for health and version
- [`src/foundation.ts`](./src/foundation.ts)
  - first product-level entities and route constants for node, mount, and cloud flows
- [`openapi/`](./openapi)
  - language-neutral source documents for future SDK generation
- [`schemas/`](./schemas)
  - JSON schema mirrors for the first shared entities

## Change rules

1. Shared API shape changes happen here first.
2. If the boundary changes, also update
   [`docs/architecture.md`](../../docs/architecture.md).
3. Prefer additive changes until all four parts are live.
4. Do not put Nextcloud-only assumptions into the core contracts unless the
   field is explicitly part of the cloud adapter.
5. Keep the first version narrow. Over-modeling early is another form of drift.
