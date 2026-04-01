# betternas

<img width="723" height="354" alt="image" src="https://github.com/user-attachments/assets/4e64fa91-315b-4a31-b191-d54ed1862ff7" />

## Architecture

The intended boundary is documented in `docs/architecture.md`. The short version is:

- Nextcloud remains an upstream storage and client-compatibility backend.
- The custom Nextcloud app is a shell and adapter layer.
- betternas business logic lives in the control-plane service.
