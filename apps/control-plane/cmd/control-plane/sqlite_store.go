package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

var (
	errUsernameTaken  = errors.New("username already taken")
	errInvalidLogin   = errors.New("invalid username or password")
	errSessionExpired = errors.New("session expired or invalid")
)

const sqliteSchema = `
CREATE TABLE IF NOT EXISTS ordinals (
	name  TEXT PRIMARY KEY,
	value INTEGER NOT NULL DEFAULT 0
);
INSERT OR IGNORE INTO ordinals (name, value) VALUES ('node', 0), ('export', 0);

CREATE TABLE IF NOT EXISTS nodes (
	id              TEXT PRIMARY KEY,
	machine_id      TEXT NOT NULL UNIQUE,
	owner_id        TEXT REFERENCES users(id),
	display_name    TEXT NOT NULL DEFAULT '',
	agent_version   TEXT NOT NULL DEFAULT '',
	status          TEXT NOT NULL DEFAULT 'online',
	last_seen_at    TEXT,
	direct_address  TEXT,
	relay_address   TEXT
);

CREATE TABLE IF NOT EXISTS node_tokens (
	node_id    TEXT PRIMARY KEY REFERENCES nodes(id),
	token_hash TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS exports (
	id         TEXT PRIMARY KEY,
	node_id    TEXT NOT NULL REFERENCES nodes(id),
	owner_id   TEXT REFERENCES users(id),
	label      TEXT NOT NULL DEFAULT '',
	path       TEXT NOT NULL,
	mount_path TEXT NOT NULL DEFAULT '',
	capacity_bytes INTEGER,
	UNIQUE(node_id, path)
);

CREATE TABLE IF NOT EXISTS export_protocols (
	export_id TEXT NOT NULL REFERENCES exports(id) ON DELETE CASCADE,
	protocol  TEXT NOT NULL,
	PRIMARY KEY (export_id, protocol)
);

CREATE TABLE IF NOT EXISTS export_tags (
	export_id TEXT NOT NULL REFERENCES exports(id) ON DELETE CASCADE,
	tag       TEXT NOT NULL,
	PRIMARY KEY (export_id, tag)
);

CREATE TABLE IF NOT EXISTS users (
	id            TEXT PRIMARY KEY,
	username      TEXT NOT NULL UNIQUE COLLATE NOCASE,
	password_hash TEXT NOT NULL,
	created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS sessions (
	token      TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
	expires_at TEXT NOT NULL
);
`

type sqliteStore struct {
	db *sql.DB
}

func newSQLiteStore(dbPath string) (*sqliteStore, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create database directory %s: %w", dir, err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open database %s: %w", dbPath, err)
	}

	if _, err := db.Exec(sqliteSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize database schema: %w", err)
	}
	if err := migrateSQLiteSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return &sqliteStore{db: db}, nil
}

func migrateSQLiteSchema(db *sql.DB) error {
	migrations := []string{
		"ALTER TABLE nodes ADD COLUMN owner_id TEXT REFERENCES users(id)",
		"ALTER TABLE exports ADD COLUMN owner_id TEXT REFERENCES users(id)",
	}
	for _, statement := range migrations {
		if _, err := db.Exec(statement); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("run sqlite migration %q: %w", statement, err)
		}
	}

	if _, err := db.Exec(`
		UPDATE exports
		SET owner_id = (
			SELECT owner_id
			FROM nodes
			WHERE nodes.id = exports.node_id
		)
		WHERE owner_id IS NULL
	`); err != nil {
		return fmt.Errorf("backfill export owners: %w", err)
	}

	return nil
}

func (s *sqliteStore) nextOrdinal(tx *sql.Tx, name string) (int, error) {
	var value int
	err := tx.QueryRow("UPDATE ordinals SET value = value + 1 WHERE name = ? RETURNING value", name).Scan(&value)
	if err != nil {
		return 0, fmt.Errorf("next ordinal %q: %w", name, err)
	}
	return value, nil
}

func ordinalToNodeID(ordinal int) string {
	if ordinal == 1 {
		return "dev-node"
	}
	return fmt.Sprintf("dev-node-%d", ordinal)
}

func ordinalToExportID(ordinal int) string {
	if ordinal == 1 {
		return "dev-export"
	}
	return fmt.Sprintf("dev-export-%d", ordinal)
}

func (s *sqliteStore) registerNode(ownerID string, request nodeRegistrationRequest, registeredAt time.Time) (nodeRegistrationResult, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nodeRegistrationResult{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if machine already registered.
	var nodeID string
	var existingOwnerID sql.NullString
	err = tx.QueryRow("SELECT id, owner_id FROM nodes WHERE machine_id = ?", request.MachineID).Scan(&nodeID, &existingOwnerID)
	if err == sql.ErrNoRows {
		ordinal, err := s.nextOrdinal(tx, "node")
		if err != nil {
			return nodeRegistrationResult{}, err
		}
		nodeID = ordinalToNodeID(ordinal)
	} else if err != nil {
		return nodeRegistrationResult{}, fmt.Errorf("lookup node by machine_id: %w", err)
	} else if existingOwnerID.Valid && strings.TrimSpace(existingOwnerID.String) != "" && existingOwnerID.String != ownerID {
		return nodeRegistrationResult{}, errNodeOwnedByAnotherUser
	}

	// Upsert node.
	_, err = tx.Exec(`
		INSERT INTO nodes (id, machine_id, owner_id, display_name, agent_version, status, last_seen_at, direct_address, relay_address)
		VALUES (?, ?, ?, ?, ?, 'online', ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			owner_id = excluded.owner_id,
			display_name = excluded.display_name,
			agent_version = excluded.agent_version,
			status = 'online',
			last_seen_at = excluded.last_seen_at,
			direct_address = excluded.direct_address,
			relay_address = excluded.relay_address
	`, nodeID, request.MachineID, ownerID, request.DisplayName, request.AgentVersion,
		registeredAt.UTC().Format(time.RFC3339),
		nullableString(request.DirectAddress), nullableString(request.RelayAddress))
	if err != nil {
		return nodeRegistrationResult{}, fmt.Errorf("upsert node: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nodeRegistrationResult{}, fmt.Errorf("commit registration: %w", err)
	}

	node, _ := s.nodeByID(nodeID)
	return nodeRegistrationResult{
		Node: node,
	}, nil
}

func (s *sqliteStore) upsertExports(nodeID string, ownerID string, request nodeExportsRequest) ([]storageExport, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Verify node exists.
	var exists bool
	err = tx.QueryRow("SELECT 1 FROM nodes WHERE id = ? AND owner_id = ?", nodeID, ownerID).Scan(&exists)
	if err != nil {
		return nil, errNodeNotFound
	}

	// Collect current export IDs for this node (by path).
	currentExports := make(map[string]string) // path -> exportID
	rows, err := tx.Query("SELECT id, path FROM exports WHERE node_id = ?", nodeID)
	if err != nil {
		return nil, fmt.Errorf("query current exports: %w", err)
	}
	for rows.Next() {
		var id, path string
		if err := rows.Scan(&id, &path); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan current export: %w", err)
		}
		currentExports[path] = id
	}
	rows.Close()

	keepPaths := make(map[string]struct{}, len(request.Exports))
	for _, input := range request.Exports {
		exportID, exists := currentExports[input.Path]
		if !exists {
			ordinal, err := s.nextOrdinal(tx, "export")
			if err != nil {
				return nil, err
			}
			exportID = ordinalToExportID(ordinal)
		}

		_, err = tx.Exec(`
			INSERT INTO exports (id, node_id, owner_id, label, path, mount_path, capacity_bytes)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				owner_id = excluded.owner_id,
				label = excluded.label,
				mount_path = excluded.mount_path,
				capacity_bytes = excluded.capacity_bytes
		`, exportID, nodeID, ownerID, input.Label, input.Path, input.MountPath, nullableInt64(input.CapacityBytes))
		if err != nil {
			return nil, fmt.Errorf("upsert export %q: %w", input.Path, err)
		}

		// Replace protocols.
		if _, err := tx.Exec("DELETE FROM export_protocols WHERE export_id = ?", exportID); err != nil {
			return nil, fmt.Errorf("clear export protocols: %w", err)
		}
		for _, protocol := range input.Protocols {
			if _, err := tx.Exec("INSERT INTO export_protocols (export_id, protocol) VALUES (?, ?)", exportID, protocol); err != nil {
				return nil, fmt.Errorf("insert export protocol: %w", err)
			}
		}

		// Replace tags.
		if _, err := tx.Exec("DELETE FROM export_tags WHERE export_id = ?", exportID); err != nil {
			return nil, fmt.Errorf("clear export tags: %w", err)
		}
		for _, tag := range input.Tags {
			if _, err := tx.Exec("INSERT INTO export_tags (export_id, tag) VALUES (?, ?)", exportID, tag); err != nil {
				return nil, fmt.Errorf("insert export tag: %w", err)
			}
		}

		keepPaths[input.Path] = struct{}{}
	}

	// Remove exports not in the input.
	for path, exportID := range currentExports {
		if _, keep := keepPaths[path]; !keep {
			if _, err := tx.Exec("DELETE FROM exports WHERE id = ?", exportID); err != nil {
				return nil, fmt.Errorf("delete stale export %q: %w", exportID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit exports: %w", err)
	}

	return s.listExportsForNode(nodeID), nil
}

func (s *sqliteStore) recordHeartbeat(nodeID string, ownerID string, request nodeHeartbeatRequest) error {
	result, err := s.db.Exec(
		"UPDATE nodes SET status = ?, last_seen_at = ? WHERE id = ? AND owner_id = ?",
		request.Status, request.LastSeenAt, nodeID, ownerID)
	if err != nil {
		return fmt.Errorf("update heartbeat: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return errNodeNotFound
	}
	return nil
}

func (s *sqliteStore) listExports(ownerID string) []storageExport {
	rows, err := s.db.Query("SELECT id, node_id, owner_id, label, path, mount_path, capacity_bytes FROM exports WHERE owner_id = ? ORDER BY id", ownerID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var exports []storageExport
	for rows.Next() {
		e := s.scanExport(rows)
		if e.ID != "" {
			exports = append(exports, e)
		}
	}
	if exports == nil {
		exports = []storageExport{}
	}

	// Load protocols and tags for each export.
	for i := range exports {
		exports[i].Protocols = s.loadExportProtocols(exports[i].ID)
		exports[i].Tags = s.loadExportTags(exports[i].ID)
	}

	return exports
}

func (s *sqliteStore) listNodes(ownerID string) []nasNode {
	rows, err := s.db.Query("SELECT id, machine_id, owner_id, display_name, agent_version, status, last_seen_at, direct_address, relay_address FROM nodes WHERE owner_id = ? ORDER BY id", ownerID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var nodes []nasNode
	for rows.Next() {
		node := s.scanNode(rows)
		if node.ID != "" {
			nodes = append(nodes, node)
		}
	}
	if nodes == nil {
		nodes = []nasNode{}
	}

	return nodes
}

func (s *sqliteStore) listExportsForNode(nodeID string) []storageExport {
	rows, err := s.db.Query("SELECT id, node_id, owner_id, label, path, mount_path, capacity_bytes FROM exports WHERE node_id = ? ORDER BY id", nodeID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var exports []storageExport
	for rows.Next() {
		e := s.scanExport(rows)
		if e.ID != "" {
			exports = append(exports, e)
		}
	}
	if exports == nil {
		exports = []storageExport{}
	}

	for i := range exports {
		exports[i].Protocols = s.loadExportProtocols(exports[i].ID)
		exports[i].Tags = s.loadExportTags(exports[i].ID)
	}

	sort.Slice(exports, func(i, j int) bool { return exports[i].ID < exports[j].ID })
	return exports
}

func (s *sqliteStore) exportContext(exportID string, ownerID string) (exportContext, bool) {
	var e storageExport
	var capacityBytes sql.NullInt64
	var exportOwnerID sql.NullString
	err := s.db.QueryRow(
		"SELECT id, node_id, owner_id, label, path, mount_path, capacity_bytes FROM exports WHERE id = ? AND owner_id = ?",
		exportID, ownerID).Scan(&e.ID, &e.NasNodeID, &exportOwnerID, &e.Label, &e.Path, &e.MountPath, &capacityBytes)
	if err != nil {
		return exportContext{}, false
	}
	if exportOwnerID.Valid {
		e.OwnerID = exportOwnerID.String
	}
	if capacityBytes.Valid {
		e.CapacityBytes = &capacityBytes.Int64
	}
	e.Protocols = s.loadExportProtocols(e.ID)
	e.Tags = s.loadExportTags(e.ID)

	node, ok := s.nodeByID(e.NasNodeID)
	if !ok {
		return exportContext{}, false
	}

	return exportContext{export: e, node: node}, true
}

func (s *sqliteStore) nodeByID(nodeID string) (nasNode, bool) {
	row := s.db.QueryRow(
		"SELECT id, machine_id, owner_id, display_name, agent_version, status, last_seen_at, direct_address, relay_address FROM nodes WHERE id = ?",
		nodeID)
	n := s.scanNode(row)
	if n.ID == "" {
		return nasNode{}, false
	}

	return n, true
}

type sqliteNodeScanner interface {
	Scan(dest ...any) error
}

func (s *sqliteStore) scanNode(scanner sqliteNodeScanner) nasNode {
	var n nasNode
	var directAddr, relayAddr sql.NullString
	var lastSeenAt sql.NullString
	var ownerID sql.NullString
	err := scanner.Scan(&n.ID, &n.MachineID, &ownerID, &n.DisplayName, &n.AgentVersion, &n.Status, &lastSeenAt, &directAddr, &relayAddr)
	if err != nil {
		return nasNode{}
	}
	if ownerID.Valid {
		n.OwnerID = ownerID.String
	}
	if lastSeenAt.Valid {
		n.LastSeenAt = lastSeenAt.String
	}
	if directAddr.Valid {
		n.DirectAddress = &directAddr.String
	}
	if relayAddr.Valid {
		n.RelayAddress = &relayAddr.String
	}
	return n
}

func (s *sqliteStore) nodeAuthByMachineID(machineID string) (nodeAuthState, bool) {
	var state nodeAuthState
	var tokenHash sql.NullString
	err := s.db.QueryRow(`
		SELECT n.id, nt.token_hash
		FROM nodes n
		LEFT JOIN node_tokens nt ON nt.node_id = n.id
		WHERE n.machine_id = ?
	`, machineID).Scan(&state.NodeID, &tokenHash)
	if err != nil {
		return nodeAuthState{}, false
	}
	if tokenHash.Valid {
		state.TokenHash = tokenHash.String
	}
	return state, true
}

func (s *sqliteStore) nodeAuthByID(nodeID string) (nodeAuthState, bool) {
	var state nodeAuthState
	var tokenHash sql.NullString
	err := s.db.QueryRow(`
		SELECT n.id, nt.token_hash
		FROM nodes n
		LEFT JOIN node_tokens nt ON nt.node_id = n.id
		WHERE n.id = ?
	`, nodeID).Scan(&state.NodeID, &tokenHash)
	if err != nil {
		return nodeAuthState{}, false
	}
	if tokenHash.Valid {
		state.TokenHash = tokenHash.String
	}
	return state, true
}

// --- helpers ---

func (s *sqliteStore) scanExport(rows *sql.Rows) storageExport {
	var e storageExport
	var capacityBytes sql.NullInt64
	var ownerID sql.NullString
	if err := rows.Scan(&e.ID, &e.NasNodeID, &ownerID, &e.Label, &e.Path, &e.MountPath, &capacityBytes); err != nil {
		return storageExport{}
	}
	if ownerID.Valid {
		e.OwnerID = ownerID.String
	}
	if capacityBytes.Valid {
		e.CapacityBytes = &capacityBytes.Int64
	}
	return e
}

func (s *sqliteStore) loadExportProtocols(exportID string) []string {
	rows, err := s.db.Query("SELECT protocol FROM export_protocols WHERE export_id = ? ORDER BY protocol", exportID)
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	var protocols []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err == nil {
			protocols = append(protocols, p)
		}
	}
	if protocols == nil {
		return []string{}
	}
	return protocols
}

func (s *sqliteStore) loadExportTags(exportID string) []string {
	rows, err := s.db.Query("SELECT tag FROM export_tags WHERE export_id = ? ORDER BY tag", exportID)
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err == nil {
			tags = append(tags, t)
		}
	}
	if tags == nil {
		return []string{}
	}
	return tags
}

func nullableString(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *p, Valid: true}
}

func nullableInt64(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}

// --- user auth ---

func (s *sqliteStore) createUser(username string, password string) (user, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return user{}, fmt.Errorf("hash password: %w", err)
	}

	id, err := newSessionToken()
	if err != nil {
		return user{}, err
	}

	var u user
	err = s.db.QueryRow(`
		INSERT INTO users (id, username, password_hash) VALUES (?, ?, ?)
		RETURNING id, username, created_at
	`, id, username, string(hash)).Scan(&u.ID, &u.Username, &u.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return user{}, errUsernameTaken
		}
		return user{}, fmt.Errorf("create user: %w", err)
	}

	return u, nil
}

func (s *sqliteStore) authenticateUser(username string, password string) (user, error) {
	var u user
	var passwordHash string
	err := s.db.QueryRow(
		"SELECT id, username, password_hash, created_at FROM users WHERE username = ?",
		username).Scan(&u.ID, &u.Username, &passwordHash, &u.CreatedAt)
	if err != nil {
		return user{}, errInvalidLogin
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return user{}, errInvalidLogin
	}

	return u, nil
}

func (s *sqliteStore) createSession(userID string, ttl time.Duration) (string, error) {
	token, err := newSessionToken()
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().UTC().Add(ttl).Format(time.RFC3339)
	_, err = s.db.Exec(
		"INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expiresAt)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	// Clean up expired sessions opportunistically.
	_, _ = s.db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now().UTC().Format(time.RFC3339))

	return token, nil
}

func (s *sqliteStore) validateSession(token string) (user, error) {
	var u user
	err := s.db.QueryRow(`
		SELECT u.id, u.username, u.created_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = ? AND s.expires_at > ?
	`, token, time.Now().UTC().Format(time.RFC3339)).Scan(&u.ID, &u.Username, &u.CreatedAt)
	if err != nil {
		return user{}, errSessionExpired
	}
	return u, nil
}

func (s *sqliteStore) deleteSession(token string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

func newSessionToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return hex.EncodeToString(raw), nil
}
