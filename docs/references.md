# betterNAS References

This file tracks the upstream repos, tools, and docs we are likely to reuse, reference, fork from, or borrow ideas from as betterNAS evolves.

The goal is simple: do not lose the external pieces that give us leverage.

## NAS node

### WebDAV server candidates

- `rclone serve webdav`
  - repo: https://github.com/rclone/rclone
  - why: fast way to stand up a WebDAV layer over existing storage

- `hacdias/webdav`
  - repo: https://github.com/hacdias/webdav
  - why: small standalone WebDAV server, easy to reason about

- Apache `mod_dav`
  - docs: https://httpd.apache.org/docs/current/mod/mod_dav.html
  - why: standard WebDAV implementation if we want conventional infra

### Nix / host configuration

- NixOS manual
  - docs: https://nixos.org/manual/nixos/stable/
  - why: host module design, service config, declarative machine setup

- Nixpkgs
  - repo: https://github.com/NixOS/nixpkgs
  - why: reference for packaging and service modules

## Control plane

### Backend and infra references

- Go routing enhancements
  - docs: https://go.dev/blog/routing-enhancements
  - why: best low-dependency baseline if we stay with the standard library

- `chi`
  - repo: https://github.com/go-chi/chi
  - why: thin stdlib-friendly router if we want middleware and route groups

- PostgreSQL
  - docs: https://www.postgresql.org/docs/
  - why: source of truth for product metadata

- `pgx`
  - repo: https://github.com/jackc/pgx
  - why: Postgres-first Go driver and toolkit

- `sqlc`
  - repo: https://github.com/sqlc-dev/sqlc
  - why: typed query generation for Go

- Redis
  - docs: https://redis.io/docs/latest/
  - why: cache, jobs, ephemeral coordination

- `go-redis`
  - repo: https://github.com/redis/go-redis
  - why: primary Redis client for Go

- `asynq`
  - repo: https://github.com/hibiken/asynq
  - why: practical Redis-backed background jobs

- `koanf`
  - repo: https://github.com/knadh/koanf
  - why: layered config if the control plane grows beyond env-only config

- `envconfig`
  - repo: https://github.com/kelseyhightower/envconfig
  - why: small env-only config loader

- `log/slog`
  - docs: https://pkg.go.dev/log/slog
  - why: structured logging without extra dependencies

- `oapi-codegen`
  - repo: https://github.com/oapi-codegen/oapi-codegen
  - why: generate Go and TS surfaces from OpenAPI with less drift

### SSH access / gateway reference

- `sshpiper`
  - repo: https://github.com/tg123/sshpiper
  - why: SSH proxy/gateway reference if we add SSH-brokered access later

## Local device

### macOS native mount references

- Apple Finder `Connect to Server`
  - docs: https://support.apple.com/en-lamr/guide/mac-help/mchlp3015/mac
  - why: baseline native mounting UX on macOS

- Apple Finder WebDAV mounting
  - docs: https://support.apple.com/is-is/guide/mac-help/mchlp1546/mac
  - why: direct WebDAV mount behavior in Finder

### macOS integration references

- Apple developer docs
  - docs: https://developer.apple.com/documentation/
  - why: Keychain, launch agents, desktop helpers, future native integration

- Keychain data protection
  - docs: https://support.apple.com/guide/security/keychain-data-protection-secb0694df1a/web
  - why: baseline secret-storage model for device credentials

- Finder Sync extensions
  - docs: https://developer.apple.com/library/archive/documentation/General/Conceptual/ExtensibilityPG/Finder.html
  - why: future helper-app integration pattern if Finder UX grows

- WebDAV RFC 4918
  - docs: https://www.rfc-editor.org/rfc/rfc4918
  - why: protocol semantics and caveats

## Cloud / web layer

### Nextcloud server and app references

- Nextcloud server
  - repo: https://github.com/nextcloud/server
  - why: cloud/web/share substrate

- Nextcloud app template
  - repo: https://github.com/nextcloud/app_template
  - why: official starting point for the thin shell app

- Nextcloud AppAPI / ExApps
  - docs: https://docs.nextcloud.com/server/latest/admin_manual/exapps_management/AppAPIAndExternalApps.html
  - why: external app integration model

### Nextcloud client references

- Nextcloud desktop
  - repo: https://github.com/nextcloud/desktop
  - why: Finder/cloud-drive style reference behavior

- Nextcloud iOS
  - repo: https://github.com/nextcloud/ios
  - why: mobile reference implementation

### Nextcloud storage and protocol references

- Nextcloud WebDAV access
  - docs: https://docs.nextcloud.com/server/latest/user_manual/en/files/access_webdav.html
  - why: protocol and client behavior reference

- Nextcloud external storage
  - docs: https://docs.nextcloud.com/server/latest/user_manual/en/external_storage/external_storage.html
  - why: storage aggregation reference

- Nextcloud theming / branded clients
  - docs: https://docs.nextcloud.com/server/latest/admin_manual/configuration_server/theming.html
  - why: future branding path if Nextcloud stays user-facing

## Frontend

- Next.js
  - repo: https://github.com/vercel/next.js
  - why: likely standalone control-plane web UI

## Working rule

Use these references in this order:

1. steal primitives that already solve boring problems
2. adapt them at the control-plane boundary
3. only fork or replace when the product meaningfully diverges
