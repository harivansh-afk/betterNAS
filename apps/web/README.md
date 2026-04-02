# betterNAS Web

Next.js control-plane UI for betterNAS.

Use this app for:

- admin and operator workflows
- user-scoped node and export visibility
- issuing mount profiles that reuse the same betterNAS account credentials
- later cloud-mode management

Do not move the product system of record into this app. It should stay a UI and
thin BFF layer over the Go control plane.
