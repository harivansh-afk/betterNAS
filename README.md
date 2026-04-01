# betterNAS

- control-plane owns policy and identity (decides)
- node-agent owns file serving (serves)
- web owns UX (consumer facing)
- nextcloud-app is optional adapter only for cloud storage in s3 n shit

## Monorepo

- `apps/web`: Next.js control-plane UI
- `apps/control-plane`: Go control-plane service
- `apps/node-agent`: Go NAS runtime / WebDAV node
- `apps/nextcloud-app`: optional Nextcloud adapter
- `packages/contracts`: canonical shared contracts
- `packages/ui`: shared React UI
- `infra/docker`: local Docker runtime

The root planning and delegation guide lives in [skeleton.md](./skeleton.md).

## Verify

Run the repo acceptance loop with:

```bash
pnpm verify
```

## Runtime loop

Bootstrap clone-local runtime settings with:

```bash
pnpm agent:bootstrap
```

If `.env.agent` is missing, bootstrap writes clone-local defaults for this checkout.

Bring the stack up, verify it, and tear it down with:

```bash
pnpm stack:up
pnpm stack:verify
pnpm stack:down --volumes
```

## Agent loop

Run the full static and integration loop with:

```bash
pnpm agent:verify
```

Create or refresh the sibling agent clones with:

```bash
pnpm clones:setup
```
