package nodeagent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	registerNodeRoute   = "/api/v1/nodes/register"
	controlPlaneTimeout = 10 * time.Second
)

type registrationLoop struct {
	cfg                  Config
	logger               *log.Logger
	client               *http.Client
	nodeID               string
	heartbeatUnsupported bool
}

type nodeRegistrationRequest struct {
	MachineID     string               `json:"machineId"`
	DisplayName   string               `json:"displayName"`
	AgentVersion  string               `json:"agentVersion"`
	DirectAddress *string              `json:"directAddress"`
	RelayAddress  *string              `json:"relayAddress"`
	Exports       []storageExportInput `json:"exports"`
}

type storageExportInput struct {
	Label         string   `json:"label"`
	Path          string   `json:"path"`
	Protocols     []string `json:"protocols"`
	CapacityBytes *int64   `json:"capacityBytes"`
	Tags          []string `json:"tags"`
}

type nodeRegistrationResponse struct {
	ID string `json:"id"`
}

type nodeHeartbeatRequest struct {
	NodeID     string `json:"nodeId"`
	Status     string `json:"status"`
	LastSeenAt string `json:"lastSeenAt"`
}

type responseStatusError struct {
	route      string
	statusCode int
	message    string
}

func (e *responseStatusError) Error() string {
	return fmt.Sprintf("%s returned %d: %s", e.route, e.statusCode, e.message)
}

func newRegistrationLoop(cfg Config, logger *log.Logger) *registrationLoop {
	return &registrationLoop{
		cfg:    cfg,
		logger: logger,
		client: &http.Client{Timeout: controlPlaneTimeout},
	}
}

func (r *registrationLoop) Run(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		r.syncOnce(ctx)
		if r.nodeID != "" && (!r.cfg.HeartbeatEnabled || r.heartbeatUnsupported) {
			return
		}

		timer.Reset(r.cfg.HeartbeatInterval)
	}
}

func (r *registrationLoop) syncOnce(ctx context.Context) {
	if r.nodeID == "" {
		if err := r.registerAndStore(ctx, "betterNAS node agent registered as %s"); err != nil {
			r.logger.Printf("betterNAS node agent registration failed: %v", err)
			return
		}
	}

	if !r.cfg.HeartbeatEnabled {
		return
	}

	if err := r.sendHeartbeat(ctx); err != nil {
		if heartbeatRouteUnsupported(err) {
			r.heartbeatUnsupported = true
			r.logger.Printf("betterNAS node agent heartbeat route is unavailable; stopping heartbeats: %v", err)
			return
		}
		if heartbeatRequiresRegistrationRefresh(err) {
			if err := r.recoverFromRejectedHeartbeat(ctx, err); err != nil {
				r.logger.Printf("betterNAS node agent %v", err)
			}
			return
		}

		r.logger.Printf("betterNAS node agent heartbeat failed: %v", err)
	}
}

func (r *registrationLoop) registerAndStore(ctx context.Context, message string) error {
	nodeID, err := r.register(ctx)
	if err != nil {
		return err
	}

	r.nodeID = nodeID
	if strings.TrimSpace(message) != "" {
		r.logger.Printf(message, r.nodeID)
	}

	return nil
}

func (r *registrationLoop) recoverFromRejectedHeartbeat(ctx context.Context, heartbeatErr error) error {
	rejectedNodeID := r.nodeID
	r.logger.Printf("betterNAS node agent heartbeat was rejected for %s; re-registering: %v", rejectedNodeID, heartbeatErr)
	r.nodeID = ""

	if err := r.registerAndStore(ctx, "betterNAS node agent re-registered as %s after heartbeat rejection"); err != nil {
		return fmt.Errorf("failed to re-register after heartbeat rejection: %w", err)
	}

	if err := r.sendHeartbeat(ctx); err != nil {
		if heartbeatRouteUnsupported(err) || heartbeatRequiresRegistrationRefresh(err) {
			r.heartbeatUnsupported = true
			return fmt.Errorf("heartbeat route did not accept the freshly registered node; stopping heartbeats: %w", err)
		}

		return fmt.Errorf("heartbeat failed after re-registration: %w", err)
	}

	return nil
}

func (r *registrationLoop) register(ctx context.Context) (string, error) {
	request := r.registrationRequest()

	var response nodeRegistrationResponse
	if err := r.postJSON(ctx, registerNodeRoute, request, http.StatusOK, &response); err != nil {
		return "", err
	}

	if strings.TrimSpace(response.ID) == "" {
		return "", fmt.Errorf("register response did not include a node id")
	}

	return response.ID, nil
}

func (r *registrationLoop) registrationRequest() nodeRegistrationRequest {
	machineID := strings.TrimSpace(r.cfg.MachineID)
	displayName := strings.TrimSpace(r.cfg.DisplayName)
	if displayName == "" {
		displayName = machineID
	}

	agentVersion := strings.TrimSpace(r.cfg.AgentVersion)
	if agentVersion == "" {
		agentVersion = defaultAgentVersion
	}

	exportLabel := strings.TrimSpace(r.cfg.ExportLabel)
	if exportLabel == "" {
		exportLabel = defaultExportLabel(r.cfg.ExportPath)
	}

	return nodeRegistrationRequest{
		MachineID:     machineID,
		DisplayName:   displayName,
		AgentVersion:  agentVersion,
		DirectAddress: optionalString(r.cfg.DirectAddress),
		RelayAddress:  optionalString(r.cfg.RelayAddress),
		Exports: []storageExportInput{
			{
				Label:         exportLabel,
				Path:          r.cfg.ExportPath,
				Protocols:     []string{"webdav"},
				CapacityBytes: detectCapacityBytes(r.cfg.ExportPath),
				Tags:          cloneStringSlice(r.cfg.ExportTags),
			},
		},
	}
}

func (r *registrationLoop) sendHeartbeat(ctx context.Context) error {
	request := nodeHeartbeatRequest{
		NodeID:     r.nodeID,
		Status:     "online",
		LastSeenAt: time.Now().UTC().Format(time.RFC3339),
	}

	return r.postJSON(ctx, heartbeatRoute(r.nodeID), request, http.StatusNoContent, nil)
}

func heartbeatRoute(nodeID string) string {
	return "/api/v1/nodes/" + url.PathEscape(nodeID) + "/heartbeat"
}

func (r *registrationLoop) postJSON(ctx context.Context, route string, payload any, wantStatus int, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s payload: %w", route, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, r.cfg.ControlPlaneURL+route, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create %s request: %w", route, err)
	}

	request.Header.Set("Content-Type", "application/json")
	if token := strings.TrimSpace(r.cfg.ControlPlaneToken); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := r.client.Do(request)
	if err != nil {
		return fmt.Errorf("post %s: %w", route, err)
	}
	defer response.Body.Close()

	if response.StatusCode != wantStatus {
		message, readErr := io.ReadAll(io.LimitReader(response.Body, 4*1024))
		if readErr != nil {
			return fmt.Errorf("%s returned %d and body read failed: %w", route, response.StatusCode, readErr)
		}

		return &responseStatusError{
			route:      route,
			statusCode: response.StatusCode,
			message:    strings.TrimSpace(string(message)),
		}
	}

	if out == nil {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil
	}

	if err := json.NewDecoder(response.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s response: %w", route, err)
	}

	return nil
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func cloneStringSlice(values []string) []string {
	return append([]string{}, values...)
}

func heartbeatRouteUnsupported(err error) bool {
	var statusErr *responseStatusError
	if !errors.As(err, &statusErr) {
		return false
	}

	switch statusErr.statusCode {
	case http.StatusMethodNotAllowed, http.StatusNotImplemented:
		return true
	default:
		return false
	}
}

func heartbeatRequiresRegistrationRefresh(err error) bool {
	var statusErr *responseStatusError
	if !errors.As(err, &statusErr) {
		return false
	}

	switch statusErr.statusCode {
	case http.StatusNotFound, http.StatusGone:
		return true
	default:
		return false
	}
}
