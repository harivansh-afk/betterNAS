# betterNAS Skeleton

This is the root build skeleton for the monorepo.

Its job is simple:

- lock the repo shape
- lock the language per runtime
- lock the first shared contract surface
- give agents a safe place to work in parallel
- keep the list of upstream references we are stealing from

## Repo shape

```text
betterNAS/
├── apps/
│   ├── web/                 # Next.js control-plane UI
│   ├── control-plane/       # Go control-plane API
│   ├── node-agent/          # Go NAS runtime + WebDAV surface
│   └── nextcloud-app/       # optional Nextcloud adapter
├── packages/
│   ├── contracts/           # canonical OpenAPI, schemas, TS types
│   ├── ui/                  # shared React UI
│   ├── eslint-config/       # shared lint config
│   └── typescript-config/   # shared TS config
├── infra/
│   └── docker/             # local runtime stack
├── docs/                   # architecture and part docs
├── scripts/                # local helper scripts
├── go.work                 # Go workspace
├── turbo.json              # Turborepo task graph
└── skeleton.md             # this file
```

## Runtime and language choices

| Part                 | Language                           | Why                                                                  |
| -------------------- | ---------------------------------- | -------------------------------------------------------------------- |
| `apps/web`           | TypeScript + Next.js               | best UI velocity, best admin/control-plane UX                        |
| `apps/control-plane` | Go                                 | strong concurrency, static binaries, operationally simple            |
| `apps/node-agent`    | Go                                 | best fit for host runtime, WebDAV service, and future Nix deployment |
| `apps/nextcloud-app` | PHP                                | native language for the Nextcloud adapter surface                    |
| `packages/contracts` | OpenAPI + JSON Schema + TypeScript | language-neutral source of truth with practical TS ergonomics        |

## Canonical contract rule

The source of truth for shared interfaces is:

1. [`docs/architecture.md`](./docs/architecture.md)
2. [`packages/contracts/openapi/betternas.v1.yaml`](./packages/contracts/openapi/betternas.v1.yaml)
3. [`packages/contracts/schemas`](./packages/contracts/schemas)
4. [`packages/contracts/src`](./packages/contracts/src)

Agents must not invent private shared request or response shapes outside those
locations.

## Parallel lanes

```text
                    shared write surface
      +-------------------------------------------+
      | docs/architecture.md                      |
      | packages/contracts/                       |
      +----------------+--------------------------+
                       |
     +-----------------+-----------------+-----------------+
     |                 |                 |                 |
     v                 v                 v                 v
  NAS node        control plane      local device      cloud layer
  lane            lane               lane              lane
```

Allowed ownership:

- NAS node lane
  - `apps/node-agent`
  - future `infra/nix/node-*`
- control-plane lane
  - `apps/control-plane`
  - DB and queue integration code later
- local-device lane
  - mount docs first
  - future helper app
- cloud layer lane
  - `apps/nextcloud-app`
  - Nextcloud mapping logic
- shared contract lane
  - `packages/contracts`
  - `docs/architecture.md`

## The first verification loop

```text
[node-agent]
  serves WebDAV export
        |
        v
[control-plane]
  registers node + export
  issues mount profile
        |
        v
[local device]
  mounts WebDAV in Finder
        |
        v
[cloud layer]
  optionally exposes same export in Nextcloud
```

If a task does not make one of those steps more real, it is probably too early.

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
  - why: keep Next.js as UI/BFF, not the system-of-record backend

### Go control plane

- Go routing enhancements
  - https://go.dev/blog/routing-enhancements
  - why: stdlib-first routing baseline

- `chi`
  - https://github.com/go-chi/chi
  - why: minimal router if stdlib patterns become too bare

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

- `koanf`
  - https://github.com/knadh/koanf
  - why: layered config if env-only config becomes too small

- `envconfig`
  - https://github.com/kelseyhightower/envconfig
  - why: tiny env-only config option

- `log/slog`
  - https://pkg.go.dev/log/slog
  - why: structured logging without adding a logging framework first

- `oapi-codegen`
  - https://github.com/oapi-codegen/oapi-codegen
  - why: generate Go and TS surfaces from OpenAPI

### NAS node and WebDAV

- Go WebDAV package
  - https://pkg.go.dev/golang.org/x/net/webdav
  - why: embeddable WebDAV server implementation

- `hacdias/webdav`
  - https://github.com/hacdias/webdav
  - why: small standalone WebDAV reference

- NixOS manual
  - https://nixos.org/manual/nixos/stable/
  - why: declarative host setup and service wiring

- Nixpkgs
  - https://github.com/NixOS/nixpkgs
  - why: service module and packaging reference

### Local device and mount UX

- Finder `Connect to Server`
  - https://support.apple.com/en-lamr/guide/mac-help/mchlp3015/mac
  - why: native mount UX baseline

- Finder WebDAV mounting
  - https://support.apple.com/is-is/guide/mac-help/mchlp1546/mac
  - why: exact v1 mount path

- Keychain data protection
  - https://support.apple.com/guide/security/keychain-data-protection-secb0694df1a/web
  - why: local credential storage model

- Finder Sync extensions
  - https://developer.apple.com/library/archive/documentation/General/Conceptual/ExtensibilityPG/Finder.html
  - why: future helper app / Finder integration reference

- WebDAV RFC 4918
  - https://www.rfc-editor.org/rfc/rfc4918
  - why: protocol semantics and edge cases

### Cloud and adapter layer

- Nextcloud app template
  - https://github.com/nextcloud/app_template
  - why: thin adapter app reference

- AppAPI / External Apps
  - https://docs.nextcloud.com/server/latest/admin_manual/exapps_management/AppAPIAndExternalApps.html
  - why: official external-app integration path

- Nextcloud WebDAV docs
  - https://docs.nextcloud.com/server/latest/user_manual/en/files/access_webdav.html
  - why: protocol/client behavior reference

- Nextcloud external storage
  - https://docs.nextcloud.com/server/latest/admin_manual/configuration_files/external_storage_configuration_gui.html
  - why: storage aggregation behavior

- Nextcloud file sharing config
  - https://docs.nextcloud.com/server/latest/admin_manual/configuration_files/file_sharing_configuration.html
  - why: share semantics reference

## What we steal vs what we own

### Steal

- Turborepo repo shape and task graph
- Next.js web-app conventions
- Go stdlib and proven Go infra libraries
- Go WebDAV implementation
- Finder native WebDAV mount UX
- Nextcloud shell-app and cloud/web primitives

### Own

- the betterNAS domain model
- the control-plane API
- the node registration and export model
- the mount profile model
- the mapping between cloud mode and mount mode
- the repo contract and shared schemas
- the root `pnpm verify` loop

## The first implementation slices after this scaffold

1. make `apps/node-agent` serve a real configurable WebDAV export
2. make `apps/control-plane` store real node/export records
3. issue real mount profiles from the control plane
4. make `apps/web` let a user pick an export and request a profile
5. keep `apps/nextcloud-app` thin and optional
