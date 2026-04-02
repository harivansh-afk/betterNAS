# betterNAS Node Agent

Go service that runs on the NAS machine.

For the scaffold it does two things:

- serves `GET /health`
- serves a WebDAV export at `/dav/`
- optionally serves multiple configured exports at deterministic `/dav/exports/<slug>/` paths via `BETTERNAS_EXPORT_PATHS_JSON`
- registers itself with the control plane and syncs its exports when
  `BETTERNAS_CONTROL_PLANE_URL` is configured
- uses `BETTERNAS_USERNAME` and `BETTERNAS_PASSWORD` both for control-plane login
  and for local WebDAV basic auth

This is the first real storage-facing surface in the monorepo.

The user-facing binary should be distributed as `betternas-node`.

Install the latest release with:

```bash
curl -fsSL https://raw.githubusercontent.com/harivansh-afk/betterNAS/main/scripts/install-betternas-node.sh | sh
```

Then connect a machine to betterNAS with:

```bash
BETTERNAS_USERNAME=your-username \
BETTERNAS_PASSWORD=your-password \
BETTERNAS_EXPORT_PATH=/path/to/export \
BETTERNAS_NODE_DIRECT_ADDRESS=https://your-public-node-url \
betternas-node
```

If `BETTERNAS_CONTROL_PLANE_URL` is not set, the node defaults to
`https://api.betternas.com`.
