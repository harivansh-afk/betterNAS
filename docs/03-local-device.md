# betterNAS Part 3: Local Device

This document describes the software and user experience on the user's Mac or other local device.

## What it is

The local device layer is how a user actually mounts and uses their NAS.

It can start simple:
- Finder + WebDAV mount
- manual `Connect to Server`

It can later grow into:
- a small desktop helper
- one-click mount flows
- auto-mount at login
- status and reconnect behavior

## What it does

- authenticates the user to betterNAS
- fetches allowed mount profiles from the control plane
- mounts approved storage exports locally
- gives the user a native-feeling way to browse files

## What it should not do

- invent its own permissions model
- hardcode NAS endpoints outside the control plane
- become tightly coupled to Nextcloud

## Diagram

```text
                            betterNAS system

   NAS node  <--------->  control plane  <--------->  [THIS DOC] local device
      |                           |                                   |
      |                           |                                   |
      +---------------------------+-----------------------+-----------+
                                                          |
                                                          v
                                                   cloud/web layer
```

## Core decisions

- V1 can rely on native Finder WebDAV mounting.
- A lightweight helper app is likely enough before a full custom client.
- The local device should consume mount profiles, not raw infrastructure details.

## User modes

### Mount mode

- user mounts a NAS export into Finder
- files are browsed as a mounted remote disk

### Cloud mode

- user accesses the same storage through browser/mobile/cloud surfaces
- this is not the same as a mounted filesystem

## TODO

- Define the mount profile format the control plane returns.
- Decide what the first local UX is: manual Finder flow, helper app, or both.
- Define credential storage and Keychain behavior.
- Define auto-mount, reconnect, and offline expectations.
- Define how the local device hands off to the cloud/web layer when mount mode is not enough.
