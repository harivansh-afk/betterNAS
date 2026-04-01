# betterNAS Part 2: Control Plane

This document describes the main backend that owns product semantics and coordinates the rest of the system.

## What it is

The control plane is the source of truth for betterNAS.

It should own:
- users
- devices
- NAS nodes
- storage exports
- access grants
- mount profiles
- cloud access profiles
- audit events

## What it does

- authenticates users and devices
- tracks which NAS nodes exist
- decides who can access which export
- issues mount instructions to local devices
- coordinates optional cloud/web access
- stores the operational model of the whole product

## What it should not do

- proxy file bytes unless absolutely necessary
- become a bottleneck in the data path
- depend on Nextcloud as its system of record

## Diagram

```text
                            betterNAS system

   NAS node  <--------->  [THIS DOC] control plane  <--------->  local device
      |                           |                                   |
      |                           |                                   |
      +---------------------------+-----------------------+-----------+
                                                          |
                                                          v
                                                   cloud/web layer
```

## Core decisions

- The control plane is the product brain.
- It should own policy and registry, not storage bytes.
- It should stay standalone even if it integrates with Nextcloud.
- It should issue access decisions, not act like a file server.

## Suggested first entities

- `User`
- `Device`
- `NasNode`
- `StorageExport`
- `AccessGrant`
- `MountProfile`
- `CloudProfile`
- `AuditEvent`

## TODO

- Define the first real domain model and database schema.
- Define auth between user device, NAS node, and control plane.
- Define the API for mount profiles and access grants.
- Define how the control plane tells the cloud/web layer what to expose.
- Define direct-access vs relay behavior for unreachable NAS nodes.
