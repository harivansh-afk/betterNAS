package nodeagent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/webdav"
)

const davPrefix = "/dav/"

type App struct {
	cfg          Config
	davFS        *exportFileSystem
	logger       *log.Logger
	server       *http.Server
	registration *registrationLoop
}

func New(cfg Config, logger *log.Logger) (*App, error) {
	if logger == nil {
		logger = log.Default()
	}

	if err := validateRuntimeConfig(cfg); err != nil {
		return nil, err
	}
	if err := ensureExportPath(cfg.ExportPath); err != nil {
		return nil, err
	}

	davFS, err := newExportFileSystem(cfg.ExportPath)
	if err != nil {
		return nil, err
	}

	app := &App{
		cfg:    cfg,
		davFS:  davFS,
		logger: logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", app.handleHealth)
	mux.HandleFunc("/dav", handleDAVRedirect)
	mux.Handle(davPrefix, http.Handler(&webdav.Handler{
		Prefix:     davPrefix,
		FileSystem: app.davFS,
		LockSystem: webdav.NewMemLS(),
	}))
	mux.HandleFunc("/", http.NotFound)

	app.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if cfg.RegisterEnabled {
		app.registration = newRegistrationLoop(cfg, logger)
	}

	return app, nil
}

func (a *App) ListenAndServe(ctx context.Context) error {
	listener, err := net.Listen("tcp", a.cfg.ListenAddress)
	if err != nil {
		a.closeDAVFS()
		return fmt.Errorf("listen on %s: %w", a.cfg.ListenAddress, err)
	}

	a.logger.Printf("betterNAS node agent serving %s at %s on %s", a.cfg.ExportPath, davPrefix, listener.Addr())
	if strings.TrimSpace(a.cfg.ListenAddress) == defaultListenAddress(a.cfg.Port) {
		a.logger.Printf("betterNAS node agent using loopback-only listen address %s by default", a.cfg.ListenAddress)
	}
	if a.registration != nil {
		a.logger.Printf("betterNAS node agent control-plane sync enabled for %s", a.cfg.ControlPlaneURL)
		if strings.TrimSpace(a.cfg.DirectAddress) == "" {
			a.logger.Printf("betterNAS node agent is not advertising a direct address; set BETTERNAS_NODE_DIRECT_ADDRESS if clients should mount this listener directly")
		}
	}

	return a.Serve(ctx, listener)
}

func (a *App) Serve(ctx context.Context, listener net.Listener) error {
	defer a.closeDAVFS()

	serverErrors := make(chan error, 1)

	go func() {
		serverErrors <- a.server.Serve(listener)
	}()

	if a.registration != nil {
		go a.registration.Run(ctx)
	}

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("shutdown node-agent server: %w", err)
	}

	err := <-serverErrors
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}

func (a *App) closeDAVFS() {
	if a.davFS == nil {
		return
	}

	davFS := a.davFS
	a.davFS = nil

	if err := davFS.Close(); err != nil {
		a.logger.Printf("betterNAS node agent failed to close export root %s: %v", a.cfg.ExportPath, err)
	}
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if r.Method != http.MethodHead {
		_, _ = io.WriteString(w, "ok\n")
	}
}

func handleDAVRedirect(w http.ResponseWriter, r *http.Request) {
	location := davPrefix
	if rawQuery := strings.TrimSpace(r.URL.RawQuery); rawQuery != "" {
		location += "?" + rawQuery
	}

	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusPermanentRedirect)
}

func ensureExportPath(exportPath string) error {
	trimmedPath := strings.TrimSpace(exportPath)
	if trimmedPath == "" {
		return fmt.Errorf("export path is required")
	}

	info, err := os.Stat(trimmedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("export path %s does not exist", trimmedPath)
		}

		return fmt.Errorf("stat export path %s: %w", trimmedPath, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("export path %s is not a directory", trimmedPath)
	}

	return nil
}
