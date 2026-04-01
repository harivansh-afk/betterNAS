## ADDED Requirements

### Requirement: Standalone betternas web surface
The platform SHALL plan for a standalone web control-plane surface outside Nextcloud.

#### Scenario: Product UI expands beyond an embedded shell
- **WHEN** betternas needs an admin or product control interface that is larger than a thin Nextcloud page
- **THEN** the plan MUST place that interface in an betternas-owned standalone web application

### Requirement: Web UI consumes betternas API
The standalone web application SHALL be designed to consume betternas-owned backend contracts rather than Nextcloud internals directly.

#### Scenario: Web product feature requires backend data
- **WHEN** the standalone web surface needs workspaces, devices, shares, or policies
- **THEN** it MUST obtain those concepts through the betternas control-plane API design rather than by binding directly to Nextcloud internal models

### Requirement: Preserve Nextcloud shell as adapter
The presence of a standalone web app SHALL not remove the need for the thin Nextcloud shell as an adapter and embedded entry surface.

#### Scenario: betternas still needs a presence inside Nextcloud
- **WHEN** the broader product grows outside the Nextcloud UI
- **THEN** the shell app MUST remain conceptually limited to integration and entry-point responsibilities rather than absorbing the full standalone product
