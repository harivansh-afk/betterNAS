package nodeagent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadConfigResolvesRelativeExportPathFromWorkspaceRoot(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	agentDir := filepath.Join(repoRoot, "apps", "node-agent")

	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("create agent dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repoRoot, "pnpm-workspace.yaml"), []byte("packages:\n  - apps/*\n"), 0o644); err != nil {
		t.Fatalf("write workspace file: %v", err)
	}

	cfg, err := loadConfig(
		mapLookup(map[string]string{
			exportPathEnvKey:                     ".state/nas/export",
			"BETTERNAS_NODE_MACHINE_ID":          "nas-machine-id",
			"BETTERNAS_EXPORT_TAGS":              "finder, photos, finder",
			"BETTERNAS_NODE_REGISTER_ENABLED":    "true",
			"BETTERNAS_NODE_HEARTBEAT_ENABLED":   "true",
			"BETTERNAS_CONTROL_PLANE_URL":        "http://127.0.0.1:8081/",
			"BETTERNAS_CONTROL_PLANE_AUTH_TOKEN": "node-auth-token",
			"BETTERNAS_NODE_HEARTBEAT_INTERVAL":  "45s",
		}),
		agentDir,
		"nas-box",
	)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	expectedExportPath := filepath.Join(repoRoot, ".state", "nas", "export")
	if cfg.ExportPath != expectedExportPath {
		t.Fatalf("export path = %q, want %q", cfg.ExportPath, expectedExportPath)
	}

	if cfg.ListenAddress != defaultListenAddress(defaultPort) {
		t.Fatalf("listen address = %q, want %q", cfg.ListenAddress, defaultListenAddress(defaultPort))
	}

	if cfg.MachineID != "nas-machine-id" {
		t.Fatalf("machine id = %q, want nas-machine-id", cfg.MachineID)
	}

	if cfg.DisplayName != "nas-machine-id" {
		t.Fatalf("display name = %q, want nas-machine-id", cfg.DisplayName)
	}

	if cfg.DirectAddress != "http://localhost:8090" {
		t.Fatalf("direct address = %q, want loopback default", cfg.DirectAddress)
	}

	if cfg.ExportLabel != "export" {
		t.Fatalf("export label = %q, want export", cfg.ExportLabel)
	}

	if len(cfg.ExportTags) != 2 || cfg.ExportTags[0] != "finder" || cfg.ExportTags[1] != "photos" {
		t.Fatalf("export tags = %#v, want [finder photos]", cfg.ExportTags)
	}

	if !cfg.RegisterEnabled {
		t.Fatalf("register enabled = false, want true")
	}

	if !cfg.HeartbeatEnabled {
		t.Fatalf("heartbeat enabled = false, want true")
	}

	if cfg.HeartbeatInterval != 45*time.Second {
		t.Fatalf("heartbeat interval = %s, want 45s", cfg.HeartbeatInterval)
	}

	if cfg.ControlPlaneURL != "http://127.0.0.1:8081" {
		t.Fatalf("control plane url = %q, want trimmed url", cfg.ControlPlaneURL)
	}

	if cfg.ControlPlaneToken != "node-auth-token" {
		t.Fatalf("control plane token = %q, want node-auth-token", cfg.ControlPlaneToken)
	}
}

func TestLoadConfigDefaultsRegistrationToDisabled(t *testing.T) {
	t.Parallel()

	cfg, err := loadConfig(
		mapLookup(map[string]string{
			exportPathEnvKey: ".state/nas/export",
		}),
		t.TempDir(),
		"nas-box",
	)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.RegisterEnabled {
		t.Fatal("register enabled = true, want false")
	}

	if cfg.HeartbeatEnabled {
		t.Fatal("heartbeat enabled = true, want false")
	}

	if cfg.ControlPlaneURL != "" {
		t.Fatalf("control plane url = %q, want empty", cfg.ControlPlaneURL)
	}

	if cfg.MachineID != "nas-box" {
		t.Fatalf("machine id = %q, want nas-box", cfg.MachineID)
	}

	if cfg.ListenAddress != defaultListenAddress(defaultPort) {
		t.Fatalf("listen address = %q, want %q", cfg.ListenAddress, defaultListenAddress(defaultPort))
	}
}

func TestLoadConfigDefaultsHeartbeatToDisabledEvenWhenRegistrationEnabled(t *testing.T) {
	t.Parallel()

	cfg, err := loadConfig(
		mapLookup(map[string]string{
			exportPathEnvKey:                  ".state/nas/export",
			"BETTERNAS_NODE_MACHINE_ID":       "nas-machine-id",
			"BETTERNAS_NODE_REGISTER_ENABLED": "true",
			"BETTERNAS_CONTROL_PLANE_URL":     "http://127.0.0.1:8081",
		}),
		t.TempDir(),
		"nas-box",
	)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if !cfg.RegisterEnabled {
		t.Fatal("register enabled = false, want true")
	}

	if cfg.HeartbeatEnabled {
		t.Fatal("heartbeat enabled = true, want false")
	}
}

func TestLoadConfigRejectsHeartbeatWithoutRegistration(t *testing.T) {
	t.Parallel()

	_, err := loadConfig(
		mapLookup(map[string]string{
			exportPathEnvKey:                   ".state/nas/export",
			"BETTERNAS_NODE_HEARTBEAT_ENABLED": "true",
		}),
		t.TempDir(),
		"nas-box",
	)
	if err == nil {
		t.Fatal("expected heartbeat-only config to fail")
	}
}

func TestLoadConfigRequiresExportPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		values map[string]string
	}{
		{
			name:   "missing",
			values: map[string]string{},
		},
		{
			name: "blank",
			values: map[string]string{
				exportPathEnvKey: "   ",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := loadConfig(mapLookup(testCase.values), t.TempDir(), "nas-box")
			if err == nil {
				t.Fatal("expected missing export path to fail")
			}

			if !strings.Contains(err.Error(), exportPathEnvKey) {
				t.Fatalf("error = %q, want %q", err.Error(), exportPathEnvKey)
			}
		})
	}
}

func TestLoadConfigDefaultsListenAddressToLoopback(t *testing.T) {
	t.Parallel()

	cfg, err := loadConfig(
		mapLookup(map[string]string{
			exportPathEnvKey: ".state/nas/export",
			"PORT":           "9100",
		}),
		t.TempDir(),
		"nas-box",
	)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.ListenAddress != "127.0.0.1:9100" {
		t.Fatalf("listen address = %q, want 127.0.0.1:9100", cfg.ListenAddress)
	}

	if cfg.DirectAddress != "http://localhost:9100" {
		t.Fatalf("direct address = %q, want http://localhost:9100", cfg.DirectAddress)
	}
}

func TestLoadConfigUsesExplicitWildcardListenAddress(t *testing.T) {
	t.Parallel()

	cfg, err := loadConfig(
		mapLookup(map[string]string{
			exportPathEnvKey:    ".state/nas/export",
			listenAddressEnvKey: ":9090",
		}),
		t.TempDir(),
		"nas-box",
	)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.ListenAddress != ":9090" {
		t.Fatalf("listen address = %q, want :9090", cfg.ListenAddress)
	}

	if cfg.DirectAddress != "" {
		t.Fatalf("direct address = %q, want empty for wildcard listener", cfg.DirectAddress)
	}
}

func TestLoadConfigDerivesDirectAddressFromExplicitHostListenAddress(t *testing.T) {
	t.Parallel()

	cfg, err := loadConfig(
		mapLookup(map[string]string{
			exportPathEnvKey:    ".state/nas/export",
			listenAddressEnvKey: "192.0.2.10:9443",
		}),
		t.TempDir(),
		"nas-box",
	)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.DirectAddress != "http://192.0.2.10:9443" {
		t.Fatalf("direct address = %q, want http://192.0.2.10:9443", cfg.DirectAddress)
	}
}

func TestLoadConfigDoesNotDeriveDirectAddressFromWildcardHostListenAddress(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		listenAddress string
	}{
		{
			name:          "ipv4 wildcard",
			listenAddress: "0.0.0.0:9443",
		},
		{
			name:          "ipv6 wildcard",
			listenAddress: "[::]:9443",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := loadConfig(
				mapLookup(map[string]string{
					exportPathEnvKey:    ".state/nas/export",
					listenAddressEnvKey: testCase.listenAddress,
				}),
				t.TempDir(),
				"nas-box",
			)
			if err != nil {
				t.Fatalf("load config: %v", err)
			}

			if cfg.DirectAddress != "" {
				t.Fatalf("direct address = %q, want empty for %q", cfg.DirectAddress, testCase.listenAddress)
			}
		})
	}
}

func TestLoadConfigRejectsInvalidListenAddress(t *testing.T) {
	t.Parallel()

	_, err := loadConfig(
		mapLookup(map[string]string{
			exportPathEnvKey:    ".state/nas/export",
			listenAddressEnvKey: "localhost",
		}),
		t.TempDir(),
		"nas-box",
	)
	if err == nil {
		t.Fatal("expected invalid listen address to fail")
	}

	if !strings.Contains(err.Error(), listenAddressEnvKey) {
		t.Fatalf("error = %q, want %q", err.Error(), listenAddressEnvKey)
	}
}

func TestLoadConfigAllowsRegistrationWithoutControlPlaneToken(t *testing.T) {
	t.Parallel()

	cfg, err := loadConfig(
		mapLookup(map[string]string{
			exportPathEnvKey:                  ".state/nas/export",
			"BETTERNAS_NODE_MACHINE_ID":       "nas-machine-id",
			"BETTERNAS_NODE_REGISTER_ENABLED": "true",
			"BETTERNAS_CONTROL_PLANE_URL":     "http://127.0.0.1:8081",
		}),
		t.TempDir(),
		"nas-box",
	)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.ControlPlaneToken != "" {
		t.Fatalf("control-plane token = %q, want empty", cfg.ControlPlaneToken)
	}
}

func TestLoadConfigRejectsRegistrationWithoutMachineID(t *testing.T) {
	t.Parallel()

	_, err := loadConfig(
		mapLookup(map[string]string{
			exportPathEnvKey:                  ".state/nas/export",
			"BETTERNAS_NODE_REGISTER_ENABLED": "true",
			"BETTERNAS_CONTROL_PLANE_URL":     "http://127.0.0.1:8081",
		}),
		t.TempDir(),
		"nas-box",
	)
	if err == nil {
		t.Fatal("expected missing machine id to fail")
	}

	if !strings.Contains(err.Error(), "BETTERNAS_NODE_MACHINE_ID") {
		t.Fatalf("error = %q, want missing-machine-id message", err.Error())
	}
}

func TestLoadConfigRejectsRegistrationWithoutControlPlaneURL(t *testing.T) {
	t.Parallel()

	_, err := loadConfig(
		mapLookup(map[string]string{
			exportPathEnvKey:                  ".state/nas/export",
			"BETTERNAS_NODE_MACHINE_ID":       "nas-machine-id",
			"BETTERNAS_NODE_REGISTER_ENABLED": "true",
		}),
		t.TempDir(),
		"nas-box",
	)
	if err == nil {
		t.Fatal("expected missing control-plane url to fail")
	}

	if !strings.Contains(err.Error(), "BETTERNAS_CONTROL_PLANE_URL") {
		t.Fatalf("error = %q, want missing-control-plane-url message", err.Error())
	}
}

func mapLookup(values map[string]string) envLookup {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
