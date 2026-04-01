# betterNAS Part 3: Local Device

This document describes the software and user experience on the user's Mac or
other local device.

## What it is

The local device layer is how a user actually mounts and uses their NAS.

It should start simple:

- browser opens the web control plane
- user gets a WebDAV mount URL
- Finder mounts the export

It can later grow into:

- a small helper app
- one-click mount flows
- auto-mount at login
- status and reconnect behavior

## What it does

- authenticates the user to betterNAS
- fetches allowed mount profiles from `control-server`
- mounts approved storage exports locally
- gives the user a native-feeling way to browse files

## What it should not do

- invent its own permissions model
- hardcode node endpoints outside the control-server
- depend on the optional cloud adapter for the core mount flow

## Diagram

```text
                         self-hosted betterNAS stack

      node-service <--------> control-server <--------> web control plane
           ^                                               ^
           |                                               |
           +------------- [THIS DOC] local device ---------+
                          browser + Finder
```

## Core decisions

- V1 relies on native Finder WebDAV mounting.
- The web UI should be enough to get the user to a mountable URL.
- A lightweight helper app is likely enough before a full native client.

## User modes

### Mount mode

- user mounts a NAS export in Finder
- files are browsed as a mounted remote disk

### Browser mode

- user manages the NAS and exports in the web control plane
- optional later: browse files in the browser

## TODO

- Define the mount profile format returned by `control-server`.
- Decide whether the first UX is manual Finder flow, helper app, or both.
- Define credential handling and Keychain behavior.
- Define reconnect and auto-mount expectations.
- Define what later native client work is actually worth doing.
