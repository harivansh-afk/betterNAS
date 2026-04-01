package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	controlPlaneBinaryOnce sync.Once
	controlPlaneBinaryPath string
	controlPlaneBinaryErr  error

	nodeAgentBinaryOnce sync.Once
	nodeAgentBinaryPath string
	nodeAgentBinaryErr  error
)

func TestControlPlaneBinaryMountLoopIntegration(t *testing.T) {
	exportDir := t.TempDir()
	writeExportFile(t, exportDir, "README.txt", "betterNAS export\n")

	nextcloud := httptest.NewServer(http.NotFoundHandler())
	defer nextcloud.Close()

	nodeAgent := startNodeAgentBinary(t, exportDir)
	controlPlane := startControlPlaneBinary(t, "runtime-test-version", nextcloud.URL)
	client := &http.Client{Timeout: 2 * time.Second}

	directAddress := nodeAgent.baseURL
	registration := registerNode(t, client, controlPlane.baseURL+"/api/v1/nodes/register", testNodeBootstrapToken, nodeRegistrationRequest{
		MachineID:     "machine-runtime-1",
		DisplayName:   "Runtime NAS",
		AgentVersion:  "1.2.3",
		DirectAddress: &directAddress,
		RelayAddress:  nil,
		Exports: []storageExportInput{{
			Label:         "Photos",
			Path:          exportDir,
			MountPath:     defaultWebDAVPath,
			Protocols:     []string{"webdav"},
			CapacityBytes: nil,
			Tags:          []string{"runtime"},
		}},
	})
	if registration.Node.ID != "dev-node" {
		t.Fatalf("expected node ID %q, got %q", "dev-node", registration.Node.ID)
	}
	if registration.NodeToken == "" {
		t.Fatal("expected runtime registration to return a node token")
	}

	exports := getJSONAuth[[]storageExport](t, client, testClientToken, controlPlane.baseURL+"/api/v1/exports")
	if len(exports) != 1 {
		t.Fatalf("expected 1 export, got %d", len(exports))
	}
	if exports[0].ID != "dev-export" {
		t.Fatalf("expected export ID %q, got %q", "dev-export", exports[0].ID)
	}
	if exports[0].Path != exportDir {
		t.Fatalf("expected exported path %q, got %q", exportDir, exports[0].Path)
	}
	if exports[0].MountPath != defaultWebDAVPath {
		t.Fatalf("expected mountPath %q, got %q", defaultWebDAVPath, exports[0].MountPath)
	}

	mount := postJSONAuth[mountProfile](t, client, testClientToken, controlPlane.baseURL+"/api/v1/mount-profiles/issue", mountProfileRequest{
		UserID:   "runtime-user",
		DeviceID: "runtime-device",
		ExportID: exports[0].ID,
	})
	if mount.MountURL != nodeAgent.baseURL+defaultWebDAVPath {
		t.Fatalf("expected runtime mount URL %q, got %q", nodeAgent.baseURL+defaultWebDAVPath, mount.MountURL)
	}

	assertHTTPStatus(t, client, "PROPFIND", mount.MountURL, http.StatusMultiStatus)
	assertMountedFileContents(t, client, mount.MountURL+"README.txt", "betterNAS export\n")

	cloud := postJSONAuth[cloudProfile](t, client, testClientToken, controlPlane.baseURL+"/api/v1/cloud-profiles/issue", cloudProfileRequest{
		UserID:   "runtime-user",
		ExportID: exports[0].ID,
		Provider: "nextcloud",
	})
	if cloud.BaseURL != nextcloud.URL {
		t.Fatalf("expected runtime cloud baseUrl %q, got %q", nextcloud.URL, cloud.BaseURL)
	}
	expectedCloudPath := cloudProfilePathForExport(exports[0].ID)
	if cloud.Path != expectedCloudPath {
		t.Fatalf("expected runtime cloud path %q, got %q", expectedCloudPath, cloud.Path)
	}

	postJSONAuthStatus(t, client, registration.NodeToken, controlPlane.baseURL+"/api/v1/nodes/"+registration.Node.ID+"/heartbeat", nodeHeartbeatRequest{
		NodeID:     registration.Node.ID,
		Status:     "online",
		LastSeenAt: "2025-01-02T03:04:05Z",
	}, http.StatusNoContent)
}

func TestControlPlaneBinaryReRegistrationReconcilesExports(t *testing.T) {
	nextcloud := httptest.NewServer(http.NotFoundHandler())
	defer nextcloud.Close()

	controlPlane := startControlPlaneBinary(t, "runtime-test-version", nextcloud.URL)
	client := &http.Client{Timeout: 2 * time.Second}

	directAddress := "http://nas.local:8090"
	firstRegistration := registerNode(t, client, controlPlane.baseURL+"/api/v1/nodes/register", testNodeBootstrapToken, nodeRegistrationRequest{
		MachineID:     "machine-runtime-2",
		DisplayName:   "Runtime NAS",
		AgentVersion:  "1.2.3",
		DirectAddress: &directAddress,
		RelayAddress:  nil,
		Exports: []storageExportInput{
			{
				Label:         "Docs",
				Path:          "/srv/docs",
				MountPath:     "/dav/exports/docs/",
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{"runtime"},
			},
			{
				Label:         "Media",
				Path:          "/srv/media",
				MountPath:     "/dav/exports/media/",
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{"runtime"},
			},
		},
	})

	initialExports := exportsByPath(getJSONAuth[[]storageExport](t, client, testClientToken, controlPlane.baseURL+"/api/v1/exports"))
	docsExport := initialExports["/srv/docs"]
	if _, ok := initialExports["/srv/media"]; !ok {
		t.Fatal("expected media export to be registered")
	}

	secondRegistration := registerNode(t, client, controlPlane.baseURL+"/api/v1/nodes/register", firstRegistration.NodeToken, nodeRegistrationRequest{
		MachineID:     "machine-runtime-2",
		DisplayName:   "Runtime NAS Updated",
		AgentVersion:  "1.2.4",
		DirectAddress: &directAddress,
		RelayAddress:  nil,
		Exports: []storageExportInput{
			{
				Label:         "Docs v2",
				Path:          "/srv/docs",
				MountPath:     "/dav/exports/docs-v2/",
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{"runtime", "updated"},
			},
			{
				Label:         "Backups",
				Path:          "/srv/backups",
				MountPath:     "/dav/exports/backups/",
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{"runtime"},
			},
		},
	})
	if secondRegistration.Node.ID != firstRegistration.Node.ID {
		t.Fatalf("expected node ID %q after re-registration, got %q", firstRegistration.Node.ID, secondRegistration.Node.ID)
	}

	updatedExports := exportsByPath(getJSONAuth[[]storageExport](t, client, testClientToken, controlPlane.baseURL+"/api/v1/exports"))
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
		t.Fatal("expected stale media export to be removed")
	}
	if _, ok := updatedExports["/srv/backups"]; !ok {
		t.Fatal("expected backups export to be present")
	}
}

func TestControlPlaneBinaryMultiExportProfilesStayDistinct(t *testing.T) {
	firstExportDir := t.TempDir()
	secondExportDir := t.TempDir()
	writeExportFile(t, firstExportDir, "README.txt", "first runtime export\n")
	writeExportFile(t, secondExportDir, "README.txt", "second runtime export\n")

	nextcloud := httptest.NewServer(http.NotFoundHandler())
	defer nextcloud.Close()

	nodeAgent := startNodeAgentBinaryWithExports(t, []string{firstExportDir, secondExportDir})
	controlPlane := startControlPlaneBinary(t, "runtime-test-version", nextcloud.URL)
	client := &http.Client{Timeout: 2 * time.Second}

	firstMountPath := nodeAgentMountPathForExport(firstExportDir, 2)
	secondMountPath := nodeAgentMountPathForExport(secondExportDir, 2)
	directAddress := nodeAgent.baseURL
	registerNode(t, client, controlPlane.baseURL+"/api/v1/nodes/register", testNodeBootstrapToken, nodeRegistrationRequest{
		MachineID:     "machine-runtime-multi",
		DisplayName:   "Runtime Multi NAS",
		AgentVersion:  "1.2.3",
		DirectAddress: &directAddress,
		RelayAddress:  nil,
		Exports: []storageExportInput{
			{
				Label:         "Docs",
				Path:          firstExportDir,
				MountPath:     firstMountPath,
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{"runtime"},
			},
			{
				Label:         "Media",
				Path:          secondExportDir,
				MountPath:     secondMountPath,
				Protocols:     []string{"webdav"},
				CapacityBytes: nil,
				Tags:          []string{"runtime"},
			},
		},
	})

	exports := exportsByPath(getJSONAuth[[]storageExport](t, client, testClientToken, controlPlane.baseURL+"/api/v1/exports"))
	firstExport := exports[firstExportDir]
	secondExport := exports[secondExportDir]

	firstMount := postJSONAuth[mountProfile](t, client, testClientToken, controlPlane.baseURL+"/api/v1/mount-profiles/issue", mountProfileRequest{
		UserID:   "runtime-user",
		DeviceID: "runtime-device",
		ExportID: firstExport.ID,
	})
	secondMount := postJSONAuth[mountProfile](t, client, testClientToken, controlPlane.baseURL+"/api/v1/mount-profiles/issue", mountProfileRequest{
		UserID:   "runtime-user",
		DeviceID: "runtime-device",
		ExportID: secondExport.ID,
	})
	if firstMount.MountURL == secondMount.MountURL {
		t.Fatalf("expected distinct runtime mount URLs, got %q", firstMount.MountURL)
	}
	if firstMount.MountURL != nodeAgent.baseURL+firstMountPath {
		t.Fatalf("expected first runtime mount URL %q, got %q", nodeAgent.baseURL+firstMountPath, firstMount.MountURL)
	}
	if secondMount.MountURL != nodeAgent.baseURL+secondMountPath {
		t.Fatalf("expected second runtime mount URL %q, got %q", nodeAgent.baseURL+secondMountPath, secondMount.MountURL)
	}

	assertHTTPStatus(t, client, "PROPFIND", firstMount.MountURL, http.StatusMultiStatus)
	assertHTTPStatus(t, client, "PROPFIND", secondMount.MountURL, http.StatusMultiStatus)
	assertMountedFileContents(t, client, firstMount.MountURL+"README.txt", "first runtime export\n")
	assertMountedFileContents(t, client, secondMount.MountURL+"README.txt", "second runtime export\n")

	firstCloud := postJSONAuth[cloudProfile](t, client, testClientToken, controlPlane.baseURL+"/api/v1/cloud-profiles/issue", cloudProfileRequest{
		UserID:   "runtime-user",
		ExportID: firstExport.ID,
		Provider: "nextcloud",
	})
	secondCloud := postJSONAuth[cloudProfile](t, client, testClientToken, controlPlane.baseURL+"/api/v1/cloud-profiles/issue", cloudProfileRequest{
		UserID:   "runtime-user",
		ExportID: secondExport.ID,
		Provider: "nextcloud",
	})
	if firstCloud.Path == secondCloud.Path {
		t.Fatalf("expected distinct runtime cloud paths, got %q", firstCloud.Path)
	}
	if firstCloud.Path != cloudProfilePathForExport(firstExport.ID) {
		t.Fatalf("expected first runtime cloud path %q, got %q", cloudProfilePathForExport(firstExport.ID), firstCloud.Path)
	}
	if secondCloud.Path != cloudProfilePathForExport(secondExport.ID) {
		t.Fatalf("expected second runtime cloud path %q, got %q", cloudProfilePathForExport(secondExport.ID), secondCloud.Path)
	}
}

type runningBinary struct {
	baseURL string
	logPath string
}

func startControlPlaneBinary(t *testing.T, version string, nextcloudBaseURL string) runningBinary {
	t.Helper()

	port := reserveTCPPort(t)
	logPath := filepath.Join(t.TempDir(), "control-plane.log")
	statePath := filepath.Join(t.TempDir(), "control-plane-state.json")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("create control-plane log file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, buildControlPlaneBinary(t))
	cmd.Env = append(
		os.Environ(),
		"PORT="+port,
		"BETTERNAS_VERSION="+version,
		"NEXTCLOUD_BASE_URL="+nextcloudBaseURL,
		"BETTERNAS_CONTROL_PLANE_STATE_PATH="+statePath,
		"BETTERNAS_CONTROL_PLANE_CLIENT_TOKEN="+testClientToken,
		"BETTERNAS_CONTROL_PLANE_NODE_BOOTSTRAP_TOKEN="+testNodeBootstrapToken,
	)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		t.Fatalf("start control-plane binary: %v", err)
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	baseURL := fmt.Sprintf("http://127.0.0.1:%s", port)
	waitForHTTPStatus(t, baseURL+"/health", waitDone, logPath, http.StatusOK)
	registerProcessCleanup(t, ctx, cancel, cmd, waitDone, logFile, logPath, "control-plane")

	return runningBinary{
		baseURL: baseURL,
		logPath: logPath,
	}
}

func startNodeAgentBinary(t *testing.T, exportPath string) runningBinary {
	return startNodeAgentBinaryWithExports(t, []string{exportPath})
}

func startNodeAgentBinaryWithExports(t *testing.T, exportPaths []string) runningBinary {
	t.Helper()

	port := reserveTCPPort(t)
	logPath := filepath.Join(t.TempDir(), "node-agent.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("create node-agent log file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, buildNodeAgentBinary(t))
	rawExportPaths, err := json.Marshal(exportPaths)
	if err != nil {
		_ = logFile.Close()
		t.Fatalf("marshal export paths: %v", err)
	}
	cmd.Env = append(
		os.Environ(),
		"PORT="+port,
		"BETTERNAS_EXPORT_PATHS_JSON="+string(rawExportPaths),
	)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		t.Fatalf("start node-agent binary: %v", err)
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	baseURL := fmt.Sprintf("http://127.0.0.1:%s", port)
	waitForHTTPStatus(t, baseURL+"/health", waitDone, logPath, http.StatusOK)
	registerProcessCleanup(t, ctx, cancel, cmd, waitDone, logFile, logPath, "node-agent")

	return runningBinary{
		baseURL: baseURL,
		logPath: logPath,
	}
}

func buildControlPlaneBinary(t *testing.T) string {
	t.Helper()

	controlPlaneBinaryOnce.Do(func() {
		_, filename, _, ok := runtime.Caller(0)
		if !ok {
			controlPlaneBinaryErr = errors.New("locate control-plane package directory")
			return
		}

		tempDir, err := os.MkdirTemp("", "betternas-control-plane-*")
		if err != nil {
			controlPlaneBinaryErr = fmt.Errorf("create build temp dir: %w", err)
			return
		}

		controlPlaneBinaryPath = filepath.Join(tempDir, "control-plane")
		cmd := exec.Command("go", "build", "-o", controlPlaneBinaryPath, ".")
		cmd.Dir = filepath.Dir(filename)
		output, err := cmd.CombinedOutput()
		if err != nil {
			controlPlaneBinaryErr = fmt.Errorf("build control-plane binary: %w\n%s", err, output)
		}
	})

	if controlPlaneBinaryErr != nil {
		t.Fatal(controlPlaneBinaryErr)
	}

	return controlPlaneBinaryPath
}

func buildNodeAgentBinary(t *testing.T) string {
	t.Helper()

	nodeAgentBinaryOnce.Do(func() {
		_, filename, _, ok := runtime.Caller(0)
		if !ok {
			nodeAgentBinaryErr = errors.New("locate control-plane package directory")
			return
		}

		tempDir, err := os.MkdirTemp("", "betternas-node-agent-*")
		if err != nil {
			nodeAgentBinaryErr = fmt.Errorf("create build temp dir: %w", err)
			return
		}

		nodeAgentBinaryPath = filepath.Join(tempDir, "node-agent")
		cmd := exec.Command("go", "build", "-o", nodeAgentBinaryPath, "./cmd/node-agent")
		cmd.Dir = filepath.Clean(filepath.Join(filepath.Dir(filename), "../../../node-agent"))
		output, err := cmd.CombinedOutput()
		if err != nil {
			nodeAgentBinaryErr = fmt.Errorf("build node-agent binary: %w\n%s", err, output)
		}
	})

	if nodeAgentBinaryErr != nil {
		t.Fatal(nodeAgentBinaryErr)
	}

	return nodeAgentBinaryPath
}

func reserveTCPPort(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve tcp port: %v", err)
	}
	defer listener.Close()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}

	return port
}

func waitForHTTPStatus(t *testing.T, endpoint string, waitDone <-chan error, logPath string, expectedStatus int) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	client := &http.Client{Timeout: 500 * time.Millisecond}

	for time.Now().Before(deadline) {
		select {
		case err := <-waitDone:
			logOutput, _ := os.ReadFile(logPath)
			t.Fatalf("process exited before %s returned %d: %v\n%s", endpoint, expectedStatus, err, logOutput)
		default:
		}

		response, err := client.Get(endpoint)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == expectedStatus {
				return
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	logOutput, _ := os.ReadFile(logPath)
	t.Fatalf("endpoint %s did not return %d in time\n%s", endpoint, expectedStatus, logOutput)
}

func registerProcessCleanup(t *testing.T, ctx context.Context, cancel context.CancelFunc, cmd *exec.Cmd, waitDone <-chan error, logFile *os.File, logPath string, processName string) {
	t.Helper()

	t.Cleanup(func() {
		cancel()
		defer func() {
			_ = logFile.Close()
			if t.Failed() {
				if logOutput, err := os.ReadFile(logPath); err == nil {
					t.Logf("%s logs:\n%s", processName, logOutput)
				}
			}
		}()

		select {
		case err := <-waitDone:
			if err != nil && ctx.Err() == nil {
				t.Fatalf("%s exited unexpectedly: %v", processName, err)
			}
		case <-time.After(5 * time.Second):
			if killErr := cmd.Process.Kill(); killErr != nil {
				t.Fatalf("kill %s: %v", processName, killErr)
			}
			if err := <-waitDone; err != nil && ctx.Err() == nil {
				t.Fatalf("%s exited unexpectedly after kill: %v", processName, err)
			}
		}
	})
}

func assertMountedFileContents(t *testing.T, client *http.Client, endpoint string, expected string) {
	t.Helper()

	response, err := client.Get(endpoint)
	if err != nil {
		t.Fatalf("get %s: %v", endpoint, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("get %s: expected status 200, got %d", endpoint, response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read %s response: %v", endpoint, err)
	}
	if string(body) != expected {
		t.Fatalf("expected %s body %q, got %q", endpoint, expected, string(body))
	}
}

func assertHTTPStatus(t *testing.T, client *http.Client, method string, endpoint string, expectedStatus int) {
	t.Helper()

	request, err := http.NewRequest(method, endpoint, nil)
	if err != nil {
		t.Fatalf("build %s request for %s: %v", method, endpoint, err)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, endpoint, err)
	}
	defer response.Body.Close()

	if response.StatusCode != expectedStatus {
		t.Fatalf("%s %s: expected status %d, got %d", method, endpoint, expectedStatus, response.StatusCode)
	}
}

func writeExportFile(t *testing.T, directory string, name string, contents string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o644); err != nil {
		t.Fatalf("write export file %s: %v", name, err)
	}
}

func nodeAgentMountPathForExport(exportPath string, exportCount int) string {
	if exportCount <= 1 {
		return defaultWebDAVPath
	}

	sum := sha256.Sum256([]byte(strings.TrimSpace(exportPath)))
	return "/dav/exports/" + hex.EncodeToString(sum[:]) + "/"
}
