## ADDED Requirements

### Requirement: Distinguish cloud-drive access from true remote mounts
The platform SHALL explicitly distinguish between cloud-drive style access and true remote mount behavior.

#### Scenario: Device access strategy is discussed
- **WHEN** aiNAS plans device-native access on macOS or mobile
- **THEN** the planning artifacts MUST state whether the capability is based on cloud-drive style client behavior, true remote mounts, or both

### Requirement: Defer custom native agent work until justified
The platform SHALL not require a custom device-native agent for the first backend and control-plane planning unless true remote mount orchestration is confirmed as a near-term requirement.

#### Scenario: Delivery sequencing is chosen
- **WHEN** implementation order is planned
- **THEN** the design MUST allow heavy reuse of Nextcloud client references before requiring a custom native device daemon

### Requirement: Keep future device agent possible
The architecture SHALL preserve a clear boundary where a future aiNAS-owned device access layer can be introduced without rewriting the control plane.

#### Scenario: Product later adds login-time mounts or stronger native behavior
- **WHEN** aiNAS decides to add explicit mount orchestration or device-native workflows
- **THEN** the design MUST place that behavior in a device access layer separate from the core control-plane domain
