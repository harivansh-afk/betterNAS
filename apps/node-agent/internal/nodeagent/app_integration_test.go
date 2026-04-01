package nodeagent

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testControlPlaneToken = "test-control-plane-token"

func TestAppServesWebDAVFromConfiguredExportPath(t *testing.T) {
	t.Parallel()

	exportPath := filepath.Join(t.TempDir(), "export")

	baseURL, stop := startTestApp(t, Config{
		Port:              "0",
		ExportPath:        exportPath,
		MachineID:         "nas-1",
		DisplayName:       "NAS 1",
		AgentVersion:      "test-version",
		ExportLabel:       "integration",
		ExportTags:        []string{"finder"},
		HeartbeatInterval: time.Second,
	})
	defer stop()

	healthResponse, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("get health: %v", err)
	}
	defer healthResponse.Body.Close()

	healthBody, err := io.ReadAll(healthResponse.Body)
	if err != nil {
		t.Fatalf("read health body: %v", err)
	}

	if healthResponse.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d, want 200", healthResponse.StatusCode)
	}

	if string(healthBody) != "ok\n" {
		t.Fatalf("health body = %q, want ok", string(healthBody))
	}

	headRequest, err := http.NewRequest(http.MethodHead, baseURL+"/health", nil)
	if err != nil {
		t.Fatalf("build health head request: %v", err)
	}

	headResponse, err := http.DefaultClient.Do(headRequest)
	if err != nil {
		t.Fatalf("head health: %v", err)
	}
	defer headResponse.Body.Close()

	headBody, err := io.ReadAll(headResponse.Body)
	if err != nil {
		t.Fatalf("read head response body: %v", err)
	}

	if len(headBody) != 0 {
		t.Fatalf("head body length = %d, want 0", len(headBody))
	}

	redirectClient := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	redirectResponse, err := redirectClient.Get(baseURL + "/dav?session=abc")
	if err != nil {
		t.Fatalf("get /dav: %v", err)
	}
	defer redirectResponse.Body.Close()

	if redirectResponse.StatusCode != http.StatusPermanentRedirect {
		t.Fatalf("redirect status = %d, want 308", redirectResponse.StatusCode)
	}

	if redirectResponse.Header.Get("Location") != davPrefix+"?session=abc" {
		t.Fatalf("redirect location = %q, want %q", redirectResponse.Header.Get("Location"), davPrefix+"?session=abc")
	}

	optionsRequest := mustRequest(t, http.MethodOptions, baseURL+davPrefix, nil)
	optionsResponse, err := http.DefaultClient.Do(optionsRequest)
	if err != nil {
		t.Fatalf("options /dav/: %v", err)
	}
	defer optionsResponse.Body.Close()

	if optionsResponse.StatusCode != http.StatusOK {
		t.Fatalf("options status = %d, want 200", optionsResponse.StatusCode)
	}

	if !strings.Contains(optionsResponse.Header.Get("Dav"), "1") {
		t.Fatalf("dav header = %q, want DAV support", optionsResponse.Header.Get("Dav"))
	}

	putRequest := mustRequest(t, http.MethodPut, baseURL+"/dav/notes.txt", strings.NewReader("hello from webdav"))
	putResponse, err := http.DefaultClient.Do(putRequest)
	if err != nil {
		t.Fatalf("put file: %v", err)
	}
	defer putResponse.Body.Close()

	if putResponse.StatusCode != http.StatusCreated {
		t.Fatalf("put status = %d, want 201", putResponse.StatusCode)
	}

	savedBytes, err := os.ReadFile(filepath.Join(exportPath, "notes.txt"))
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}

	if string(savedBytes) != "hello from webdav" {
		t.Fatalf("saved file = %q, want file content", string(savedBytes))
	}

	mkcolRequest, err := http.NewRequest("MKCOL", baseURL+"/dav/docs", nil)
	if err != nil {
		t.Fatalf("build mkcol request: %v", err)
	}

	mkcolResponse, err := http.DefaultClient.Do(mkcolRequest)
	if err != nil {
		t.Fatalf("mkcol docs: %v", err)
	}
	defer mkcolResponse.Body.Close()

	if mkcolResponse.StatusCode != http.StatusCreated {
		t.Fatalf("mkcol status = %d, want 201", mkcolResponse.StatusCode)
	}

	propfindRequest, err := http.NewRequest("PROPFIND", baseURL+"/dav/docs", nil)
	if err != nil {
		t.Fatalf("build propfind request: %v", err)
	}
	propfindRequest.Header.Set("Depth", "0")

	propfindResponse, err := http.DefaultClient.Do(propfindRequest)
	if err != nil {
		t.Fatalf("propfind docs: %v", err)
	}
	defer propfindResponse.Body.Close()

	propfindBody, err := io.ReadAll(propfindResponse.Body)
	if err != nil {
		t.Fatalf("read propfind body: %v", err)
	}

	if propfindResponse.StatusCode != http.StatusMultiStatus {
		t.Fatalf("propfind status = %d, want 207", propfindResponse.StatusCode)
	}

	if !strings.Contains(string(propfindBody), "<D:href>/dav/docs/</D:href>") {
		t.Fatalf("propfind body = %q, want docs href", string(propfindBody))
	}

	getResponse, err := doWebDAVRequest(baseURL, http.MethodGet, "/dav/notes.txt", nil)
	if err != nil {
		t.Fatalf("get file: %v", err)
	}
	defer getResponse.Body.Close()

	getBody, err := io.ReadAll(getResponse.Body)
	if err != nil {
		t.Fatalf("read get body: %v", err)
	}

	if getResponse.StatusCode != http.StatusOK {
		t.Fatalf("get file status = %d, want 200", getResponse.StatusCode)
	}

	if string(getBody) != "hello from webdav" {
		t.Fatalf("get file body = %q, want file content", string(getBody))
	}

	deleteRequest := mustRequest(t, http.MethodDelete, baseURL+"/dav/notes.txt", nil)
	deleteResponse, err := http.DefaultClient.Do(deleteRequest)
	if err != nil {
		t.Fatalf("delete file: %v", err)
	}
	defer deleteResponse.Body.Close()

	if deleteResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", deleteResponse.StatusCode)
	}

	if _, err := os.Stat(filepath.Join(exportPath, "notes.txt")); !os.IsNotExist(err) {
		t.Fatalf("deleted file still exists or stat failed: %v", err)
	}
}

func TestAppServesSymlinksThatStayWithinExportRoot(t *testing.T) {
	t.Parallel()

	exportPath := filepath.Join(t.TempDir(), "export")
	if err := os.MkdirAll(exportPath, 0o755); err != nil {
		t.Fatalf("create export dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(exportPath, "plain.txt"), []byte("inside export"), 0o644); err != nil {
		t.Fatalf("write export file: %v", err)
	}

	if err := os.Symlink("plain.txt", filepath.Join(exportPath, "alias.txt")); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}

	baseURL, stop := startTestApp(t, Config{
		Port:       "0",
		ExportPath: exportPath,
	})
	defer stop()

	response, err := doWebDAVRequest(baseURL, http.MethodGet, "/dav/alias.txt", nil)
	if err != nil {
		t.Fatalf("get symlinked file: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read symlinked file body: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("symlink get status = %d, want 200", response.StatusCode)
	}

	if string(body) != "inside export" {
		t.Fatalf("symlink get body = %q, want inside export", string(body))
	}
}

func TestAppRejectsSymlinksThatEscapeExportRoot(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	exportPath := filepath.Join(tempDir, "export")
	outsidePath := filepath.Join(tempDir, "outside.txt")

	if err := os.MkdirAll(exportPath, 0o755); err != nil {
		t.Fatalf("create export dir: %v", err)
	}

	if err := os.WriteFile(outsidePath, []byte("outside"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	if err := os.Symlink("../outside.txt", filepath.Join(exportPath, "escape.txt")); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}

	baseURL, stop := startTestApp(t, Config{
		Port:       "0",
		ExportPath: exportPath,
	})
	defer stop()

	getResponse, err := doWebDAVRequest(baseURL, http.MethodGet, "/dav/escape.txt", nil)
	if err != nil {
		t.Fatalf("get escaped symlink: %v", err)
	}
	defer getResponse.Body.Close()

	if getResponse.StatusCode < http.StatusBadRequest {
		t.Fatalf("escaped symlink get status = %d, want 4xx or 5xx", getResponse.StatusCode)
	}

	putRequest := mustRequest(t, http.MethodPut, baseURL+"/dav/escape.txt", strings.NewReader("should-not-write"))
	putResponse, err := http.DefaultClient.Do(putRequest)
	if err != nil {
		t.Fatalf("put escaped symlink: %v", err)
	}
	defer putResponse.Body.Close()

	if putResponse.StatusCode < http.StatusBadRequest {
		t.Fatalf("escaped symlink put status = %d, want 4xx or 5xx", putResponse.StatusCode)
	}

	outsideBytes, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside file: %v", err)
	}

	if string(outsideBytes) != "outside" {
		t.Fatalf("outside file = %q, want unchanged content", string(outsideBytes))
	}
}

func TestAppRegistersAndHeartbeatsAgainstControlPlane(t *testing.T) {
	t.Parallel()

	registerRequests := make(chan nodeRegistrationRequest, 1)
	heartbeatRequests := make(chan nodeHeartbeatRequest, 4)

	controlPlane := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+testControlPlaneToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			t.Errorf("authorization header = %q, want Bearer token", got)
			return
		}

		switch r.URL.EscapedPath() {
		case registerNodeRoute:
			var request nodeRegistrationRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode register request: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			registerRequests <- request
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"node/123"}`)
		case heartbeatRoute("node/123"):
			var request nodeHeartbeatRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode heartbeat request: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			heartbeatRequests <- request
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer controlPlane.Close()

	exportPath := filepath.Join(t.TempDir(), "export")

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	cfg := Config{
		Port:              "0",
		ListenAddress:     listener.Addr().String(),
		ExportPath:        exportPath,
		MachineID:         "nas-42",
		DisplayName:       "Garage NAS",
		AgentVersion:      "test-version",
		DirectAddress:     "http://" + listener.Addr().String(),
		ExportLabel:       "archive",
		ExportTags:        []string{"photos", "finder"},
		ControlPlaneURL:   controlPlane.URL,
		ControlPlaneToken: testControlPlaneToken,
		RegisterEnabled:   true,
		HeartbeatEnabled:  true,
		HeartbeatInterval: 50 * time.Millisecond,
	}

	stop := serveWithListener(t, listener, cfg)
	defer stop()

	registerRequest := awaitValue(t, registerRequests, 2*time.Second, "register request")
	if registerRequest.MachineID != cfg.MachineID {
		t.Fatalf("machine id = %q, want %q", registerRequest.MachineID, cfg.MachineID)
	}

	if registerRequest.DisplayName != cfg.DisplayName {
		t.Fatalf("display name = %q, want %q", registerRequest.DisplayName, cfg.DisplayName)
	}

	if registerRequest.AgentVersion != cfg.AgentVersion {
		t.Fatalf("agent version = %q, want %q", registerRequest.AgentVersion, cfg.AgentVersion)
	}

	if registerRequest.DirectAddress == nil || *registerRequest.DirectAddress != cfg.DirectAddress {
		t.Fatalf("direct address = %#v, want %q", registerRequest.DirectAddress, cfg.DirectAddress)
	}

	if registerRequest.RelayAddress != nil {
		t.Fatalf("relay address = %#v, want nil", registerRequest.RelayAddress)
	}

	if len(registerRequest.Exports) != 1 {
		t.Fatalf("exports length = %d, want 1", len(registerRequest.Exports))
	}

	export := registerRequest.Exports[0]
	if export.Label != cfg.ExportLabel {
		t.Fatalf("export label = %q, want %q", export.Label, cfg.ExportLabel)
	}

	if export.Path != cfg.ExportPath {
		t.Fatalf("export path = %q, want %q", export.Path, cfg.ExportPath)
	}

	if len(export.Protocols) != 1 || export.Protocols[0] != "webdav" {
		t.Fatalf("export protocols = %#v, want [webdav]", export.Protocols)
	}

	if len(export.Tags) != 2 || export.Tags[0] != "photos" || export.Tags[1] != "finder" {
		t.Fatalf("export tags = %#v, want [photos finder]", export.Tags)
	}

	heartbeatRequest := awaitValue(t, heartbeatRequests, 2*time.Second, "heartbeat request")
	if heartbeatRequest.NodeID != "node/123" {
		t.Fatalf("heartbeat node id = %q, want node/123", heartbeatRequest.NodeID)
	}

	if heartbeatRequest.Status != "online" {
		t.Fatalf("heartbeat status = %q, want online", heartbeatRequest.Status)
	}

	if heartbeatRequest.LastSeenAt == "" {
		t.Fatal("heartbeat lastSeenAt is empty")
	}
}

func TestAppRegistersWithoutControlPlaneTokenWhenUnset(t *testing.T) {
	t.Parallel()

	registerRequests := make(chan nodeRegistrationRequest, 1)

	controlPlane := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			http.Error(w, "unexpected authorization header", http.StatusBadRequest)
			t.Errorf("authorization header = %q, want empty", got)
			return
		}

		if r.URL.EscapedPath() != registerNodeRoute {
			http.NotFound(w, r)
			return
		}

		var request nodeRegistrationRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode register request: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		registerRequests <- request
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"node/no-token"}`)
	}))
	defer controlPlane.Close()

	exportPath := filepath.Join(t.TempDir(), "export")

	_, stop := startTestApp(t, Config{
		Port:            "0",
		ExportPath:      exportPath,
		MachineID:       "nas-no-token",
		DisplayName:     "No Token NAS",
		AgentVersion:    "test-version",
		ExportLabel:     "register-only",
		ControlPlaneURL: controlPlane.URL,
		RegisterEnabled: true,
	})
	defer stop()

	registerRequest := awaitValue(t, registerRequests, 2*time.Second, "register request")
	if registerRequest.MachineID != "nas-no-token" {
		t.Fatalf("machine id = %q, want nas-no-token", registerRequest.MachineID)
	}
}

func TestHeartbeatRejectedNodeReregistersAndRecovers(t *testing.T) {
	t.Parallel()

	registerRequests := make(chan nodeRegistrationRequest, 4)
	heartbeatRequests := make(chan nodeHeartbeatRequest, 4)
	registerCount := 0

	controlPlane := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+testControlPlaneToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			t.Errorf("authorization header = %q, want Bearer token", got)
			return
		}

		switch r.URL.EscapedPath() {
		case registerNodeRoute:
			registerCount++

			var request nodeRegistrationRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode register request: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			registerRequests <- request
			w.Header().Set("Content-Type", "application/json")
			if registerCount == 1 {
				_, _ = io.WriteString(w, `{"id":"node/stale"}`)
				return
			}

			_, _ = io.WriteString(w, `{"id":"node/fresh"}`)
		case heartbeatRoute("node/stale"), heartbeatRoute("node/fresh"):
			var request nodeHeartbeatRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode heartbeat request: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			heartbeatRequests <- request
			if r.URL.EscapedPath() == heartbeatRoute("node/stale") {
				http.NotFound(w, r)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer controlPlane.Close()

	exportPath := filepath.Join(t.TempDir(), "export")

	baseURL, stop := startTestApp(t, Config{
		Port:              "0",
		ExportPath:        exportPath,
		MachineID:         "nas-stale",
		DisplayName:       "NAS stale",
		AgentVersion:      "test-version",
		ExportLabel:       "resilient",
		ControlPlaneURL:   controlPlane.URL,
		ControlPlaneToken: testControlPlaneToken,
		RegisterEnabled:   true,
		HeartbeatEnabled:  true,
		HeartbeatInterval: 50 * time.Millisecond,
	})
	defer stop()

	firstRegister := awaitValue(t, registerRequests, 2*time.Second, "first register request")
	if firstRegister.MachineID != "nas-stale" {
		t.Fatalf("first register machine id = %q, want nas-stale", firstRegister.MachineID)
	}

	secondRegister := awaitValue(t, registerRequests, 2*time.Second, "second register request")
	if secondRegister.MachineID != "nas-stale" {
		t.Fatalf("second register machine id = %q, want nas-stale", secondRegister.MachineID)
	}

	firstHeartbeat := awaitValue(t, heartbeatRequests, 2*time.Second, "stale heartbeat request")
	if firstHeartbeat.NodeID != "node/stale" {
		t.Fatalf("stale heartbeat node id = %q, want node/stale", firstHeartbeat.NodeID)
	}

	secondHeartbeat := awaitValue(t, heartbeatRequests, 2*time.Second, "fresh heartbeat request")
	if secondHeartbeat.NodeID != "node/fresh" {
		t.Fatalf("fresh heartbeat node id = %q, want node/fresh", secondHeartbeat.NodeID)
	}

	propfindRequest, err := http.NewRequest("PROPFIND", baseURL+"/dav/", nil)
	if err != nil {
		t.Fatalf("build WebDAV root propfind after heartbeat recovery: %v", err)
	}
	propfindRequest.Header.Set("Depth", "0")

	propfindResponse, err := http.DefaultClient.Do(propfindRequest)
	if err != nil {
		t.Fatalf("propfind WebDAV root after heartbeat recovery: %v", err)
	}
	defer propfindResponse.Body.Close()

	if propfindResponse.StatusCode != http.StatusMultiStatus {
		t.Fatalf("propfind WebDAV root status after heartbeat recovery = %d, want 207", propfindResponse.StatusCode)
	}
}

func TestHeartbeatRouteUnavailableStopsAfterFreshReregistrationWithoutStoppingWebDAV(t *testing.T) {
	t.Parallel()

	registerRequests := make(chan nodeRegistrationRequest, 4)
	heartbeatAttempts := make(chan nodeHeartbeatRequest, 4)

	controlPlane := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+testControlPlaneToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			t.Errorf("authorization header = %q, want Bearer token", got)
			return
		}

		switch r.URL.EscapedPath() {
		case registerNodeRoute:
			var request nodeRegistrationRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode register request: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			registerRequests <- request
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"node/404"}`)
		case heartbeatRoute("node/404"):
			var request nodeHeartbeatRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode heartbeat request: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			heartbeatAttempts <- request
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer controlPlane.Close()

	exportPath := filepath.Join(t.TempDir(), "export")

	baseURL, stop := startTestApp(t, Config{
		Port:              "0",
		ExportPath:        exportPath,
		MachineID:         "nas-404",
		DisplayName:       "NAS 404",
		AgentVersion:      "test-version",
		ExportLabel:       "resilient",
		ControlPlaneURL:   controlPlane.URL,
		ControlPlaneToken: testControlPlaneToken,
		RegisterEnabled:   true,
		HeartbeatEnabled:  true,
		HeartbeatInterval: 50 * time.Millisecond,
	})
	defer stop()

	firstRegister := awaitValue(t, registerRequests, 2*time.Second, "first register request")
	if firstRegister.MachineID != "nas-404" {
		t.Fatalf("first register machine id = %q, want nas-404", firstRegister.MachineID)
	}

	secondRegister := awaitValue(t, registerRequests, 2*time.Second, "second register request")
	if secondRegister.MachineID != "nas-404" {
		t.Fatalf("second register machine id = %q, want nas-404", secondRegister.MachineID)
	}

	firstHeartbeat := awaitValue(t, heartbeatAttempts, 2*time.Second, "first heartbeat attempt")
	if firstHeartbeat.NodeID != "node/404" {
		t.Fatalf("first heartbeat node id = %q, want node/404", firstHeartbeat.NodeID)
	}

	secondHeartbeat := awaitValue(t, heartbeatAttempts, 2*time.Second, "second heartbeat attempt")
	if secondHeartbeat.NodeID != "node/404" {
		t.Fatalf("second heartbeat node id = %q, want node/404", secondHeartbeat.NodeID)
	}

	time.Sleep(150 * time.Millisecond)
	if extraAttempts := len(heartbeatAttempts); extraAttempts != 0 {
		t.Fatalf("heartbeat attempts after unsupported route = %d, want 0", extraAttempts)
	}
	if extraRegistrations := len(registerRequests); extraRegistrations != 0 {
		t.Fatalf("register attempts after unsupported route = %d, want 0", extraRegistrations)
	}

	putRequest := mustRequest(t, http.MethodPut, baseURL+"/dav/after-heartbeat.txt", strings.NewReader("still-serving"))
	putResponse, err := http.DefaultClient.Do(putRequest)
	if err != nil {
		t.Fatalf("put after heartbeat failure: %v", err)
	}
	defer putResponse.Body.Close()

	if putResponse.StatusCode != http.StatusCreated {
		t.Fatalf("put status after heartbeat failure = %d, want 201", putResponse.StatusCode)
	}

	savedBytes, err := os.ReadFile(filepath.Join(exportPath, "after-heartbeat.txt"))
	if err != nil {
		t.Fatalf("read saved file after heartbeat failure: %v", err)
	}

	if string(savedBytes) != "still-serving" {
		t.Fatalf("saved bytes after heartbeat failure = %q, want still-serving", string(savedBytes))
	}
}

func startTestApp(t *testing.T, cfg Config) (string, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	if cfg.ListenAddress == "" {
		cfg.ListenAddress = listener.Addr().String()
	}
	if cfg.DirectAddress == "" {
		cfg.DirectAddress = "http://" + listener.Addr().String()
	}

	stop := serveWithListener(t, listener, cfg)
	return "http://" + listener.Addr().String(), stop
}

func serveWithListener(t *testing.T, listener net.Listener, cfg Config) func() {
	t.Helper()

	if cfg.ListenAddress == "" {
		cfg.ListenAddress = listener.Addr().String()
	}

	if err := os.MkdirAll(cfg.ExportPath, 0o755); err != nil {
		listener.Close()
		t.Fatalf("create export path: %v", err)
	}

	app, err := New(cfg, log.New(io.Discard, "", 0))
	if err != nil {
		listener.Close()
		t.Fatalf("new app: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	serverErrors := make(chan error, 1)

	go func() {
		serverErrors <- app.Serve(ctx, listener)
	}()

	waitForCondition(t, 2*time.Second, "app health", func() bool {
		response, err := http.Get("http://" + listener.Addr().String() + "/health")
		if err != nil {
			return false
		}
		defer response.Body.Close()

		return response.StatusCode == http.StatusOK
	})

	return func() {
		cancel()
		if err := <-serverErrors; err != nil {
			t.Fatalf("serve app: %v", err)
		}
	}
}

func doWebDAVRequest(baseURL, method, requestPath string, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequest(method, baseURL+requestPath, body)
	if err != nil {
		return nil, err
	}

	return http.DefaultClient.Do(request)
}

func mustRequest(t *testing.T, method, target string, body io.Reader) *http.Request {
	t.Helper()

	request, err := http.NewRequest(method, target, body)
	if err != nil {
		t.Fatalf("build request %s %s: %v", method, target, err)
	}

	return request
}

func awaitValue[T any](t *testing.T, values <-chan T, timeout time.Duration, label string) T {
	t.Helper()

	select {
	case value := <-values:
		return value
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for %s", label)
		var zero T
		return zero
	}
}

func waitForCondition(t *testing.T, timeout time.Duration, label string, check func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %s", label)
}
