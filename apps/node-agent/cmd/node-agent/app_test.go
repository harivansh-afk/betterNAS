package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSingleExportServesDefaultAndScopedMountPaths(t *testing.T) {
	t.Parallel()

	exportDir := t.TempDir()
	writeExportFile(t, exportDir, "README.txt", "single export\n")

	app, err := newApp(appConfig{exportPaths: []string{exportDir}})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	server := httptest.NewServer(app.handler())
	defer server.Close()

	assertHTTPStatus(t, server.Client(), "PROPFIND", server.URL+defaultWebDAVPath, http.StatusMultiStatus)
	assertHTTPStatus(t, server.Client(), "PROPFIND", server.URL+scopedMountPathForExport(exportDir), http.StatusMultiStatus)
	assertMountedFileContents(t, server.Client(), server.URL+defaultWebDAVPath+"README.txt", "single export\n")
	assertMountedFileContents(t, server.Client(), server.URL+scopedMountPathForExport(exportDir)+"README.txt", "single export\n")
}

func TestMultipleExportsServeDistinctScopedMountPaths(t *testing.T) {
	t.Parallel()

	firstExportDir := t.TempDir()
	secondExportDir := t.TempDir()
	writeExportFile(t, firstExportDir, "README.txt", "first export\n")
	writeExportFile(t, secondExportDir, "README.txt", "second export\n")

	app, err := newApp(appConfig{exportPaths: []string{firstExportDir, secondExportDir}})
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

	assertHTTPStatus(t, server.Client(), "PROPFIND", server.URL+firstMountPath, http.StatusMultiStatus)
	assertHTTPStatus(t, server.Client(), "PROPFIND", server.URL+secondMountPath, http.StatusMultiStatus)
	assertMountedFileContents(t, server.Client(), server.URL+firstMountPath+"README.txt", "first export\n")
	assertMountedFileContents(t, server.Client(), server.URL+secondMountPath+"README.txt", "second export\n")

	response, err := server.Client().Get(server.URL + defaultWebDAVPath)
	if err != nil {
		t.Fatalf("get default multi-export mount path: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected %s to return 404 for multi-export config, got %d", defaultWebDAVPath, response.StatusCode)
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

func writeExportFile(t *testing.T, directory string, name string, contents string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o644); err != nil {
		t.Fatalf("write export file %s: %v", name, err)
	}
}
