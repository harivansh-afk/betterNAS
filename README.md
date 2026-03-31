# aiNAS

aiNAS is a storage control-plane project that uses vanilla Nextcloud as an upstream backend instead of forking the core server. This repository starts with the foundational pieces we need to build our own product surfaces while keeping file primitives, sync compatibility, and existing client integrations delegated to Nextcloud.

## Repository Layout

- `docker/`: local development runtime for Nextcloud and aiNAS services
- `apps/ainas-controlplane/`: thin Nextcloud shell app
- `exapps/control-plane/`: aiNAS-owned control-plane service
- `packages/contracts/`: shared API contracts used by aiNAS services and adapters
- `docs/`: architecture and development notes
- `scripts/`: repeatable developer workflows

## Local Development

Requirements:
- Docker with Compose support
- Node.js 22+
- npm 10+

Bootstrap the JavaScript workspace:

```bash
npm install
```

Start the local stack:

```bash
./scripts/dev-up
```

Stop the local stack:

```bash
./scripts/dev-down
```

Once the stack is up:
- Nextcloud: `http://localhost:8080`
- aiNAS control plane: `http://localhost:3001`

The `dev-up` script waits for Nextcloud installation to finish and then enables the `ainascontrolplane` custom app inside the container.

## Architecture

The intended boundary is documented in `docs/architecture.md`. The short version is:

- Nextcloud remains an upstream storage and client-compatibility backend.
- The custom Nextcloud app is a shell and adapter layer.
- aiNAS business logic lives in the control-plane service.
