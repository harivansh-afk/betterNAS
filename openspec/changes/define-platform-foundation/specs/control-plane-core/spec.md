## ADDED Requirements

### Requirement: betternas control plane as product system of record
The system SHALL define an betternas-owned control plane as the authoritative home for product-level domain concepts.

#### Scenario: Product semantics require persistence
- **WHEN** betternas needs to represent devices, workspaces, storage sources, shares, mount profiles, policies, or audit history
- **THEN** the planning artifacts MUST place those concepts inside the betternas control-plane domain model

### Requirement: High-level domain contracts
The platform SHALL define a first high-level contract map for core entities and API categories before implementation proceeds.

#### Scenario: Future implementation work needs a stable conceptual frame
- **WHEN** engineers start implementing the control plane
- **THEN** the planning artifacts MUST already identify the first core entities and the first API categories those entities imply

### Requirement: Standalone service posture
The control plane SHALL remain architecturally standalone even if it is temporarily packaged or surfaced through Nextcloud-compatible mechanisms.

#### Scenario: betternas backend is consumed by multiple surfaces
- **WHEN** a standalone web app, Nextcloud shell, or future device client needs backend behavior
- **THEN** the design MUST treat the control plane as a reusable betternas service rather than as logic conceptually trapped inside the Nextcloud app model

### Requirement: Single modular backend first
The platform SHALL prefer one modular backend service before splitting the control plane into multiple distributed services.

#### Scenario: Engineers plan service topology
- **WHEN** the first real control-plane backend is planned
- **THEN** the default architecture MUST be one service with explicit internal modules rather than many microservices
