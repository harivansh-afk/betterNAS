## ADDED Requirements

### Requirement: Repository boundary scaffold
The repository SHALL provide a top-level scaffold that separates infrastructure, Nextcloud app code, betternas-owned service code, shared contracts, documentation, and automation scripts.

#### Scenario: Fresh clone exposes expected boundaries
- **WHEN** a developer inspects the repository after applying this change
- **THEN** the repository MUST include dedicated locations for Docker runtime assets, the Nextcloud shell app, the control-plane service, shared contracts, documentation, and scripts

### Requirement: Local development platform
The repository SHALL provide a local development runtime that starts a vanilla Nextcloud instance together with its required backing services and the betternas control-plane service.

#### Scenario: Developer boots the local stack
- **WHEN** a developer runs the documented local startup flow
- **THEN** the system MUST start Nextcloud and the betternas service dependencies without requiring a forked Nextcloud build

### Requirement: Shared contract package
The repository SHALL include a shared contract location for schemas and service interfaces used between the Nextcloud shell app and betternas-owned services.

#### Scenario: Interface changes are modeled centrally
- **WHEN** betternas defines an internal API or payload exchanged between the shell app and the control-plane service
- **THEN** the schema MUST be represented in the shared contracts location rather than duplicated ad hoc across codebases
