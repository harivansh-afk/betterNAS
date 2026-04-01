# betterNAS Control Plane

Go service that owns the product control plane.

It is intentionally small for now:

- `GET /health`
- `GET /version`
- `POST /api/v1/nodes/register`
- `GET /api/v1/exports`
- `POST /api/v1/mount-profiles/issue`
- `POST /api/v1/cloud-profiles/issue`

The request and response shapes must follow the contracts in
[`packages/contracts`](../../packages/contracts).
