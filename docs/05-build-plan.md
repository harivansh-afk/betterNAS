# betterNAS Part 5: Build Plan

This document ties the other four parts together.

It answers four questions:

- how the full system fits together
- where each part starts
- what we should steal from existing tools
- what we must write ourselves

## The full system

```text
                           self-hosted betterNAS

                            [3] web control plane
                    +--------------------------------+
                    | onboarding / management / UX   |
                    +---------------+----------------+
                                    |
                                    v
                            [2] control-server
                    +--------------------------------+
                    | auth / nodes / exports         |
                    | grants / mount profiles        |
                    +---------------+----------------+
                                    |
                                    v
                             [1] node-service
                    +--------------------------------+
                    | WebDAV + export runtime        |
                    | real storage                   |
                    +---------------+----------------+
                                    ^
                                    |
                              [4] local device
                    +--------------------------------+
                    | browser + Finder mount         |
                    +--------------------------------+

 optional later:
 - Nextcloud adapter
 - hosted control plane
 - hosted web UI
```

## The core rule

`control-server` owns product semantics.

The other three parts are execution surfaces:

- `node-service` serves storage
- `web control plane` exposes management and mount UX
- `local device` consumes the issued mount flow

## What we steal vs write

| Part              | Steal first                                                       | Write ourselves                                                     |
| ----------------- | ----------------------------------------------------------------- | ------------------------------------------------------------------- |
| node-service      | Go WebDAV primitives, Docker packaging, later Nix module patterns | node runtime, export model, node enrollment                         |
| control-server    | Go stdlib routing, pgx/sqlc, Redis helpers, OpenAPI codegen       | product domain model, policy engine, mount APIs, registry           |
| web control plane | Next.js app conventions, shared UI primitives                     | product UI, onboarding, node/export flows, mount UX                 |
| local device      | Finder WebDAV mount flow, macOS Keychain later                    | helper app or mount launcher later                                  |
| optional adapter  | Nextcloud server and app template                                 | betterNAS mapping layer if we decide to keep a cloud/mobile surface |

## Where each part should start

## 1. node-service

Start from:

- one Go binary
- one export root
- one WebDAV surface
- one deployable self-hosted runtime

Do not start by writing:

- a custom storage protocol
- a custom sync engine
- a complex relay stack

## 2. control-server

Start from:

- Go
- one API
- one durable data model
- node registration and mount profile issuance

Do not start by writing:

- microservices
- file proxying by default
- hosted-only assumptions

## 3. web control plane

Start from:

- sign in
- list nodes and exports
- show mount URL and mount instructions

Do not start by writing:

- a large browser file manager
- a second backend hidden inside Next.js

## 4. local device

Start from:

- Finder `Connect to Server`
- WebDAV mount URL issued by `control-server`

Then later add:

- one-click helper
- Keychain integration
- auto-mount at login

## Recommended build order

### Phase A: make the self-hosted mount path real

1. node-service exposes a directory over WebDAV
2. control-server registers the node and its exports
3. web control plane shows the export and mount action
4. local device mounts the export in Finder

### Phase B: make the product real

1. add durable users, nodes, exports, grants, mount profiles
2. add auth and token lifecycle
3. add a proper web UI for admin and user control flows

### Phase C: make deployment real

1. define Docker self-hosting shape
2. define Nix-based NAS host install shape
3. define remote access story for non-local usage

### Phase D: add optional adapter surfaces

1. add Nextcloud only if browser/share/mobile value justifies it
2. keep it out of the critical mount path

## Build goal for V1

V1 should prove one clean loop:

```text
user opens betterNAS web UI
-> sees a registered export
-> requests mount instructions
-> Finder mounts the WebDAV export
-> user sees and uses files from the NAS
```
