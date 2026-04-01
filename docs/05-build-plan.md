# betterNAS Part 5: Build Plan

This document ties the other four parts together.

It answers four questions:

- how the full system fits together
- where each part starts
- what we should steal from existing tools
- what we must write ourselves

## The full system

```text
                              betterNAS build plan

                                 [2] control plane
                         +--------------------------------+
                         | API + policy + registry + UI   |
                         +--------+---------------+-------+
                                  |               |
                     control/API  |               | cloud adapter
                                  v               v
                         [1] NAS node         [4] cloud/web layer
                    +------------------+      +-------------------+
                    | WebDAV + agent   |      | Nextcloud adapter |
                    | real storage     |      | browser/mobile    |
                    +---------+--------+      +---------+---------+
                              |                         ^
                              | mount profile           |
                              v                         |
                        [3] local device ---------------+
                    +----------------------+
                    | Finder mount/helper  |
                    | native user entry    |
                    +----------------------+
```

## The core rule

The control plane owns product semantics.

The other three parts are execution surfaces:

- the NAS node serves storage
- the local device mounts and uses storage
- the cloud/web layer exposes storage through browser and mobile-friendly flows

## What we steal vs write

| Part            | Steal first                                                                                 | Write ourselves                                                                            |
| --------------- | ------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| NAS node        | NixOS/Nix module patterns, existing WebDAV servers                                          | node agent, export model, node registration flow                                           |
| Control plane   | Go stdlib routing, pgx/sqlc, go-redis/asynq, OpenAPI codegen                                | product domain model, policy engine, mount/cloud APIs, registry                            |
| Local device    | Finder WebDAV mount, macOS Keychain, later maybe launch agent patterns                      | helper app, mount profile handling, auto-mount UX                                          |
| Cloud/web layer | Nextcloud server, Nextcloud shell app, Nextcloud share/file UI, Nextcloud mobile references | betterNAS integration layer, mapping between product model and Nextcloud, later branded UI |

## Where each part should start

## 1. NAS node

Start from:

- Nix flake / module
- a standard WebDAV server
- a very small agent process

Do not start by writing:

- custom storage protocol
- custom file server
- custom sync engine

The NAS node should be boring and reproducible.

## 2. Control plane

Start from:

- Go
- standard library routing first
- Postgres via `pgx` and `sqlc`
- Redis via `go-redis`
- OpenAPI-driven contracts
- standalone API mindset

Do not start by writing:

- microservices
- custom file transport
- a proxy that sits in the middle of every file transfer

This is the first real thing we should build.

## 3. Local device

Start from:

- native Finder `Connect to Server`
- WebDAV mount URLs issued by the control plane

Then later add:

- a lightweight helper app
- Keychain integration
- auto-mount at login

Do not start by writing:

- a full custom desktop sync client
- a Finder extension
- a new filesystem driver

## 4. Cloud / web layer

Start from:

- stock Nextcloud
- current shell app
- Nextcloud browser/share/mobile primitives

Then later add:

- betterNAS-specific integration pages
- standalone control-plane web UI
- custom branding or replacement UI where justified

Do not start by writing:

- a full custom browser file manager
- a custom mobile client
- a custom sharing stack

## Recommended build order

### Phase A: make the storage path real

1. NAS node can expose a directory over WebDAV
2. control plane can register the node and its exports
3. local device can mount that export in Finder

This is the shortest path to a real product loop.

### Phase B: make the product model real

1. add users, devices, NAS nodes, exports, grants, mount profiles
2. add auth and policy
3. add a simple standalone web UI for admin/control use

This is where betterNAS becomes its own product.

### Phase C: add cloud mode

1. connect the same storage into Nextcloud
2. expose browser/mobile/share flows
3. map Nextcloud behavior back to betterNAS product semantics

This is high leverage, but should not block Phase A.

## External parts we should deliberately reuse

### NAS node

- WebDAV server implementation
- Nix module patterns

### Control plane

- Go API service scaffold
- Postgres
- Redis

### Local device

- Finder's native WebDAV mounting
- macOS credential storage

### Cloud/web layer

- Nextcloud server
- Nextcloud app shell
- Nextcloud share/browser behavior
- Nextcloud mobile and desktop references

## From-scratch parts we should deliberately own

### NAS node

- node enrollment
- export registration
- machine identity and health reporting

### Control plane

- full backend domain model
- access and policy model
- mount profile generation
- cloud profile generation
- audit and registry

### Local device

- user-friendly mounting workflow
- helper app if needed
- local mount orchestration

### Cloud/web layer

- betterNAS-to-Nextcloud mapping layer
- standalone betterNAS product UI over time

## First scaffolds to use

| Part            | First scaffold                                                |
| --------------- | ------------------------------------------------------------- |
| NAS node        | Nix flake/module + WebDAV server service config               |
| Control plane   | Go service + OpenAPI contract + Postgres/Redis adapters later |
| Local device    | documented Finder mount flow, then lightweight helper app     |
| Cloud/web layer | current Nextcloud scaffold and shell app                      |

## What not to overbuild early

- custom sync engine
- custom desktop client
- custom mobile app
- many backend services
- control-plane-in-the-data-path file proxy

Those can come later if the simpler stack proves insufficient.

## Build goal for V1

V1 should prove one clean loop:

```text
user picks NAS export in betterNAS UI
-> control plane issues mount profile
-> local device mounts WebDAV export
-> user sees and uses files in Finder
-> optional Nextcloud surface exposes the same storage in cloud mode
```

If that loop works, the architecture is sound.

## TODO

- Choose the exact WebDAV server for the NAS node.
- Decide the first Nix module layout for node installation.
- Define the first database-backed control-plane entities.
- Decide whether the local device starts as documentation-only or a helper app.
- Decide when the Nextcloud cloud/web layer becomes user-facing in v1.
