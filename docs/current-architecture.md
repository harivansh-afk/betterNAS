# betternas Architecture Boundary

## Core Decision

betternas treats Nextcloud as an upstream backend, not as the place where betternas product logic should accumulate.

That leads to three explicit boundaries:

1. `apps/betternas-controlplane/` is a thin shell inside Nextcloud.
2. `exapps/control-plane/` owns betternas business logic and internal APIs.
3. `packages/contracts/` defines the interface between the shell app and the control plane.

## Why This Boundary Exists

Forking `nextcloud/server` would force betternas to own upstream patching and compatibility work too early. Pushing betternas logic into a traditional Nextcloud app would make the product harder to evolve outside the PHP monolith. The scaffold in this repository is designed to avoid both traps.

## Responsibilities

### Nextcloud shell app

The shell app is responsible for:
- navigation entries
- branded entry pages inside Nextcloud
- admin-facing integration surfaces
- adapter calls into the betternas control plane

The shell app is not responsible for:
- storage policy rules
- orchestration logic
- betternas-native RBAC decisions
- product workflows that may later be reused by desktop, iOS, or standalone web clients

### Control-plane service

The control plane is responsible for:
- domain logic
- policy decisions
- internal APIs consumed by betternas surfaces
- Nextcloud integration adapters kept at the service boundary

### Shared contracts

Contracts live in `packages/contracts/` so request and response shapes do not get duplicated between PHP and TypeScript codebases.

## Local Runtime

The local development stack uses Docker Compose so developers can bring up:
- Nextcloud
- PostgreSQL
- Redis
- the betternas control-plane service

The Nextcloud shell app is mounted as a custom app and enabled through `./scripts/dev-up`.

