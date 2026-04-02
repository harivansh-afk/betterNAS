package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var testControlPlaneNow = time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)

const (
	testPassword           = "password123"
	testClientToken        = "test-client-token"
	testNodeBootstrapToken = "test-node-session-token"
)

type registeredNode struct {
	Node      nasNode
	NodeToken string
}

func TestControlPlaneHealthAndVersion(t *testing.T) {
	t.Parallel()

	_, server := newTestControlPlaneServer(t, appConfig{
		version:          "test-version",
		nextcloudBaseURL: "http://nextcloud.test",
	})
	defer server.Close()

	health := getJSON[controlPlaneHealthResponse](t, server.Client(), server.URL+"/health")
	if health.Service != "control-plane" {
		t.Fatalf("expected service control-plane, got %q", health.Service)
	}
	if health.Status != "ok" {
		t.Fatalf("expected status ok, got %q", health.Status)
	}
	if health.Timestamp != testControlPlaneNow.Format(time.RFC3339) {
		t.Fatalf("expected timestamp %q, got %q", testControlPlaneNow.Format(time.RFC3339), health.Timestamp)
	}
	if health.UptimeSeconds != 0 {
		t.Fatalf("expected uptimeSeconds 0, got %d", health.UptimeSeconds)
	}
	if !health.Nextcloud.Configured {
		t.Fatal("expected nextcloud.configured to be true")
	}
	if health.Nextcloud.BaseURL != "http://nextcloud.test" {
		t.Fatalf("expected baseUrl http://nextcloud.test, got %q", health.Nextcloud.BaseURL)
	}
	if health.Nextcloud.Provider != "nextcloud" {
		t.Fatalf("expected provider nextcloud, got %q", health.Nextcloud.Provider)
	}

	version := getJSON[controlPlaneVersionResponse](t, server.Client(), server.URL+"/version")
	if version.Service != "control-plane" {
		t.Fatalf("expected version service control-plane, got %q", version.Service)
	}
	if version.Version != "test-version" {
		t.Fatalf("expected version test-version, got %q", version.Version)
	}
	if version.APIVersion != "v1" {
		t.Fatalf("expected apiVersion v1, got %q", version.APIVersion)
	}

	getStatusWithAuth(t, server.Client(), "", server.URL+"/api/v1/exports", http.StatusUnauthorized)

	exports := getJSONAuth[[]storageExport](t, server.Client(), testClientToken, server.URL+"/api/v1/exports")
	if len(exports) != 0 {
		t.Fatalf("expected no exports before registration, got %d", len(exports))
	}
}

func TestControlPlaneRegistrationProfilesAndHeartbeat(t *testing.T) {
	t.Parallel()

	app, server := newTestControlPlaneServer(t, appConfig{
		version:          "test-version",
		nextcloudBaseURL: "http://nextcloud.test",
	})
	defer server.Close()

	directAddress := "http://nas.local:8090"
	relayAddress := "http://nas.internal:8090"
	registration := registerNode(t, server.Client(), server.URL+"/api/v1/nodes/register", testNodeBootstrapToken, nodeRegistrationRequest{
		MachineID:     "machine-1",
		DisplayName:   "Primary NAS",
		AgentVersion:  "1.2.3",
		DirectAddress: &directAddress,
		RelayAddress:  &relayAddress,
	})
	if registration.NodeToken == "" {
		t.Fatal("expected node registration to preserve the session token")
	}

	syncedExports := syncNodeExports(t, server.Client(), registration.NodeToken, server.URL+"/api/v1/nodes/"+registration.Node.ID+"/exports", nodeExportsRequest{
		Exports: []storageExportInput{{
			Label:         "Photos",
			Path:          "/srv/photos",
			MountPath:     "/dav/",
			Protocols:     []string{"webdav"},
			CapacityBytes: nil,
			Tags:          []string{"family"},
		}},
	})
	if len(syncedExports) != 1 {
		t.Fatalf("expected sync to return 1 export, got %d", len(syncedExports))
	}

	node := registration.Node
	if node.ID != "dev-node" {
		t.Fatalf("expected first node ID %q, got %q", "dev-node", node.ID)
	}
	if node.Status != "online" {
		t.Fatalf("expected registered node to be online, got %q", node.Status)
	}
	if node.LastSeenAt != testControlPlaneNow.Format(time.RFC3339) {
		t.Fatalf("expected lastSeenAt %q, got %q", testControlPlaneNow.Format(time.RFC3339), node.LastSeenAt)
	}
	if node.DirectAddress == nil || *node.DirectAddress != directAddress {
		t.Fatalf("expected directAddress %q, got %#v", directAddress, node.DirectAddress)
	}
	if node.RelayAddress == nil || *node.RelayAddress != relayAddress {
		t.Fatalf("expected relayAddress %q, got %#v", relayAddress, node.RelayAddress)
	}

	exports := getJSONAuth[[]storageExport](t, server.Client(), testClientToken, server.URL+"/api/v1/exports")
	if len(exports) != 1 {
		t.Fatalf("expected 1 export, got %d", len(exports))
	}
	if exports[0].ID != "dev-export" {
		t.Fatalf("expected first export ID %q, got %q", "dev-export", exports[0].ID)
	}
	if exports[0].NasNodeID != node.ID {
		t.Fatalf("expected export to belong to %q, got %q", node.ID, exports[0].NasNodeID)
	}
	if exports[0].Label != "Photos" {
		t.Fatalf("expected export label Photos, got %q", exports[0].Label)
	}
	if exports[0].Path != "/srv/photos" {
		t.Fatalf("expected export path %q, got %q", "/srv/photos", exports[0].Path)
	}
	if exports[0].MountPath != "/dav/" {
		t.Fatalf("expected mountPath %q, got %q", "/dav/", exports[0].MountPath)
	}

	mount := postJSONAuth[mountProfile](t, server.Client(), testClientToken, server.URL+"/api/v1/mount-profiles/issue", mountProfileRequest{
		ExportID: exports[0].ID,
	})
	if mount.ExportID != exports[0].ID {
		t.Fatalf("expected mount profile exportId %q, got %q", exports[0].ID, mount.ExportID)
	}
	if mount.Protocol != "webdav" {
		t.Fatalf("expected mount protocol webdav, got %q", mount.Protocol)
	}
	if mount.DisplayName != "Photos" {
		t.Fatalf("expected mount display name Photos, got %q", mount.DisplayName)
	}
	if mount.MountURL != "http://nas.local:8090/dav/" {
		t.Fatalf("expected mount URL %q, got %q", "http://nas.local:8090/dav/", mount.MountURL)
	}
	if mount.Readonly {
		t.Fatal("expected mount profile to be read-write")
	}
	if mount.Credential.Mode != mountCredentialModeBasicAuth {
		t.Fatalf("expected credential mode %q, got %q", mountCredentialModeBasicAuth, mount.Credential.Mode)
	}
	if mount.Credential.Username != "fixture" {
		t.Fatalf("expected mount credential username %q, got %q", "fixture", mount.Credential.Username)
	}
	if mount.Credential.Password != "" {
		t.Fatalf("expected mount credential password to be blank, got %q", mount.Credential.Password)
	}
	if mount.Credential.ExpiresAt != "" {
		t.Fatalf("expected mount credential expiry to be blank, got %q", mount.Credential.ExpiresAt)
	}

	cloud := postJSONAuth[cloudProfile](t, server.Client(), testClientToken, server.URL+"/api/v1/cloud-profiles/issue", cloudProfileRequest{
		UserID:   "user-1",
		ExportID: exports[0].ID,
		Provider: "nextcloud",
	})
	if cloud.ExportID != exports[0].ID {
		t.Fatalf("expected cloud profile exportId %q, got %q", exports[0].ID, cloud.ExportID)
	}
	if cloud.Provider != "nextcloud" {
		t.Fatalf("expected provider nextcloud, got %q", cloud.Provider)
	}
	if cloud.BaseURL != "http://nextcloud.test" {
		t.Fatalf("expected baseUrl http://nextcloud.test, got %q", cloud.BaseURL)
	}
	expectedCloudPath := cloudProfilePathForExport(exports[0].ID)
	if cloud.Path != expectedCloudPath {
		t.Fatalf("expected cloud profile path %q, got %q", expectedCloudPath, cloud.Path)
	}

	postJSONAuthStatus(t, server.Client(), registration.NodeToken, server.URL+"/api/v1/nodes/"+node.ID+"/heartbeat", nodeHeartbeatRequest{
		NodeID:     node.ID,
		Status:     "degraded",
		LastSeenAt: "2025-01-02T03:04:05Z",
	}, http.StatusNoContent)

	updatedNode, ok := app.store.nodeByID(node.ID)
	if !ok {
		t.Fatalf("expected node %q to exist after heartbeat", node.ID)
	}
	if updatedNode.Status != "degraded" {
		t.Fatalf("expected heartbeat to update status to degraded, got %q", updatedNode.Status)
	}
	if updatedNode.LastSeenAt != "2025-01-02T03:04:05Z" {
		t.Fatalf("expected heartbeat to update lastSeenAt, got %q", updatedNode.LastSeenAt)
	}
}

func TestControlPlaneExportSyncReconcilesExportsAndKeepsStableIDs(t *testing.T) {
	t.Parallel()

	app, server := newTestControlPlaneServer(t, appConfig{version: "test-version"})
	defer server.Close()

	directAddress := "http://nas.local:8090"
	firstRegistration := registerNode(t, server.Client(), server.URL+"/api/v1/nodes/register", testNodeBootstrapToken, nodeRegistrationRequest{
		MachineID:     "machine-1",
		DisplayName:   "Primary NAS",
		AgentVersion:  "1.2.3",
		DirectAddress: &directAddress,
		RelayAddress:  nil,
	})

	putJSONAuthStatus(t, server.Client(), "wrong-session-token", server.URL+"/api/v1/nodes/"+firstRegistration.Node.ID+"/exports", nodeExportsRequest{
		Exports: []storageExportInput{
			{
				Label:         "Docs",
				Path:          "/srv/docs",
				MountPath:     "/dav/exports/docs/",
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{"work"},
			},
			{
				Label:         "Media",
				Path:          "/srv/media",
				MountPath:     "/dav/exports/media/",
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{"personal"},
			},
		},
	}, http.StatusUnauthorized)

	syncNodeExports(t, server.Client(), firstRegistration.NodeToken, server.URL+"/api/v1/nodes/"+firstRegistration.Node.ID+"/exports", nodeExportsRequest{
		Exports: []storageExportInput{
			{
				Label:         "Docs",
				Path:          "/srv/docs",
				MountPath:     "/dav/exports/docs/",
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{"work"},
			},
			{
				Label:         "Media",
				Path:          "/srv/media",
				MountPath:     "/dav/exports/media/",
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{"personal"},
			},
		},
	})

	initialExports := exportsByPath(getJSONAuth[[]storageExport](t, server.Client(), testClientToken, server.URL+"/api/v1/exports"))
	docsExport := initialExports["/srv/docs"]
	mediaExport := initialExports["/srv/media"]

	secondRegistration := registerNode(t, server.Client(), server.URL+"/api/v1/nodes/register", firstRegistration.NodeToken, nodeRegistrationRequest{
		MachineID:     "machine-1",
		DisplayName:   "Primary NAS Updated",
		AgentVersion:  "1.2.4",
		DirectAddress: &directAddress,
		RelayAddress:  nil,
	})

	putJSONAuthStatus(t, server.Client(), "wrong-session-token", server.URL+"/api/v1/nodes/"+firstRegistration.Node.ID+"/exports", nodeExportsRequest{
		Exports: []storageExportInput{
			{
				Label:         "Docs v2",
				Path:          "/srv/docs",
				MountPath:     "/dav/exports/docs-v2/",
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{"work", "updated"},
			},
			{
				Label:         "Backups",
				Path:          "/srv/backups",
				MountPath:     "/dav/exports/backups/",
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{"system"},
			},
		},
	}, http.StatusUnauthorized)

	syncNodeExports(t, server.Client(), firstRegistration.NodeToken, server.URL+"/api/v1/nodes/"+firstRegistration.Node.ID+"/exports", nodeExportsRequest{
		Exports: []storageExportInput{
			{
				Label:         "Docs v2",
				Path:          "/srv/docs",
				MountPath:     "/dav/exports/docs-v2/",
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{"work", "updated"},
			},
			{
				Label:         "Backups",
				Path:          "/srv/backups",
				MountPath:     "/dav/exports/backups/",
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{"system"},
			},
		},
	})

	if secondRegistration.Node.ID != firstRegistration.Node.ID {
		t.Fatalf("expected re-registration to keep node ID %q, got %q", firstRegistration.Node.ID, secondRegistration.Node.ID)
	}
	if secondRegistration.NodeToken != firstRegistration.NodeToken {
		t.Fatalf("expected re-registration to keep the existing session token %q, got %q", firstRegistration.NodeToken, secondRegistration.NodeToken)
	}

	updatedExports := exportsByPath(getJSONAuth[[]storageExport](t, server.Client(), testClientToken, server.URL+"/api/v1/exports"))
	if len(updatedExports) != 2 {
		t.Fatalf("expected 2 exports after re-registration, got %d", len(updatedExports))
	}
	if updatedExports["/srv/docs"].ID != docsExport.ID {
		t.Fatalf("expected docs export to keep ID %q, got %q", docsExport.ID, updatedExports["/srv/docs"].ID)
	}
	if updatedExports["/srv/docs"].Label != "Docs v2" {
		t.Fatalf("expected docs export label to update, got %q", updatedExports["/srv/docs"].Label)
	}
	if updatedExports["/srv/docs"].MountPath != "/dav/exports/docs-v2/" {
		t.Fatalf("expected docs export mountPath to update, got %q", updatedExports["/srv/docs"].MountPath)
	}
	if _, ok := updatedExports["/srv/media"]; ok {
		t.Fatalf("expected stale media export %q to be removed", mediaExport.ID)
	}
	if updatedExports["/srv/backups"].ID == docsExport.ID {
		t.Fatal("expected backups export to get a distinct ID")
	}

	storedNode, ok := app.store.nodeByID(firstRegistration.Node.ID)
	if !ok {
		t.Fatalf("expected node %q to exist after re-registration", firstRegistration.Node.ID)
	}
	if storedNode.DisplayName != "Primary NAS Updated" {
		t.Fatalf("expected updated display name, got %q", storedNode.DisplayName)
	}
	if storedNode.AgentVersion != "1.2.4" {
		t.Fatalf("expected updated agent version, got %q", storedNode.AgentVersion)
	}
}

func TestControlPlaneProfilesRemainExportSpecificForConfiguredMountPaths(t *testing.T) {
	t.Parallel()

	_, server := newTestControlPlaneServer(t, appConfig{
		version:          "test-version",
		nextcloudBaseURL: "http://nextcloud.test",
	})
	defer server.Close()

	directAddress := "http://nas.local:8090"
	registration := registerNode(t, server.Client(), server.URL+"/api/v1/nodes/register", testNodeBootstrapToken, nodeRegistrationRequest{
		MachineID:     "machine-multi",
		DisplayName:   "Multi Export NAS",
		AgentVersion:  "1.2.3",
		DirectAddress: &directAddress,
		RelayAddress:  nil,
	})
	syncNodeExports(t, server.Client(), registration.NodeToken, server.URL+"/api/v1/nodes/"+registration.Node.ID+"/exports", nodeExportsRequest{
		Exports: []storageExportInput{
			{
				Label:         "Docs",
				Path:          "/srv/docs",
				MountPath:     "/dav/exports/docs/",
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{"work"},
			},
			{
				Label:         "Media",
				Path:          "/srv/media",
				MountPath:     "/dav/exports/media/",
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{"personal"},
			},
		},
	})

	exports := exportsByPath(getJSONAuth[[]storageExport](t, server.Client(), testClientToken, server.URL+"/api/v1/exports"))
	docsExport := exports["/srv/docs"]
	mediaExport := exports["/srv/media"]

	docsMount := postJSONAuth[mountProfile](t, server.Client(), testClientToken, server.URL+"/api/v1/mount-profiles/issue", mountProfileRequest{ExportID: docsExport.ID})
	mediaMount := postJSONAuth[mountProfile](t, server.Client(), testClientToken, server.URL+"/api/v1/mount-profiles/issue", mountProfileRequest{ExportID: mediaExport.ID})
	if docsMount.MountURL == mediaMount.MountURL {
		t.Fatalf("expected distinct mount URLs for configured export paths, got %q", docsMount.MountURL)
	}
	if docsMount.MountURL != "http://nas.local:8090/dav/exports/docs/" {
		t.Fatalf("expected docs mount URL %q, got %q", "http://nas.local:8090/dav/exports/docs/", docsMount.MountURL)
	}
	if mediaMount.MountURL != "http://nas.local:8090/dav/exports/media/" {
		t.Fatalf("expected media mount URL %q, got %q", "http://nas.local:8090/dav/exports/media/", mediaMount.MountURL)
	}

	docsCloud := postJSONAuth[cloudProfile](t, server.Client(), testClientToken, server.URL+"/api/v1/cloud-profiles/issue", cloudProfileRequest{
		UserID:   "user-1",
		ExportID: docsExport.ID,
		Provider: "nextcloud",
	})
	mediaCloud := postJSONAuth[cloudProfile](t, server.Client(), testClientToken, server.URL+"/api/v1/cloud-profiles/issue", cloudProfileRequest{
		UserID:   "user-1",
		ExportID: mediaExport.ID,
		Provider: "nextcloud",
	})
	if docsCloud.Path == mediaCloud.Path {
		t.Fatalf("expected distinct cloud profile paths for multi-export node, got %q", docsCloud.Path)
	}
	if docsCloud.Path != cloudProfilePathForExport(docsExport.ID) {
		t.Fatalf("expected docs cloud path %q, got %q", cloudProfilePathForExport(docsExport.ID), docsCloud.Path)
	}
	if mediaCloud.Path != cloudProfilePathForExport(mediaExport.ID) {
		t.Fatalf("expected media cloud path %q, got %q", cloudProfilePathForExport(mediaExport.ID), mediaCloud.Path)
	}
}

func TestControlPlaneMountProfilesUseRelayAndPreserveBasePath(t *testing.T) {
	t.Parallel()

	_, server := newTestControlPlaneServer(t, appConfig{version: "test-version"})
	defer server.Close()

	relayAddress := "https://nas.example.test/control"
	registration := registerNode(t, server.Client(), server.URL+"/api/v1/nodes/register", testNodeBootstrapToken, nodeRegistrationRequest{
		MachineID:     "machine-relay",
		DisplayName:   "Relay NAS",
		AgentVersion:  "1.2.3",
		DirectAddress: nil,
		RelayAddress:  &relayAddress,
	})
	syncNodeExports(t, server.Client(), registration.NodeToken, server.URL+"/api/v1/nodes/"+registration.Node.ID+"/exports", nodeExportsRequest{
		Exports: []storageExportInput{{
			Label:         "Relay",
			Path:          "/srv/relay",
			MountPath:     "/dav/relay/",
			Protocols:     []string{"webdav"},
			CapacityBytes: nil,
			Tags:          []string{},
		}},
	})

	mount := postJSONAuth[mountProfile](t, server.Client(), testClientToken, server.URL+"/api/v1/mount-profiles/issue", mountProfileRequest{ExportID: "dev-export"})
	if mount.MountURL != "https://nas.example.test/control/dav/relay/" {
		t.Fatalf("expected relay mount URL %q, got %q", "https://nas.example.test/control/dav/relay/", mount.MountURL)
	}

	registration = registerNode(t, server.Client(), server.URL+"/api/v1/nodes/register", testNodeBootstrapToken, nodeRegistrationRequest{
		MachineID:     "machine-no-target",
		DisplayName:   "No Target NAS",
		AgentVersion:  "1.2.3",
		DirectAddress: nil,
		RelayAddress:  nil,
	})
	syncNodeExports(t, server.Client(), registration.NodeToken, server.URL+"/api/v1/nodes/"+registration.Node.ID+"/exports", nodeExportsRequest{
		Exports: []storageExportInput{{
			Label:         "Offline",
			Path:          "/srv/offline",
			MountPath:     "/dav/",
			Protocols:     []string{"webdav"},
			CapacityBytes: nil,
			Tags:          []string{},
		}},
	})

	postJSONAuthStatus(t, server.Client(), testClientToken, server.URL+"/api/v1/mount-profiles/issue", mountProfileRequest{ExportID: "dev-export-2"}, http.StatusServiceUnavailable)
}

func TestControlPlaneCloudProfilesRequireConfiguredBaseURLAndExistingExport(t *testing.T) {
	t.Parallel()

	_, server := newTestControlPlaneServer(t, appConfig{version: "test-version"})
	defer server.Close()

	directAddress := "http://nas.local:8090"
	registration := registerNode(t, server.Client(), server.URL+"/api/v1/nodes/register", testNodeBootstrapToken, nodeRegistrationRequest{
		MachineID:     "machine-cloud",
		DisplayName:   "Cloud NAS",
		AgentVersion:  "1.2.3",
		DirectAddress: &directAddress,
		RelayAddress:  nil,
	})
	syncNodeExports(t, server.Client(), registration.NodeToken, server.URL+"/api/v1/nodes/"+registration.Node.ID+"/exports", nodeExportsRequest{
		Exports: []storageExportInput{{
			Label:         "Photos",
			Path:          "/srv/photos",
			MountPath:     "/dav/",
			Protocols:     []string{"webdav"},
			CapacityBytes: nil,
			Tags:          []string{},
		}},
	})

	postJSONAuthStatus(t, server.Client(), testClientToken, server.URL+"/api/v1/cloud-profiles/issue", cloudProfileRequest{
		UserID:   "user-1",
		ExportID: "dev-export",
		Provider: "nextcloud",
	}, http.StatusServiceUnavailable)

	_, serverWithNextcloud := newTestControlPlaneServer(t, appConfig{
		version:          "test-version",
		nextcloudBaseURL: "http://nextcloud.test",
	})
	defer serverWithNextcloud.Close()

	postJSONAuthStatus(t, serverWithNextcloud.Client(), testClientToken, serverWithNextcloud.URL+"/api/v1/cloud-profiles/issue", cloudProfileRequest{
		UserID:   "user-1",
		ExportID: "missing-export",
		Provider: "nextcloud",
	}, http.StatusNotFound)
}

func TestControlPlanePersistsRegistryAcrossAppRestart(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "control-plane.db")
	directAddress := "http://nas.local:8090"

	_, firstServer := newTestControlPlaneServer(t, appConfig{
		version: "test-version",
		dbPath:  dbPath,
	})
	registration := registerNode(t, firstServer.Client(), firstServer.URL+"/api/v1/nodes/register", testNodeBootstrapToken, nodeRegistrationRequest{
		MachineID:     "machine-persisted",
		DisplayName:   "Persisted NAS",
		AgentVersion:  "1.2.3",
		DirectAddress: &directAddress,
		RelayAddress:  nil,
	})
	syncNodeExports(t, firstServer.Client(), registration.NodeToken, firstServer.URL+"/api/v1/nodes/"+registration.Node.ID+"/exports", nodeExportsRequest{
		Exports: []storageExportInput{{
			Label:         "Docs",
			Path:          "/srv/docs",
			MountPath:     "/dav/persisted/",
			Protocols:     []string{"webdav"},
			CapacityBytes: nil,
			Tags:          []string{"work"},
		}},
	})
	firstServer.Close()

	_, secondServer := newTestControlPlaneServer(t, appConfig{
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

	mount := postJSONAuth[mountProfile](t, secondServer.Client(), testClientToken, secondServer.URL+"/api/v1/mount-profiles/issue", mountProfileRequest{ExportID: exports[0].ID})
	if mount.MountURL != "http://nas.local:8090/dav/persisted/" {
		t.Fatalf("expected persisted mount URL %q, got %q", "http://nas.local:8090/dav/persisted/", mount.MountURL)
	}

	reRegistration := registerNode(t, secondServer.Client(), secondServer.URL+"/api/v1/nodes/register", registration.NodeToken, nodeRegistrationRequest{
		MachineID:     "machine-persisted",
		DisplayName:   "Persisted NAS Updated",
		AgentVersion:  "1.2.4",
		DirectAddress: &directAddress,
		RelayAddress:  nil,
	})
	syncNodeExports(t, secondServer.Client(), registration.NodeToken, secondServer.URL+"/api/v1/nodes/"+registration.Node.ID+"/exports", nodeExportsRequest{
		Exports: []storageExportInput{{
			Label:         "Docs Updated",
			Path:          "/srv/docs",
			MountPath:     "/dav/persisted/",
			Protocols:     []string{"webdav"},
			CapacityBytes: nil,
			Tags:          []string{"work"},
		}},
	})
	if reRegistration.Node.ID != registration.Node.ID {
		t.Fatalf("expected persisted node ID %q, got %q", registration.Node.ID, reRegistration.Node.ID)
	}
}

func TestControlPlaneRejectsInvalidRequestsAndEnforcesAuth(t *testing.T) {
	t.Parallel()

	_, server := newTestControlPlaneServer(t, appConfig{version: "test-version"})
	defer server.Close()

	postRawJSONStatus(t, server.Client(), server.URL+"/api/v1/nodes/register", `{
		"machineId":"machine-1",
		"displayName":"Primary NAS",
		"agentVersion":"1.2.3",
		"directAddress":"http://nas.local:8090",
		"relayAddress":null
	}`, http.StatusUnauthorized)

	postRawJSONAuthStatus(t, server.Client(), testNodeBootstrapToken, server.URL+"/api/v1/nodes/register", `{
		"machineId":"machine-1",
		"displayName":"Primary NAS",
		"agentVersion":"1.2.3",
		"relayAddress":null
	}`, http.StatusBadRequest)

	postRawJSONAuthStatus(t, server.Client(), testNodeBootstrapToken, server.URL+"/api/v1/nodes/register", `{
		"machineId":"machine-1",
		"displayName":"Primary NAS",
		"agentVersion":"1.2.3",
		"directAddress":"nas.local:8090",
	}`, http.StatusBadRequest)

	response := postRawJSONAuth(t, server.Client(), testNodeBootstrapToken, server.URL+"/api/v1/nodes/register", `{
		"machineId":"machine-1",
		"displayName":"Primary NAS",
		"agentVersion":"1.2.3",
		"directAddress":"http://nas.local:8090",
		"relayAddress":null,
		"ignoredTopLevel":"ok"
	}`)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("post %s: expected status 200, got %d: %s", server.URL+"/api/v1/nodes/register", response.StatusCode, body)
	}

	var node nasNode
	if err := json.NewDecoder(response.Body).Decode(&node); err != nil {
		t.Fatalf("decode registration response: %v", err)
	}
	nodeToken := testNodeBootstrapToken
	if node.ID != "dev-node" {
		t.Fatalf("expected node ID %q, got %q", "dev-node", node.ID)
	}

	putJSONAuthStatus(t, server.Client(), "wrong-session-token", server.URL+"/api/v1/nodes/"+node.ID+"/exports", nodeExportsRequest{
		Exports: []storageExportInput{{
			Label:         "Docs",
			Path:          "/srv/docs",
			MountPath:     "/dav/docs/",
			Protocols:     []string{"webdav"},
			CapacityBytes: nil,
			Tags:          []string{},
		}},
	}, http.StatusUnauthorized)

	putJSONAuthStatus(t, server.Client(), nodeToken, server.URL+"/api/v1/nodes/"+node.ID+"/exports", nodeExportsRequest{
		Exports: []storageExportInput{
			{
				Label:         "Docs",
				Path:          "/srv/docs",
				MountPath:     "/dav/docs/",
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{},
			},
			{
				Label:         "Media",
				Path:          "/srv/media",
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{},
			},
		},
	}, http.StatusBadRequest)

	putJSONAuthStatus(t, server.Client(), nodeToken, server.URL+"/api/v1/nodes/"+node.ID+"/exports", nodeExportsRequest{
		Exports: []storageExportInput{
			{
				Label:         "Docs",
				Path:          "/srv/docs",
				MountPath:     "/dav/docs/",
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{},
			},
			{
				Label:         "Docs Duplicate",
				Path:          "/srv/docs-2",
				MountPath:     "/dav/docs/",
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{},
			},
		},
	}, http.StatusBadRequest)

	postJSONAuthStatus(t, server.Client(), "wrong-session-token", server.URL+"/api/v1/nodes/"+node.ID+"/heartbeat", nodeHeartbeatRequest{
		NodeID:     node.ID,
		Status:     "online",
		LastSeenAt: "2025-01-02T03:04:05Z",
	}, http.StatusUnauthorized)

	postJSONAuthStatus(t, server.Client(), nodeToken, server.URL+"/api/v1/nodes/"+node.ID+"/heartbeat", nodeHeartbeatRequest{
		NodeID:     "node-other",
		Status:     "online",
		LastSeenAt: "2025-01-02T03:04:05Z",
	}, http.StatusBadRequest)

	postJSONAuthStatus(t, server.Client(), nodeToken, server.URL+"/api/v1/nodes/"+node.ID+"/heartbeat", nodeHeartbeatRequest{
		NodeID:     node.ID,
		Status:     "broken",
		LastSeenAt: "2025-01-02T03:04:05Z",
	}, http.StatusBadRequest)

	postJSONAuthStatus(t, server.Client(), nodeToken, server.URL+"/api/v1/nodes/"+node.ID+"/heartbeat", nodeHeartbeatRequest{
		NodeID:     node.ID,
		Status:     "online",
		LastSeenAt: "not-a-timestamp",
	}, http.StatusBadRequest)

	postJSONAuthStatus(t, server.Client(), nodeToken, server.URL+"/api/v1/nodes/missing-node/heartbeat", nodeHeartbeatRequest{
		NodeID:     "missing-node",
		Status:     "online",
		LastSeenAt: "2025-01-02T03:04:05Z",
	}, http.StatusNotFound)

	getStatusWithAuth(t, server.Client(), "", server.URL+"/api/v1/exports", http.StatusUnauthorized)
	getStatusWithAuth(t, server.Client(), "wrong-client-token", server.URL+"/api/v1/exports", http.StatusUnauthorized)

	postRawJSONAuthStatus(t, server.Client(), testClientToken, server.URL+"/api/v1/mount-profiles/issue", `{}`, http.StatusBadRequest)

	postJSONAuthStatus(t, server.Client(), testClientToken, server.URL+"/api/v1/mount-profiles/issue", mountProfileRequest{
		ExportID: "missing-export",
	}, http.StatusNotFound)

	postJSONAuthStatus(t, server.Client(), testClientToken, server.URL+"/api/v1/cloud-profiles/issue", cloudProfileRequest{
		UserID:   "user-1",
		ExportID: "missing-export",
		Provider: "nextcloud",
	}, http.StatusNotFound)
}

func newTestControlPlaneServer(t *testing.T, config appConfig) (*app, *httptest.Server) {
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
	app.now = func() time.Time {
		return testControlPlaneNow
	}
	seedDefaultSessionUser(t, app)

	server := httptest.NewServer(app.handler())
	return app, server
}

func seedDefaultSessionUser(t *testing.T, app *app) {
	t.Helper()

	u, err := app.store.createUser("fixture", testPassword)
	if err != nil && !errors.Is(err, errUsernameTaken) {
		t.Fatalf("seed default test user: %v", err)
	}
	if errors.Is(err, errUsernameTaken) {
		u, err = app.store.authenticateUser("fixture", testPassword)
		if err != nil {
			t.Fatalf("authenticate seeded test user: %v", err)
		}
	}

	sqliteStore, ok := app.store.(*sqliteStore)
	if !ok {
		return
	}

	createdAt := time.Now().UTC().Format(time.RFC3339)
	expiresAt := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	for _, token := range []string{testClientToken, testNodeBootstrapToken} {
		if _, err := sqliteStore.db.Exec(
			"INSERT OR REPLACE INTO sessions (token, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)",
			token,
			u.ID,
			createdAt,
			expiresAt,
		); err != nil {
			t.Fatalf("seed session %s: %v", token, err)
		}
	}
}

func exportsByPath(exports []storageExport) map[string]storageExport {
	byPath := make(map[string]storageExport, len(exports))
	for _, export := range exports {
		byPath[export.Path] = export
	}

	return byPath
}

func registerNode(t *testing.T, client *http.Client, endpoint string, token string, payload nodeRegistrationRequest) registeredNode {
	t.Helper()

	response := postJSONAuthResponse(t, client, token, endpoint, payload)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(response.Body)
		t.Fatalf("post %s: expected status 200, got %d: %s", endpoint, response.StatusCode, responseBody)
	}

	var node nasNode
	if err := json.NewDecoder(response.Body).Decode(&node); err != nil {
		t.Fatalf("decode %s response: %v", endpoint, err)
	}

	return registeredNode{
		Node:      node,
		NodeToken: strings.TrimSpace(token),
	}
}

func registerSessionUser(t *testing.T, client *http.Client, baseURL string, username string) authLoginResponse {
	t.Helper()

	return postJSONAuthCreated[authLoginResponse](t, client, "", baseURL+"/api/v1/auth/register", authRegisterRequest{
		Username: username,
		Password: testPassword,
	})
}

func syncNodeExports(t *testing.T, client *http.Client, token string, endpoint string, payload nodeExportsRequest) []storageExport {
	t.Helper()

	response := putJSONAuthResponse(t, client, token, endpoint, payload)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(response.Body)
		t.Fatalf("put %s: expected status 200, got %d: %s", endpoint, response.StatusCode, responseBody)
	}

	var exports []storageExport
	if err := json.NewDecoder(response.Body).Decode(&exports); err != nil {
		t.Fatalf("decode %s response: %v", endpoint, err)
	}

	return exports
}

func getJSON[T any](t *testing.T, client *http.Client, endpoint string) T {
	t.Helper()

	response := doRequest(t, client, http.MethodGet, endpoint, nil, nil)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("get %s: expected status 200, got %d: %s", endpoint, response.StatusCode, body)
	}

	var payload T
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode %s response: %v", endpoint, err)
	}

	return payload
}

func getJSONAuth[T any](t *testing.T, client *http.Client, token string, endpoint string) T {
	t.Helper()

	response := doRequest(t, client, http.MethodGet, endpoint, nil, authHeaders(token))
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("get %s: expected status 200, got %d: %s", endpoint, response.StatusCode, body)
	}

	var payload T
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode %s response: %v", endpoint, err)
	}

	return payload
}

func getStatusWithAuth(t *testing.T, client *http.Client, token string, endpoint string, expectedStatus int) {
	t.Helper()

	response := doRequest(t, client, http.MethodGet, endpoint, nil, authHeaders(token))
	defer response.Body.Close()

	if response.StatusCode != expectedStatus {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("get %s: expected status %d, got %d: %s", endpoint, expectedStatus, response.StatusCode, body)
	}
}

func postJSONAuth[T any](t *testing.T, client *http.Client, token string, endpoint string, payload any) T {
	t.Helper()

	response := postJSONAuthResponse(t, client, token, endpoint, payload)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(response.Body)
		t.Fatalf("post %s: expected status 200, got %d: %s", endpoint, response.StatusCode, responseBody)
	}

	var decoded T
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode %s response: %v", endpoint, err)
	}

	return decoded
}

func postJSONAuthStatus(t *testing.T, client *http.Client, token string, endpoint string, payload any, expectedStatus int) {
	t.Helper()

	response := postJSONAuthResponse(t, client, token, endpoint, payload)
	defer response.Body.Close()

	if response.StatusCode != expectedStatus {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("post %s: expected status %d, got %d: %s", endpoint, expectedStatus, response.StatusCode, body)
	}
}

func putJSONAuthStatus(t *testing.T, client *http.Client, token string, endpoint string, payload any, expectedStatus int) {
	t.Helper()

	response := putJSONAuthResponse(t, client, token, endpoint, payload)
	defer response.Body.Close()

	if response.StatusCode != expectedStatus {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("put %s: expected status %d, got %d: %s", endpoint, expectedStatus, response.StatusCode, body)
	}
}

func postJSONAuthResponse(t *testing.T, client *http.Client, token string, endpoint string, payload any) *http.Response {
	t.Helper()

	return jsonAuthResponse(t, client, http.MethodPost, token, endpoint, payload)
}

func putJSONAuthResponse(t *testing.T, client *http.Client, token string, endpoint string, payload any) *http.Response {
	t.Helper()

	return jsonAuthResponse(t, client, http.MethodPut, token, endpoint, payload)
}

func jsonAuthResponse(t *testing.T, client *http.Client, method string, token string, endpoint string, payload any) *http.Response {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload for %s: %v", endpoint, err)
	}

	return doRequest(t, client, method, endpoint, bytes.NewReader(body), authHeaders(token))
}

func postRawJSONAuthStatus(t *testing.T, client *http.Client, token string, endpoint string, raw string, expectedStatus int) {
	t.Helper()

	response := postRawJSONAuth(t, client, token, endpoint, raw)
	defer response.Body.Close()

	if response.StatusCode != expectedStatus {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("post %s: expected status %d, got %d: %s", endpoint, expectedStatus, response.StatusCode, body)
	}
}

func postRawJSONStatus(t *testing.T, client *http.Client, endpoint string, raw string, expectedStatus int) {
	t.Helper()

	response := doRequest(t, client, http.MethodPost, endpoint, strings.NewReader(raw), nil)
	defer response.Body.Close()

	if response.StatusCode != expectedStatus {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("post %s: expected status %d, got %d: %s", endpoint, expectedStatus, response.StatusCode, body)
	}
}

func postRawJSONAuth(t *testing.T, client *http.Client, token string, endpoint string, raw string) *http.Response {
	t.Helper()

	return doRequest(t, client, http.MethodPost, endpoint, strings.NewReader(raw), authHeaders(token))
}

func doRequest(t *testing.T, client *http.Client, method string, endpoint string, body io.Reader, headers map[string]string) *http.Response {
	t.Helper()

	request, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		t.Fatalf("build %s request for %s: %v", method, endpoint, err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, endpoint, err)
	}

	return response
}

func authHeaders(token string) map[string]string {
	if strings.TrimSpace(token) == "" {
		return nil
	}

	return map[string]string{
		authorizationHeader: bearerScheme + " " + token,
	}
}
