# betterNAS Part 4: Cloud / Web Layer

This document describes the optional browser, mobile, and cloud-drive style access layer.

## What it is

The cloud/web layer is the part of betterNAS that makes storage accessible beyond local mounts.

This is where we can reuse Nextcloud heavily for:
- browser file UI
- uploads and downloads
- sharing links
- WebDAV-based cloud access
- mobile reference behavior

## What it does

- gives users a browser-based file experience
- supports sharing and link-based access
- gives us a cloud mode in addition to mount mode
- can act as a reference surface while the main betterNAS product grows

## What it should not do

- own the product system of record
- become the only way users access storage
- swallow control-plane logic that should stay in betterNAS

## Diagram

```text
                            betterNAS system

   NAS node  <--------->  control plane  <--------->  local device
      |                           |                                   |
      |                           |                                   |
      +---------------------------+-----------------------+-----------+
                                                          |
                                                          v
                                              +----------------------+
                                              | [THIS DOC] cloud/web |
                                              |----------------------|
                                              | Nextcloud adapter    |
                                              | browser UI           |
                                              | sharing / mobile     |
                                              +----------------------+
```

## Core decisions

- The cloud/web layer is optional but very high leverage.
- Nextcloud is a strong fit here because it already gives us file UI and sharing primitives.
- It should sit beside mount mode, not replace it.

## Likely role of Nextcloud

- browser-based file UI
- share and link management
- optional mobile and cloud-drive style access
- adapter over the same storage exports the control plane knows about

## TODO

- Decide whether Nextcloud is directly user-facing in v1 or mostly an adapter behind betterNAS.
- Define how storage exports from the NAS node appear in the cloud/web layer.
- Define how shares in this layer map back to control-plane access grants.
- Define what mobile access looks like in v1.
- Define branding and how much of the cloud/web layer stays stock vs customized.
