# betterNAS Production Deployment Plan

## Overview

Deploy the betterNAS control-plane as a production service on netty (Netcup VPS) with SQLite-backed user auth, NGINX reverse proxy at `api.betternas.com`, and the web frontend on Vercel at `betternas.com`. Replaces the current dev Docker Compose setup with a NixOS-native systemd service matching the existing deployment pattern (forgejo, vaultwarden, sandbox-agent).

## Current State

- Control-plane is a Go binary running in Docker on netty (port 3001->3000)
- State is an in-memory store backed by a JSON file
- Auth is static tokens from environment variables (no user accounts)
- Web frontend reads env vars to find the control-plane URL and client token
- Node-agent runs in Docker, connects to control-plane over Docker network
- NGINX on netty already reverse-proxies 3 domains with ACME/Let's Encrypt
- NixOS config is at `/home/rathi/Documents/GitHub/nix/hosts/netty/configuration.nix`
- `betternas.com` is registered on Vercel with nameservers pointed to Vercel DNS

## Desired End State

- `api.betternas.com` serves the control-plane Go binary behind NGINX with TLS
- `betternas.com` serves the Next.js web UI from Vercel
- All state (users, sessions, nodes, exports) lives in SQLite at `/var/lib/betternas/control-plane/betternas.db`
- Users log in with username/password on the web UI, get a session cookie
- One-click mount: logged-in user clicks an export, backend issues WebDAV credentials using the user's session
- Node-agent connects to `api.betternas.com` over HTTPS
- Deployment is declarative via NixOS configuration.nix

### Verification:

1. `curl https://api.betternas.com/health` returns `ok`
2. Web UI at `betternas.com` loads, shows login page
3. User can register, log in, see exports, one-click mount
4. Node-agent on netty registers and syncs exports to `api.betternas.com`
5. WebDAV mount from Finder works with issued credentials

## What We're NOT Doing

- Multi-tenant / multi-user RBAC (just simple username/password accounts)
- OAuth / SSO / social login
- Email verification or password reset flows
- Migrating existing JSON state (fresh SQLite DB)
- Nextcloud integration (can add later)
- CI/CD pipeline (manual deploy via `nixos-rebuild switch`)
- Rate limiting or request throttling

## Implementation Approach

Five phases, each independently deployable and testable:

1. **SQLite store** - Replace memoryStore with sqliteStore for all existing state
2. **User auth** - Add users/sessions tables, login/register endpoints, session middleware
3. **CORS + frontend auth** - Wire the web UI to use session-based auth against `api.betternas.com`
4. **NixOS deployment** - Systemd service, NGINX vhost, ACME cert, DNS
5. **Vercel deployment** - Deploy web UI, configure domain and env vars

---

## Phase 1: SQLite Store

### Overview

Replace `memoryStore` (in-memory + JSON file) with a `sqliteStore` using `modernc.org/sqlite` (pure Go, no CGo, `database/sql` compatible). This keeps all existing API behavior identical while switching the persistence layer.

### Schema

```sql
-- Ordinal counters (replaces NextNodeOrdinal / NextExportOrdinal)
CREATE TABLE ordinals (
    name    TEXT PRIMARY KEY,
    value   INTEGER NOT NULL DEFAULT 0
);
INSERT INTO ordinals (name, value) VALUES ('node', 0), ('export', 0);

-- Nodes
CREATE TABLE nodes (
    id               TEXT PRIMARY KEY,
    machine_id       TEXT NOT NULL UNIQUE,
    display_name     TEXT NOT NULL DEFAULT '',
    agent_version    TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'online',
    last_seen_at     TEXT,
    direct_address   TEXT,
    relay_address    TEXT,
    created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Node auth tokens (hashed)
CREATE TABLE node_tokens (
    node_id     TEXT PRIMARY KEY REFERENCES nodes(id),
    token_hash  TEXT NOT NULL
);

-- Storage exports
CREATE TABLE exports (
    id          TEXT PRIMARY KEY,
    node_id     TEXT NOT NULL REFERENCES nodes(id),
    label       TEXT NOT NULL DEFAULT '',
    path        TEXT NOT NULL,
    mount_path  TEXT NOT NULL DEFAULT '',
    capacity_bytes INTEGER,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(node_id, path)
);

-- Export protocols (normalized from JSON array)
CREATE TABLE export_protocols (
    export_id  TEXT NOT NULL REFERENCES exports(id) ON DELETE CASCADE,
    protocol   TEXT NOT NULL,
    PRIMARY KEY (export_id, protocol)
);

-- Export tags (normalized from JSON array)
CREATE TABLE export_tags (
    export_id  TEXT NOT NULL REFERENCES exports(id) ON DELETE CASCADE,
    tag        TEXT NOT NULL,
    PRIMARY KEY (export_id, tag)
);
```

### Changes Required

#### 1. Add SQLite dependency

**File**: `apps/control-plane/go.mod`

```
go get modernc.org/sqlite
```

#### 2. New file: `sqlite_store.go`

**File**: `apps/control-plane/cmd/control-plane/sqlite_store.go`

Implements the same operations as `memoryStore` but backed by SQLite:

- `newSQLiteStore(dbPath string) (*sqliteStore, error)` - opens DB, runs migrations
- `registerNode(...)` - INSERT/UPDATE node + token hash in a transaction
- `upsertExports(...)` - DELETE removed exports, UPSERT current ones in a transaction
- `recordHeartbeat(...)` - UPDATE node status/lastSeenAt
- `listExports()` - SELECT all exports with protocols/tags joined
- `exportContext(exportID)` - SELECT export + its node
- `nodeAuthByMachineID(machineID)` - SELECT node_id + token_hash by machine_id
- `nodeAuthByID(nodeID)` - SELECT token_hash by node_id
- `nextOrdinal(name)` - UPDATE ordinals SET value = value + 1 RETURNING value

Key design decisions:

- Use `database/sql` with `modernc.org/sqlite` driver
- WAL mode enabled at connection: `PRAGMA journal_mode=WAL`
- Foreign keys enabled: `PRAGMA foreign_keys=ON`
- Schema migrations run on startup (embed SQL with `//go:embed`)
- All multi-table mutations wrapped in transactions
- No ORM - raw SQL with prepared statements

#### 3. Update `app.go` to use SQLite store

**File**: `apps/control-plane/cmd/control-plane/app.go`

Replace `memoryStore` initialization with `sqliteStore`:

```go
// Replace:
//   store, err := newMemoryStore(statePath)
// With:
//   store, err := newSQLiteStore(dbPath)
```

New env var: `BETTERNAS_CONTROL_PLANE_DB_PATH` (default: `/var/lib/betternas/control-plane/betternas.db`)

#### 4. Update `server.go` to use new store interface

**File**: `apps/control-plane/cmd/control-plane/server.go`

The server handlers currently call methods directly on `*memoryStore`. These need to call the equivalent methods on the new store. If the method signatures match, this is a straight swap. If not, introduce a `store` interface that both implement during migration, then delete `memoryStore`.

### Success Criteria

#### Automated Verification:

- [ ] `go build ./apps/control-plane/cmd/control-plane/` compiles with `CGO_ENABLED=0`
- [ ] `go test ./apps/control-plane/cmd/control-plane/ -v` passes all existing tests
- [ ] New SQLite store tests pass (register node, upsert exports, list exports, auth lookup)
- [ ] `curl` against a local instance: register node, sync exports, issue mount profile - all return expected responses

#### Manual Verification:

- [ ] Start control-plane locally, SQLite file is created at configured path
- [ ] Restart control-plane - state persists across restarts
- [ ] Node-agent can register and sync exports against the SQLite-backed control-plane

---

## Phase 2: User Auth

### Overview

Add user accounts with username/password (bcrypt) and session tokens stored in SQLite. The session token replaces the static `BETTERNAS_CONTROL_PLANE_CLIENT_TOKEN` for web UI access. Node-agent auth (bootstrap token + node token) is unchanged.

### Additional Schema

```sql
-- Users
CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE COLLATE NOCASE,
    password_hash TEXT NOT NULL,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Sessions
CREATE TABLE sessions (
    token      TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at TEXT NOT NULL
);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);
```

### New API Endpoints

```
POST /api/v1/auth/register    - Create account (username, password)
POST /api/v1/auth/login       - Login, returns session token + sets cookie
POST /api/v1/auth/logout      - Invalidate session
GET  /api/v1/auth/me          - Return current user info (session validation)
```

### Changes Required

#### 1. New file: `auth.go`

**File**: `apps/control-plane/cmd/control-plane/auth.go`

```go
// Dependencies: golang.org/x/crypto/bcrypt, crypto/rand

func (s *sqliteStore) createUser(username, password string) (user, error)
// - Validate username (3-64 chars, alphanumeric + underscore/hyphen)
// - bcrypt hash the password (cost 10)
// - INSERT into users with generated ID
// - Return user struct

func (s *sqliteStore) authenticateUser(username, password string) (user, error)
// - SELECT user by username
// - bcrypt.CompareHashAndPassword
// - Return user or error

func (s *sqliteStore) createSession(userID string, ttl time.Duration) (string, error)
// - Generate 32-byte random token, hex-encode
// - INSERT into sessions with expires_at = now + ttl
// - Return token

func (s *sqliteStore) validateSession(token string) (user, error)
// - SELECT session JOIN users WHERE token = ? AND expires_at > now
// - Return user or error

func (s *sqliteStore) deleteSession(token string) error
// - DELETE FROM sessions WHERE token = ?

func (s *sqliteStore) cleanExpiredSessions() error
// - DELETE FROM sessions WHERE expires_at < now
// - Run periodically (e.g., on each request or via goroutine)
```

#### 2. New env vars

```
BETTERNAS_SESSION_TTL          # Session duration (default: "720h" = 30 days)
BETTERNAS_REGISTRATION_ENABLED # Allow new registrations (default: "true")
```

#### 3. Update `server.go` - auth middleware and routes

**File**: `apps/control-plane/cmd/control-plane/server.go`

Add auth routes:

```go
mux.HandleFunc("POST /api/v1/auth/register", s.handleRegister)
mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
mux.HandleFunc("GET /api/v1/auth/me", s.handleMe)
```

Update client-auth middleware:

```go
// Currently: checks Bearer token against static BETTERNAS_CONTROL_PLANE_CLIENT_TOKEN
// New: checks Bearer token against sessions table first, falls back to static token
// This preserves backwards compatibility during migration
func (s *server) requireClientAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := extractBearerToken(r)

        // Try session-based auth first
        user, err := s.store.validateSession(token)
        if err == nil {
            ctx := context.WithValue(r.Context(), userContextKey, user)
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }

        // Fall back to static client token (for backwards compat / scripts)
        if secureStringEquals(token, s.config.clientToken) {
            next.ServeHTTP(w, r)
            return
        }

        writeUnauthorized(w)
    })
}
```

### Success Criteria

#### Automated Verification:

- [ ] `go test` passes for auth endpoints (register, login, logout, me)
- [ ] `go test` passes for session middleware (valid token, expired token, invalid token)
- [ ] Existing client token auth still works (backwards compat)
- [ ] Existing node auth unchanged

#### Manual Verification:

- [ ] Register a user via curl, login, use session token to list exports
- [ ] Session expires after TTL
- [ ] Logout invalidates session immediately
- [ ] Registration can be disabled via env var

---

## Phase 3: CORS + Frontend Auth Integration

### Overview

Add CORS headers to the control-plane so the Vercel-hosted frontend can make API calls. Update the web frontend to use session-based auth (login page, session cookie/token management).

### Changes Required

#### 1. CORS middleware in control-plane

**File**: `apps/control-plane/cmd/control-plane/server.go`

```go
// New env var: BETTERNAS_CORS_ORIGIN (e.g., "https://betternas.com")

func corsMiddleware(allowedOrigin string, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
        w.Header().Set("Access-Control-Allow-Credentials", "true")
        w.Header().Set("Access-Control-Max-Age", "86400")

        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusNoContent)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

#### 2. Frontend auth flow

**Files**: `apps/web/`

New pages/components:

- `app/login/page.tsx` - Login form (username + password)
- `app/register/page.tsx` - Registration form (if enabled)
- `lib/auth.ts` - Client-side auth helpers (store token, attach to requests)

Update `lib/control-plane.ts`:

- Remove `.env.agent` file reading (production doesn't need it)
- Read `NEXT_PUBLIC_BETTERNAS_API_URL` env var for the backend URL
- Use session token from localStorage/cookie instead of static client token
- Add login/register/logout API calls

```typescript
// lib/auth.ts
const TOKEN_KEY = "betternas_session";

export function getSessionToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(TOKEN_KEY);
}

export function setSessionToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearSessionToken(): void {
  localStorage.removeItem(TOKEN_KEY);
}

export async function login(
  apiUrl: string,
  username: string,
  password: string,
): Promise<string> {
  const res = await fetch(`${apiUrl}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  if (!res.ok) throw new Error("Login failed");
  const data = await res.json();
  setSessionToken(data.token);
  return data.token;
}
```

Update `lib/control-plane.ts`:

```typescript
// Replace the current getControlPlaneConfig with:
export function getControlPlaneConfig(): ControlPlaneConfig {
  const baseUrl = process.env.NEXT_PUBLIC_BETTERNAS_API_URL || null;
  const clientToken = getSessionToken();
  return { baseUrl, clientToken };
}
```

#### 3. Auth-gated layout

**File**: `apps/web/app/layout.tsx` or a middleware

Redirect to `/login` if no valid session. The `/login` and `/register` pages are public.

### Success Criteria

#### Automated Verification:

- [ ] CORS preflight (OPTIONS) returns correct headers
- [ ] Frontend builds: `cd apps/web && pnpm build`
- [ ] No TypeScript errors

#### Manual Verification:

- [ ] Open `betternas.com` (or localhost:3000) - redirected to login
- [ ] Register a new account, login, see exports dashboard
- [ ] Click an export, get mount credentials
- [ ] Logout, confirm redirected to login
- [ ] API calls from frontend include correct CORS headers

---

## Phase 4: NixOS Deployment (netty)

### Overview

Deploy the control-plane as a NixOS-managed systemd service on netty, behind NGINX with ACME TLS at `api.betternas.com`. Stop the Docker Compose stack.

### Changes Required

#### 1. DNS: Point `api.betternas.com` to netty

Run from local machine (Vercel CLI):

```bash
vercel dns add betternas.com api A 152.53.195.59
```

#### 2. Build the Go binary for Linux

**File**: `apps/control-plane/Dockerfile` (or local cross-compile)

For NixOS, we can either:

- (a) Cross-compile locally: `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o control-plane ./cmd/control-plane`
- (b) Build a Nix package (cleaner, but more work)
- (c) Build on netty directly from the git repo

Recommendation: **(c) Build on netty** from the cloned repo. Simple, works now. Add a Nix package later if desired.

#### 3. NixOS configuration changes

**File**: `/home/rathi/Documents/GitHub/nix/hosts/netty/configuration.nix`

Add these blocks (following the existing forgejo/vaultwarden pattern):

```nix
  # --- betterNAS control-plane ---
  betternasDomain = "api.betternas.com";

  # In services.nginx.virtualHosts:
  virtualHosts.${betternasDomain} = {
    enableACME = true;
    forceSSL = true;
    locations."/".proxyPass = "http://127.0.0.1:3100";
    locations."/".extraConfig = ''
      proxy_set_header X-Forwarded-Proto $scheme;
    '';
  };

  # Systemd service:
  systemd.services.betternas-control-plane = {
    description = "betterNAS Control Plane";
    after = [ "network-online.target" ];
    wants = [ "network-online.target" ];
    wantedBy = [ "multi-user.target" ];
    serviceConfig = {
      Type = "simple";
      User = username;
      Group = "users";
      WorkingDirectory = "/var/lib/betternas/control-plane";
      ExecStart = "/home/${username}/Documents/GitHub/betterNAS/betterNAS/apps/control-plane/dist/control-plane";
      EnvironmentFile = "/var/lib/betternas/control-plane/control-plane.env";
      Restart = "on-failure";
      RestartSec = 5;
      StateDirectory = "betternas/control-plane";
    };
  };
```

#### 4. Environment file on netty

**File**: `/var/lib/betternas/control-plane/control-plane.env`

```bash
PORT=3100
BETTERNAS_VERSION=0.1.0
BETTERNAS_CONTROL_PLANE_DB_PATH=/var/lib/betternas/control-plane/betternas.db
BETTERNAS_CONTROL_PLANE_CLIENT_TOKEN=<generate-strong-token>
BETTERNAS_CONTROL_PLANE_NODE_BOOTSTRAP_TOKEN=<generate-strong-token>
BETTERNAS_DAV_AUTH_SECRET=<generate-strong-secret>
BETTERNAS_DAV_CREDENTIAL_TTL=24h
BETTERNAS_SESSION_TTL=720h
BETTERNAS_REGISTRATION_ENABLED=true
BETTERNAS_CORS_ORIGIN=https://betternas.com
BETTERNAS_NODE_DIRECT_ADDRESS=https://api.betternas.com
```

#### 5. Build and deploy script

**File**: `apps/control-plane/scripts/deploy-netty.sh`

```bash
#!/usr/bin/env bash
set -euo pipefail

REMOTE="netty"
REPO="/home/rathi/Documents/GitHub/betterNAS/betterNAS"
DIST="$REPO/apps/control-plane/dist"

ssh "$REMOTE" "cd $REPO && git pull && \
  mkdir -p $DIST && \
  cd apps/control-plane && \
  CGO_ENABLED=0 go build -o $DIST/control-plane ./cmd/control-plane && \
  sudo systemctl restart betternas-control-plane && \
  sleep 2 && \
  sudo systemctl status betternas-control-plane --no-pager"
```

#### 6. Stop Docker Compose stack

After the systemd service is running and verified:

```bash
ssh netty 'bash -c "cd /home/rathi/Documents/GitHub/betterNAS/betterNAS && source scripts/lib/runtime-env.sh && compose down"'
```

### Success Criteria

#### Automated Verification:

- [ ] `curl https://api.betternas.com/health` returns `ok`
- [ ] `curl https://api.betternas.com/version` returns version JSON
- [ ] TLS certificate is valid (Let's Encrypt)
- [ ] `systemctl status betternas-control-plane` shows active

#### Manual Verification:

- [ ] Node-agent can register against `https://api.betternas.com`
- [ ] Mount credentials issued via the API work in Finder
- [ ] Service survives restart: `sudo systemctl restart betternas-control-plane`
- [ ] State persists in SQLite across restarts

---

## Phase 5: Vercel Deployment

### Overview

Deploy the Next.js web UI to Vercel at `betternas.com`.

### Changes Required

#### 1. Create Vercel project

```bash
cd apps/web
vercel link  # or vercel --yes
```

#### 2. Configure environment variables on Vercel

```bash
vercel env add NEXT_PUBLIC_BETTERNAS_API_URL production
# Value: https://api.betternas.com
```

#### 3. Configure domain

```bash
vercel domains add betternas.com
# Already have wildcard ALIAS to vercel-dns, so this should work
```

#### 4. Deploy

```bash
cd apps/web
vercel --prod
```

#### 5. Verify CORS

The backend at `api.betternas.com` must have `BETTERNAS_CORS_ORIGIN=https://betternas.com` set (done in Phase 4).

### Success Criteria

#### Automated Verification:

- [ ] `curl -I https://betternas.com` returns 200
- [ ] CORS preflight from `betternas.com` to `api.betternas.com` succeeds

#### Manual Verification:

- [ ] Visit `betternas.com` - see login page
- [ ] Register, login, see exports, issue mount credentials
- [ ] Mount from Finder using issued credentials

---

## Node-Agent Deployment (post-phases)

After the control-plane is running at `api.betternas.com`, update the node-agent on netty to connect to it:

1. Build node-agent: `cd apps/node-agent && CGO_ENABLED=0 go build -o dist/node-agent ./cmd/node-agent`
2. Create systemd service similar to control-plane
3. Environment: `BETTERNAS_CONTROL_PLANE_URL=https://api.betternas.com`
4. NGINX vhost for WebDAV if needed (or direct port exposure)

This is a follow-up task, not part of the initial deployment.

---

## Testing Strategy

### Unit Tests (Go):

- SQLite store: CRUD operations, transactions, concurrent access
- Auth: registration, login, session validation, expiry, logout
- Migration: schema creates cleanly on empty DB

### Integration Tests:

- Full API flow: register user -> login -> list exports -> issue mount profile
- Node registration + export sync against SQLite store
- Session expiry and cleanup

### Manual Testing:

1. Fresh deploy: start control-plane with empty DB
2. Register first user via API
3. Login from web UI
4. Connect node-agent, verify exports appear
5. Issue mount credentials, mount in Finder
6. Restart control-plane, verify all state persisted

## Performance Considerations

- SQLite WAL mode for concurrent reads during writes
- Session cleanup: delete expired sessions on a timer (every 10 minutes), not on every request
- Connection pool: single writer, multiple readers (SQLite default with WAL)
- For a single-NAS deployment, SQLite performance is more than sufficient

## Go Dependencies to Add

```
modernc.org/sqlite          # Pure Go SQLite driver
golang.org/x/crypto/bcrypt  # Password hashing
```

Both are well-maintained, widely used, and have no CGo requirement.

## References

- NixOS config: `/home/rathi/Documents/GitHub/nix/hosts/netty/configuration.nix`
- Control-plane server: `apps/control-plane/cmd/control-plane/server.go`
- Control-plane store: `apps/control-plane/cmd/control-plane/store.go`
- Web frontend API client: `apps/web/lib/control-plane.ts`
- Docker compose (current dev): `infra/docker/compose.dev.yml`
