## ADDED Requirements

### Requirement: Nextcloud substrate boundary
The system SHALL explicitly define which storage, sharing, and client primitives betternas adopts from Nextcloud and which concerns remain betternas-owned.

#### Scenario: Product planning references Nextcloud capabilities
- **WHEN** betternas decides whether to build or reuse a capability
- **THEN** the planning artifacts MUST classify the capability as either Nextcloud substrate, betternas-owned logic, or a later optional fork/reference path

### Requirement: Reuse external storage backends
The platform SHALL treat Nextcloud external storage support as the first candidate substrate for connecting backend storage systems.

#### Scenario: betternas selects initial backend storage types
- **WHEN** betternas chooses the first storage backends to support
- **THEN** the plan MUST assume reuse of Nextcloud-supported external storage backends before proposing custom storage ingestion infrastructure

### Requirement: Reuse desktop and mobile references first
The platform SHALL treat the public Nextcloud desktop and iOS clients as the first reference implementations for cloud-drive style access before planning fully custom clients.

#### Scenario: betternas evaluates native device access
- **WHEN** the product needs Finder-style or mobile file access
- **THEN** the plan MUST document whether Nextcloud clients are being used directly, referenced, branded later, or intentionally replaced

### Requirement: Keep Nextcloud as substrate, not system of record
The platform SHALL not let Nextcloud become the long-term system of record for betternas-specific product semantics.

#### Scenario: New product concept is introduced
- **WHEN** betternas introduces workspaces, devices, policies, mount profiles, or similar product concepts
- **THEN** the design MUST model those concepts in betternas-owned contracts rather than relying on implicit Nextcloud-only representations
