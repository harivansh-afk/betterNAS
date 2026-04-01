# aiNAS Platform Foundation

This document is the north-star planning artifact for the next phase of aiNAS.

The scaffold phase is done. We now have a verified local Nextcloud runtime, a thin Nextcloud shell app, and a minimal aiNAS control-plane service. The next phase is about deciding what we will steal from Nextcloud, what we will own ourselves, and how the product should evolve without turning Nextcloud into the center of the system.

## Product Stance

aiNAS is not "a custom Nextcloud theme."

aiNAS is a storage control plane with:
- its own product semantics
- its own API
- its own web surface
- its own future device and mount model

Nextcloud remains valuable, but it is an upstream substrate and reference implementation, not the long-term system of record for the product.

## High-Level Model

```text
                           aiNAS platform model

         users / browser / desktop / mobile / cli
                          |
                          v
               +--------------------------+
               | aiNAS control plane      |
               |--------------------------|
               | identity                 |
               | workspaces               |
               | devices                  |
               | storage sources          |
               | shares                   |
               | mount profiles           |
               | policies                 |
               | audit + jobs             |
               +------------+-------------+
                            |
             +--------------+---------------+
             |                              |
             v                              v
   +----------------------+      +------------------------+
   | Nextcloud adapter    |      | device access layer    |
   |----------------------|      |------------------------|
   | files/share/web UI   |      | desktop / mobile / cli |
   | external storage     |      | native mount/sync      |
   | webdav / user shell  |      | device orchestration   |
   +----------+-----------+      +-----------+------------+
              |                              |
              +---------------+--------------+
                              |
                              v
                   +-------------------------+
                   | storage backends        |
                   | SMB NFS S3 WebDAV Local |
                   +-------------------------+
```

## What We Steal From Nextcloud

We should deliberately reuse as much as possible before building our own equivalents.

### Server-side primitives

Nextcloud is a strong source of reusable server-side primitives:
- external storage support for Amazon S3, FTP/FTPS, Local, Nextcloud, OpenStack Object Storage, SFTP, SMB/CIFS, and WebDAV
- web file UI and sharing UI
- WebDAV and OCS/external API surfaces
- built-in user/session/admin runtime
- branding and theming hooks
- custom app shell inside the main web experience

### Client-side references

Nextcloud is also a strong source of reference implementations:
- desktop client for macOS and other desktop platforms
- macOS Virtual Files / Finder integration
- iOS app

The desktop client is especially high leverage because the official docs describe it as appearing as a dedicated location in the Finder sidebar, with offline controls, file previews, sharing, server-side actions, and automatic change detection.

## What aiNAS Must Own

Even with heavy reuse, the product still needs its own control plane.

aiNAS should own:
- the domain model for users, devices, workspaces, storage sources, shares, policies, and mount profiles
- the product API used by future web, desktop, and mobile surfaces
- device and mount orchestration semantics
- product-specific RBAC and policy logic
- audit and operational workflows

Nextcloud should not become the system of record for those concerns.

## Recommended Technical Shape

### 1. Control plane

Start with one modular service, not many microservices.

Recommended stack:
- TypeScript for the control-plane API
- Postgres for product metadata
- Redis for cache and background work

Why:
- the repo already uses TypeScript for contracts and the initial control-plane service
- a modular monolith keeps boundaries explicit without premature service sprawl
- Postgres and Redis fit the operational model well

### 2. Web product

Build a standalone control-plane web app outside Nextcloud.

Recommended stack:
- Next.js

Why:
- it pairs well with a TypeScript backend
- it gives us a real product web surface instead of living forever inside the Nextcloud shell
- the Nextcloud app can remain a thin adapter and optional embedded surface

### 3. Device layer

Treat device-native mounts and sync as a separate concern from the control plane.

Recommended direction:
- defer a custom device agent until we know we need true mount orchestration
- when we do need one, prefer Go for the first device-side daemon

Why:
- Finder-style cloud-drive presence can be heavily referenced from Nextcloud desktop first
- true mount-at-login behavior is a different product problem and likely needs its own agent

## Product Modes

There are two product shapes hidden inside the idea. We should be explicit about them.

### Mode A: cloud-drive style

Characteristics:
- Finder presence
- virtual files / files-on-demand
- sync-like behavior
- lower custom device work

This mode aligns well with Nextcloud desktop and should be the first reference path.

### Mode B: true remote mount

Characteristics:
- explicit filesystem mount
- login-time mount orchestration
- stronger native OS coupling
- more custom device-side work

This mode should be treated as a later capability unless it becomes the immediate core differentiator.

## System of Record

The working assumption for the next phase is:

- aiNAS is the system of record for product semantics
- Nextcloud is the upstream file/share/storage substrate

This keeps the architecture clean and avoids accidentally letting the adapter become the product.

## Recommended Delivery Sequence

1. Define the Nextcloud substrate we are officially adopting.
2. Define the aiNAS control-plane domain model and API.
3. Build the first real control-plane backend around Postgres and Redis.
4. Build a standalone Next.js control-plane web app.
5. Deepen the Nextcloud adapter so it mirrors aiNAS-owned semantics.
6. Only then decide how much custom device and mount orchestration we need.

## Decision Matrix

| Area | Use Nextcloud first | Own in aiNAS |
|---|---|---|
| file and sharing web UX | yes | later, only if needed |
| storage backend aggregation | yes | overlay policy, source catalog, and orchestration |
| macOS Finder-style cloud presence | yes, reference desktop client first | later, if branded/native client is required |
| iOS app | yes, reference Nextcloud iOS first | later, if branded/native client is required |
| product API | no | yes |
| device model | no | yes |
| mount model | no | yes |
| policy / RBAC semantics | baseline from Nextcloud is acceptable | real product semantics belong in aiNAS |
| admin/control UI | partial in Nextcloud | full standalone control plane should be ours |

## Open Questions

These questions should be explicitly resolved in the next planning change:

- Is v1 cloud-drive-first, mount-first, or hybrid?
- Which storage backends are in scope first: SMB, S3, WebDAV, local, or all of them?
- What should aiNAS identity own in v1 versus what should be delegated to Nextcloud users/groups?
- Should Nextcloud remain part of the end-user workflow in v1, or mostly act as a backend adapter?
- When do we fork or brand the desktop and mobile clients, if ever?

## Reference Sources

- Nextcloud AppAPI / ExApps
- Nextcloud external storage administration docs
- Nextcloud external API / OCS docs
- Nextcloud theming and branded client links
- Nextcloud macOS Virtual Files docs
- public Nextcloud desktop and iOS repositories
