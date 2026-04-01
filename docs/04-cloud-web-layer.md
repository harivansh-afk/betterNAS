# betterNAS Part 4: Web Control Plane and Optional Cloud Layer

This document describes the browser UI that users interact with, plus the
optional cloud adapter layer that may exist later.

## What it is

The web control plane is part of the core product.

It should provide:

- onboarding
- node and export management
- mount instructions
- sharing and browser file access later

An optional cloud adapter may later provide:

- Nextcloud-backed browser file UI
- mobile-friendly access
- share and link workflows

## What it does

- gives users a browser-based entry point into betterNAS
- talks only to `control-server`
- exposes the mount flow cleanly
- optionally layers on cloud/mobile/share behavior later

## What it should not do

- own product state separately from `control-server`
- become the only way users access their storage
- make the optional cloud adapter part of the core mount path

## Diagram

```text
                         self-hosted betterNAS stack

      node-service <--------> control-server <--------> [THIS DOC] web control plane
           ^                               |
           |                               |
           +----------- Finder mount flow -+

      optional later:
      Nextcloud adapter / cloud/mobile/share surface
```

## Core decisions

- The web control plane is part of the core product now.
- Nextcloud is optional and secondary.
- The first user value is managing exports and getting a mount URL, not a full
  browser file manager.

## Likely near-term role of the web control plane

- sign in
- see available NAS nodes
- see available exports
- request mount instructions
- copy or launch the WebDAV mount flow

## TODO

- Define the first user-facing screens for nodes, exports, and mount actions.
- Define how auth/session works in the web UI.
- Decide whether browser file viewing is part of V1 or follows later.
- Decide whether Nextcloud remains an internal adapter or becomes user-facing.
- Define what sharing means before adding any cloud/mobile layer.
