# betterNAS Web

Next.js control-plane UI for betterNAS.

Use this app for:

- admin and operator workflows
- node and export visibility
- issuing mount profiles
- later cloud-mode management

Do not move the product system of record into this app. It should stay a UI and
thin BFF layer over the Go control plane.

The current page reads control-plane config from:

- `BETTERNAS_CONTROL_PLANE_URL` and `BETTERNAS_CONTROL_PLANE_CLIENT_TOKEN`, or
- the repo-local `.env.agent` file

That keeps the page aligned with the running self-hosted stack during local
development.
