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

const controlPlaneNodeTokenHeader = "X-BetterNAS-Node-Token"

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
	controlPlaneURL, err := requiredEnv("BETTERNAS_CONTROL_PLANE_URL")
	if err != nil {
		return bootstrapResult{}, err
	}

	bootstrapToken, err := requiredEnv("BETTERNAS_CONTROL_PLANE_NODE_BOOTSTRAP_TOKEN")
	if err != nil {
		return bootstrapResult{}, err
	}

	nodeTokenPath, err := requiredEnv("BETTERNAS_NODE_TOKEN_PATH")
	if err != nil {
		return bootstrapResult{}, err
	}

	machineID, err := requiredEnv("BETTERNAS_NODE_MACHINE_ID")
	if err != nil {
		return bootstrapResult{}, err
	}

	displayName := strings.TrimSpace(env("BETTERNAS_NODE_DISPLAY_NAME", machineID))
	if displayName == "" {
		displayName = machineID
	}

	client := &http.Client{Timeout: 5 * time.Second}
	nodeToken, err := readNodeToken(nodeTokenPath)
	if err != nil {
		return bootstrapResult{}, err
	}

	authToken := nodeToken
	if authToken == "" {
		authToken = bootstrapToken
	}

	registration, issuedNodeToken, err := registerNodeWithControlPlane(client, controlPlaneURL, authToken, nodeRegistrationRequest{
		MachineID:     machineID,
		DisplayName:   displayName,
		AgentVersion:  env("BETTERNAS_VERSION", "0.1.0-dev"),
		DirectAddress: optionalEnvPointer("BETTERNAS_NODE_DIRECT_ADDRESS"),
		RelayAddress:  optionalEnvPointer("BETTERNAS_NODE_RELAY_ADDRESS"),
	})
	if err != nil {
		return bootstrapResult{}, err
	}

	if strings.TrimSpace(issuedNodeToken) != "" {
		if err := writeNodeToken(nodeTokenPath, issuedNodeToken); err != nil {
			return bootstrapResult{}, err
		}
		authToken = issuedNodeToken
	}

	if err := syncNodeExportsWithControlPlane(client, controlPlaneURL, authToken, registration.ID, buildStorageExportInputs(exportPaths)); err != nil {
		return bootstrapResult{}, err
	}
	if err := sendNodeHeartbeat(client, controlPlaneURL, authToken, registration.ID); err != nil {
		return bootstrapResult{}, err
	}

	return bootstrapResult{nodeID: registration.ID}, nil
}

func registerNodeWithControlPlane(client *http.Client, baseURL string, token string, payload nodeRegistrationRequest) (nodeRegistrationResponse, string, error) {
	response, err := doControlPlaneJSONRequest(client, http.MethodPost, controlPlaneEndpoint(baseURL, "/api/v1/nodes/register"), token, payload)
	if err != nil {
		return nodeRegistrationResponse{}, "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nodeRegistrationResponse{}, "", controlPlaneResponseError("register node", response)
	}

	var registration nodeRegistrationResponse
	if err := json.NewDecoder(response.Body).Decode(&registration); err != nil {
		return nodeRegistrationResponse{}, "", fmt.Errorf("decode register node response: %w", err)
	}

	return registration, strings.TrimSpace(response.Header.Get(controlPlaneNodeTokenHeader)), nil
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
	request.Header.Set("Authorization", "Bearer "+token)

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

func readNodeToken(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}

		return "", fmt.Errorf("read node token %s: %w", path, err)
	}

	return strings.TrimSpace(string(data)), nil
}

func writeNodeToken(path string, token string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create node token directory %s: %w", filepath.Dir(path), err)
	}

	tempFile, err := os.CreateTemp(filepath.Dir(path), ".node-token-*.tmp")
	if err != nil {
		return fmt.Errorf("create node token temp file in %s: %w", filepath.Dir(path), err)
	}

	tempFilePath := tempFile.Name()
	cleanupTempFile := true
	defer func() {
		if cleanupTempFile {
			_ = os.Remove(tempFilePath)
		}
	}()

	if err := tempFile.Chmod(0o600); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("chmod node token temp file %s: %w", tempFilePath, err)
	}
	if _, err := tempFile.WriteString(strings.TrimSpace(token) + "\n"); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("write node token temp file %s: %w", tempFilePath, err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close node token temp file %s: %w", tempFilePath, err)
	}
	if err := os.Rename(tempFilePath, path); err != nil {
		return fmt.Errorf("replace node token %s: %w", path, err)
	}

	cleanupTempFile = false
	return nil
}
