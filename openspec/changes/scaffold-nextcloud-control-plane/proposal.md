## Why

betternas needs an initial architecture and repository scaffold that lets us build our own storage control plane without inheriting the maintenance cost of a Nextcloud core fork. We want to move quickly on product-specific business logic, but still stand on top of a mature backend for files, sync, sharing primitives, and existing clients.

## What Changes

- Create an initial betternas platform scaffold centered on vanilla Nextcloud running in Docker for local development.
- Define a thin Nextcloud app shell that owns betternas-specific integration points, branded surfaces, and adapters into the Nextcloud backend.
- Define a control-plane service boundary where betternas business logic, policy, and future orchestration will live outside the Nextcloud monolith.
- Establish a repository layout for Docker infrastructure, Nextcloud app code, ExApp/service code, and shared API contracts.
- Document the decision to treat Nextcloud as an upstream backend dependency rather than a forked application baseline.

## Capabilities

### New Capabilities
- `workspace-scaffold`: Repository structure and local development platform for running betternas with Nextcloud, service containers, and shared packages.
- `nextcloud-shell-app`: Thin betternas app inside Nextcloud for navigation, settings, branded entry points, and backend integration hooks.
- `control-plane-service`: External betternas service layer that owns business logic and exposes internal APIs used by the Nextcloud shell and future clients.

### Modified Capabilities
- None.

## Impact

- Affected code: new repository layout under `docker/`, `apps/`, `exapps/`, `packages/`, `docs/`, and `scripts/`
- Affected systems: local developer workflow, Docker-based service orchestration, Nextcloud runtime, AppAPI/ExApp integration path
- Dependencies: Nextcloud, Docker Compose, AppAPI/ExApps, shared contract definitions
- APIs: new internal control-plane APIs and service boundaries for future desktop, iOS, and web clients
