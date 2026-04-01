# betterNAS Skeleton

This is the root build skeleton for the monorepo.

Its job is simple:

- lock the repo shape
- lock the language per runtime
- lock the first shared contract surface
- keep the self-hosted stack clear
- make later scoped execution runs easier

## Repo shape

```text
betterNAS/
├── apps/
│   ├── web/                 # Next.js web control plane
│   ├── control-plane/       # Go control-server
│   ├── node-agent/          # Go node-service and WebDAV runtime
│   └── nextcloud-app/       # optional Nextcloud adapter
├── packages/
│   ├── contracts/           # canonical OpenAPI, schemas, TS types
│   ├── ui/                  # shared React UI
│   ├── eslint-config/       # shared lint config
│   └── typescript-config/   # shared TS config
├── infra/
│   └── docker/              # self-hosted stack for local proof
├── docs/                    # architecture and build docs
├── scripts/                 # bootstrap, verify, and stack helpers
├── go.work                  # Go workspace
├── turbo.json               # Turborepo task graph
└── skeleton.md              # this file
```

## Runtime and language choices

| Part                 | Language                           | Why                                                                 |
| -------------------- | ---------------------------------- | ------------------------------------------------------------------- |
| `apps/web`           | TypeScript + Next.js               | fastest way to build the control-plane UI                           |
| `apps/control-plane` | Go                                 | strong backend baseline, static binaries, simple self-hosting       |
| `apps/node-agent`    | Go                                 | best fit for NAS runtime, WebDAV serving, and future Nix deployment |
| `apps/nextcloud-app` | PHP                                | native language for an optional Nextcloud adapter                   |
| `packages/contracts` | OpenAPI + JSON Schema + TypeScript | language-neutral source of truth with practical frontend ergonomics |

## Default deployment model

The default product story is self-hosted:

```text
                  self-hosted betterNAS stack on user's NAS

               +--------------------------------------------+
               | web control plane                          |
               | user opens this in browser                 |
               +-------------------+------------------------+
                                   |
                                   v
               +--------------------------------------------+
               | control-server                             |
               | auth / nodes / exports / grants            |
               | mount profile issuance                     |
               +-------------------+------------------------+
                                   |
                                   v
               +--------------------------------------------+
               | node-service                               |
               | WebDAV export runtime                      |
               | real NAS files                             |
               +--------------------------------------------+

 user Mac
   |
   +--> browser -> web control plane
   |
   +--> Finder -> issued WebDAV mount URL
```

Optional later shape:

- hosted control-server
- hosted web control plane
- optional Nextcloud adapter for cloud/mobile/share surfaces

Those are not required for the core betterNAS product loop.

## Canonical contract rule

The source of truth for shared interfaces is:

1. [`docs/architecture.md`](./docs/architecture.md)
2. [`packages/contracts/openapi/betternas.v1.yaml`](./packages/contracts/openapi/betternas.v1.yaml)
3. [`packages/contracts/schemas`](./packages/contracts/schemas)
4. [`packages/contracts/src`](./packages/contracts/src)

Agents must not invent shared request or response shapes outside those
locations.

## Implementation lanes

```text
                   shared write surface
      +-------------------------------------------+
      | docs/architecture.md                      |
      | packages/contracts/                       |
      +----------------+--------------------------+
                       |
       +---------------+----------------+----------------+
       |                                |                |
       v                                v                v
  node-service                    control-server   web control plane

                      optional later:
                           nextcloud adapter
```

Allowed ownership:

- node-service lane
  - `apps/node-agent`
  - future `infra/nix` host module work
- control-server lane
  - `apps/control-plane`
- web control plane lane
  - `apps/web`
- optional adapter lane
  - `apps/nextcloud-app`
- shared contract lane
  - `packages/contracts`
  - `docs/architecture.md`

## The first verification loop

```text
[node-service]
  serves WebDAV export
        |
        v
[control-server]
  registers node + export
  issues mount profile
        |
        v
[web control plane]
  shows export and mount action
        |
        v
[local device]
  mounts in Finder
```

This is the main product loop.

## Upstream references to steal from

### Monorepo and web

- Turborepo `create-turbo`
  - https://turborepo.dev/repo/docs/reference/create-turbo
  - why: monorepo base scaffold

- Turborepo structuring guide
  - https://turborepo.dev/repo/docs/crafting-your-repository/structuring-a-repository
  - why: package boundaries and task graph rules

- Next.js backend-for-frontend guide
  - https://nextjs.org/docs/app/guides/backend-for-frontend
  - why: keep Next.js as UI and orchestration surface, not the source-of-truth backend

### Go control-server

- Go routing enhancements
  - https://go.dev/blog/routing-enhancements
  - why: stdlib-first routing baseline

- `pgx`
  - https://github.com/jackc/pgx
  - why: Postgres-first Go driver

- `sqlc`
  - https://github.com/sqlc-dev/sqlc
  - why: typed query generation

- `go-redis`
  - https://github.com/redis/go-redis
  - why: primary Redis client

- `asynq`
  - https://github.com/hibiken/asynq
  - why: practical Redis-backed job system

- `oapi-codegen`
  - https://github.com/oapi-codegen/oapi-codegen
  - why: generate surfaces from OpenAPI with less drift

### Node-service and WebDAV

- Go WebDAV package
  - https://pkg.go.dev/golang.org/x/net/webdav
  - why: embeddable WebDAV server implementation

- `hacdias/webdav`
  - https://github.com/hacdias/webdav
  - why: small standalone WebDAV reference

- NixOS manual
  - https://nixos.org/manual/nixos/stable/
  - why: declarative host setup and service wiring

### Local mount UX

- Finder `Connect to Server`
  - https://support.apple.com/en-lamr/guide/mac-help/mchlp3015/mac
  - why: native mount UX baseline

- Finder WebDAV mounting
  - https://support.apple.com/is-is/guide/mac-help/mchlp1546/mac
  - why: exact v1 mount path

- Keychain data protection
  - https://support.apple.com/guide/security/keychain-data-protection-secb0694df1a/web
  - why: local credential storage model

- WebDAV RFC 4918
  - https://www.rfc-editor.org/rfc/rfc4918
  - why: protocol semantics and edge cases

### Optional cloud adapter

- Nextcloud app template
  - https://github.com/nextcloud/app_template
  - why: thin adapter app reference

- Nextcloud WebDAV docs
  - https://docs.nextcloud.com/server/latest/user_manual/en/files/access_webdav.html
  - why: protocol/client behavior reference

## What we steal vs what we own

### Steal

- Turborepo repo shape and task graph
- Next.js web-app conventions
- Go stdlib and proven Go infra libraries
- Go WebDAV implementation
- Finder native WebDAV mount UX
- optional Nextcloud adapter primitives later

### Own

- the betterNAS domain model
- the control-server API
- the node registration and export model
- the mount profile model
- the self-hosted stack wiring
- the repo contract and shared schemas
- the root `pnpm verify` loop

## The next implementation slices

1. make `apps/web` expose the real mount flow to a user
2. add durable control-server storage for nodes, exports, and grants
3. define the self-hosted NAS install shape for `apps/node-agent`
4. keep the optional cloud adapter out of the critical path
