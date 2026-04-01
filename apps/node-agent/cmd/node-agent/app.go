package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/webdav"
)

const (
	defaultWebDAVPath        = "/dav/"
	exportScopedWebDAVPrefix = "/dav/exports/"
)

type appConfig struct {
	exportPaths []string
}

type app struct {
	exportMounts []exportMount
}

type exportMount struct {
	exportPath string
	mountPath  string
}

func newApp(config appConfig) (*app, error) {
	exportMounts, err := buildExportMounts(config.exportPaths)
	if err != nil {
		return nil, err
	}

	return &app{exportMounts: exportMounts}, nil
}

func newAppFromEnv() (*app, error) {
	exportPaths, err := exportPathsFromEnv()
	if err != nil {
		return nil, err
	}

	return newApp(appConfig{exportPaths: exportPaths})
}

func exportPathsFromEnv() ([]string, error) {
	rawValue, _ := os.LookupEnv("BETTERNAS_EXPORT_PATHS_JSON")
	raw := strings.TrimSpace(rawValue)
	if raw == "" {
		return []string{env("BETTERNAS_EXPORT_PATH", ".")}, nil
	}

	var exportPaths []string
	if err := json.Unmarshal([]byte(raw), &exportPaths); err != nil {
		return nil, fmt.Errorf("BETTERNAS_EXPORT_PATHS_JSON must be a JSON array of strings: %w", err)
	}
	if len(exportPaths) == 0 {
		return nil, errors.New("BETTERNAS_EXPORT_PATHS_JSON must not be empty")
	}

	return exportPaths, nil
}

func buildExportMounts(exportPaths []string) ([]exportMount, error) {
	if len(exportPaths) == 0 {
		return nil, errors.New("at least one export path is required")
	}

	normalizedPaths := make([]string, len(exportPaths))
	seenPaths := make(map[string]struct{}, len(exportPaths))
	for index, exportPath := range exportPaths {
		normalizedPath := strings.TrimSpace(exportPath)
		if normalizedPath == "" {
			return nil, fmt.Errorf("exportPaths[%d] is required", index)
		}
		if _, ok := seenPaths[normalizedPath]; ok {
			return nil, fmt.Errorf("exportPaths[%d] must be unique", index)
		}

		seenPaths[normalizedPath] = struct{}{}
		normalizedPaths[index] = normalizedPath
	}

	mounts := make([]exportMount, 0, len(normalizedPaths)+1)
	if len(normalizedPaths) == 1 {
		singleExportPath := normalizedPaths[0]
		mounts = append(mounts, exportMount{
			exportPath: singleExportPath,
			mountPath:  defaultWebDAVPath,
		})
		mounts = append(mounts, exportMount{
			exportPath: singleExportPath,
			mountPath:  scopedMountPathForExport(singleExportPath),
		})

		return mounts, nil
	}

	for _, exportPath := range normalizedPaths {
		mounts = append(mounts, exportMount{
			exportPath: exportPath,
			mountPath:  scopedMountPathForExport(exportPath),
		})
	}

	return mounts, nil
}

func (a *app) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})

	for _, mount := range a.exportMounts {
		mountPathPrefix := strings.TrimSuffix(mount.mountPath, "/")
		dav := &webdav.Handler{
			Prefix:     mountPathPrefix,
			FileSystem: webdav.Dir(mount.exportPath),
			LockSystem: webdav.NewMemLS(),
		}
		mux.Handle(mount.mountPath, dav)
	}

	return mux
}

func mountProfilePathForExport(exportPath string, exportCount int) string {
	// Keep /dav/ stable for the common single-export case while exposing distinct
	// scoped roots when a node serves more than one export.
	if exportCount <= 1 {
		return defaultWebDAVPath
	}

	return scopedMountPathForExport(exportPath)
}

func scopedMountPathForExport(exportPath string) string {
	return exportScopedWebDAVPrefix + exportRouteSlug(exportPath) + "/"
}

func exportRouteSlug(exportPath string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(exportPath)))
	return hex.EncodeToString(sum[:])
}
