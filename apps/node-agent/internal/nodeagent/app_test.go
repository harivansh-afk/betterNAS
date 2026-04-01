package nodeagent

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRejectsMissingExportDirectory(t *testing.T) {
	t.Parallel()

	exportPath := filepath.Join(t.TempDir(), "missing-export")

	_, err := New(Config{
		ExportPath:    exportPath,
		ListenAddress: defaultListenAddress(defaultPort),
	}, log.New(io.Discard, "", 0))
	if err == nil {
		t.Fatal("expected missing export directory to fail")
	}

	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("error = %q, want missing-directory message", err.Error())
	}
}

func TestNewRejectsFileExportPath(t *testing.T) {
	t.Parallel()

	exportPath := filepath.Join(t.TempDir(), "export.txt")
	if err := os.WriteFile(exportPath, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write export file: %v", err)
	}

	_, err := New(Config{
		ExportPath:    exportPath,
		ListenAddress: defaultListenAddress(defaultPort),
	}, log.New(io.Discard, "", 0))
	if err == nil {
		t.Fatal("expected file export path to fail")
	}

	if !strings.Contains(err.Error(), "is not a directory") {
		t.Fatalf("error = %q, want not-a-directory message", err.Error())
	}
}

func TestNewRejectsInvalidListenAddress(t *testing.T) {
	t.Parallel()

	_, err := New(Config{
		ExportPath:    t.TempDir(),
		ListenAddress: "localhost",
	}, log.New(io.Discard, "", 0))
	if err == nil {
		t.Fatal("expected invalid listen address to fail")
	}

	if !strings.Contains(err.Error(), listenAddressEnvKey) {
		t.Fatalf("error = %q, want %q", err.Error(), listenAddressEnvKey)
	}
}

func TestNewAcceptsLoopbackListenAddressByDefault(t *testing.T) {
	t.Parallel()

	_, err := New(Config{
		ExportPath:    t.TempDir(),
		ListenAddress: defaultListenAddress(defaultPort),
	}, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
}

func TestNewRejectsRegistrationWithoutMachineID(t *testing.T) {
	t.Parallel()

	_, err := New(Config{
		ExportPath:      t.TempDir(),
		ListenAddress:   defaultListenAddress(defaultPort),
		RegisterEnabled: true,
		ControlPlaneURL: "http://127.0.0.1:8081",
	}, log.New(io.Discard, "", 0))
	if err == nil {
		t.Fatal("expected missing machine id to fail")
	}

	if !strings.Contains(err.Error(), "BETTERNAS_NODE_MACHINE_ID") {
		t.Fatalf("error = %q, want missing-machine-id message", err.Error())
	}
}
