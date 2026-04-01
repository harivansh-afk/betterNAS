# Control

This clone is the main repo.

Use it for:

- shared contracts
- repo guardrails
- runtime scripts
- integration verification
- architecture and coordination

Planned clone layout:

```text
/home/rathi/Documents/GitHub/betterNAS/
  betterNAS
  betterNAS-runtime
  betterNAS-control
  betterNAS-node
```

Clone roles:

- `betterNAS`
  - main coordination repo
  - owns contracts, scripts, and shared verification rules
- `betterNAS-runtime`
  - owns Docker Compose, stack env, readiness checks, and end-to-end runtime verification
- `betterNAS-control`
  - owns the Go control plane and contract-backed API behavior
- `betterNAS-node`
  - owns the node agent, WebDAV serving, and NAS-side registration/export behavior

Rules:

- shared interface changes land in `packages/contracts` first
- runtime verification must stay green in the main repo
- feature agents should stay inside their assigned clone unless a contract change is required

Agent command surface:

- main repo creates or refreshes sibling clones with `pnpm clones:setup`
- each clone bootstraps itself with `pnpm agent:bootstrap`
- each clone runs the full loop with `pnpm agent:verify`

Agent prompts live in:

- `docs/agents/runtime-agent.md`
- `docs/agents/control-plane-agent.md`
- `docs/agents/node-agent.md`
