package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testDAVAuthSecret = "test-dav-auth-secret"

func TestSingleExportServesDefaultAndScopedMountPathsWithValidCredentials(t *testing.T) {
	t.Parallel()

	exportDir := t.TempDir()
	writeExportFile(t, exportDir, "README.txt", "single export\n")

	app, err := newApp(appConfig{
		exportPaths:   []string{exportDir},
		nodeID:        "node-1",
		davAuthSecret: testDAVAuthSecret,
	})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	server := httptest.NewServer(app.handler())
	defer server.Close()

	defaultUsername, defaultPassword := issueTestMountCredential(t, "node-1", defaultWebDAVPath, false)
	scopedMountPath := scopedMountPathForExport(exportDir)
	scopedUsername, scopedPassword := issueTestMountCredential(t, "node-1", scopedMountPath, false)

	assertHTTPStatusWithBasicAuth(t, server.Client(), "PROPFIND", server.URL+defaultWebDAVPath, defaultUsername, defaultPassword, http.StatusMultiStatus)
	assertHTTPStatusWithBasicAuth(t, server.Client(), "PROPFIND", server.URL+scopedMountPath, scopedUsername, scopedPassword, http.StatusMultiStatus)
	assertMountedFileContentsWithBasicAuth(t, server.Client(), server.URL+defaultWebDAVPath+"README.txt", defaultUsername, defaultPassword, "single export\n")
	assertMountedFileContentsWithBasicAuth(t, server.Client(), server.URL+scopedMountPath+"README.txt", scopedUsername, scopedPassword, "single export\n")
}

func TestMultipleExportsServeDistinctScopedMountPathsWithValidCredentials(t *testing.T) {
	t.Parallel()

	firstExportDir := t.TempDir()
	secondExportDir := t.TempDir()
	writeExportFile(t, firstExportDir, "README.txt", "first export\n")
	writeExportFile(t, secondExportDir, "README.txt", "second export\n")

	app, err := newApp(appConfig{
		exportPaths:   []string{firstExportDir, secondExportDir},
		nodeID:        "node-1",
		davAuthSecret: testDAVAuthSecret,
	})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	server := httptest.NewServer(app.handler())
	defer server.Close()

	firstMountPath := mountProfilePathForExport(firstExportDir, 2)
	secondMountPath := mountProfilePathForExport(secondExportDir, 2)
	if firstMountPath == secondMountPath {
		t.Fatal("expected distinct mount paths for multiple exports")
	}

	firstUsername, firstPassword := issueTestMountCredential(t, "node-1", firstMountPath, false)
	secondUsername, secondPassword := issueTestMountCredential(t, "node-1", secondMountPath, false)

	assertHTTPStatusWithBasicAuth(t, server.Client(), "PROPFIND", server.URL+firstMountPath, firstUsername, firstPassword, http.StatusMultiStatus)
	assertHTTPStatusWithBasicAuth(t, server.Client(), "PROPFIND", server.URL+secondMountPath, secondUsername, secondPassword, http.StatusMultiStatus)
	assertMountedFileContentsWithBasicAuth(t, server.Client(), server.URL+firstMountPath+"README.txt", firstUsername, firstPassword, "first export\n")
	assertMountedFileContentsWithBasicAuth(t, server.Client(), server.URL+secondMountPath+"README.txt", secondUsername, secondPassword, "second export\n")

	response, err := server.Client().Get(server.URL + defaultWebDAVPath)
	if err != nil {
		t.Fatalf("get default multi-export mount path: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected %s to return 404 for multi-export config, got %d", defaultWebDAVPath, response.StatusCode)
	}
}

func TestDAVAuthRejectsMissingInvalidAndReadonlyCredentials(t *testing.T) {
	t.Parallel()

	exportDir := t.TempDir()
	writeExportFile(t, exportDir, "README.txt", "readonly export\n")

	app, err := newApp(appConfig{
		exportPaths:   []string{exportDir},
		nodeID:        "node-1",
		davAuthSecret: testDAVAuthSecret,
	})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	server := httptest.NewServer(app.handler())
	defer server.Close()

	assertHTTPStatusWithBasicAuth(t, server.Client(), "PROPFIND", server.URL+defaultWebDAVPath, "", "", http.StatusUnauthorized)

	wrongMountUsername, wrongMountPassword := issueTestMountCredential(t, "node-1", "/dav/wrong/", false)
	assertHTTPStatusWithBasicAuth(t, server.Client(), "PROPFIND", server.URL+defaultWebDAVPath, wrongMountUsername, wrongMountPassword, http.StatusUnauthorized)

	expiredUsername, expiredPassword := issueExpiredTestMountCredential(t, "node-1", defaultWebDAVPath, false)
	assertHTTPStatusWithBasicAuth(t, server.Client(), "PROPFIND", server.URL+defaultWebDAVPath, expiredUsername, expiredPassword, http.StatusUnauthorized)

	readonlyUsername, readonlyPassword := issueTestMountCredential(t, "node-1", defaultWebDAVPath, true)
	request, err := http.NewRequest(http.MethodPut, server.URL+defaultWebDAVPath+"README.txt", strings.NewReader("updated\n"))
	if err != nil {
		t.Fatalf("build PUT request: %v", err)
	}
	request.SetBasicAuth(readonlyUsername, readonlyPassword)
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("PUT %s: %v", request.URL.String(), err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected readonly credential to return 403, got %d", response.StatusCode)
	}
}

func TestBuildExportMountsRejectsInvalidConfigs(t *testing.T) {
	t.Parallel()

	if _, err := buildExportMounts(nil); err == nil {
		t.Fatal("expected empty export paths to fail")
	}
	if _, err := buildExportMounts([]string{"  "}); err == nil {
		t.Fatal("expected blank export path to fail")
	}
	if _, err := buildExportMounts([]string{"/srv/docs", "/srv/docs"}); err == nil {
		t.Fatal("expected duplicate export paths to fail")
	}
}

func assertHTTPStatusWithBasicAuth(t *testing.T, client *http.Client, method string, endpoint string, username string, password string, expectedStatus int) {
	t.Helper()

	request, err := http.NewRequest(method, endpoint, nil)
	if err != nil {
		t.Fatalf("build %s request for %s: %v", method, endpoint, err)
	}
	if username != "" || password != "" {
		request.SetBasicAuth(username, password)
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

func issueTestMountCredential(t *testing.T, nodeID string, mountPath string, readonly bool) (string, string) {
	t.Helper()

	claims := signedMountCredentialClaims{
		Version:   1,
		NodeID:    nodeID,
		MountPath: mountPath,
		Username:  "mount-test-user",
		Readonly:  readonly,
		ExpiresAt: time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
	}
	password, err := encodeTestMountCredential(claims)
	if err != nil {
		t.Fatalf("issue test mount credential: %v", err)
	}

	return claims.Username, password
}

func issueExpiredTestMountCredential(t *testing.T, nodeID string, mountPath string, readonly bool) (string, string) {
	t.Helper()

	claims := signedMountCredentialClaims{
		Version:   1,
		NodeID:    nodeID,
		MountPath: mountPath,
		Username:  "mount-expired-user",
		Readonly:  readonly,
		ExpiresAt: time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
	}
	password, err := encodeTestMountCredential(claims)
	if err != nil {
		t.Fatalf("issue expired test mount credential: %v", err)
	}

	return claims.Username, password
}

func encodeTestMountCredential(claims signedMountCredentialClaims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	return encodedPayload + "." + signMountCredentialPayload(testDAVAuthSecret, encodedPayload), nil
}

func writeExportFile(t *testing.T, directory string, name string, contents string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o644); err != nil {
		t.Fatalf("write export file %s: %v", name, err)
	}
}
