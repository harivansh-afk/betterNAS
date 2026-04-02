package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/webdav"
)

const (
	defaultWebDAVPath        = "/dav/"
	exportScopedWebDAVPrefix = "/dav/exports/"
)

type appConfig struct {
	exportPaths  []string
	authUsername string
	authPassword string
}

type app struct {
	authUsername string
	authPassword string
	exportMounts []exportMount
	controlPlane *controlPlaneSession
}

type exportMount struct {
	exportPath string
	mountPath  string
}

func newApp(config appConfig) (*app, error) {
	config.authUsername = strings.TrimSpace(config.authUsername)
	if config.authUsername == "" {
		return nil, errors.New("authUsername is required")
	}
	if config.authPassword == "" {
		return nil, errors.New("authPassword is required")
	}

	exportMounts, err := buildExportMounts(config.exportPaths)
	if err != nil {
		return nil, err
	}

	return &app{
		authUsername: config.authUsername,
		authPassword: config.authPassword,
		exportMounts: exportMounts,
	}, nil
}

func newAppFromEnv() (*app, error) {
	exportPaths, err := exportPathsFromEnv()
	if err != nil {
		return nil, err
	}

	authUsername, err := requiredEnv("BETTERNAS_USERNAME")
	if err != nil {
		return nil, err
	}
	authPassword, err := requiredEnv("BETTERNAS_PASSWORD")
	if err != nil {
		return nil, err
	}
	var controlPlane *controlPlaneSession
	if strings.TrimSpace(env("BETTERNAS_CONTROL_PLANE_URL", "")) != "" {
		session, err := bootstrapNodeAgentFromEnv(exportPaths)
		if err != nil {
			return nil, err
		}
		controlPlane = &session
	}

	app, err := newApp(appConfig{
		exportPaths:  exportPaths,
		authUsername: authUsername,
		authPassword: authPassword,
	})
	if err != nil {
		return nil, err
	}
	app.controlPlane = controlPlane
	return app, nil
}

func (a *app) startControlPlaneLoop(ctx context.Context) {
	if a.controlPlane == nil {
		return
	}

	go runNodeHeartbeatLoop(
		ctx,
		&http.Client{Timeout: 5 * time.Second},
		a.controlPlane.controlPlaneURL,
		a.controlPlane.sessionToken,
		a.controlPlane.nodeID,
		a.controlPlane.heartbeatInterval,
		time.Now,
		log.Default(),
	)
}

func (a *app) controlPlaneEnabled() bool {
	return a.controlPlane != nil
}

func defaultNodeHeartbeatInterval() time.Duration {
	return 30 * time.Second
}

func heartbeatIntervalFromEnv() (time.Duration, error) {
	rawInterval := strings.TrimSpace(env("BETTERNAS_NODE_HEARTBEAT_INTERVAL", "30s"))
	if rawInterval == "" {
		return defaultNodeHeartbeatInterval(), nil
	}

	interval, err := time.ParseDuration(rawInterval)
	if err != nil {
		return 0, err
	}
	if interval <= 0 {
		return 0, errors.New("BETTERNAS_NODE_HEARTBEAT_INTERVAL must be greater than zero")
	}

	return interval, nil
}

func runNodeHeartbeatLoop(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	sessionToken string,
	nodeID string,
	interval time.Duration,
	now func() time.Time,
	logger *log.Logger,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := sendNodeHeartbeatAt(client, baseURL, sessionToken, nodeID, now().UTC()); err != nil && logger != nil {
				logger.Printf("betternas node heartbeat failed: %v", err)
			}
		}
	}
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
		lockSystem := webdav.NewMemLS()
		dav := &webdav.Handler{
			Prefix:     mountPathPrefix,
			FileSystem: fs,
			LockSystem: lockSystem,
		}
		mux.Handle(mount.mountPath, a.requireDAVAuth(mount, finderCompatible(dav, fs, mountPathPrefix)))

		// Register a username-scoped handler at {mountPath}{username}/ so
		// Finder shows the username as the volume name in the sidebar.
		userScopedPath := mount.mountPath + a.authUsername + "/"
		userScopedPrefix := strings.TrimSuffix(userScopedPath, "/")
		userDav := &webdav.Handler{
			Prefix:     userScopedPrefix,
			FileSystem: fs,
			LockSystem: lockSystem,
		}
		mux.Handle(userScopedPath, a.requireDAVAuth(mount, finderCompatible(userDav, fs, userScopedPrefix)))
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
		if !a.matchesAccountCredential(username, password) {
			writeDAVUnauthorized(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *app) matchesAccountCredential(username string, password string) bool {
	return subtle.ConstantTimeCompare([]byte(strings.TrimSpace(username)), []byte(a.authUsername)) == 1 &&
		subtle.ConstantTimeCompare([]byte(password), []byte(a.authPassword)) == 1
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
