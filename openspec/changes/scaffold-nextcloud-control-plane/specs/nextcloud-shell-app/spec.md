## ADDED Requirements

### Requirement: betterNAS shell app inside Nextcloud
The system SHALL provide a dedicated betterNAS shell app inside Nextcloud that establishes branded entry points for betterNAS-owned product surfaces.

#### Scenario: betterNAS surface is visible in Nextcloud
- **WHEN** the betterNAS app is installed in a local development environment
- **THEN** Nextcloud MUST expose an betterNAS-branded application surface that can be used as the integration shell for future product flows

### Requirement: Thin adapter responsibility
The betterNAS shell app SHALL act as an adapter layer and MUST keep core business logic outside the Nextcloud monolith.

#### Scenario: Product decision requires domain logic
- **WHEN** the shell app needs information about policy, orchestration, or future product rules
- **THEN** it MUST obtain that information through betterNAS-owned service boundaries instead of embedding the decision logic directly in the app

### Requirement: Nextcloud integration hooks
The betterNAS shell app SHALL provide the minimal integration hooks required to connect betterNAS-owned services to Nextcloud runtime surfaces such as navigation, settings, and backend access points.

#### Scenario: betterNAS needs a Nextcloud-native entry point
- **WHEN** betterNAS introduces a new product flow that starts from a Nextcloud-rendered page
- **THEN** the shell app MUST provide a supported hook or page boundary where the flow can enter betterNAS-controlled logic
