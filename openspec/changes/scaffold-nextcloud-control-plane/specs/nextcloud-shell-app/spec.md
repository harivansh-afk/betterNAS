## ADDED Requirements

### Requirement: betternas shell app inside Nextcloud
The system SHALL provide a dedicated betternas shell app inside Nextcloud that establishes branded entry points for betternas-owned product surfaces.

#### Scenario: betternas surface is visible in Nextcloud
- **WHEN** the betternas app is installed in a local development environment
- **THEN** Nextcloud MUST expose an betternas-branded application surface that can be used as the integration shell for future product flows

### Requirement: Thin adapter responsibility
The betternas shell app SHALL act as an adapter layer and MUST keep core business logic outside the Nextcloud monolith.

#### Scenario: Product decision requires domain logic
- **WHEN** the shell app needs information about policy, orchestration, or future product rules
- **THEN** it MUST obtain that information through betternas-owned service boundaries instead of embedding the decision logic directly in the app

### Requirement: Nextcloud integration hooks
The betternas shell app SHALL provide the minimal integration hooks required to connect betternas-owned services to Nextcloud runtime surfaces such as navigation, settings, and backend access points.

#### Scenario: betternas needs a Nextcloud-native entry point
- **WHEN** betternas introduces a new product flow that starts from a Nextcloud-rendered page
- **THEN** the shell app MUST provide a supported hook or page boundary where the flow can enter betternas-controlled logic
