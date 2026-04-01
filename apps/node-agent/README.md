# betterNAS Node Agent

Go service that runs on the NAS machine.

For the scaffold it does two things:

- serves `GET /health`
- serves a WebDAV export at `/dav/`

This is the first real storage-facing surface in the monorepo.
