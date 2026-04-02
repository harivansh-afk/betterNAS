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

const runtimeUsername = "runtime-user"

func TestControlPlaneBinaryMountLoopIntegration(t *testing.T) {
	exportDir := t.TempDir()
	writeExportFile(t, exportDir, "README.txt", "betterNAS export\n")

	nextcloud := httptest.NewServer(http.NotFoundHandler())
	defer nextcloud.Close()

	controlPlane := startControlPlaneBinary(t, "runtime-test-version", nextcloud.URL)
	nodeAgent := startNodeAgentBinaryWithExports(t, controlPlane.baseURL, []string{exportDir}, "machine-runtime-1")
	client := &http.Client{Timeout: 2 * time.Second}

	exports := waitForExportsByPath(t, client, controlPlane.sessionToken, controlPlane.baseURL+"/api/v1/exports", []string{exportDir})
	export := exports[exportDir]
	if export.ID != "dev-export" {
		t.Fatalf("expected export ID %q, got %q", "dev-export", export.ID)
	}
	if export.MountPath != defaultWebDAVPath {
		t.Fatalf("expected mountPath %q, got %q", defaultWebDAVPath, export.MountPath)
	}

	mount := postJSONAuth[mountProfile](t, client, controlPlane.sessionToken, controlPlane.baseURL+"/api/v1/mount-profiles/issue", mountProfileRequest{
		ExportID: export.ID,
	})
	if mount.MountURL != nodeAgent.baseURL+defaultWebDAVPath {
		t.Fatalf("expected runtime mount URL %q, got %q", nodeAgent.baseURL+defaultWebDAVPath, mount.MountURL)
	}
	if mount.Credential.Mode != mountCredentialModeBasicAuth {
		t.Fatalf("expected mount credential mode %q, got %q", mountCredentialModeBasicAuth, mount.Credential.Mode)
	}

	assertHTTPStatusWithBasicAuth(t, client, "PROPFIND", mount.MountURL, controlPlane.username, controlPlane.password, http.StatusMultiStatus)
	assertMountedFileContentsWithBasicAuth(t, client, mount.MountURL+"README.txt", controlPlane.username, controlPlane.password, "betterNAS export\n")

	cloud := postJSONAuth[cloudProfile](t, client, controlPlane.sessionToken, controlPlane.baseURL+"/api/v1/cloud-profiles/issue", cloudProfileRequest{
		UserID:   controlPlane.userID,
		ExportID: export.ID,
		Provider: "nextcloud",
	})
	if cloud.BaseURL != nextcloud.URL {
		t.Fatalf("expected runtime cloud baseUrl %q, got %q", nextcloud.URL, cloud.BaseURL)
	}
	if cloud.Path != cloudProfilePathForExport(export.ID) {
		t.Fatalf("expected runtime cloud path %q, got %q", cloudProfilePathForExport(export.ID), cloud.Path)
	}
}

func TestControlPlaneBinaryMultiExportProfilesStayDistinct(t *testing.T) {
	firstExportDir := t.TempDir()
	secondExportDir := t.TempDir()
	writeExportFile(t, firstExportDir, "README.txt", "first runtime export\n")
	writeExportFile(t, secondExportDir, "README.txt", "second runtime export\n")

	nextcloud := httptest.NewServer(http.NotFoundHandler())
	defer nextcloud.Close()

	controlPlane := startControlPlaneBinary(t, "runtime-test-version", nextcloud.URL)
	nodeAgent := startNodeAgentBinaryWithExports(t, controlPlane.baseURL, []string{firstExportDir, secondExportDir}, "machine-runtime-multi")
	client := &http.Client{Timeout: 2 * time.Second}

	firstMountPath := nodeAgentMountPathForExport(firstExportDir, 2)
	secondMountPath := nodeAgentMountPathForExport(secondExportDir, 2)
	exports := waitForExportsByPath(t, client, controlPlane.sessionToken, controlPlane.baseURL+"/api/v1/exports", []string{firstExportDir, secondExportDir})
	firstExport := exports[firstExportDir]
	secondExport := exports[secondExportDir]

	firstMount := postJSONAuth[mountProfile](t, client, controlPlane.sessionToken, controlPlane.baseURL+"/api/v1/mount-profiles/issue", mountProfileRequest{ExportID: firstExport.ID})
	secondMount := postJSONAuth[mountProfile](t, client, controlPlane.sessionToken, controlPlane.baseURL+"/api/v1/mount-profiles/issue", mountProfileRequest{ExportID: secondExport.ID})
	if firstMount.MountURL == secondMount.MountURL {
		t.Fatalf("expected distinct runtime mount URLs, got %q", firstMount.MountURL)
	}
	if firstMount.MountURL != nodeAgent.baseURL+firstMountPath {
		t.Fatalf("expected first runtime mount URL %q, got %q", nodeAgent.baseURL+firstMountPath, firstMount.MountURL)
	}
	if secondMount.MountURL != nodeAgent.baseURL+secondMountPath {
		t.Fatalf("expected second runtime mount URL %q, got %q", nodeAgent.baseURL+secondMountPath, secondMount.MountURL)
	}

	assertHTTPStatusWithBasicAuth(t, client, "PROPFIND", firstMount.MountURL, controlPlane.username, controlPlane.password, http.StatusMultiStatus)
	assertHTTPStatusWithBasicAuth(t, client, "PROPFIND", secondMount.MountURL, controlPlane.username, controlPlane.password, http.StatusMultiStatus)
	assertMountedFileContentsWithBasicAuth(t, client, firstMount.MountURL+"README.txt", controlPlane.username, controlPlane.password, "first runtime export\n")
	assertMountedFileContentsWithBasicAuth(t, client, secondMount.MountURL+"README.txt", controlPlane.username, controlPlane.password, "second runtime export\n")

	firstCloud := postJSONAuth[cloudProfile](t, client, controlPlane.sessionToken, controlPlane.baseURL+"/api/v1/cloud-profiles/issue", cloudProfileRequest{
		UserID:   controlPlane.userID,
		ExportID: firstExport.ID,
		Provider: "nextcloud",
	})
	secondCloud := postJSONAuth[cloudProfile](t, client, controlPlane.sessionToken, controlPlane.baseURL+"/api/v1/cloud-profiles/issue", cloudProfileRequest{
		UserID:   controlPlane.userID,
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
	baseURL      string
	logPath      string
	sessionToken string
	username     string
	password     string
	userID       string
}

func startControlPlaneBinary(t *testing.T, version string, nextcloudBaseURL string) runningBinary {
	t.Helper()

	port := reserveTCPPort(t)
	logPath := filepath.Join(t.TempDir(), "control-plane.log")
	dbPath := filepath.Join(t.TempDir(), "control-plane.db")
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
		"BETTERNAS_CONTROL_PLANE_DB_PATH="+dbPath,
		"BETTERNAS_REGISTRATION_ENABLED=true",
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
	session := registerRuntimeUser(t, &http.Client{Timeout: 2 * time.Second}, baseURL)
	registerProcessCleanup(t, ctx, cancel, cmd, waitDone, logFile, logPath, "control-plane")

	return runningBinary{
		baseURL:      baseURL,
		logPath:      logPath,
		sessionToken: session.Token,
		username:     runtimeUsername,
		password:     testPassword,
		userID:       session.User.ID,
	}
}

func startNodeAgentBinaryWithExports(t *testing.T, controlPlaneBaseURL string, exportPaths []string, machineID string) runningBinary {
	t.Helper()

	port := reserveTCPPort(t)
	baseURL := fmt.Sprintf("http://127.0.0.1:%s", port)
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
		"BETTERNAS_CONTROL_PLANE_URL="+controlPlaneBaseURL,
		"BETTERNAS_USERNAME="+runtimeUsername,
		"BETTERNAS_PASSWORD="+testPassword,
		"BETTERNAS_NODE_MACHINE_ID="+machineID,
		"BETTERNAS_NODE_DISPLAY_NAME="+machineID,
		"BETTERNAS_NODE_DIRECT_ADDRESS="+baseURL,
		"BETTERNAS_VERSION=runtime-test-version",
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

	waitForHTTPStatus(t, baseURL+"/health", waitDone, logPath, http.StatusOK)
	registerProcessCleanup(t, ctx, cancel, cmd, waitDone, logFile, logPath, "node-agent")

	return runningBinary{
		baseURL: baseURL,
		logPath: logPath,
	}
}

func waitForExportsByPath(t *testing.T, client *http.Client, token string, endpoint string, expectedPaths []string) map[string]storageExport {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		exports := getJSONAuth[[]storageExport](t, client, token, endpoint)
		exportsByPath := exportsByPath(exports)
		allPresent := true
		for _, expectedPath := range expectedPaths {
			if _, ok := exportsByPath[expectedPath]; !ok {
				allPresent = false
				break
			}
		}
		if allPresent {
			return exportsByPath
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("exports for %v did not appear in time", expectedPaths)
	return nil
}

func registerRuntimeUser(t *testing.T, client *http.Client, baseURL string) authLoginResponse {
	t.Helper()

	return postJSONAuthCreated[authLoginResponse](t, client, "", baseURL+"/api/v1/auth/register", authRegisterRequest{
		Username: runtimeUsername,
		Password: testPassword,
	})
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
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
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
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
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

func assertMountedFileContentsWithBasicAuth(t *testing.T, client *http.Client, endpoint string, username string, password string, expected string) {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatalf("build GET request for %s: %v", endpoint, err)
	}
	request.SetBasicAuth(username, password)

	response, err := client.Do(request)
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

func assertHTTPStatusWithBasicAuth(t *testing.T, client *http.Client, method string, endpoint string, username string, password string, expectedStatus int) {
	t.Helper()

	request, err := http.NewRequest(method, endpoint, nil)
	if err != nil {
		t.Fatalf("build %s request for %s: %v", method, endpoint, err)
	}
	request.SetBasicAuth(username, password)

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
