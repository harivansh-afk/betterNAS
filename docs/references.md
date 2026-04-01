# betterNAS References

This file tracks the upstream repos, tools, and docs we are likely to reuse,
reference, fork from, or borrow ideas from as betterNAS evolves.

The ordering matters:

1. self-hosted WebDAV stack first
2. control-server and web control plane second
3. optional cloud adapter later

## Primary now: self-hosted DAV stack

### Node-service and WebDAV

- Go WebDAV package
  - docs: https://pkg.go.dev/golang.org/x/net/webdav
  - why: embeddable WebDAV implementation for the NAS runtime

- `hacdias/webdav`
  - repo: https://github.com/hacdias/webdav
  - why: small standalone WebDAV reference

- `rclone serve webdav`
  - repo: https://github.com/rclone/rclone
  - why: useful reference for standing up WebDAV over existing storage

### Self-hosting and NAS configuration

- NixOS manual
  - docs: https://nixos.org/manual/nixos/stable/
  - why: host module design and declarative machine setup

- Nixpkgs
  - repo: https://github.com/NixOS/nixpkgs
  - why: service module and packaging reference

- Docker Compose docs
  - docs: https://docs.docker.com/compose/
  - why: current self-hosted runtime packaging baseline

## Primary now: control-server

### Backend and infra references

- Go routing enhancements
  - docs: https://go.dev/blog/routing-enhancements
  - why: low-dependency baseline for the API

- `chi`
  - repo: https://github.com/go-chi/chi
  - why: thin router if stdlib becomes too bare

- PostgreSQL
  - docs: https://www.postgresql.org/docs/
  - why: source of truth for product metadata

- `pgx`
  - repo: https://github.com/jackc/pgx
  - why: Postgres-first Go driver

- `sqlc`
  - repo: https://github.com/sqlc-dev/sqlc
  - why: typed query generation for Go

- Redis
  - docs: https://redis.io/docs/latest/
  - why: cache, jobs, and ephemeral coordination

- `go-redis`
  - repo: https://github.com/redis/go-redis
  - why: primary Redis client

- `asynq`
  - repo: https://github.com/hibiken/asynq
  - why: practical Redis-backed background jobs

- `oapi-codegen`
  - repo: https://github.com/oapi-codegen/oapi-codegen
  - why: generate Go and TS surfaces from OpenAPI with less drift

## Primary now: web control plane and local device

### Web control plane

- Next.js
  - repo: https://github.com/vercel/next.js
  - why: control-plane web UI

- Turborepo
  - docs: https://turborepo.dev/repo/docs/crafting-your-repository/structuring-a-repository
  - why: monorepo boundaries and task graph rules

### macOS mount UX

- Apple Finder `Connect to Server`
  - docs: https://support.apple.com/en-lamr/guide/mac-help/mchlp3015/mac
  - why: baseline native mount UX on macOS

- Apple Finder WebDAV mounting
  - docs: https://support.apple.com/is-is/guide/mac-help/mchlp1546/mac
  - why: direct WebDAV mount behavior in Finder

- Apple developer docs
  - docs: https://developer.apple.com/documentation/
  - why: Keychain, helper apps, launch agents, and later native integration

- Keychain data protection
  - docs: https://support.apple.com/guide/security/keychain-data-protection-secb0694df1a/web
  - why: baseline secret-storage model for device credentials

- WebDAV RFC 4918
  - docs: https://www.rfc-editor.org/rfc/rfc4918
  - why: protocol semantics and caveats

## Optional later: cloud adapter

### Nextcloud server and app references

- Nextcloud server
  - repo: https://github.com/nextcloud/server
  - why: optional browser/share/mobile substrate

- Nextcloud app template
  - repo: https://github.com/nextcloud/app_template
  - why: official starting point for the thin adapter app

- Nextcloud AppAPI / ExApps
  - docs: https://docs.nextcloud.com/server/latest/admin_manual/exapps_management/AppAPIAndExternalApps.html
  - why: external app integration model

- Nextcloud WebDAV access
  - docs: https://docs.nextcloud.com/server/latest/user_manual/en/files/access_webdav.html
  - why: protocol and client behavior reference

- Nextcloud external storage
  - docs: https://docs.nextcloud.com/server/latest/user_manual/en/external_storage/external_storage.html
  - why: storage aggregation reference

## Working rule

Use these references in this order:

1. steal primitives that solve the self-hosted DAV problem first
2. adapt them at the control-server boundary
3. only pull in optional cloud layers when the core mount product is solid
