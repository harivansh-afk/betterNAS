# Runtime Agent Prompt

```text
You are working in /home/rathi/Documents/GitHub/betterNAS/betterNAS-runtime.

Goal:
Make the clone-local runtime and integration loop deterministic and green on this machine.

Primary scope:
- infra/docker/**
- scripts/**
- README.md
- control.md only if the command surface changes

Do not change:
- packages/contracts/**
- docs/architecture.md
- app behavior unless a tiny startup or health fix is strictly required to get the runtime green

Rules:
- keep this clone isolated and clone-safe
- do not hardcode ports or paths outside .env.agent
- do not invent new contracts
- prefer fixing runtime wiring, readiness, healthchecks, compose config, and verification scripts

Required command surface:
- pnpm agent:bootstrap
- pnpm verify
- pnpm stack:up
- pnpm stack:verify
- pnpm stack:down --volumes

Acceptance criteria:
1. From a fresh clone, pnpm agent:bootstrap succeeds.
2. pnpm verify succeeds.
3. pnpm stack:up succeeds.
4. pnpm stack:verify succeeds.
5. pnpm stack:down --volumes succeeds.
6. After a full reset, stack:up and stack:verify succeeds again.
7. The runtime stays deterministic and clone-safe.

If blocked:
- inspect the actual failing service logs
- make the smallest necessary fix
- keep fixes inside runtime-owned files unless a tiny startup fix is unavoidable

Deliverable:
A green runtime loop for this clone on this machine.
```
