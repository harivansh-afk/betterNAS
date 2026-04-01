# betterNAS

<img width="723" height="354" alt="image" src="https://github.com/user-attachments/assets/4e64fa91-315b-4a31-b191-d54ed1862ff7" />

## Start here

The canonical repo contract lives in [docs/architecture.md](/home/rathi/Documents/GitHub/betterNAS/docs/architecture.md).

Read these in order:

1. [docs/architecture.md](/home/rathi/Documents/GitHub/betterNAS/docs/architecture.md)
2. [docs/01-nas-node.md](/home/rathi/Documents/GitHub/betterNAS/docs/01-nas-node.md)
3. [docs/02-control-plane.md](/home/rathi/Documents/GitHub/betterNAS/docs/02-control-plane.md)
4. [docs/03-local-device.md](/home/rathi/Documents/GitHub/betterNAS/docs/03-local-device.md)
5. [docs/04-cloud-web-layer.md](/home/rathi/Documents/GitHub/betterNAS/docs/04-cloud-web-layer.md)
6. [docs/05-build-plan.md](/home/rathi/Documents/GitHub/betterNAS/docs/05-build-plan.md)
7. [docs/references.md](/home/rathi/Documents/GitHub/betterNAS/docs/references.md)

## Current direction

- betterNAS is WebDAV-first for mount mode.
- the control plane is the system of record.
- the NAS node serves bytes directly whenever possible.
- Nextcloud is an optional cloud/web adapter, not the product center.

## Monorepo

- `apps/web`: Next.js control-plane UI
- `apps/control-plane`: Go control-plane service
- `apps/node-agent`: Go NAS runtime / WebDAV node
- `apps/nextcloud-app`: optional Nextcloud adapter
- `packages/contracts`: canonical shared contracts
- `packages/sdk-ts`: TypeScript SDK surface for the web app
- `packages/ui`: shared React UI
- `infra/docker`: local Docker runtime

The root planning and delegation guide lives in [skeleton.md](/home/rathi/Documents/GitHub/betterNAS/skeleton.md).
