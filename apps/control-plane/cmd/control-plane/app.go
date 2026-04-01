package main

import (
	"errors"
	"strings"
	"time"
)

type appConfig struct {
	version            string
	nextcloudBaseURL   string
	statePath          string
	clientToken        string
	nodeBootstrapToken string
}

type app struct {
	startedAt time.Time
	now       func() time.Time
	config    appConfig
	store     *memoryStore
}

func newApp(config appConfig, startedAt time.Time) (*app, error) {
	config.clientToken = strings.TrimSpace(config.clientToken)
	if config.clientToken == "" {
		return nil, errors.New("client token is required")
	}

	config.nodeBootstrapToken = strings.TrimSpace(config.nodeBootstrapToken)
	if config.nodeBootstrapToken == "" {
		return nil, errors.New("node bootstrap token is required")
	}

	store, err := newMemoryStore(config.statePath)
	if err != nil {
		return nil, err
	}

	return &app{
		startedAt: startedAt,
		now:       time.Now,
		config:    config,
		store:     store,
	}, nil
}

type nextcloudBackendStatus struct {
	Configured bool   `json:"configured"`
	BaseURL    string `json:"baseUrl"`
	Provider   string `json:"provider"`
}

type controlPlaneHealthResponse struct {
	Service       string                 `json:"service"`
	Status        string                 `json:"status"`
	Timestamp     string                 `json:"timestamp"`
	UptimeSeconds int                    `json:"uptimeSeconds"`
	Nextcloud     nextcloudBackendStatus `json:"nextcloud"`
}

type controlPlaneVersionResponse struct {
	Service    string `json:"service"`
	Version    string `json:"version"`
	APIVersion string `json:"apiVersion"`
}

type nasNode struct {
	ID            string  `json:"id"`
	MachineID     string  `json:"machineId"`
	DisplayName   string  `json:"displayName"`
	AgentVersion  string  `json:"agentVersion"`
	Status        string  `json:"status"`
	LastSeenAt    string  `json:"lastSeenAt"`
	DirectAddress *string `json:"directAddress"`
	RelayAddress  *string `json:"relayAddress"`
}

type storageExport struct {
	ID            string   `json:"id"`
	NasNodeID     string   `json:"nasNodeId"`
	Label         string   `json:"label"`
	Path          string   `json:"path"`
	MountPath     string   `json:"mountPath,omitempty"`
	Protocols     []string `json:"protocols"`
	CapacityBytes *int64   `json:"capacityBytes"`
	Tags          []string `json:"tags"`
}

type mountProfile struct {
	ID             string `json:"id"`
	ExportID       string `json:"exportId"`
	Protocol       string `json:"protocol"`
	DisplayName    string `json:"displayName"`
	MountURL       string `json:"mountUrl"`
	Readonly       bool   `json:"readonly"`
	CredentialMode string `json:"credentialMode"`
}

type cloudProfile struct {
	ID       string `json:"id"`
	ExportID string `json:"exportId"`
	Provider string `json:"provider"`
	BaseURL  string `json:"baseUrl"`
	Path     string `json:"path"`
}

type storageExportInput struct {
	Label         string   `json:"label"`
	Path          string   `json:"path"`
	MountPath     string   `json:"mountPath,omitempty"`
	Protocols     []string `json:"protocols"`
	CapacityBytes *int64   `json:"capacityBytes"`
	Tags          []string `json:"tags"`
}

type nodeRegistrationRequest struct {
	MachineID     string               `json:"machineId"`
	DisplayName   string               `json:"displayName"`
	AgentVersion  string               `json:"agentVersion"`
	DirectAddress *string              `json:"directAddress"`
	RelayAddress  *string              `json:"relayAddress"`
	Exports       []storageExportInput `json:"exports"`
}

type nodeHeartbeatRequest struct {
	NodeID     string `json:"nodeId"`
	Status     string `json:"status"`
	LastSeenAt string `json:"lastSeenAt"`
}

type mountProfileRequest struct {
	UserID   string `json:"userId"`
	DeviceID string `json:"deviceId"`
	ExportID string `json:"exportId"`
}

type cloudProfileRequest struct {
	UserID   string `json:"userId"`
	ExportID string `json:"exportId"`
	Provider string `json:"provider"`
}

type exportContext struct {
	export storageExport
	node   nasNode
}

func copyStringPointer(value *string) *string {
	if value == nil {
		return nil
	}

	copied := *value
	return &copied
}

func copyInt64Pointer(value *int64) *int64 {
	if value == nil {
		return nil
	}

	copied := *value
	return &copied
}

func copyStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	copied := make([]string, len(values))
	copy(copied, values)

	return copied
}
