## 1. Repository and local platform scaffold

- [x] 1.1 Create the top-level repository structure for `docker/`, `apps/`, `exapps/`, `packages/`, `docs/`, and `scripts/`
- [x] 1.2 Add a Docker Compose development stack for vanilla Nextcloud and its required backing services
- [x] 1.3 Add the aiNAS control-plane service container to the local development stack
- [x] 1.4 Add repeatable developer scripts and documentation for booting and stopping the local stack

## 2. Nextcloud shell app scaffold

- [x] 2.1 Generate the aiNAS Nextcloud app scaffold into `apps/betternas-controlplane/`
- [x] 2.2 Configure the shell app with aiNAS branding, navigation entry points, and basic settings surface
- [x] 2.3 Add an adapter layer in the shell app for calling aiNAS-owned service endpoints
- [x] 2.4 Verify the shell app installs and loads in the local Nextcloud runtime

## 3. Control-plane service scaffold

- [x] 3.1 Scaffold the aiNAS control-plane service in `exapps/control-plane/`
- [x] 3.2 Add a minimal internal HTTP API surface with health and version endpoints
- [x] 3.3 Create a dedicated Nextcloud adapter boundary inside the service for backend integrations
- [x] 3.4 Wire local service configuration so the shell app can discover and call the control-plane service

## 4. Shared contracts and verification

- [x] 4.1 Create the shared contracts package for internal API schemas and payload definitions
- [x] 4.2 Define the initial contracts used between the shell app and the control-plane service
- [x] 4.3 Document the architectural boundary that keeps business logic out of the Nextcloud app
- [x] 4.4 Verify end-to-end local startup with Nextcloud, the shell app, and the control-plane service all reachable
