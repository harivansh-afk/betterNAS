# betterNAS Architecture Contract

This file is the canonical contract for the repository.

If the planning docs, scaffold code, or future tasks disagree, this file and
[`packages/contracts`](../packages/contracts) win.

## Product default

betterNAS is self-hosted first.

For the current product shape, the user should be able to run the whole stack on
their NAS machine:

- `node-service` serves the real files over WebDAV
- `control-server` owns auth, nodes, exports, grants, and mount profiles
- `web control plane` is the browser UI over the control-server
- the local device mounts an issued WebDAV URL in Finder

Optional hosted deployments can come later. Optional Nextcloud integration can
come later.

## The core system

```text
                     betterNAS canonical contract

                       self-hosted on user's NAS

                 +--------------------------------------+
                 | [2] control-server                  |
                 | system of record                    |
                 | auth / nodes / exports / grants     |
                 | mount sessions / audit              |
                 +------------------+-------------------+
                                    |
                                    v
                 +--------------------------------------+
                 | [1] node-service                     |
                 | WebDAV export runtime                |
                 | real file bytes                      |
                 +------------------+-------------------+
                                    ^
                                    |
                 +------------------+-------------------+
                 | [3] web control plane               |
                 | onboarding / management / mount UX  |
                 +------------------+-------------------+
                                    ^
                                    |
                              user browser

 user local device
   |
   +-----------------------------------------------> Finder mount
                                                    via issued WebDAV URL

 [4] optional cloud adapter
   |
   +--> secondary browser/mobile/share layer
        not part of the core mount path
```

## Non-negotiable rules

1. `control-server` is the system of record.
2. `node-service` serves the bytes.
3. `web control plane` is a UI over `control-server`, not a second policy
   backend.
4. The main data path should be `local device <-> node-service` whenever
   possible.
5. `control-server` should issue access, grants, and mount profiles. It should
   not become the default file proxy.
6. The self-hosted stack should work without Nextcloud.
7. Nextcloud, if used, is an optional adapter and secondary surface.

## Canonical sources of truth

Use these in this order:

1. [`docs/architecture.md`](./architecture.md)
   for boundaries, ownership, and delivery rules
2. [`packages/contracts`](../packages/contracts)
   for machine-readable types, schemas, and route constants
3. the part docs:
   - [`docs/01-nas-node.md`](./01-nas-node.md)
   - [`docs/02-control-plane.md`](./02-control-plane.md)
   - [`docs/03-local-device.md`](./03-local-device.md)
   - [`docs/04-cloud-web-layer.md`](./04-cloud-web-layer.md)
   - [`docs/05-build-plan.md`](./05-build-plan.md)

## Repo lanes

The monorepo is split into these primary implementation lanes:

- [`apps/node-agent`](../apps/node-agent)
- [`apps/control-plane`](../apps/control-plane)
- [`apps/web`](../apps/web)
- [`apps/nextcloud-app`](../apps/nextcloud-app)
- [`packages/contracts`](../packages/contracts)

The first three are core. `apps/nextcloud-app` is optional and should not drive
the main architecture.

## The contract surface we need first

The first shared contract set should cover only the seams needed for the
self-hosted mount flow.

### Node-service -> control-server

- node registration
- node heartbeat
- export inventory via a dedicated sync endpoint

### Web control plane -> control-server

- auth/session bootstrapping
- list nodes and exports
- issue mount profile
- issue share or cloud profile later

### Local device -> control-server

- fetch mount instructions
- receive issued WebDAV URL and standard WebDAV credentials
  - username
  - password
  - expiresAt

## Initial backend route sketch

The first backend contract should stay narrow:

- `POST /api/v1/nodes/register`
- `POST /api/v1/nodes/{nodeId}/heartbeat`
- `PUT /api/v1/nodes/{nodeId}/exports`
- `GET /api/v1/exports`
- `POST /api/v1/mount-profiles/issue`

### Control-server internal

- health
- version
- the first domain entities:
  - `NasNode`
  - `StorageExport`
  - `AccessGrant`
  - `MountProfile`
  - `AuditEvent`

## Parallel work boundaries

| Part              | Owns                                             | May read                       | Must not own                   |
| ----------------- | ------------------------------------------------ | ------------------------------ | ------------------------------ |
| node-service      | NAS runtime, WebDAV serving, export reporting    | contracts, control-server docs | product policy                 |
| control-server    | domain model, grants, profile issuance, registry | everything                     | direct file serving by default |
| web control plane | onboarding, node/export management, mount UX     | contracts, control-server docs | source of truth                |
| optional adapter  | Nextcloud mapping and cloud surfaces             | contracts, control-server docs | core mount path                |

The shared write surface across parts should stay narrow:

- [`packages/contracts`](../packages/contracts)
- this file when architecture changes

## Verification loop

This is the main loop every near-term task should support.

```text
[node-service]
  serves a WebDAV export
        |
        v
[control-server]
  registers the node and export
  issues a mount profile
        |
        v
[web control plane]
  shows the export and mount action
        |
        v
[local device]
  mounts the issued WebDAV URL in Finder
```

If a task does not make one of those steps more real, it is probably too early.

## Definition of done for the current foundation

The current foundation is in good shape when:

- the self-hosted stack boots locally
- the control-server can represent nodes, exports, grants, and mount profiles
- the node-service serves a real WebDAV export
- the web control plane can expose the mount flow
- a local Mac can mount the export in Finder

## Rules for future tasks and agents

1. No part may invent private request or response shapes for shared flows.
2. Contract changes must update [`packages/contracts`](../packages/contracts)
   first.
3. Architecture changes must update this file in the same change.
4. Additive contract changes are preferred over breaking ones.
5. Prioritize the self-hosted mount loop before optional cloud/mobile work.
