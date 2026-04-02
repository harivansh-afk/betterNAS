# TODO

- [x] Remove the temporary TypeScript SDK layer so shared interfaces only come from `packages/contracts`.
- [x] Switch the monorepo from `npm` workspaces to `pnpm`.
- [x] Add root formatting, verification, and Go formatting rails.
- [x] Add hard boundary checks so apps and packages cannot drift across lanes with private imports.
- [x] Make the first contract-backed mount loop real: node registration, export inventory, mount profile issuance, and a Finder-mountable WebDAV export.
- [x] Prove the first manual remote-host WebDAV mount from a Mac over SSH tunnel.
- [ ] Surface exports and issued mount URLs in the web control plane.
- [ ] Add durable control-server storage for nodes, exports, grants, and mount profiles.
- [ ] Define the self-hosted deployment shape for the full stack on a NAS device.
- [ ] Define the Nix/module shape for installing the node-service on a NAS host.
- [ ] Decide whether the node-service should self-register or stay bootstrap-registered.
- [ ] Decide whether browser file viewing belongs in V1 web control plane or later.
- [ ] Define if and when the optional Nextcloud adapter comes back into scope.
