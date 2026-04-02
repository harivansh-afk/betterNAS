package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type bootstrapResult struct {
	nodeID string
}

type nodeRegistrationRequest struct {
	MachineID     string  `json:"machineId"`
	DisplayName   string  `json:"displayName"`
	AgentVersion  string  `json:"agentVersion"`
	DirectAddress *string `json:"directAddress"`
	RelayAddress  *string `json:"relayAddress"`
}

type nodeRegistrationResponse struct {
	ID string `json:"id"`
}

type authLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authLoginResponse struct {
	Token string `json:"token"`
}

type nodeExportsRequest struct {
	Exports []storageExportInput `json:"exports"`
}

type storageExportInput struct {
	Label         string   `json:"label"`
	Path          string   `json:"path"`
	MountPath     string   `json:"mountPath"`
	Protocols     []string `json:"protocols"`
	CapacityBytes *int64   `json:"capacityBytes"`
	Tags          []string `json:"tags"`
}

type nodeHeartbeatRequest struct {
	NodeID     string `json:"nodeId"`
	Status     string `json:"status"`
	LastSeenAt string `json:"lastSeenAt"`
}

func bootstrapNodeAgentFromEnv(exportPaths []string) (bootstrapResult, error) {
	controlPlaneURL := strings.TrimSpace(env("BETTERNAS_CONTROL_PLANE_URL", "https://api.betternas.com"))
	if controlPlaneURL == "" {
		return bootstrapResult{}, fmt.Errorf("BETTERNAS_CONTROL_PLANE_URL is required")
	}

	username, err := requiredEnv("BETTERNAS_USERNAME")
	if err != nil {
		return bootstrapResult{}, err
	}
	password, err := requiredEnv("BETTERNAS_PASSWORD")
	if err != nil {
		return bootstrapResult{}, err
	}

	machineID := strings.TrimSpace(env("BETTERNAS_NODE_MACHINE_ID", defaultNodeMachineID(username)))
	displayName := strings.TrimSpace(env("BETTERNAS_NODE_DISPLAY_NAME", defaultNodeDisplayName(machineID)))
	if displayName == "" {
		displayName = machineID
	}

	client := &http.Client{Timeout: 5 * time.Second}
	sessionToken, err := loginWithControlPlane(client, controlPlaneURL, username, password)
	if err != nil {
		return bootstrapResult{}, err
	}

	registration, err := registerNodeWithControlPlane(client, controlPlaneURL, sessionToken, nodeRegistrationRequest{
		MachineID:     machineID,
		DisplayName:   displayName,
		AgentVersion:  env("BETTERNAS_VERSION", "0.1.0-dev"),
		DirectAddress: optionalEnvPointer("BETTERNAS_NODE_DIRECT_ADDRESS"),
		RelayAddress:  optionalEnvPointer("BETTERNAS_NODE_RELAY_ADDRESS"),
	})
	if err != nil {
		return bootstrapResult{}, err
	}

	if err := syncNodeExportsWithControlPlane(client, controlPlaneURL, sessionToken, registration.ID, buildStorageExportInputs(exportPaths)); err != nil {
		return bootstrapResult{}, err
	}
	if err := sendNodeHeartbeat(client, controlPlaneURL, sessionToken, registration.ID); err != nil {
		return bootstrapResult{}, err
	}

	return bootstrapResult{nodeID: registration.ID}, nil
}

func loginWithControlPlane(client *http.Client, baseURL string, username string, password string) (string, error) {
	response, err := doControlPlaneJSONRequest(client, http.MethodPost, controlPlaneEndpoint(baseURL, "/api/v1/auth/login"), "", authLoginRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", controlPlaneResponseError("login", response)
	}

	var auth authLoginResponse
	if err := json.NewDecoder(response.Body).Decode(&auth); err != nil {
		return "", fmt.Errorf("decode login response: %w", err)
	}
	if strings.TrimSpace(auth.Token) == "" {
		return "", fmt.Errorf("login: missing session token")
	}

	return strings.TrimSpace(auth.Token), nil
}

func registerNodeWithControlPlane(client *http.Client, baseURL string, token string, payload nodeRegistrationRequest) (nodeRegistrationResponse, error) {
	response, err := doControlPlaneJSONRequest(client, http.MethodPost, controlPlaneEndpoint(baseURL, "/api/v1/nodes/register"), token, payload)
	if err != nil {
		return nodeRegistrationResponse{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nodeRegistrationResponse{}, controlPlaneResponseError("register node", response)
	}

	var registration nodeRegistrationResponse
	if err := json.NewDecoder(response.Body).Decode(&registration); err != nil {
		return nodeRegistrationResponse{}, fmt.Errorf("decode register node response: %w", err)
	}

	return registration, nil
}

func syncNodeExportsWithControlPlane(client *http.Client, baseURL string, token string, nodeID string, exports []storageExportInput) error {
	response, err := doControlPlaneJSONRequest(client, http.MethodPut, controlPlaneEndpoint(baseURL, "/api/v1/nodes/"+nodeID+"/exports"), token, nodeExportsRequest{
		Exports: exports,
	})
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return controlPlaneResponseError("sync node exports", response)
	}

	_, _ = io.Copy(io.Discard, response.Body)
	return nil
}

func sendNodeHeartbeat(client *http.Client, baseURL string, token string, nodeID string) error {
	response, err := doControlPlaneJSONRequest(client, http.MethodPost, controlPlaneEndpoint(baseURL, "/api/v1/nodes/"+nodeID+"/heartbeat"), token, nodeHeartbeatRequest{
		NodeID:     nodeID,
		Status:     "online",
		LastSeenAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return controlPlaneResponseError("send node heartbeat", response)
	}

	return nil
}

func doControlPlaneJSONRequest(client *http.Client, method string, endpoint string, token string, payload any) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal %s %s payload: %w", method, endpoint, err)
	}

	request, err := http.NewRequest(method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build %s %s request: %w", method, endpoint, err)
	}
	request.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(token) != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, endpoint, err)
	}

	return response, nil
}

func controlPlaneEndpoint(baseURL string, suffix string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/") + suffix
}

func controlPlaneResponseError(action string, response *http.Response) error {
	body, _ := io.ReadAll(response.Body)
	return fmt.Errorf("%s: unexpected status %d: %s", action, response.StatusCode, strings.TrimSpace(string(body)))
}

func buildStorageExportInputs(exportPaths []string) []storageExportInput {
	inputs := make([]storageExportInput, len(exportPaths))
	for index, exportPath := range exportPaths {
		inputs[index] = storageExportInput{
			Label:         exportLabel(exportPath),
			Path:          strings.TrimSpace(exportPath),
			MountPath:     mountProfilePathForExport(exportPath, len(exportPaths)),
			Protocols:     []string{"webdav"},
			CapacityBytes: nil,
			Tags:          []string{},
		}
	}

	return inputs
}

func exportLabel(exportPath string) string {
	base := filepath.Base(filepath.Clean(strings.TrimSpace(exportPath)))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "export"
	}

	return base
}

func stableExportRouteSlug(exportPath string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(exportPath)))
	return hex.EncodeToString(sum[:])
}

func requiredEnv(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", key)
	}

	return strings.TrimSpace(value), nil
}

func optionalEnvPointer(key string) *string {
	value := strings.TrimSpace(env(key, ""))
	if value == "" {
		return nil
	}

	return &value
}

func defaultNodeMachineID(username string) string {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		return strings.TrimSpace(username) + "@node"
	}

	return strings.TrimSpace(username) + "@" + strings.TrimSpace(hostname)
}

func defaultNodeDisplayName(machineID string) string {
	_, displayName, ok := strings.Cut(strings.TrimSpace(machineID), "@")
	if ok && strings.TrimSpace(displayName) != "" {
		return strings.TrimSpace(displayName)
	}

	return strings.TrimSpace(machineID)
}
