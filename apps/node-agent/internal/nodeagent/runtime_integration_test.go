package nodeagent

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type startedProcess struct {
	cmd    *exec.Cmd
	output *lockedBuffer
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.b.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.b.String()
}

func TestRuntimeBinaryBindsToLoopbackByDefault(t *testing.T) {
	repoRoot := testRepoRoot(t)
	nodeAgentBin := buildNodeAgentBinary(t, repoRoot)
	nodeAgentPort := freePort(t)
	exportPath := filepath.Join(t.TempDir(), "export")
	if err := os.MkdirAll(exportPath, 0o755); err != nil {
		t.Fatalf("create export path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(exportPath, "seed.txt"), []byte("seed"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}

	nodeAgentURL := "http://127.0.0.1:" + strconv.Itoa(nodeAgentPort)
	nodeAgent := startBinaryProcess(t, repoRoot, nodeAgentBin, []string{
		"PORT=" + strconv.Itoa(nodeAgentPort),
		"BETTERNAS_EXPORT_PATH=" + exportPath,
	})
	defer nodeAgent.stop(t)

	waitForHTTPStatus(t, nodeAgentURL+"/health", http.StatusOK)

	propfindRequest, err := http.NewRequest("PROPFIND", nodeAgentURL+"/dav/", nil)
	if err != nil {
		t.Fatalf("build propfind request: %v", err)
	}
	propfindRequest.Header.Set("Depth", "0")

	propfindResponse, err := http.DefaultClient.Do(propfindRequest)
	if err != nil {
		t.Fatalf("propfind WebDAV root: %v", err)
	}
	defer propfindResponse.Body.Close()

	if propfindResponse.StatusCode != http.StatusMultiStatus {
		t.Fatalf("propfind status = %d, want 207", propfindResponse.StatusCode)
	}

	getResponse, err := http.Get(nodeAgentURL + "/dav/seed.txt")
	if err != nil {
		t.Fatalf("get WebDAV file: %v", err)
	}
	defer getResponse.Body.Close()

	getBody, err := io.ReadAll(getResponse.Body)
	if err != nil {
		t.Fatalf("read WebDAV body: %v", err)
	}

	if getResponse.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want 200", getResponse.StatusCode)
	}
	if string(getBody) != "seed" {
		t.Fatalf("get body = %q, want seed", string(getBody))
	}

	host, ok := firstNonLoopbackIPv4()
	if !ok {
		t.Skip("no non-loopback IPv4 address available to verify loopback-only binding")
	}

	client := &http.Client{Timeout: 500 * time.Millisecond}
	_, err = client.Get("http://" + host + ":" + strconv.Itoa(nodeAgentPort) + "/health")
	if err == nil {
		t.Fatalf("expected loopback-only listener to reject non-loopback host %s", host)
	}
}

func TestRuntimeBinaryServesWebDAVWithExplicitListenAddress(t *testing.T) {
	repoRoot := testRepoRoot(t)
	nodeAgentBin := buildNodeAgentBinary(t, repoRoot)
	nodeAgentPort := freePort(t)
	exportPath := filepath.Join(t.TempDir(), "export")
	if err := os.MkdirAll(exportPath, 0o755); err != nil {
		t.Fatalf("create export path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(exportPath, "seed.txt"), []byte("seed"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}

	nodeAgentURL := "http://127.0.0.1:" + strconv.Itoa(nodeAgentPort)
	nodeAgent := startBinaryProcess(t, repoRoot, nodeAgentBin, []string{
		"PORT=" + strconv.Itoa(nodeAgentPort),
		"BETTERNAS_EXPORT_PATH=" + exportPath,
		listenAddressEnvKey + "=:" + strconv.Itoa(nodeAgentPort),
		"BETTERNAS_NODE_DIRECT_ADDRESS=" + nodeAgentURL,
	})
	defer nodeAgent.stop(t)

	waitForHTTPStatus(t, nodeAgentURL+"/health", http.StatusOK)

	propfindRequest, err := http.NewRequest("PROPFIND", nodeAgentURL+"/dav/", nil)
	if err != nil {
		t.Fatalf("build propfind request: %v", err)
	}
	propfindRequest.Header.Set("Depth", "0")

	propfindResponse, err := http.DefaultClient.Do(propfindRequest)
	if err != nil {
		t.Fatalf("propfind WebDAV root: %v", err)
	}
	defer propfindResponse.Body.Close()

	if propfindResponse.StatusCode != http.StatusMultiStatus {
		t.Fatalf("propfind status = %d, want 207", propfindResponse.StatusCode)
	}

	getResponse, err := doRuntimeWebDAVRequest(nodeAgentURL, http.MethodGet, "/dav/seed.txt", nil)
	if err != nil {
		t.Fatalf("get WebDAV file: %v", err)
	}
	defer getResponse.Body.Close()

	getBody, err := io.ReadAll(getResponse.Body)
	if err != nil {
		t.Fatalf("read WebDAV body: %v", err)
	}

	if getResponse.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want 200", getResponse.StatusCode)
	}
	if string(getBody) != "seed" {
		t.Fatalf("get body = %q, want seed", string(getBody))
	}
}

func TestRuntimeBinaryOmitsDirectAddressForWildcardListenAddress(t *testing.T) {
	repoRoot := testRepoRoot(t)
	nodeAgentBin := buildNodeAgentBinary(t, repoRoot)
	nodeAgentPort := freePort(t)
	exportPath := filepath.Join(t.TempDir(), "export")
	if err := os.MkdirAll(exportPath, 0o755); err != nil {
		t.Fatalf("create export path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(exportPath, "seed.txt"), []byte("seed"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}

	registerRequests := make(chan nodeRegistrationRequest, 1)
	controlPlane := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		_, _ = io.WriteString(w, `{"id":"runtime-node"}`)
	}))
	defer controlPlane.Close()

	nodeAgentURL := "http://127.0.0.1:" + strconv.Itoa(nodeAgentPort)
	nodeAgent := startBinaryProcess(t, repoRoot, nodeAgentBin, []string{
		"PORT=" + strconv.Itoa(nodeAgentPort),
		"BETTERNAS_EXPORT_PATH=" + exportPath,
		"BETTERNAS_NODE_MACHINE_ID=runtime-machine",
		"BETTERNAS_CONTROL_PLANE_URL=" + controlPlane.URL,
		"BETTERNAS_NODE_REGISTER_ENABLED=true",
		listenAddressEnvKey + "=:" + strconv.Itoa(nodeAgentPort),
	})
	defer nodeAgent.stop(t)

	waitForHTTPStatus(t, nodeAgentURL+"/health", http.StatusOK)

	registerRequest := awaitValue(t, registerRequests, 5*time.Second, "register request")
	if registerRequest.DirectAddress != nil {
		t.Fatalf("direct address = %#v, want nil for wildcard listener", registerRequest.DirectAddress)
	}

	getResponse, err := doRuntimeWebDAVRequest(nodeAgentURL, http.MethodGet, "/dav/seed.txt", nil)
	if err != nil {
		t.Fatalf("get WebDAV file: %v", err)
	}
	defer getResponse.Body.Close()

	getBody, err := io.ReadAll(getResponse.Body)
	if err != nil {
		t.Fatalf("read WebDAV body: %v", err)
	}

	if getResponse.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want 200", getResponse.StatusCode)
	}
	if string(getBody) != "seed" {
		t.Fatalf("get body = %q, want seed", string(getBody))
	}
}

func TestRuntimeBinaryRejectsInvalidListenAddress(t *testing.T) {
	repoRoot := testRepoRoot(t)
	nodeAgentBin := buildNodeAgentBinary(t, repoRoot)
	nodeAgentPort := freePort(t)
	exportPath := filepath.Join(t.TempDir(), "export")
	if err := os.MkdirAll(exportPath, 0o755); err != nil {
		t.Fatalf("create export path: %v", err)
	}

	command := exec.Command(nodeAgentBin)
	command.Dir = repoRoot
	command.Env = mergedEnv([]string{
		"PORT=" + strconv.Itoa(nodeAgentPort),
		"BETTERNAS_EXPORT_PATH=" + exportPath,
		listenAddressEnvKey + "=localhost",
	})
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("expected node-agent to reject invalid listen address")
	}

	if !strings.Contains(string(output), listenAddressEnvKey) {
		t.Fatalf("output = %q, want %q guidance", string(output), listenAddressEnvKey)
	}
}

func TestRuntimeBinaryUsesOptionalControlPlaneSync(t *testing.T) {
	repoRoot := testRepoRoot(t)
	nodeAgentBin := buildNodeAgentBinary(t, repoRoot)
	nodeAgentPort := freePort(t)
	exportPath := filepath.Join(t.TempDir(), "export")
	if err := os.MkdirAll(exportPath, 0o755); err != nil {
		t.Fatalf("create export path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(exportPath, "seed.txt"), []byte("seed"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}

	const (
		machineID         = "runtime-machine"
		controlPlaneToken = "runtime-control-plane-token"
	)

	registerRequests := make(chan nodeRegistrationRequest, 1)
	heartbeatRequests := make(chan nodeHeartbeatRequest, 4)
	controlPlane := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+controlPlaneToken {
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
			_, _ = io.WriteString(w, `{"id":"runtime-node"}`)
		case heartbeatRoute("runtime-node"):
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

	nodeAgentURL := "http://127.0.0.1:" + strconv.Itoa(nodeAgentPort)
	nodeAgent := startBinaryProcess(t, repoRoot, nodeAgentBin, []string{
		"PORT=" + strconv.Itoa(nodeAgentPort),
		"BETTERNAS_EXPORT_PATH=" + exportPath,
		"BETTERNAS_VERSION=test-version",
		"BETTERNAS_NODE_MACHINE_ID=" + machineID,
		"BETTERNAS_NODE_DISPLAY_NAME=Runtime NAS",
		"BETTERNAS_EXPORT_LABEL=runtime-export",
		"BETTERNAS_EXPORT_TAGS=runtime,finder",
		"BETTERNAS_NODE_DIRECT_ADDRESS=" + nodeAgentURL,
		"BETTERNAS_CONTROL_PLANE_URL=" + controlPlane.URL,
		"BETTERNAS_CONTROL_PLANE_AUTH_TOKEN=" + controlPlaneToken,
		"BETTERNAS_NODE_REGISTER_ENABLED=true",
		"BETTERNAS_NODE_HEARTBEAT_ENABLED=true",
		"BETTERNAS_NODE_HEARTBEAT_INTERVAL=100ms",
	})
	defer nodeAgent.stop(t)

	waitForHTTPStatus(t, nodeAgentURL+"/health", http.StatusOK)

	getResponse, err := doRuntimeWebDAVRequest(nodeAgentURL, http.MethodGet, "/dav/seed.txt", nil)
	if err != nil {
		t.Fatalf("get WebDAV file: %v", err)
	}
	defer getResponse.Body.Close()

	getBody, err := io.ReadAll(getResponse.Body)
	if err != nil {
		t.Fatalf("read WebDAV body: %v", err)
	}
	if getResponse.StatusCode != http.StatusOK {
		t.Fatalf("get WebDAV status = %d, want 200", getResponse.StatusCode)
	}
	if string(getBody) != "seed" {
		t.Fatalf("get WebDAV body = %q, want seed", string(getBody))
	}

	registerRequest := awaitValue(t, registerRequests, 5*time.Second, "register request")
	if registerRequest.MachineID != machineID {
		t.Fatalf("machine id = %q, want %q", registerRequest.MachineID, machineID)
	}
	if registerRequest.DisplayName != "Runtime NAS" {
		t.Fatalf("display name = %q, want Runtime NAS", registerRequest.DisplayName)
	}
	if registerRequest.AgentVersion != "test-version" {
		t.Fatalf("agent version = %q, want test-version", registerRequest.AgentVersion)
	}
	if registerRequest.DirectAddress == nil || *registerRequest.DirectAddress != nodeAgentURL {
		t.Fatalf("direct address = %#v, want %q", registerRequest.DirectAddress, nodeAgentURL)
	}
	if registerRequest.RelayAddress != nil {
		t.Fatalf("relay address = %#v, want nil", registerRequest.RelayAddress)
	}
	if len(registerRequest.Exports) != 1 {
		t.Fatalf("exports length = %d, want 1", len(registerRequest.Exports))
	}
	if registerRequest.Exports[0].Label != "runtime-export" {
		t.Fatalf("export label = %q, want runtime-export", registerRequest.Exports[0].Label)
	}
	if registerRequest.Exports[0].Path != exportPath {
		t.Fatalf("export path = %q, want %q", registerRequest.Exports[0].Path, exportPath)
	}
	if len(registerRequest.Exports[0].Protocols) != 1 || registerRequest.Exports[0].Protocols[0] != "webdav" {
		t.Fatalf("export protocols = %#v, want [webdav]", registerRequest.Exports[0].Protocols)
	}
	if len(registerRequest.Exports[0].Tags) != 2 || registerRequest.Exports[0].Tags[0] != "runtime" || registerRequest.Exports[0].Tags[1] != "finder" {
		t.Fatalf("export tags = %#v, want [runtime finder]", registerRequest.Exports[0].Tags)
	}

	heartbeatRequest := awaitValue(t, heartbeatRequests, 5*time.Second, "heartbeat request")
	if heartbeatRequest.NodeID != "runtime-node" {
		t.Fatalf("heartbeat node id = %q, want runtime-node", heartbeatRequest.NodeID)
	}
	if heartbeatRequest.Status != "online" {
		t.Fatalf("heartbeat status = %q, want online", heartbeatRequest.Status)
	}
	if _, err := time.Parse(time.RFC3339, heartbeatRequest.LastSeenAt); err != nil {
		t.Fatalf("heartbeat lastSeenAt parse: %v", err)
	}
}

func TestRuntimeBinaryIntegratesWithRealControlPlane(t *testing.T) {
	repoRoot := testRepoRoot(t)
	nodeAgentBin := buildNodeAgentBinary(t, repoRoot)
	controlPlaneBin := buildControlPlaneBinary(t, repoRoot)
	nodeAgentPort := freePort(t)
	controlPlanePort := freePort(t)
	exportPath := filepath.Join(t.TempDir(), "export")
	if err := os.MkdirAll(exportPath, 0o755); err != nil {
		t.Fatalf("create export path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(exportPath, "seed.txt"), []byte("seed"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}

	controlPlaneURL := "http://127.0.0.1:" + strconv.Itoa(controlPlanePort)
	nodeAgentURL := "http://127.0.0.1:" + strconv.Itoa(nodeAgentPort)
	controlPlane := startBinaryProcess(t, repoRoot, controlPlaneBin, []string{
		"PORT=" + strconv.Itoa(controlPlanePort),
		"BETTERNAS_VERSION=test-version",
		"BETTERNAS_EXAMPLE_MOUNT_URL=" + nodeAgentURL + "/dav/",
		"BETTERNAS_NODE_DIRECT_ADDRESS=" + nodeAgentURL,
	})
	defer controlPlane.stop(t)

	waitForHTTPStatus(t, controlPlaneURL+"/health", http.StatusOK)

	nodeAgent := startBinaryProcess(t, repoRoot, nodeAgentBin, []string{
		"PORT=" + strconv.Itoa(nodeAgentPort),
		"BETTERNAS_EXPORT_PATH=" + exportPath,
		"BETTERNAS_NODE_MACHINE_ID=runtime-machine",
		"BETTERNAS_NODE_DISPLAY_NAME=Runtime NAS",
		"BETTERNAS_EXPORT_LABEL=runtime-export",
		"BETTERNAS_NODE_DIRECT_ADDRESS=" + nodeAgentURL,
		"BETTERNAS_CONTROL_PLANE_URL=" + controlPlaneURL,
		"BETTERNAS_NODE_REGISTER_ENABLED=true",
		"BETTERNAS_NODE_HEARTBEAT_ENABLED=true",
		"BETTERNAS_NODE_HEARTBEAT_INTERVAL=100ms",
	})
	defer nodeAgent.stop(t)

	waitForHTTPStatus(t, nodeAgentURL+"/health", http.StatusOK)
	waitForProcessOutput(t, nodeAgent, 5*time.Second, "registered as dev-node")
	waitForProcessOutput(t, nodeAgent, 5*time.Second, "stopping heartbeats")

	mountProfileRequest, err := http.NewRequest(http.MethodPost, controlPlaneURL+"/api/v1/mount-profiles/issue", strings.NewReader(`{"userId":"integration-user","deviceId":"integration-device","exportId":"dev-export"}`))
	if err != nil {
		t.Fatalf("build mount profile request: %v", err)
	}
	mountProfileRequest.Header.Set("Content-Type", "application/json")

	mountProfileResponse, err := http.DefaultClient.Do(mountProfileRequest)
	if err != nil {
		t.Fatalf("issue mount profile: %v", err)
	}
	defer mountProfileResponse.Body.Close()

	if mountProfileResponse.StatusCode != http.StatusOK {
		t.Fatalf("mount profile status = %d, want 200", mountProfileResponse.StatusCode)
	}

	var mountProfile struct {
		Protocol string `json:"protocol"`
		MountURL string `json:"mountUrl"`
	}
	if err := json.NewDecoder(mountProfileResponse.Body).Decode(&mountProfile); err != nil {
		t.Fatalf("decode mount profile: %v", err)
	}

	if mountProfile.Protocol != "webdav" {
		t.Fatalf("mount profile protocol = %q, want webdav", mountProfile.Protocol)
	}
	if mountProfile.MountURL != nodeAgentURL+"/dav/" {
		t.Fatalf("mount profile url = %q, want %q", mountProfile.MountURL, nodeAgentURL+"/dav/")
	}

	propfindRequest, err := http.NewRequest("PROPFIND", mountProfile.MountURL, nil)
	if err != nil {
		t.Fatalf("build mount-url propfind request: %v", err)
	}
	propfindRequest.Header.Set("Depth", "0")

	propfindResponse, err := http.DefaultClient.Do(propfindRequest)
	if err != nil {
		t.Fatalf("propfind mount profile url: %v", err)
	}
	defer propfindResponse.Body.Close()

	if propfindResponse.StatusCode != http.StatusMultiStatus {
		t.Fatalf("propfind status = %d, want 207", propfindResponse.StatusCode)
	}

	getResponse, err := doRuntimeWebDAVRequest(nodeAgentURL, http.MethodGet, "/dav/seed.txt", nil)
	if err != nil {
		t.Fatalf("get WebDAV file after control-plane sync: %v", err)
	}
	defer getResponse.Body.Close()

	getBody, err := io.ReadAll(getResponse.Body)
	if err != nil {
		t.Fatalf("read WebDAV body after control-plane sync: %v", err)
	}

	if getResponse.StatusCode != http.StatusOK {
		t.Fatalf("get status after control-plane sync = %d, want 200", getResponse.StatusCode)
	}
	if string(getBody) != "seed" {
		t.Fatalf("get body after control-plane sync = %q, want seed", string(getBody))
	}
}

func testRepoRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve runtime integration test filename")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", ".."))
}

func buildNodeAgentBinary(t *testing.T, repoRoot string) string {
	t.Helper()

	binDir := t.TempDir()
	nodeAgentBin := filepath.Join(binDir, binaryName("node-agent"))
	buildBinary(t, repoRoot, "./apps/node-agent/cmd/node-agent", nodeAgentBin)
	return nodeAgentBin
}

func buildControlPlaneBinary(t *testing.T, repoRoot string) string {
	t.Helper()

	binDir := t.TempDir()
	controlPlaneBin := filepath.Join(binDir, binaryName("control-plane"))
	buildBinary(t, repoRoot, "./apps/control-plane/cmd/control-plane", controlPlaneBin)
	return controlPlaneBin
}

func binaryName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}

	return base
}

func buildBinary(t *testing.T, repoRoot, packagePath, outputPath string) {
	t.Helper()

	command := exec.Command("go", "build", "-o", outputPath, packagePath)
	command.Dir = repoRoot
	command.Env = mergedEnv([]string{"CGO_ENABLED=0"})
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("build %s: %v\n%s", packagePath, err, string(output))
	}
}

func startBinaryProcess(t *testing.T, repoRoot, binaryPath string, env []string) *startedProcess {
	t.Helper()

	output := &lockedBuffer{}
	command := exec.Command(binaryPath)
	command.Dir = repoRoot
	command.Env = mergedEnv(env)
	command.Stdout = output
	command.Stderr = output

	if err := command.Start(); err != nil {
		t.Fatalf("start %s: %v", binaryPath, err)
	}

	return &startedProcess{
		cmd:    command,
		output: output,
	}
}

func (p *startedProcess) stop(t *testing.T) {
	t.Helper()

	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return
	}

	_ = p.cmd.Process.Kill()

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- p.cmd.Wait()
	}()

	select {
	case err := <-waitDone:
		if err != nil && !strings.Contains(err.Error(), "signal: killed") {
			t.Fatalf("wait for %s: %v\n%s", p.cmd.Path, err, p.output.String())
		}
	case <-time.After(5 * time.Second):
		_ = p.cmd.Process.Kill()
		err := <-waitDone
		if err != nil && !strings.Contains(err.Error(), "signal: killed") {
			t.Fatalf("kill %s: %v\n%s", p.cmd.Path, err, p.output.String())
		}
	}
}

func freePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for free port: %v", err)
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port
}

func waitForHTTPStatus(t *testing.T, target string, wantStatus int) {
	t.Helper()

	waitForCondition(t, 10*time.Second, target, func() bool {
		response, err := http.Get(target)
		if err != nil {
			return false
		}
		defer response.Body.Close()

		return response.StatusCode == wantStatus
	})
}

func waitForProcessOutput(t *testing.T, process *startedProcess, timeout time.Duration, fragment string) {
	t.Helper()

	waitForCondition(t, timeout, "process output "+fragment, func() bool {
		return strings.Contains(process.output.String(), fragment)
	})
}

func doRuntimeWebDAVRequest(baseURL, method, requestPath string, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequest(method, baseURL+requestPath, body)
	if err != nil {
		return nil, err
	}

	return http.DefaultClient.Do(request)
}

func firstNonLoopbackIPv4() (string, bool) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", false
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch value := addr.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			if ipv4 := ip.To4(); ipv4 != nil {
				return ipv4.String(), true
			}
		}
	}

	return "", false
}

func mergedEnv(overrides []string) []string {
	values := make(map[string]string)

	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		values[key] = value
	}

	for _, entry := range overrides {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		values[key] = value
	}

	merged := make([]string, 0, len(values))
	for key, value := range values {
		merged = append(merged, key+"="+value)
	}
	sort.Strings(merged)
	return merged
}
