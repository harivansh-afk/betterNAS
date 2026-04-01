## Context

betterNAS is starting as a greenfield project with a clear product boundary: we want our own storage control plane, product UX, and business logic while using Nextcloud as an upstream backend for file storage, sync primitives, sharing primitives, and existing client compatibility. The repository is effectively empty today, so this change needs to establish both the architectural stance and the initial developer scaffold.

The main constraint is maintenance ownership. Forking `nextcloud/server` would move security patches, upstream upgrade churn, and internal compatibility risk onto betterNAS too early. At the same time, pushing all product logic into a traditional Nextcloud app would make our business rules hard to evolve and tightly couple the product to the PHP monolith. The design therefore needs to leave us with a thin in-Nextcloud surface and a separate betterNAS-owned service layer.

## Goals / Non-Goals

**Goals:**
- Create a repository scaffold that supports local development with vanilla Nextcloud and betterNAS-owned services.
- Define a thin Nextcloud shell app that handles navigation, branded entry points, and backend integration hooks.
- Define an betterNAS control-plane service boundary for business logic, policy, and orchestration.
- Keep interfaces typed and explicit so future web, desktop, and iOS clients can target betterNAS services rather than Nextcloud internals.
- Make the initial architecture easy to extend without forcing a Nextcloud core fork.

**Non-Goals:**
- Implement end-user storage features such as mounts, sync semantics, or sharing workflows in this change.
- Build custom desktop or iOS clients in this change.
- Replace Nextcloud's file storage, sync engine, or existing client stack.
- Finalize long-term production deployment topology or multi-node scaling.

## Decisions

### 1. Use vanilla Nextcloud as an upstream backend, not a fork

betterNAS will run a stock Nextcloud instance in local development and future environments. We will extend it through a dedicated betterNAS app and service integrations instead of modifying core server code.

Rationale:
- Keeps upstream upgrades and security patches tractable.
- Lets us reuse mature file storage and client compatibility immediately.
- Preserves an exit ramp if we later replace parts of the backend.

Alternatives considered:
- Fork `nextcloud/server`: rejected due to long-term maintenance cost.
- Build a custom storage platform first: rejected because it delays product iteration on higher-value workflows.

### 2. Keep the Nextcloud app thin and treat it as an adapter shell

The generated Nextcloud app will own betterNAS-specific UI entry points inside Nextcloud, settings pages, and integration hooks, but SHALL not become the home of core business logic. It will call betterNAS-owned APIs/services for control-plane decisions.

Rationale:
- Keeps PHP app code small and replaceable.
- Makes future non-Nextcloud clients first-class instead of afterthoughts.
- Allows us to rewrite business logic without continually reshaping the shell app.

Alternatives considered:
- Put most logic directly in the app: rejected because it couples product evolution to the monolith.

### 3. Scaffold an betterNAS control-plane service from the start

The repo will include a control-plane service that exposes internal HTTP APIs, owns domain models, and encapsulates policy and orchestration logic. In the first scaffold, this service may be packaged in an ExApp-compatible container, but the code structure SHALL keep Nextcloud-specific integration at the boundary rather than in domain logic.

Rationale:
- Matches the product direction of betterNAS owning the control plane.
- Gives one place for RBAC, storage policy, and orchestration logic to live.
- Supports future desktop, iOS, and standalone web surfaces without coupling them to Nextcloud-rendered pages.

Alternatives considered:
- Delay service creation and start with only a Nextcloud app: rejected because it encourages logic to accumulate in the wrong place.
- Build multiple services immediately: rejected because one control-plane service is enough to establish the boundary.

### 4. Use a monorepo with explicit top-level boundaries

The initial scaffold will create clear top-level directories for infrastructure, app code, service code, shared contracts, docs, and scripts. The exact framework choices inside those directories can evolve, but the boundary layout should exist from day one.

Initial structure:
- `docker/`: local orchestration and container assets
- `apps/ainas-controlplane/`: generated Nextcloud shell app
- `exapps/control-plane/`: betterNAS control-plane service, packaged for Nextcloud-compatible dev flows
- `packages/contracts/`: shared schemas and API contracts
- `docs/`: architecture and product model notes
- `scripts/`: repeatable developer entry points

Rationale:
- Makes ownership and coupling visible in the filesystem.
- Supports gradual expansion into more services or clients without a repo rewrite.
- Keeps the local developer story coherent.

Alternatives considered:
- Single-app repo only: rejected because it hides important boundaries.
- Many services on day one: rejected because it adds overhead before we know the cut lines.

### 5. Standardize on a Docker-based local platform first

The first scaffold will target a Docker Compose development environment that starts Nextcloud, its required backing services, and the betterNAS control-plane service. This gives a repeatable local runtime before we decide on production deployment.

Rationale:
- Aligns with Nextcloud's easiest local development path.
- Lowers friction for bootstrapping the first app and service.
- Keeps infrastructure complexity proportional to the stage of the project.

Alternatives considered:
- Nix-only local orchestration: rejected for now because the project needs a portable first runtime.
- Production-like Kubernetes dev environment: rejected as premature.

## Risks / Trade-offs

- [Nextcloud coupling leaks into betterNAS service design] → Keep all Nextcloud-specific API calls and payload translation in adapter modules at the edge of the control-plane service.
- [The shell app grows into a second control plane] → Enforce a rule that product decisions and persistent domain logic live in the control-plane service, not the Nextcloud app.
- [ExApp packaging constrains future independence] → Structure the service so container packaging is a deployment concern rather than the application architecture.
- [Initial repo layout may be wrong in details] → Optimize for a small number of strong boundaries now; revisit internal package names later without collapsing ownership boundaries.
- [Docker dev environment differs from production NAS setups] → Treat the first environment as a development harness and keep storage/network assumptions explicit in docs.

## Migration Plan

1. Add the proposal artifacts that establish the architecture and scaffold requirements.
2. Create the top-level repository layout and a Docker Compose development environment.
3. Generate the Nextcloud shell app into `apps/ainas-controlplane/`.
4. Scaffold the control-plane service and shared contracts package.
5. Verify local startup, service discovery, and basic health paths before implementing product features.

Rollback strategy:
- Because this is a greenfield scaffold, rollback is simply removing the generated directories and Compose wiring if the architectural choice changes early.

## Open Questions

- Should the first control-plane service be implemented in Go, Python, or Node/TypeScript?
- What authentication boundary should exist between the Nextcloud shell app and the control-plane service in local development?
- Which parts of future sharing and RBAC behavior should remain delegated to Nextcloud, and which should be modeled natively in betterNAS?
- Do we want the first web product surface to live inside Nextcloud pages, outside Nextcloud as a separate frontend, or both?
