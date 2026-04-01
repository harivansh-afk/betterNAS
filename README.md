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

The root planning and delegation guide lives in [skeleton.md](/home/rathi/Documents/GitHub/betterNAS/skeleton.md).

## Verify

Run the repo acceptance loop with:

```bash
pnpm verify
```
