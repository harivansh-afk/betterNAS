# betterNAS Architecture Contract

This file is the canonical contract for the repository.

If the planning docs, scaffold code, or future tasks disagree, this file and
[`packages/contracts`](../packages/contracts)
win.

## The single first task

Before splitting work across agents, do one foundation task:

- scaffold the four product parts
- lock the shared contracts
- define one end-to-end verification loop
- enforce clear ownership boundaries

That first task should leave the repo in a state where later work can be
parallelized without interface drift.

## The four parts

```text
                         betterNAS canonical contract

                           [2] control plane
                  +-----------------------------------+
                  | system of record                  |
                  | users / devices / nodes / grants  |
                  | mount profiles / cloud profiles   |
                  +---------+---------------+---------+
                            |               |
             control/API    |               | cloud adapter
                            v               v
                  [1] NAS node          [4] cloud / web layer
             +-------------------+      +----------------------+
             | WebDAV + node     |      | Nextcloud adapter    |
             | real file bytes   |      | browser / mobile     |
             +---------+---------+      +----------+-----------+
                       |                           ^
                       | mount profile             |
                       v                           |
                   [3] local device --------------+
             +----------------------+
             | Finder mount/helper  |
             | native user entry    |
             +----------------------+
```

## Non-negotiable rules

1. The control plane is the system of record.
2. File bytes should flow as directly as possible between the NAS node and the
   local device.
3. The control plane should issue policy, grants, and profiles. It should not
   become the default file proxy.
4. The NAS node should serve WebDAV directly whenever possible.
5. The local device consumes mount profiles. It does not hardcode infra details.
6. The cloud/web layer is optional and secondary. Nextcloud is an adapter, not
   the product center.

## Canonical sources of truth

Use these in this order:

1. [`docs/architecture.md`](./architecture.md)
   for boundaries, ownership, and delivery rules
2. [`packages/contracts`](../packages/contracts)
   for machine-readable types, schemas, and route constants
3. the part docs for local detail:
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

Every parallel task should primarily stay inside one of those lanes unless it is
an explicit contract task.

## The contract surface we need first

The first shared contract set should cover only the seams that let all four
parts exist at once.

### NAS node -> control plane

- node registration
- node heartbeat
- export inventory

### Local device -> control plane

- list allowed exports
- issue mount profile

### Cloud/web layer -> control plane

- issue cloud profile
- read export metadata

### Control plane internal

- health
- version
- the first domain entities:
  - `NasNode`
  - `StorageExport`
  - `AccessGrant`
  - `MountProfile`
  - `CloudProfile`

## Parallel work boundaries

Each area gets an owner and a narrow write surface.

| Part            | Owns                                             | May read                      | Must not own                   |
| --------------- | ------------------------------------------------ | ----------------------------- | ------------------------------ |
| NAS node        | node runtime, export reporting, WebDAV config    | contracts, control-plane docs | product policy                 |
| Control plane   | domain model, grants, profile issuance, registry | everything                    | direct file serving by default |
| Local device    | mount UX, helper flows, credential handling      | contracts, control-plane docs | access policy                  |
| Cloud/web layer | Nextcloud adapter, browser/mobile integration    | contracts, control-plane docs | source of truth                |

The only shared write surface across teams should be:

- [`packages/contracts`](../packages/contracts)
- this file when the architecture contract changes

## Verification loop

This is the first loop every scaffold and agent should target.

```text
[1] mock or real NAS node exposes a WebDAV export
-> [2] control plane registers the node and export
-> [3] local device asks for a mount profile
-> [3] local device receives a WebDAV mount URL
-> user can mount the export in Finder
-> [4] optional cloud/web layer can expose the same export in cloud mode
```

If a task does not help one of those steps become real, it is probably too
early.

## Definition of done for the foundation scaffold

The initial scaffold is complete when:

- all four parts have a documented entry point
- the control plane can represent nodes, exports, grants, and profiles
- the contracts package exports the first shared shapes and schemas
- local verification can prove the mount-profile loop end to end
- future agents can work inside one part without inventing new interfaces

## Rules for future tasks and agents

1. No part may invent private request or response shapes for shared flows.
2. Contract changes must update
   [`packages/contracts`](../packages/contracts)
   first.
3. Architecture changes must update this file in the same change.
4. Additive contract changes are preferred over breaking ones.
5. New tasks should target one part at a time unless they are explicitly
   contract tasks.
