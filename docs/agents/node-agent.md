# Node Agent Prompt

```text
You are working in /home/rathi/Documents/GitHub/betterNAS/betterNAS-node.

Goal:
Make the node agent a clean NAS-side runtime for the first loop, with reliable WebDAV behavior and export configuration.

Primary scope:
- apps/node-agent/**

Read-only references:
- packages/contracts/**
- docs/architecture.md
- control.md
- TODO.md

Do not change:
- packages/contracts/**
- docs/architecture.md
- runtime scripts
- control-plane code

Use the existing contracts as fixed for this task.

Implement cleanly:
- stable WebDAV serving from BETTERNAS_EXPORT_PATH
- Finder-friendly behavior at /dav/
- clean env-driven node identity and export metadata
  - machine id
  - display name
  - export label
  - tags if useful
- keep the health endpoint clean
- add tests where practical

Optional only if it fits cleanly without changing contracts:
- node self-registration and heartbeat client wiring behind env configuration

Constraints:
- do not invent new shared APIs
- keep this a boring, reliable NAS-side service
- prefer correctness and configurability over features

Acceptance criteria:
1. pnpm agent:bootstrap succeeds.
2. pnpm verify succeeds.
3. WebDAV behavior is reliable against the configured export path.
4. If the runtime loop is already green on this machine, pnpm stack:up and pnpm stack:verify also stays green in this clone.

Deliverable:
A clean node agent that serves the first real WebDAV export path without cross-lane drift.
```
