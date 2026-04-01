# betterNAS Node Agent

Go service that runs on the NAS machine.

It keeps the NAS-side runtime intentionally small:

- serves `GET /health`
- serves a WebDAV export rooted at `BETTERNAS_EXPORT_PATH` on `/dav/`
- redirects `/dav` to `/dav/` for Finder-friendly trailing-slash behavior
- optionally self-registers and sends heartbeats to the control plane when env wiring is enabled

This is the first real storage-facing surface in the monorepo.

## Runtime configuration

The node agent keeps its runtime surface env-driven:

- `PORT`: HTTP port used for the default listen address and direct-address fallback. Defaults to `8090`.
- `BETTERNAS_NODE_LISTEN_ADDRESS`: HTTP listen address. The app falls back to `127.0.0.1:${PORT}` so a locally launched agent is loopback-only by default. The container image sets `:8090` so published Docker ports stay reachable. Set an explicit interface only when you intentionally want to expose WebDAV beyond the machine.
- `BETTERNAS_EXPORT_PATH`: required export root on disk. The path must already exist. Relative paths resolve from the workspace root when the agent is running inside this repo. WebDAV follows symlinks that stay inside the export root and rejects links that escape it.
- `BETTERNAS_NODE_MACHINE_ID`: stable node identity sent during registration. Required when registration is enabled; otherwise defaults to the current hostname.
- `BETTERNAS_NODE_DISPLAY_NAME`: human-readable node name. Defaults to `BETTERNAS_NODE_MACHINE_ID`.
- `BETTERNAS_EXPORT_LABEL`: export label sent during registration. Defaults to the export directory name.
- `BETTERNAS_EXPORT_TAGS`: comma-separated export tags. Defaults to no tags.
- `BETTERNAS_VERSION`: agent version sent during registration.
- `BETTERNAS_NODE_DIRECT_ADDRESS`: direct WebDAV base address advertised during registration. Loopback listeners default to `http://localhost:${PORT}`; explicit host listeners default to that host and port. Wildcard listeners such as `:8090` or `0.0.0.0:8090` do not advertise a direct address unless you set this explicitly, which keeps container and NAS deployments from self-registering a loopback-only URL by mistake.
- `BETTERNAS_NODE_RELAY_ADDRESS`: optional relay address for future remote access wiring.

## Optional control-plane sync

Registration stays best-effort so WebDAV serving is not blocked by control-plane reachability.
If heartbeats are rejected after a fresh re-registration, the agent logs it once
and keeps serving WebDAV.

- `BETTERNAS_CONTROL_PLANE_URL`: control-plane base URL. Required when registration is enabled.
- `BETTERNAS_CONTROL_PLANE_AUTH_TOKEN`: optional bearer token to attach to registration and heartbeat requests when a deployment expects one.
- `BETTERNAS_NODE_MACHINE_ID`: must be set explicitly before enabling registration so the node keeps a stable identity across restarts.
- `BETTERNAS_NODE_REGISTER_ENABLED`: enables node self-registration. Defaults to `false`.
- `BETTERNAS_NODE_HEARTBEAT_ENABLED`: enables heartbeats after registration. Defaults to `false`.
- `BETTERNAS_NODE_HEARTBEAT_INTERVAL`: retry and heartbeat interval. Defaults to `30s`.
