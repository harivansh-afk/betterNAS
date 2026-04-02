package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testUsername = "alice"
	testPassword = "password123"
)

func TestSingleExportServesDefaultAndScopedMountPathsWithValidCredentials(t *testing.T) {
	t.Parallel()

	exportDir := t.TempDir()
	writeExportFile(t, exportDir, "README.txt", "single export\n")

	app, err := newApp(appConfig{
		exportPaths:  []string{exportDir},
		authUsername: testUsername,
		authPassword: testPassword,
	})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	server := httptest.NewServer(app.handler())
	defer server.Close()

	scopedMountPath := scopedMountPathForExport(exportDir)

	assertHTTPStatusWithBasicAuth(t, server.Client(), "PROPFIND", server.URL+defaultWebDAVPath, testUsername, testPassword, http.StatusMultiStatus)
	assertHTTPStatusWithBasicAuth(t, server.Client(), "PROPFIND", server.URL+scopedMountPath, testUsername, testPassword, http.StatusMultiStatus)
	assertMountedFileContentsWithBasicAuth(t, server.Client(), server.URL+defaultWebDAVPath+"README.txt", testUsername, testPassword, "single export\n")
	assertMountedFileContentsWithBasicAuth(t, server.Client(), server.URL+scopedMountPath+"README.txt", testUsername, testPassword, "single export\n")
}

func TestMultipleExportsServeDistinctScopedMountPathsWithValidCredentials(t *testing.T) {
	t.Parallel()

	firstExportDir := t.TempDir()
	secondExportDir := t.TempDir()
	writeExportFile(t, firstExportDir, "README.txt", "first export\n")
	writeExportFile(t, secondExportDir, "README.txt", "second export\n")

	app, err := newApp(appConfig{
		exportPaths:  []string{firstExportDir, secondExportDir},
		authUsername: testUsername,
		authPassword: testPassword,
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

	assertHTTPStatusWithBasicAuth(t, server.Client(), "PROPFIND", server.URL+firstMountPath, testUsername, testPassword, http.StatusMultiStatus)
	assertHTTPStatusWithBasicAuth(t, server.Client(), "PROPFIND", server.URL+secondMountPath, testUsername, testPassword, http.StatusMultiStatus)
	assertMountedFileContentsWithBasicAuth(t, server.Client(), server.URL+firstMountPath+"README.txt", testUsername, testPassword, "first export\n")
	assertMountedFileContentsWithBasicAuth(t, server.Client(), server.URL+secondMountPath+"README.txt", testUsername, testPassword, "second export\n")

	response, err := server.Client().Get(server.URL + defaultWebDAVPath)
	if err != nil {
		t.Fatalf("get default multi-export mount path: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected %s to return 404 for multi-export config, got %d", defaultWebDAVPath, response.StatusCode)
	}
}

func TestDAVAuthRejectsMissingAndInvalidCredentials(t *testing.T) {
	t.Parallel()

	exportDir := t.TempDir()
	writeExportFile(t, exportDir, "README.txt", "mutable export\n")

	app, err := newApp(appConfig{
		exportPaths:  []string{exportDir},
		authUsername: testUsername,
		authPassword: testPassword,
	})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	server := httptest.NewServer(app.handler())
	defer server.Close()

	assertHTTPStatusWithBasicAuth(t, server.Client(), "PROPFIND", server.URL+defaultWebDAVPath, "", "", http.StatusUnauthorized)
	assertHTTPStatusWithBasicAuth(t, server.Client(), "PROPFIND", server.URL+defaultWebDAVPath, "wrong-user", testPassword, http.StatusUnauthorized)
	assertHTTPStatusWithBasicAuth(t, server.Client(), "PROPFIND", server.URL+defaultWebDAVPath, testUsername, "wrong-password", http.StatusUnauthorized)

	request, err := http.NewRequest(http.MethodPut, server.URL+defaultWebDAVPath+"README.txt", strings.NewReader("updated\n"))
	if err != nil {
		t.Fatalf("build PUT request: %v", err)
	}
	request.SetBasicAuth(testUsername, testPassword)
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("PUT %s: %v", request.URL.String(), err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusNoContent && response.StatusCode != http.StatusOK {
		t.Fatalf("expected write with valid credentials to succeed, got %d", response.StatusCode)
	}

	assertMountedFileContentsWithBasicAuth(t, server.Client(), server.URL+defaultWebDAVPath+"README.txt", testUsername, testPassword, "updated\n")
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

func writeExportFile(t *testing.T, directory string, name string, contents string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o644); err != nil {
		t.Fatalf("write export file %s: %v", name, err)
	}
}
