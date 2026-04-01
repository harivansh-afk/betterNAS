package nodeagent

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPort              = "8090"
	defaultAgentVersion      = "0.1.0-dev"
	defaultHeartbeatInterval = 30 * time.Second
	defaultListenHost        = "127.0.0.1"
	exportPathEnvKey         = "BETTERNAS_EXPORT_PATH"
	listenAddressEnvKey      = "BETTERNAS_NODE_LISTEN_ADDRESS"
)

type Config struct {
	Port              string
	ListenAddress     string
	ExportPath        string
	MachineID         string
	DisplayName       string
	AgentVersion      string
	DirectAddress     string
	RelayAddress      string
	ExportLabel       string
	ExportTags        []string
	ControlPlaneURL   string
	ControlPlaneToken string
	RegisterEnabled   bool
	HeartbeatEnabled  bool
	HeartbeatInterval time.Duration
}

type envLookup func(string) (string, bool)

func LoadConfigFromEnv() (Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Config{}, fmt.Errorf("get working directory: %w", err)
	}

	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "betternas-node"
	}

	return loadConfig(os.LookupEnv, cwd, hostname)
}

func loadConfig(lookup envLookup, cwd, hostname string) (Config, error) {
	port := envOrDefault(lookup, "PORT", defaultPort)

	rawExportPath, err := envRequired(lookup, exportPathEnvKey)
	if err != nil {
		return Config{}, err
	}

	exportPath, err := resolveExportPath(rawExportPath, cwd)
	if err != nil {
		return Config{}, err
	}

	listenAddress := envOrDefault(lookup, listenAddressEnvKey, defaultListenAddress(port))

	registerEnabled, err := envBool(lookup, "BETTERNAS_NODE_REGISTER_ENABLED", false)
	if err != nil {
		return Config{}, err
	}

	heartbeatEnabled, err := envBool(lookup, "BETTERNAS_NODE_HEARTBEAT_ENABLED", false)
	if err != nil {
		return Config{}, err
	}

	heartbeatInterval, err := envDuration(lookup, "BETTERNAS_NODE_HEARTBEAT_INTERVAL", defaultHeartbeatInterval)
	if err != nil {
		return Config{}, err
	}

	machineID, machineIDProvided := envOptional(lookup, "BETTERNAS_NODE_MACHINE_ID")
	if !machineIDProvided {
		machineID = hostname
	}

	if registerEnabled && !machineIDProvided {
		return Config{}, fmt.Errorf("BETTERNAS_NODE_MACHINE_ID is required when BETTERNAS_NODE_REGISTER_ENABLED=true")
	}

	displayName := envOrDefault(lookup, "BETTERNAS_NODE_DISPLAY_NAME", machineID)
	agentVersion := envOrDefault(lookup, "BETTERNAS_VERSION", defaultAgentVersion)
	directAddress := envOrDefault(lookup, "BETTERNAS_NODE_DIRECT_ADDRESS", defaultDirectAddress(listenAddress, port))
	relayAddress := envOrDefault(lookup, "BETTERNAS_NODE_RELAY_ADDRESS", "")
	exportLabel := envOrDefault(lookup, "BETTERNAS_EXPORT_LABEL", defaultExportLabel(exportPath))
	exportTags := parseCSVList(envOrDefault(lookup, "BETTERNAS_EXPORT_TAGS", ""))
	controlPlaneURL := strings.TrimRight(envOrDefault(lookup, "BETTERNAS_CONTROL_PLANE_URL", ""), "/")
	controlPlaneToken := envOrDefault(lookup, "BETTERNAS_CONTROL_PLANE_AUTH_TOKEN", "")

	cfg := Config{
		Port:              port,
		ListenAddress:     listenAddress,
		ExportPath:        exportPath,
		MachineID:         machineID,
		DisplayName:       displayName,
		AgentVersion:      agentVersion,
		DirectAddress:     directAddress,
		RelayAddress:      relayAddress,
		ExportLabel:       exportLabel,
		ExportTags:        exportTags,
		ControlPlaneURL:   controlPlaneURL,
		ControlPlaneToken: controlPlaneToken,
		RegisterEnabled:   registerEnabled,
		HeartbeatEnabled:  heartbeatEnabled,
		HeartbeatInterval: heartbeatInterval,
	}

	if err := validateRuntimeConfig(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func resolveExportPath(rawPath, cwd string) (string, error) {
	exportPath := strings.TrimSpace(rawPath)
	if exportPath == "" {
		return "", fmt.Errorf("export path is required")
	}

	if !filepath.IsAbs(exportPath) {
		basePath := cwd
		if workspaceRoot, ok := findWorkspaceRoot(cwd); ok {
			basePath = workspaceRoot
		}

		exportPath = filepath.Join(basePath, exportPath)
	}

	absolutePath, err := filepath.Abs(exportPath)
	if err != nil {
		return "", fmt.Errorf("resolve export path %q: %w", rawPath, err)
	}

	return filepath.Clean(absolutePath), nil
}

func envRequired(lookup envLookup, key string) (string, error) {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", key)
	}

	return strings.TrimSpace(value), nil
}

func envOptional(lookup envLookup, key string) (string, bool) {
	value, ok := lookup(key)
	if !ok {
		return "", false
	}

	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}

	return trimmed, true
}

func defaultListenAddress(port string) string {
	return net.JoinHostPort(defaultListenHost, port)
}

func defaultDirectAddress(listenAddress, fallbackPort string) string {
	if strings.TrimSpace(listenAddress) == defaultListenAddress(fallbackPort) {
		return httpURL("localhost", fallbackPort)
	}

	host, port, err := net.SplitHostPort(strings.TrimSpace(listenAddress))
	if err != nil || strings.TrimSpace(port) == "" {
		return ""
	}

	host = strings.TrimSpace(host)
	if isWildcardListenHost(host) {
		return ""
	}

	return httpURL(host, port)
}

func isWildcardListenHost(host string) bool {
	trimmed := strings.TrimSpace(host)
	if trimmed == "" {
		return true
	}

	ip := net.ParseIP(trimmed)
	return ip != nil && ip.IsUnspecified()
}

func httpURL(host, port string) string {
	return (&url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, port),
	}).String()
}

func findWorkspaceRoot(start string) (string, bool) {
	current := filepath.Clean(start)

	for {
		if hasPath(filepath.Join(current, "pnpm-workspace.yaml")) || hasPath(filepath.Join(current, "go.work")) || hasPath(filepath.Join(current, ".git")) {
			return current, true
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}

		current = parent
	}
}

func hasPath(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func defaultExportLabel(exportPath string) string {
	label := filepath.Base(exportPath)
	if label == "." || label == string(filepath.Separator) || label == "" {
		return "export"
	}

	return label
}

func parseCSVList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}

	values := make([]string, 0)
	seen := make(map[string]struct{})

	for _, part := range strings.Split(raw, ",") {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}

		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		values = append(values, value)
	}

	return values
}

func envOrDefault(lookup envLookup, key, fallback string) string {
	value, ok := lookup(key)
	if !ok {
		return fallback
	}

	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}

	return trimmed
}

func validateRuntimeConfig(cfg Config) error {
	if err := validateListenAddress(cfg.ListenAddress); err != nil {
		return err
	}

	if cfg.RegisterEnabled && strings.TrimSpace(cfg.ControlPlaneURL) == "" {
		return fmt.Errorf("BETTERNAS_CONTROL_PLANE_URL is required when BETTERNAS_NODE_REGISTER_ENABLED=true")
	}

	if cfg.RegisterEnabled && strings.TrimSpace(cfg.MachineID) == "" {
		return fmt.Errorf("BETTERNAS_NODE_MACHINE_ID is required when BETTERNAS_NODE_REGISTER_ENABLED=true")
	}

	if cfg.HeartbeatEnabled && !cfg.RegisterEnabled {
		return fmt.Errorf("BETTERNAS_NODE_HEARTBEAT_ENABLED requires BETTERNAS_NODE_REGISTER_ENABLED=true")
	}

	if cfg.HeartbeatEnabled && cfg.HeartbeatInterval <= 0 {
		return fmt.Errorf("BETTERNAS_NODE_HEARTBEAT_INTERVAL must be greater than zero")
	}

	if cfg.RegisterEnabled && cfg.HeartbeatInterval <= 0 {
		return fmt.Errorf("BETTERNAS_NODE_HEARTBEAT_INTERVAL must be greater than zero when registration is enabled")
	}

	return nil
}

func validateListenAddress(address string) error {
	trimmed := strings.TrimSpace(address)
	if trimmed == "" {
		return fmt.Errorf("%s is required", listenAddressEnvKey)
	}

	_, port, err := net.SplitHostPort(trimmed)
	if err != nil {
		return fmt.Errorf("parse %s: %w", listenAddressEnvKey, err)
	}

	if strings.TrimSpace(port) == "" {
		return fmt.Errorf("%s must include a port", listenAddressEnvKey)
	}

	return nil
}

func envBool(lookup envLookup, key string, fallback bool) (bool, error) {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", key, err)
	}

	return parsed, nil
}

func envDuration(lookup envLookup, key string, fallback time.Duration) (time.Duration, error) {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}

	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", key)
	}

	return parsed, nil
}
