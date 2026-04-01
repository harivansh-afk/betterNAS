# betterNAS Part 1: NAS Node

This document describes the software that runs on the actual NAS machine, VM, or workstation that owns the files.

## What it is

The NAS node is the machine that actually has the storage.

It should run:

- a WebDAV server
- a small betterNAS node agent
- declarative config via Nix
- optional tunnel or relay connection if the machine is not directly reachable

It should expose one or more storage exports such as:

- `/data`
- `/media`
- `/backups`
- `/vm-images`

## What it does

- serves the real file bytes
- exposes chosen directories over WebDAV
- registers itself with the control plane
- reports health, identity, and available exports
- optionally keeps an outbound connection alive for remote access

## What it should not do

- own user-facing product logic
- decide permissions by itself
- become the system of record for shares, devices, or policies

## Diagram

```text
                            betterNAS system

   local device  <------->  control plane  <------->  cloud/web layer
        |                         |                          |
        |                         |                          |
        +-------------------------+--------------------------+
                                  |
                                  v
                      +---------------------------+
                      | [THIS DOC] NAS node       |
                      |---------------------------|
                      | WebDAV server             |
                      | node agent                |
                      | exported directories      |
                      | optional tunnel/relay     |
                      +---------------------------+
```

## Core decisions

- The NAS node should be where WebDAV is served from whenever possible.
- The control plane should configure access, but file bytes should flow from the node to the user device as directly as possible.
- The node should be installable with a Nix module or flake so setup is reproducible.

## TODO

- Choose the WebDAV server we will standardize on for the node.
- Define the node agent responsibilities and API back to the control plane.
- Define the storage export model: path, label, capacity, tags, protocol support.
- Define direct-access vs relayed-access behavior.
- Define how the node connects to the cloud/web layer for optional Nextcloud integration.
