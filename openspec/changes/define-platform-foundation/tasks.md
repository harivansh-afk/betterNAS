## 1. Planning and boundary definition

- [ ] 1.1 Review the new planning artifacts and resolve the remaining open questions around v1 product mode, identity ownership, and first storage backends
- [ ] 1.2 Confirm the official Nextcloud primitives betternas will adopt in v1 and which ones are only reference implementations
- [ ] 1.3 Confirm the control plane as the system of record for product semantics

## 2. Control-plane architecture planning

- [ ] 2.1 Refine the first control-plane entity model for users, workspaces, devices, storage sources, shares, mount profiles, policies, and audit events
- [ ] 2.2 Define the first high-level API categories and authentication assumptions for the standalone control plane
- [ ] 2.3 Choose the initial implementation stack for the real control plane, including TypeScript backend, Postgres, Redis, and the standalone web stack

## 3. Product surface planning

- [ ] 3.1 Define the role of the standalone Next.js control-plane web app versus the embedded Nextcloud shell
- [ ] 3.2 Define the v1 desktop and mobile posture: direct reuse, branded later, or fully custom
- [ ] 3.3 Decide whether true remote mounts are a v1 requirement or a later device-layer capability

## 4. Follow-on implementation changes

- [ ] 4.1 Propose the next implementation change for defining the Nextcloud substrate in concrete runtime terms
- [ ] 4.2 Propose the implementation change for the first real betternas control-plane backend
- [ ] 4.3 Propose the implementation change for the standalone control-plane web UI
