# betterNAS Part 2: Control Server

This document describes the main backend that owns product semantics and
coordinates the rest of the system.

## What it is

`control-server` is the source of truth for betterNAS.

It should own:

- users
- devices
- NAS nodes
- storage exports
- access grants
- mount profiles
- later, share flows and audit events

## What it does

- authenticates users and devices
- tracks which NAS nodes exist
- decides who can access which export
- issues mount instructions to local devices
- drives the web control plane
- stores the operational model of the product

## What it should not do

- proxy file bytes by default
- become the only data path between the Mac and the NAS
- depend on Nextcloud as its source of truth

## Diagram

```text
                         self-hosted betterNAS stack

      node-service <--------> [THIS DOC] control-server <--------> web control plane
           ^                               |
           |                               |
           +----------- Finder mount flow -+
```

## Core decisions

- `control-server` is the product brain.
- It owns policy and registry, not storage bytes.
- It should stay deployable on the user's NAS in the default product shape.
- The web UI should remain a consumer of this service, not a second backend.

## Suggested first entities

- `User`
- `Device`
- `NasNode`
- `StorageExport`
- `AccessGrant`
- `MountProfile`
- `AuditEvent`

## TODO

- Define the first durable database schema.
- Define auth between user browser, user device, NAS node, and control-server.
- Define the API for node registration, export inventory, and mount issuance.
- Define how mount tokens or credentials are issued and rotated.
- Define what optional cloud/share integration looks like later.
