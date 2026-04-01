# betterNAS

betterNAS is a self-hostable WebDAV stack for mounting NAS exports in Finder.

The default product shape is:

- `node-service` serves the real files from the NAS over WebDAV
- `control-server` owns auth, nodes, exports, grants, and mount profile issuance
- `web control plane` lets the user manage the NAS and get mount instructions
- `macOS client` starts as native Finder WebDAV mounting, with a thin helper later

For now, the whole stack should be able to run on the user's NAS device.

## Current repo shape

- `apps/node-agent`
  - NAS-side Go runtime and WebDAV server
- `apps/control-plane`
  - Go backend for auth, registry, and mount profile issuance
- `apps/web`
  - Next.js web control plane
- `apps/nextcloud-app`
  - optional Nextcloud adapter, not the product center
- `packages/contracts`
  - canonical shared contracts
- `infra/docker`
  - self-hosted local stack

The main planning docs are:

- [docs/architecture.md](./docs/architecture.md)
- [skeleton.md](./skeleton.md)
- [docs/05-build-plan.md](./docs/05-build-plan.md)

## Default runtime model

```text
                   self-hosted betterNAS on the user's NAS

                         +------------------------------+
                         | web control plane            |
                         | Next.js UI                   |
                         +--------------+---------------+
                                        |
                                        v
                         +------------------------------+
                         | control-server               |
                         | auth / nodes / exports       |
                         | grants / mount profiles      |
                         +--------------+---------------+
                                        |
                                        v
                         +------------------------------+
                         | node-service                 |
                         | WebDAV + export runtime      |
                         | real NAS bytes               |
                         +------------------------------+

 user Mac
   |
   +--> browser -> web control plane
   |
   +--> Finder -> WebDAV mount URL from control-server
```

## Verify

Static verification:

```bash
pnpm verify
```

Bootstrap clone-local runtime settings:

```bash
pnpm agent:bootstrap
```

Bring the self-hosted stack up, verify it, and tear it down:

```bash
pnpm stack:up
pnpm stack:verify
pnpm stack:down --volumes
```

Run the full loop:

```bash
pnpm agent:verify
```

## Current end-to-end slice

The first proven slice is:

1. boot the stack with `pnpm stack:up`
2. verify it with `pnpm stack:verify`
3. get the WebDAV mount URL
4. mount it in Finder

If the stack is running on a remote machine, tunnel the WebDAV port first, then
use Finder `Connect to Server` with the tunneled URL.

## Product boundary

The default betterNAS product is self-hosted and WebDAV-first.

Nextcloud remains optional and secondary:

- useful later for browser/mobile/share surfaces
- not required for the core mount flow
- not the system of record
