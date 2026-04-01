## Context

aiNAS now has a verified local scaffold:
- a stock Nextcloud runtime
- a thin aiNAS app inside Nextcloud
- a minimal aiNAS control-plane service
- shared contracts

That scaffold proves the direction, but it does not yet answer the more important product questions. We need to decide what the actual backend is, what should remain delegated to Nextcloud, how the standalone product should take shape, and which work has the highest leverage.

The user goal is clear: build "file storage everywhere" with native-feeling access on devices, a strong web UX, sharing and RBAC, and room to rewrite product logic freely over time. The design therefore needs to preserve leverage from Nextcloud without letting it become the center of the product.

## Goals / Non-Goals

**Goals**
- Define which primitives we will adopt from Nextcloud versus own ourselves.
- Define the aiNAS control plane as the long-term system of record for product semantics.
- Define the first real backend domain at a high level.
- Define the first standalone web product direction outside Nextcloud.
- Define how device-native access fits into the architecture without forcing premature implementation.

**Non-Goals**
- Implement the backend or web UI in this change.
- Finalize every database field or endpoint shape.
- Decide final distribution details for desktop or mobile clients.
- Commit to a microservice topology.

## Core Decisions

### 1. aiNAS owns product semantics

aiNAS should be the system of record for:
- workspaces
- devices
- storage sources
- shares as product objects
- mount profiles
- policy and RBAC semantics
- audit and orchestration flows

Nextcloud should not become the long-term system of record for those concepts.

Rationale:
- keeps product evolution independent from the Nextcloud monolith
- supports future standalone clients and surfaces
- makes the Nextcloud adapter replaceable over time

### 2. Nextcloud remains an upstream substrate

Nextcloud is valuable because it already provides:
- external storage support for multiple backend types
- a strong web file and sharing surface
- WebDAV and OCS APIs
- a desktop client with Finder integration
- a mobile app

aiNAS should deliberately reuse those primitives where they reduce time-to-product, while keeping product-specific logic outside of Nextcloud.

Rationale:
- highest leverage path
- strongest reference implementations
- avoids rebuilding low-differentiation components too early

### 3. The control plane should remain a standalone aiNAS service

The current ExApp-compatible service is useful, but the long-term shape should be a standalone aiNAS backend that happens to have a Nextcloud adapter, not a backend that is conceptually trapped inside the AppAPI/ExApp model.

Rationale:
- the control plane should be reusable by future standalone clients
- aiNAS will likely want richer event, device, and auth flows than a pure Nextcloud extension mindset encourages
- keeping the service standalone reduces accidental architectural lock-in

### 4. Start with one modular backend service

Do not split into many services yet.

The first real backend should be a modular monolith with internal modules for:
- identity
- devices
- workspaces
- storage sources
- shares
- policies
- mount profiles
- audit/jobs

Rationale:
- keeps implementation velocity high
- preserves clear boundaries without creating distributed-system overhead
- still allows future extraction if needed

### 5. Prefer TypeScript for the first real control plane

Recommended stack:
- TypeScript backend
- Postgres for persistent product metadata
- Redis for ephemeral state and jobs
- Next.js for the standalone web surface

Rationale:
- matches the current repo direction
- minimizes coordination cost between API, contracts, and web
- keeps the first implementation path simple

### 6. Treat device-native mount orchestration as a separate later concern

There are two different product modes:

```text
Mode A: cloud-drive style
- Finder sidebar presence
- virtual files / file-provider behavior
- lower custom device work

Mode B: true remote mount
- explicit mounted filesystem
- login-time mount orchestration
- stronger device-native behavior
- more custom agent work
```

The design should keep both modes possible, but the first backend and web planning should not depend on solving Mode B immediately.

Rationale:
- Nextcloud already gives a stronger starting point for Mode A
- Mode B likely requires a custom agent and more device-specific work

## High-Level Architecture

```text
                         aiNAS target architecture

             browser / desktop / mobile / cli
                            |
                            v
                +---------------------------+
                | aiNAS control plane       |
                |---------------------------|
                | identity + sessions       |
                | workspaces                |
                | devices                   |
                | storage sources           |
                | shares                    |
                | mount profiles            |
                | policies                  |
                | audit + jobs              |
                +-------------+-------------+
                              |
             +----------------+----------------+
             |                                 |
             v                                 v
   +------------------------+       +-------------------------+
   | Nextcloud adapter      |       | device access layer     |
   |------------------------|       |-------------------------|
   | files / sharing / UI   |       | sync / mount behavior   |
   | external storage       |       | desktop/mobile helpers  |
   | webdav / OCS surfaces  |       | future native agent     |
   +-----------+------------+       +------------+------------+
               |                                 |
               +----------------+----------------+
                                |
                                v
                     +-------------------------+
                     | storage backends        |
                     | SMB NFS S3 WebDAV Local |
                     +-------------------------+
```

## High-Level Domain Contracts

The first control-plane design should center on these entities:

- `User`
- `Workspace`
- `Device`
- `StorageSource`
- `Share`
- `MountProfile`
- `Policy`
- `AuditEvent`

Relationship sketch:

```text
User
  |
  +-- Device
  |
  +-- Workspace
        |
        +-- StorageSource
        +-- Share
        +-- MountProfile
        +-- Policy
        +-- AuditEvent
```

The first API categories should likely map to those entities:
- `/api/me`
- `/api/workspaces`
- `/api/storage-sources`
- `/api/shares`
- `/api/devices`
- `/api/mount-profiles`
- `/api/policies`

This is intentionally high level. The exact path layout, auth model, and payload schema should be refined in later implementation changes.

## Risks / Trade-offs

- If we let Nextcloud users/groups become the permanent product identity model, aiNAS will be harder to evolve independently.
- If we overbuild the control plane before deciding which Nextcloud primitives we are actually keeping, we may duplicate useful substrate capabilities unnecessarily.
- If we force a custom device agent too early, we may spend time on native integration before proving the backend and product semantics.
- If we defer too much ownership to Nextcloud, the product may never fully become aiNAS.

## Sequencing

Recommended change sequence:

1. Define the Nextcloud substrate we are officially adopting.
2. Define the control-plane core domain and high-level API.
3. Build the first real control-plane backend around Postgres and Redis.
4. Build the standalone Next.js control-plane web surface.
5. Deepen the Nextcloud adapter.
6. Decide how much custom device access and mount logic is needed for v1 and v2.

## Open Questions

- Is v1 cloud-drive-first, mount-first, or explicitly hybrid?
- Which storage backends are in scope first?
- How much identity should be delegated to Nextcloud in v1?
- Should end users interact directly with Nextcloud in v1, or mostly through aiNAS surfaces?
- When, if ever, should the desktop or mobile clients be branded or forked?
