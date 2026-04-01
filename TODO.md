# TODO

- [x] Remove the temporary TypeScript SDK layer so shared interfaces only come from `packages/contracts`.
- [x] Switch the monorepo from `npm` workspaces to `pnpm`.
- [x] Add root formatting, verification, and Go formatting rails.
- [x] Add hard boundary checks so apps and packages cannot drift across lanes with private imports.
- [x] Make the first contract-backed mount loop real: node registration, export inventory, mount profile issuance, and a Finder-mountable WebDAV export.
- [ ] Add a manual E2E runbook for remote-host WebDAV testing from a Mac over SSH tunnel.
- [ ] Surface exports and issued mount URLs in the web control plane.
- [ ] Define the Nix/module shape for installing the node agent and export runtime on a NAS host.
- [ ] Decide whether the node agent should self-register or stay control-plane registered by bootstrap tooling.
