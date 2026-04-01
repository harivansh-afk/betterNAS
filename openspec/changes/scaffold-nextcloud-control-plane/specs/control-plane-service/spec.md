## ADDED Requirements

### Requirement: Dedicated control-plane service
The system SHALL provide an betternas-owned control-plane service that is separate from the Nextcloud shell app and owns product domain logic.

#### Scenario: betternas adds a new control-plane rule
- **WHEN** a new business rule for storage policy, RBAC, orchestration, or future client behavior is introduced
- **THEN** the rule MUST be implemented in the control-plane service rather than as primary logic inside the Nextcloud app

### Requirement: Client-agnostic internal API
The control-plane service SHALL expose internal APIs that can be consumed by the Nextcloud shell app and future betternas clients without requiring direct coupling to Nextcloud internals.

#### Scenario: New betternas client consumes control-plane behavior
- **WHEN** betternas adds a web, desktop, or iOS surface outside Nextcloud
- **THEN** that surface MUST be able to consume control-plane behavior through documented betternas service interfaces

### Requirement: Nextcloud backend adapter boundary
The control-plane service SHALL isolate Nextcloud-specific integration at its boundary so that storage and sharing backends remain replaceable over time.

#### Scenario: Service calls the Nextcloud backend
- **WHEN** the control-plane service needs to interact with file or sharing primitives provided by Nextcloud
- **THEN** the interaction MUST pass through a dedicated adapter boundary instead of spreading Nextcloud-specific calls across unrelated domain code
