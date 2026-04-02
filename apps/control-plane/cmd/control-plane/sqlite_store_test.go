package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func newTestSQLiteApp(t *testing.T, config appConfig) (*app, *httptest.Server) {
	t.Helper()

	if config.dbPath == "" {
		config.dbPath = filepath.Join(t.TempDir(), "test.db")
	}

	if config.version == "" {
		config.version = "test-version"
	}

	app, err := newApp(config, testControlPlaneNow)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	app.now = func() time.Time { return testControlPlaneNow }
	seedDefaultSessionUser(t, app)

	server := httptest.NewServer(app.handler())
	return app, server
}

func TestSQLiteHealthAndVersion(t *testing.T) {
	t.Parallel()

	_, server := newTestSQLiteApp(t, appConfig{
		version:          "test-version",
		nextcloudBaseURL: "http://nextcloud.test",
	})
	defer server.Close()

	health := getJSON[controlPlaneHealthResponse](t, server.Client(), server.URL+"/health")
	if health.Status != "ok" {
		t.Fatalf("expected status ok, got %q", health.Status)
	}

	exports := getJSONAuth[[]storageExport](t, server.Client(), testClientToken, server.URL+"/api/v1/exports")
	if len(exports) != 0 {
		t.Fatalf("expected no exports, got %d", len(exports))
	}
}

func TestSQLiteRegistrationAndExports(t *testing.T) {
	t.Parallel()

	_, server := newTestSQLiteApp(t, appConfig{
		version:          "test-version",
		nextcloudBaseURL: "http://nextcloud.test",
	})
	defer server.Close()

	directAddress := "http://nas.local:8090"
	registration := registerNode(t, server.Client(), server.URL+"/api/v1/nodes/register", testNodeBootstrapToken, nodeRegistrationRequest{
		MachineID:     "machine-1",
		DisplayName:   "Primary NAS",
		AgentVersion:  "1.2.3",
		DirectAddress: &directAddress,
		RelayAddress:  nil,
	})
	if registration.NodeToken == "" {
		t.Fatal("expected node registration to preserve the session token")
	}
	if registration.Node.ID != "dev-node" {
		t.Fatalf("expected node ID %q, got %q", "dev-node", registration.Node.ID)
	}

	syncedExports := syncNodeExports(t, server.Client(), registration.NodeToken, server.URL+"/api/v1/nodes/"+registration.Node.ID+"/exports", nodeExportsRequest{
		Exports: []storageExportInput{
			{
				Label:         "Docs",
				Path:          "/srv/docs",
				MountPath:     "/dav/docs/",
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{"work"},
			},
		},
	})
	if len(syncedExports) != 1 {
		t.Fatalf("expected 1 export, got %d", len(syncedExports))
	}
	if syncedExports[0].ID != "dev-export" {
		t.Fatalf("expected export ID %q, got %q", "dev-export", syncedExports[0].ID)
	}
	if syncedExports[0].Label != "Docs" {
		t.Fatalf("expected label %q, got %q", "Docs", syncedExports[0].Label)
	}

	allExports := getJSONAuth[[]storageExport](t, server.Client(), testClientToken, server.URL+"/api/v1/exports")
	if len(allExports) != 1 {
		t.Fatalf("expected 1 export in list, got %d", len(allExports))
	}

	mount := postJSONAuth[mountProfile](t, server.Client(), testClientToken, server.URL+"/api/v1/mount-profiles/issue", mountProfileRequest{ExportID: "dev-export"})
	if mount.MountURL != "http://nas.local:8090/dav/docs/fixture/" {
		t.Fatalf("expected mount URL %q, got %q", "http://nas.local:8090/dav/docs/fixture/", mount.MountURL)
	}
}

func TestSQLiteReRegistrationKeepsNodeID(t *testing.T) {
	t.Parallel()

	_, server := newTestSQLiteApp(t, appConfig{version: "test-version"})
	defer server.Close()

	directAddress := "http://nas.local:8090"
	first := registerNode(t, server.Client(), server.URL+"/api/v1/nodes/register", testNodeBootstrapToken, nodeRegistrationRequest{
		MachineID:     "machine-1",
		DisplayName:   "NAS",
		AgentVersion:  "1.0.0",
		DirectAddress: &directAddress,
	})

	second := registerNode(t, server.Client(), server.URL+"/api/v1/nodes/register", first.NodeToken, nodeRegistrationRequest{
		MachineID:     "machine-1",
		DisplayName:   "NAS Updated",
		AgentVersion:  "1.0.1",
		DirectAddress: &directAddress,
	})

	if second.Node.ID != first.Node.ID {
		t.Fatalf("expected re-registration to keep node ID %q, got %q", first.Node.ID, second.Node.ID)
	}
	if second.NodeToken != first.NodeToken {
		t.Fatalf("expected re-registration to keep the existing session token %q, got %q", first.NodeToken, second.NodeToken)
	}
	if second.Node.DisplayName != "NAS Updated" {
		t.Fatalf("expected updated display name, got %q", second.Node.DisplayName)
	}
}

func TestSQLiteExportSyncRemovesStaleExports(t *testing.T) {
	t.Parallel()

	_, server := newTestSQLiteApp(t, appConfig{version: "test-version"})
	defer server.Close()

	directAddress := "http://nas.local:8090"
	reg := registerNode(t, server.Client(), server.URL+"/api/v1/nodes/register", testNodeBootstrapToken, nodeRegistrationRequest{
		MachineID:     "machine-stale",
		DisplayName:   "NAS",
		AgentVersion:  "1.0.0",
		DirectAddress: &directAddress,
	})

	syncNodeExports(t, server.Client(), reg.NodeToken, server.URL+"/api/v1/nodes/"+reg.Node.ID+"/exports", nodeExportsRequest{
		Exports: []storageExportInput{
			{Label: "A", Path: "/a", MountPath: "/dav/a/", Protocols: []string{"webdav"}, Tags: []string{}},
			{Label: "B", Path: "/b", MountPath: "/dav/b/", Protocols: []string{"webdav"}, Tags: []string{}},
		},
	})

	exports := getJSONAuth[[]storageExport](t, server.Client(), testClientToken, server.URL+"/api/v1/exports")
	if len(exports) != 2 {
		t.Fatalf("expected 2 exports, got %d", len(exports))
	}

	// Sync with only A - B should be removed.
	syncNodeExports(t, server.Client(), reg.NodeToken, server.URL+"/api/v1/nodes/"+reg.Node.ID+"/exports", nodeExportsRequest{
		Exports: []storageExportInput{
			{Label: "A Updated", Path: "/a", MountPath: "/dav/a/", Protocols: []string{"webdav"}, Tags: []string{}},
		},
	})

	exports = getJSONAuth[[]storageExport](t, server.Client(), testClientToken, server.URL+"/api/v1/exports")
	if len(exports) != 1 {
		t.Fatalf("expected 1 export after stale removal, got %d", len(exports))
	}
	if exports[0].Label != "A Updated" {
		t.Fatalf("expected updated label, got %q", exports[0].Label)
	}
}

func TestSQLiteHeartbeat(t *testing.T) {
	t.Parallel()

	app, server := newTestSQLiteApp(t, appConfig{version: "test-version"})
	defer server.Close()
	_ = app

	directAddress := "http://nas.local:8090"
	reg := registerNode(t, server.Client(), server.URL+"/api/v1/nodes/register", testNodeBootstrapToken, nodeRegistrationRequest{
		MachineID:     "machine-hb",
		DisplayName:   "NAS",
		AgentVersion:  "1.0.0",
		DirectAddress: &directAddress,
	})

	postJSONAuthStatus(t, server.Client(), reg.NodeToken, server.URL+"/api/v1/nodes/"+reg.Node.ID+"/heartbeat", nodeHeartbeatRequest{
		NodeID:     reg.Node.ID,
		Status:     "online",
		LastSeenAt: "2025-06-01T12:00:00Z",
	}, http.StatusNoContent)

	node, ok := app.store.nodeByID(reg.Node.ID)
	if !ok {
		t.Fatal("expected node to exist after heartbeat")
	}
	if node.LastSeenAt != "2025-06-01T12:00:00Z" {
		t.Fatalf("expected updated lastSeenAt, got %q", node.LastSeenAt)
	}
}

func TestSQLitePersistsAcrossRestart(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "persist.db")
	directAddress := "http://nas.local:8090"

	_, firstServer := newTestSQLiteApp(t, appConfig{
		version: "test-version",
		dbPath:  dbPath,
	})
	registration := registerNode(t, firstServer.Client(), firstServer.URL+"/api/v1/nodes/register", testNodeBootstrapToken, nodeRegistrationRequest{
		MachineID:     "machine-persist",
		DisplayName:   "Persisted NAS",
		AgentVersion:  "1.2.3",
		DirectAddress: &directAddress,
	})
	syncNodeExports(t, firstServer.Client(), registration.NodeToken, firstServer.URL+"/api/v1/nodes/"+registration.Node.ID+"/exports", nodeExportsRequest{
		Exports: []storageExportInput{{
			Label:     "Docs",
			Path:      "/srv/docs",
			MountPath: "/dav/persisted/",
			Protocols: []string{"webdav"},
			Tags:      []string{"work"},
		}},
	})
	firstServer.Close()

	// Restart with same DB path.
	_, secondServer := newTestSQLiteApp(t, appConfig{
		version: "test-version",
		dbPath:  dbPath,
	})
	defer secondServer.Close()

	exports := getJSONAuth[[]storageExport](t, secondServer.Client(), testClientToken, secondServer.URL+"/api/v1/exports")
	if len(exports) != 1 {
		t.Fatalf("expected persisted export after restart, got %d", len(exports))
	}
	if exports[0].ID != "dev-export" {
		t.Fatalf("expected persisted export ID %q, got %q", "dev-export", exports[0].ID)
	}
	if exports[0].MountPath != "/dav/persisted/" {
		t.Fatalf("expected persisted mountPath %q, got %q", "/dav/persisted/", exports[0].MountPath)
	}
	if len(exports[0].Tags) != 1 || exports[0].Tags[0] != "work" {
		t.Fatalf("expected persisted tags [work], got %v", exports[0].Tags)
	}

	// Re-register with the original node token.
	reReg := registerNode(t, secondServer.Client(), secondServer.URL+"/api/v1/nodes/register", registration.NodeToken, nodeRegistrationRequest{
		MachineID:     "machine-persist",
		DisplayName:   "Persisted NAS Updated",
		AgentVersion:  "1.2.4",
		DirectAddress: &directAddress,
	})
	if reReg.Node.ID != registration.Node.ID {
		t.Fatalf("expected persisted node ID %q, got %q", registration.Node.ID, reReg.Node.ID)
	}
}

func TestSQLiteAuthEnforcement(t *testing.T) {
	t.Parallel()

	_, server := newTestSQLiteApp(t, appConfig{version: "test-version"})
	defer server.Close()

	getStatusWithAuth(t, server.Client(), "", server.URL+"/api/v1/exports", http.StatusUnauthorized)
	getStatusWithAuth(t, server.Client(), "wrong-token", server.URL+"/api/v1/exports", http.StatusUnauthorized)

	postJSONAuthStatus(t, server.Client(), testClientToken, server.URL+"/api/v1/mount-profiles/issue", mountProfileRequest{
		ExportID: "missing-export",
	}, http.StatusNotFound)
}
