package main

import (
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
	exportPaths   []string
	nodeID        string
	davAuthSecret string
}

type app struct {
	nodeID        string
	davAuthSecret string
	exportMounts  []exportMount
}

type exportMount struct {
	exportPath string
	mountPath  string
}

func newApp(config appConfig) (*app, error) {
	config.nodeID = strings.TrimSpace(config.nodeID)
	if config.nodeID == "" {
		return nil, errors.New("nodeID is required")
	}

	config.davAuthSecret = strings.TrimSpace(config.davAuthSecret)
	if config.davAuthSecret == "" {
		return nil, errors.New("davAuthSecret is required")
	}

	exportMounts, err := buildExportMounts(config.exportPaths)
	if err != nil {
		return nil, err
	}

	return &app{
		nodeID:        config.nodeID,
		davAuthSecret: config.davAuthSecret,
		exportMounts:  exportMounts,
	}, nil
}

func newAppFromEnv() (*app, error) {
	exportPaths, err := exportPathsFromEnv()
	if err != nil {
		return nil, err
	}

	davAuthSecret, err := requiredEnv("BETTERNAS_DAV_AUTH_SECRET")
	if err != nil {
		return nil, err
	}

	nodeID := strings.TrimSpace(env("BETTERNAS_NODE_ID", ""))
	if strings.TrimSpace(env("BETTERNAS_CONTROL_PLANE_URL", "")) != "" {
		bootstrapResult, err := bootstrapNodeAgentFromEnv(exportPaths)
		if err != nil {
			return nil, err
		}
		nodeID = bootstrapResult.nodeID
	}

	return newApp(appConfig{
		exportPaths:   exportPaths,
		nodeID:        nodeID,
		davAuthSecret: davAuthSecret,
	})
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
		fs := webdav.Dir(mount.exportPath)
		dav := &webdav.Handler{
			Prefix:     mountPathPrefix,
			FileSystem: fs,
			LockSystem: webdav.NewMemLS(),
		}
		mux.Handle(mount.mountPath, a.requireDAVAuth(mount, finderCompatible(dav, fs, mountPathPrefix)))
	}

	return mux
}

func (a *app) requireDAVAuth(mount exportMount, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// OPTIONS must bypass auth so the WebDAV handler can return its
		// DAV: 1, 2 compliance header. macOS Finder sends an unauthenticated
		// OPTIONS first and refuses to connect unless it sees that header.
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		username, password, ok := r.BasicAuth()
		if !ok {
			writeDAVUnauthorized(w)
			return
		}

		claims, err := verifyMountCredential(a.davAuthSecret, password)
		if err != nil {
			writeDAVUnauthorized(w)
			return
		}
		if claims.NodeID != a.nodeID || claims.MountPath != mount.mountPath || claims.Username != username {
			writeDAVUnauthorized(w)
			return
		}
		if claims.Readonly && !isDAVReadMethod(r.Method) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
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
	return stableExportRouteSlug(exportPath)
}
