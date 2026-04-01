# Control Plane Agent Prompt

```text
You are working in /home/rathi/Documents/GitHub/betterNAS/betterNAS-control.

Goal:
Make the Go control plane implement the current first-loop contracts for node registration, export inventory, heartbeat, mount profile issuance, and cloud profile issuance.

Primary scope:
- apps/control-plane/**

Read-only references:
- packages/contracts/**
- docs/architecture.md
- control.md
- TODO.md

Do not change:
- packages/contracts/**
- docs/architecture.md
- runtime scripts
- node-agent code

Use the existing contracts as fixed for this task.

Implement cleanly:
- POST /api/v1/nodes/register
  - store or update a node
  - store or update its exports
- POST /api/v1/nodes/{nodeId}/heartbeat
  - update status and lastSeenAt
- GET /api/v1/exports
  - return registered exports
- POST /api/v1/mount-profiles/issue
  - validate that the export exists
  - return a mount profile for that export
- POST /api/v1/cloud-profiles/issue
  - validate that the export exists
  - return a Nextcloud cloud profile for that export

Constraints:
- simplest correct implementation first
- in-memory storage is acceptable for this slice
- add tests
- do not invent new request or response shapes
- if you discover a real contract gap, stop and report the exact required contract change instead of patching around it

Acceptance criteria:
1. pnpm agent:bootstrap succeeds.
2. pnpm verify succeeds.
3. control-plane tests cover the implemented API behavior.
4. If the runtime loop is already green on this machine, pnpm stack:up and pnpm stack:verify also stays green in this clone.

Deliverable:
A real contract-backed control plane for the first mount loop, without contract drift.
```
