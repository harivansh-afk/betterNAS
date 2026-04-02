package main

import (
	"sort"
	"strings"
	"time"
)

type appConfig struct {
	version             string
	nextcloudBaseURL    string
	statePath           string
	dbPath              string
	sessionTTL          time.Duration
	nodeOfflineThreshold time.Duration
	registrationEnabled bool
	corsOrigin          string
}

type app struct {
	startedAt time.Time
	now       func() time.Time
	config    appConfig
	store     store
}

const defaultNodeOfflineThreshold = 2 * time.Minute

func newApp(config appConfig, startedAt time.Time) (*app, error) {
	if config.nodeOfflineThreshold <= 0 {
		config.nodeOfflineThreshold = defaultNodeOfflineThreshold
	}

	var s store
	var err error
	if config.dbPath != "" {
		s, err = newSQLiteStore(config.dbPath)
	} else {
		s, err = newMemoryStore(config.statePath)
	}
	if err != nil {
		return nil, err
	}

	return &app{
		startedAt: startedAt,
		now:       time.Now,
		config:    config,
		store:     s,
	}, nil
}

func (a *app) presentedNode(node nasNode) nasNode {
	presented := copyNasNode(node)
	if !nodeHeartbeatIsFresh(presented.LastSeenAt, a.now().UTC(), a.config.nodeOfflineThreshold) {
		presented.Status = "offline"
	}
	return presented
}

func (a *app) listNodes(ownerID string) []nasNode {
	nodes := a.store.listNodes(ownerID)
	presented := make([]nasNode, 0, len(nodes))
	for _, node := range nodes {
		presented = append(presented, a.presentedNode(node))
	}

	sort.Slice(presented, func(i, j int) bool {
		return presented[i].ID < presented[j].ID
	})

	return presented
}

func (a *app) listConnectedExports(ownerID string) []storageExport {
	exports := a.store.listExports(ownerID)
	connected := make([]storageExport, 0, len(exports))
	for _, export := range exports {
		context, ok := a.store.exportContext(export.ID, ownerID)
		if !ok {
			continue
		}
		if !nodeIsConnected(a.presentedNode(context.node)) {
			continue
		}
		connected = append(connected, export)
	}

	return connected
}

func nodeHeartbeatIsFresh(lastSeenAt string, referenceTime time.Time, threshold time.Duration) bool {
	lastSeenAt = strings.TrimSpace(lastSeenAt)
	if threshold <= 0 || lastSeenAt == "" {
		return false
	}

	parsedLastSeenAt, err := time.Parse(time.RFC3339, lastSeenAt)
	if err != nil {
		return false
	}

	referenceTime = referenceTime.UTC()
	if parsedLastSeenAt.After(referenceTime) {
		return true
	}

	return referenceTime.Sub(parsedLastSeenAt) <= threshold
}

func nodeIsConnected(node nasNode) bool {
	return node.Status == "online" || node.Status == "degraded"
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
	OwnerID       string  `json:"-"`
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
	OwnerID       string   `json:"-"`
}

type mountProfile struct {
	ID          string          `json:"id"`
	ExportID    string          `json:"exportId"`
	Protocol    string          `json:"protocol"`
	DisplayName string          `json:"displayName"`
	MountURL    string          `json:"mountUrl"`
	Readonly    bool            `json:"readonly"`
	Credential  mountCredential `json:"credential"`
}

type mountCredential struct {
	Mode      string `json:"mode"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	ExpiresAt string `json:"expiresAt"`
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
	MachineID     string  `json:"machineId"`
	DisplayName   string  `json:"displayName"`
	AgentVersion  string  `json:"agentVersion"`
	DirectAddress *string `json:"directAddress"`
	RelayAddress  *string `json:"relayAddress"`
}

type nodeExportsRequest struct {
	Exports []storageExportInput `json:"exports"`
}

type nodeHeartbeatRequest struct {
	NodeID     string `json:"nodeId"`
	Status     string `json:"status"`
	LastSeenAt string `json:"lastSeenAt"`
}

type mountProfileRequest struct {
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

type user struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	CreatedAt string `json:"createdAt"`
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
