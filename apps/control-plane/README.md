# betterNAS Control Plane

Go service that owns the product control plane.

It is intentionally small for now:

- `GET /health`
- `GET /version`
- `POST /api/v1/nodes/register`
- `POST /api/v1/nodes/{nodeId}/heartbeat`
- `PUT /api/v1/nodes/{nodeId}/exports`
- `GET /api/v1/exports`
- `POST /api/v1/mount-profiles/issue`
- `POST /api/v1/cloud-profiles/issue`

The request and response shapes must follow the contracts in
[`packages/contracts`](../../packages/contracts).

`/api/v1/*` endpoints require bearer auth. New nodes register with
the same username and password session that users use in the web app.
`BETTERNAS_USERNAME` and `BETTERNAS_PASSWORD` may be provided to seed a default
account for local or self-hosted setups. Nodes and exports are owned by users,
and mount profiles return the account username plus the mount URL so Finder can
authenticate with that same betterNAS password. Multi-export sync should send
an explicit `mountPath` per export so mount profiles can stay stable across
runtimes.
