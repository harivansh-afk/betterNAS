# betterNAS Part 1: NAS Node

This document describes the software that runs on the actual NAS machine, VM,
or workstation that owns the files.

## What it is

The NAS node is the machine that actually has the storage.

It should run:

- the `node-service`
- a WebDAV server surface
- export configuration
- optional enrollment or heartbeat back to `control-server`
- later, a reproducible install path such as Docker or Nix

It should expose one or more storage exports such as:

- `/data`
- `/media`
- `/backups`
- `/vm-images`

## What it does

- serves the real file bytes
- exposes chosen directories over WebDAV
- reports identity, health, and exports to `control-server`
- stays simple enough to self-host on a single NAS box

## What it should not do

- own product policy
- decide user access rules by itself
- become the system of record for users, grants, or shares

## Diagram

```text
                      self-hosted betterNAS stack

      web control plane ---> control-server ---> [THIS DOC] node-service
               ^                                        |
               |                                        |
               +---------------- user browser ----------+

      local Mac ---------------- Finder mount ----------+
```

## Core decisions

- The NAS node should be where WebDAV is served from whenever possible.
- The node should be installable as one boring runtime on the user's machine.
- The node should expose exports, not product semantics.

## TODO

- Define the self-hosted install shape: Docker first, Nix second, or both.
- Define the node identity and enrollment model.
- Define the storage export model: path, label, tags, permissions, capacity.
- Define when the node self-registers vs when bootstrap tooling registers it.
- Define direct-access vs relay-access behavior for remote use.
