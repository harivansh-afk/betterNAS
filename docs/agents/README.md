# Agent Prompts

These prompts are for the three sibling Devin clones:

```text
/home/rathi/Documents/GitHub/betterNAS/
  betterNAS
  betterNAS-runtime
  betterNAS-control
  betterNAS-node
```

Use them in this order:

1. start the runtime agent first
2. wait until the runtime loop is green
3. start the control and node agents in parallel

Rules that apply to all three:

- `packages/contracts/**` is frozen for this wave
- `docs/architecture.md` is frozen for this wave
- if an agent finds a real contract gap, it should stop and report the exact change instead of freelancing a workaround
- each agent should stay inside its assigned lane unless a tiny unblocker is strictly required
- each agent must verify with real commands, not only code inspection

Prompt files:

- [`runtime-agent.md`](./runtime-agent.md)
- [`control-plane-agent.md`](./control-plane-agent.md)
- [`node-agent.md`](./node-agent.md)
