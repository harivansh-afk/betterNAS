# betterNAS Node Agent

Go service that runs on the NAS machine.

For the scaffold it does two things:

- serves `GET /health`
- serves a WebDAV export at `/dav/`
- optionally serves multiple configured exports at deterministic `/dav/exports/<slug>/` paths via `BETTERNAS_EXPORT_PATHS_JSON`
- registers itself with the control plane and syncs its exports when
  `BETTERNAS_CONTROL_PLANE_URL` is configured
- enforces issued WebDAV basic-auth mount credentials

This is the first real storage-facing surface in the monorepo.
