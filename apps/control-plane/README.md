# betterNAS Control Plane

Go service that owns the product control plane.

It is intentionally small for now:

- `GET /health`
- `GET /version`
- `POST /api/v1/nodes/register`
- `POST /api/v1/nodes/{nodeId}/heartbeat`
- `GET /api/v1/exports`
- `POST /api/v1/mount-profiles/issue`
- `POST /api/v1/cloud-profiles/issue`

The request and response shapes must follow the contracts in
[`packages/contracts`](../../packages/contracts).

`/api/v1/*` endpoints require bearer auth. New nodes register with
`BETTERNAS_CONTROL_PLANE_NODE_BOOTSTRAP_TOKEN`, client flows use
`BETTERNAS_CONTROL_PLANE_CLIENT_TOKEN`, and node registration returns an
`X-BetterNAS-Node-Token` header for subsequent node-scoped register and
heartbeat calls. Multi-export registrations should also send an explicit `mountPath` per export so mount profiles can stay stable across runtimes.
