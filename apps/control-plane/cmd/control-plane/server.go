package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

var (
	errCloudProfileUnavailable = errors.New("nextcloud base URL is not configured")
	errExportNotFound          = errors.New("export not found")
	errMountTargetUnavailable  = errors.New("mount target is not available")
	errNodeIDMismatch          = errors.New("node id path and body must match")
	errNodeNotFound            = errors.New("node not found")
	errNodeOwnedByAnotherUser  = errors.New("node is already owned by another user")
)

const (
	authorizationHeader = "Authorization"
	bearerScheme        = "Bearer"
)

func (a *app) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.handleHealth)
	mux.HandleFunc("GET /version", a.handleVersion)
	mux.HandleFunc("POST /api/v1/auth/register", a.handleAuthRegister)
	mux.HandleFunc("POST /api/v1/auth/login", a.handleAuthLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", a.handleAuthLogout)
	mux.HandleFunc("GET /api/v1/auth/me", a.handleAuthMe)
	mux.HandleFunc("POST /api/v1/nodes/register", a.handleNodeRegister)
	mux.HandleFunc("POST /api/v1/nodes/{nodeId}/heartbeat", a.handleNodeHeartbeat)
	mux.HandleFunc("PUT /api/v1/nodes/{nodeId}/exports", a.handleNodeExports)
	mux.HandleFunc("GET /api/v1/exports", a.handleExportsList)
	mux.HandleFunc("POST /api/v1/mount-profiles/issue", a.handleMountProfileIssue)
	mux.HandleFunc("POST /api/v1/cloud-profiles/issue", a.handleCloudProfileIssue)

	var handler http.Handler = mux
	if a.config.corsOrigin != "" {
		handler = corsMiddleware(a.config.corsOrigin, handler)
	}

	return handler
}

func (a *app) handleHealth(w http.ResponseWriter, _ *http.Request) {
	now := a.now().UTC()
	writeJSON(w, http.StatusOK, controlPlaneHealthResponse{
		Service:       "control-plane",
		Status:        "ok",
		Timestamp:     now.Format(time.RFC3339),
		UptimeSeconds: int(now.Sub(a.startedAt).Seconds()),
		Nextcloud: nextcloudBackendStatus{
			Configured: hasConfiguredNextcloudBaseURL(a.config.nextcloudBaseURL),
			BaseURL:    a.config.nextcloudBaseURL,
			Provider:   "nextcloud",
		},
	})
}

func (a *app) handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, controlPlaneVersionResponse{
		Service:    "control-plane",
		Version:    a.config.version,
		APIVersion: "v1",
	})
}

func (a *app) handleNodeRegister(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := a.requireSessionUser(w, r)
	if !ok {
		return
	}

	request, err := decodeNodeRegistrationRequest(w, r)
	if err != nil {
		writeDecodeError(w, err)
		return
	}

	if err := validateNodeRegistrationRequest(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := a.store.registerNode(currentUser.ID, request, a.now())
	if err != nil {
		if errors.Is(err, errNodeOwnedByAnotherUser) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, result.Node)
}

func (a *app) handleNodeHeartbeat(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := a.requireSessionUser(w, r)
	if !ok {
		return
	}

	nodeID := r.PathValue("nodeId")

	var request nodeHeartbeatRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}

	if err := validateNodeHeartbeatRequest(nodeID, request); err != nil {
		statusCode := http.StatusBadRequest
		if errors.Is(err, errNodeNotFound) {
			statusCode = http.StatusNotFound
		}
		http.Error(w, err.Error(), statusCode)
		return
	}

	if err := a.store.recordHeartbeat(nodeID, currentUser.ID, request); err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, errNodeNotFound) {
			statusCode = http.StatusNotFound
		}
		http.Error(w, err.Error(), statusCode)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleNodeExports(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := a.requireSessionUser(w, r)
	if !ok {
		return
	}

	nodeID := r.PathValue("nodeId")

	request, err := decodeNodeExportsRequest(w, r)
	if err != nil {
		writeDecodeError(w, err)
		return
	}

	if err := validateNodeExportsRequest(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	exports, err := a.store.upsertExports(nodeID, currentUser.ID, request)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, errNodeNotFound) {
			statusCode = http.StatusNotFound
		}
		http.Error(w, err.Error(), statusCode)
		return
	}

	writeJSON(w, http.StatusOK, exports)
}

func (a *app) handleExportsList(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := a.requireSessionUser(w, r)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, a.store.listExports(currentUser.ID))
}

func (a *app) handleMountProfileIssue(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := a.requireSessionUser(w, r)
	if !ok {
		return
	}

	var request mountProfileRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}

	if err := validateMountProfileRequest(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	context, found := a.store.exportContext(request.ExportID, currentUser.ID)
	if !found {
		http.Error(w, errExportNotFound.Error(), http.StatusNotFound)
		return
	}

	mountURL, err := buildMountURL(context)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	writeJSON(w, http.StatusOK, mountProfile{
		ID:          context.export.ID,
		ExportID:    context.export.ID,
		Protocol:    "webdav",
		DisplayName: context.export.Label,
		MountURL:    mountURL,
		Readonly:    false,
		Credential:  buildAccountMountCredential(currentUser.Username),
	})
}

func (a *app) handleCloudProfileIssue(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := a.requireSessionUser(w, r)
	if !ok {
		return
	}

	var request cloudProfileRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}

	if err := validateCloudProfileRequest(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	context, found := a.store.exportContext(request.ExportID, currentUser.ID)
	if !found {
		http.Error(w, errExportNotFound.Error(), http.StatusNotFound)
		return
	}

	baseURL, err := buildCloudProfileBaseURL(a.config.nextcloudBaseURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	writeJSON(w, http.StatusOK, cloudProfile{
		ID:       fmt.Sprintf("cloud-%s-%s", currentUser.ID, context.export.ID),
		ExportID: context.export.ID,
		Provider: "nextcloud",
		BaseURL:  baseURL,
		Path:     buildCloudProfilePath(context.export.ID),
	})
}

type rawObject map[string]json.RawMessage

const maxRequestBodyBytes = 1 << 20

func decodeNodeRegistrationRequest(w http.ResponseWriter, r *http.Request) (nodeRegistrationRequest, error) {
	object, err := decodeRawObjectRequest(w, r)
	if err != nil {
		return nodeRegistrationRequest{}, err
	}
	if err := object.validateRequiredKeys(
		"machineId",
		"displayName",
		"agentVersion",
		"directAddress",
		"relayAddress",
	); err != nil {
		return nodeRegistrationRequest{}, err
	}

	request := nodeRegistrationRequest{}

	request.MachineID, err = object.stringField("machineId")
	if err != nil {
		return nodeRegistrationRequest{}, err
	}

	request.DisplayName, err = object.stringField("displayName")
	if err != nil {
		return nodeRegistrationRequest{}, err
	}

	request.AgentVersion, err = object.stringField("agentVersion")
	if err != nil {
		return nodeRegistrationRequest{}, err
	}

	request.DirectAddress, err = object.nullableStringField("directAddress")
	if err != nil {
		return nodeRegistrationRequest{}, err
	}

	request.RelayAddress, err = object.nullableStringField("relayAddress")
	if err != nil {
		return nodeRegistrationRequest{}, err
	}

	return request, nil
}

func decodeNodeExportsRequest(w http.ResponseWriter, r *http.Request) (nodeExportsRequest, error) {
	object, err := decodeRawObjectRequest(w, r)
	if err != nil {
		return nodeExportsRequest{}, err
	}
	if err := object.validateRequiredKeys("exports"); err != nil {
		return nodeExportsRequest{}, err
	}

	request := nodeExportsRequest{}
	request.Exports, err = object.storageExportInputsField("exports")
	if err != nil {
		return nodeExportsRequest{}, err
	}

	return request, nil
}

func decodeRawObjectRequest(w http.ResponseWriter, r *http.Request) (rawObject, error) {
	var object rawObject
	if err := decodeJSON(w, r, &object); err != nil {
		return nil, err
	}
	if object == nil {
		return nil, errors.New("request body must be a JSON object")
	}

	return object, nil
}

func decodeStorageExportInput(data json.RawMessage) (storageExportInput, error) {
	object, err := decodeRawObject(data)
	if err != nil {
		return storageExportInput{}, err
	}
	if err := object.validateRequiredKeys(
		"label",
		"path",
		"protocols",
		"capacityBytes",
		"tags",
	); err != nil {
		return storageExportInput{}, err
	}

	input := storageExportInput{}

	input.Label, err = object.stringField("label")
	if err != nil {
		return storageExportInput{}, err
	}

	input.Path, err = object.stringField("path")
	if err != nil {
		return storageExportInput{}, err
	}

	input.MountPath, err = object.optionalStringField("mountPath")
	if err != nil {
		return storageExportInput{}, err
	}

	input.Protocols, err = object.stringSliceField("protocols")
	if err != nil {
		return storageExportInput{}, err
	}

	input.CapacityBytes, err = object.nullableInt64Field("capacityBytes")
	if err != nil {
		return storageExportInput{}, err
	}

	input.Tags, err = object.stringSliceField("tags")
	if err != nil {
		return storageExportInput{}, err
	}

	return input, nil
}

func decodeRawObject(data json.RawMessage) (rawObject, error) {
	var object rawObject
	if err := json.Unmarshal(data, &object); err != nil {
		return nil, errors.New("must be a JSON object")
	}
	if object == nil {
		return nil, errors.New("must be a JSON object")
	}

	return object, nil
}

func (o rawObject) validateRequiredKeys(fieldNames ...string) error {
	for _, fieldName := range fieldNames {
		if _, ok := o[fieldName]; !ok {
			return fmt.Errorf("%s is required", fieldName)
		}
	}

	return nil
}

func (o rawObject) rawField(name string) (json.RawMessage, error) {
	raw, ok := o[name]
	if !ok {
		return nil, fmt.Errorf("%s is required", name)
	}

	return raw, nil
}

func (o rawObject) stringField(name string) (string, error) {
	raw, err := o.rawField(name)
	if err != nil {
		return "", err
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("%s must be a string", name)
	}

	return value, nil
}

func (o rawObject) nullableStringField(name string) (*string, error) {
	raw, err := o.rawField(name)
	if err != nil {
		return nil, err
	}
	if isJSONNull(raw) {
		return nil, nil
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%s must be a string or null", name)
	}

	return &value, nil
}

func (o rawObject) optionalStringField(name string) (string, error) {
	raw, ok := o[name]
	if !ok || isJSONNull(raw) {
		return "", nil
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("%s must be a string", name)
	}

	return value, nil
}

func (o rawObject) stringSliceField(name string) ([]string, error) {
	raw, err := o.rawField(name)
	if err != nil {
		return nil, err
	}
	if isJSONNull(raw) {
		return nil, fmt.Errorf("%s must be an array of strings", name)
	}

	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("%s must be an array of strings", name)
	}

	return values, nil
}

func (o rawObject) nullableInt64Field(name string) (*int64, error) {
	raw, err := o.rawField(name)
	if err != nil {
		return nil, err
	}
	if isJSONNull(raw) {
		return nil, nil
	}

	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%s must be an integer or null", name)
	}

	return &value, nil
}

func (o rawObject) storageExportInputsField(name string) ([]storageExportInput, error) {
	raw, err := o.rawField(name)
	if err != nil {
		return nil, err
	}
	if isJSONNull(raw) {
		return nil, fmt.Errorf("%s must be an array", name)
	}

	var rawExports []json.RawMessage
	if err := json.Unmarshal(raw, &rawExports); err != nil {
		return nil, fmt.Errorf("%s must be an array", name)
	}

	exports := make([]storageExportInput, len(rawExports))
	for index, rawExport := range rawExports {
		export, err := decodeStorageExportInput(rawExport)
		if err != nil {
			return nil, fmt.Errorf("%s[%d].%w", name, index, err)
		}
		exports[index] = export
	}

	return exports, nil
}

func isJSONNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

func validateNodeRegistrationRequest(request *nodeRegistrationRequest) error {
	request.MachineID = strings.TrimSpace(request.MachineID)
	if request.MachineID == "" {
		return errors.New("machineId is required")
	}

	request.DisplayName = strings.TrimSpace(request.DisplayName)
	if request.DisplayName == "" {
		return errors.New("displayName is required")
	}

	request.AgentVersion = strings.TrimSpace(request.AgentVersion)
	if request.AgentVersion == "" {
		return errors.New("agentVersion is required")
	}

	var err error
	request.DirectAddress, err = normalizeOptionalAbsoluteHTTPURL("directAddress", request.DirectAddress)
	if err != nil {
		return err
	}

	request.RelayAddress, err = normalizeOptionalAbsoluteHTTPURL("relayAddress", request.RelayAddress)
	if err != nil {
		return err
	}

	return nil
}

func validateNodeExportsRequest(request nodeExportsRequest) error {
	return validateStorageExportInputs(request.Exports)
}

func validateStorageExportInputs(exports []storageExportInput) error {
	seenPaths := make(map[string]struct{}, len(exports))
	seenMountPaths := make(map[string]struct{}, len(exports))
	for index := range exports {
		export := &exports[index]
		export.Label = strings.TrimSpace(export.Label)
		if export.Label == "" {
			return fmt.Errorf("exports[%d].label is required", index)
		}

		export.Path = strings.TrimSpace(export.Path)
		if export.Path == "" {
			return fmt.Errorf("exports[%d].path is required", index)
		}
		if _, ok := seenPaths[export.Path]; ok {
			return fmt.Errorf("exports[%d].path must be unique", index)
		}
		seenPaths[export.Path] = struct{}{}

		export.MountPath = strings.TrimSpace(export.MountPath)
		if len(exports) > 1 && export.MountPath == "" {
			return fmt.Errorf("exports[%d].mountPath is required when registering multiple exports", index)
		}
		if export.MountPath != "" {
			normalizedMountPath, err := normalizeAbsoluteURLPath(export.MountPath)
			if err != nil {
				return fmt.Errorf("exports[%d].mountPath %w", index, err)
			}
			export.MountPath = normalizedMountPath
			if _, ok := seenMountPaths[export.MountPath]; ok {
				return fmt.Errorf("exports[%d].mountPath must be unique", index)
			}
			seenMountPaths[export.MountPath] = struct{}{}
		}

		if len(export.Protocols) == 0 {
			return fmt.Errorf("exports[%d].protocols must not be empty", index)
		}
		for protocolIndex, protocol := range export.Protocols {
			if protocol != "webdav" {
				return fmt.Errorf("exports[%d].protocols[%d] must be webdav", index, protocolIndex)
			}
		}

		if export.CapacityBytes != nil && *export.CapacityBytes < 0 {
			return fmt.Errorf("exports[%d].capacityBytes must be greater than or equal to 0", index)
		}
	}

	return nil
}

func validateNodeHeartbeatRequest(nodeID string, request nodeHeartbeatRequest) error {
	if strings.TrimSpace(nodeID) == "" {
		return errNodeNotFound
	}
	if strings.TrimSpace(request.NodeID) == "" {
		return errors.New("nodeId is required")
	}
	if request.NodeID != nodeID {
		return errNodeIDMismatch
	}
	if request.Status != "online" && request.Status != "offline" && request.Status != "degraded" {
		return errors.New("status must be one of online, offline, or degraded")
	}
	if _, err := time.Parse(time.RFC3339, request.LastSeenAt); err != nil {
		return errors.New("lastSeenAt must be a valid RFC3339 timestamp")
	}

	return nil
}

func validateMountProfileRequest(request mountProfileRequest) error {
	if strings.TrimSpace(request.ExportID) == "" {
		return errors.New("exportId is required")
	}

	return nil
}

func validateCloudProfileRequest(request cloudProfileRequest) error {
	if strings.TrimSpace(request.UserID) == "" {
		return errors.New("userId is required")
	}
	if strings.TrimSpace(request.ExportID) == "" {
		return errors.New("exportId is required")
	}
	if request.Provider != "nextcloud" {
		return errors.New("provider must be nextcloud")
	}

	return nil
}

func normalizeAbsoluteURLPath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("must be an absolute URL path")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", errors.New("must be an absolute URL path")
	}
	if parsed.Scheme != "" || parsed.Opaque != "" || parsed.Host != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("must be an absolute URL path")
	}
	if !strings.HasPrefix(parsed.Path, "/") {
		return "", errors.New("must be an absolute URL path")
	}

	normalized := path.Clean(parsed.Path)
	if !strings.HasPrefix(normalized, "/") {
		return "", errors.New("must be an absolute URL path")
	}
	if !strings.HasSuffix(normalized, "/") {
		normalized += "/"
	}

	return normalized, nil
}

func normalizeOptionalAbsoluteHTTPURL(fieldName string, value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}

	normalized, err := normalizeAbsoluteHTTPURL(*value)
	if err != nil {
		return nil, fmt.Errorf("%s %w", fieldName, err)
	}

	return &normalized, nil
}

func hasConfiguredNextcloudBaseURL(baseURL string) bool {
	if strings.TrimSpace(baseURL) == "" {
		return false
	}

	_, err := normalizeAbsoluteHTTPURL(baseURL)
	return err == nil
}

func buildMountURL(context exportContext) (string, error) {
	address, ok := firstAddress(context.node.DirectAddress, context.node.RelayAddress)
	if !ok {
		return "", errMountTargetUnavailable
	}

	mountURL, err := buildAbsoluteHTTPURLWithPath(address, mountProfilePathForExport(context.export.MountPath))
	if err != nil {
		return "", errMountTargetUnavailable
	}

	return mountURL, nil
}

func buildCloudProfileBaseURL(baseURL string) (string, error) {
	if strings.TrimSpace(baseURL) == "" {
		return "", errCloudProfileUnavailable
	}

	normalized, err := normalizeAbsoluteHTTPURL(baseURL)
	if err != nil {
		return "", errCloudProfileUnavailable
	}

	return normalized, nil
}

func buildCloudProfilePath(exportID string) string {
	return cloudProfilePathForExport(exportID)
}

func firstAddress(addresses ...*string) (string, bool) {
	for _, address := range addresses {
		if address == nil {
			continue
		}

		normalized, err := normalizeAbsoluteHTTPURL(*address)
		if err == nil {
			return normalized, true
		}
	}

	return "", false
}

func buildAbsoluteHTTPURLWithPath(baseAddress string, absolutePath string) (string, error) {
	parsedBaseAddress, err := parseAbsoluteHTTPURL(baseAddress)
	if err != nil {
		return "", err
	}

	normalizedPath, err := joinAbsoluteURLPaths(parsedBaseAddress.Path, absolutePath)
	if err != nil {
		return "", err
	}

	parsedBaseAddress.Path = normalizedPath
	parsedBaseAddress.RawPath = ""
	return parsedBaseAddress.String(), nil
}

func joinAbsoluteURLPaths(basePath string, suffixPath string) (string, error) {
	if strings.TrimSpace(basePath) == "" {
		basePath = "/"
	}

	normalizedBasePath, err := normalizeAbsoluteURLPath(basePath)
	if err != nil {
		return "", err
	}

	normalizedSuffixPath, err := normalizeAbsoluteURLPath(suffixPath)
	if err != nil {
		return "", err
	}

	return normalizeAbsoluteURLPath(
		path.Join(normalizedBasePath, strings.TrimPrefix(normalizedSuffixPath, "/")),
	)
}

func normalizeAbsoluteHTTPURL(raw string) (string, error) {
	parsed, err := parseAbsoluteHTTPURL(raw)
	if err != nil {
		return "", err
	}

	return parsed.String(), nil
}

func parseAbsoluteHTTPURL(raw string) (*url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("must be null or an absolute http(s) URL")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, errors.New("must be null or an absolute http(s) URL")
	}
	if parsed.Opaque != "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("must be null or an absolute http(s) URL")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("must be null or an absolute http(s) URL without user info, query, or fragment")
	}

	return parsed, nil
}

func env(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	return value
}

func requiredEnv(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", key)
	}

	return value, nil
}

func parseRequiredDurationEnv(key string) (time.Duration, error) {
	value, err := requiredEnv(key)
	if err != nil {
		return 0, err
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", key, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be greater than 0", key)
	}

	return duration, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	defer r.Body.Close()

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(destination); err != nil {
		return err
	}

	var extraValue struct{}
	if err := decoder.Decode(&extraValue); err != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}

	return nil
}

func writeDecodeError(w http.ResponseWriter, err error) {
	var maxBytesErr *http.MaxBytesError
	statusCode := http.StatusBadRequest
	if errors.As(err, &maxBytesErr) {
		statusCode = http.StatusRequestEntityTooLarge
	}

	http.Error(w, err.Error(), statusCode)
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	if err := encoder.Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if _, err := w.Write(buffer.Bytes()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// --- auth handlers ---

type authRegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authLoginResponse struct {
	Token string `json:"token"`
	User  user   `json:"user"`
}

func (a *app) handleAuthRegister(w http.ResponseWriter, r *http.Request) {
	if !a.config.registrationEnabled {
		http.Error(w, "registration is disabled", http.StatusForbidden)
		return
	}

	var request authRegisterRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}

	username := strings.TrimSpace(request.Username)
	if len(username) < 3 || len(username) > 64 {
		http.Error(w, "username must be between 3 and 64 characters", http.StatusBadRequest)
		return
	}
	if len(request.Password) < 8 {
		http.Error(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	u, err := a.store.createUser(username, request.Password)
	if err != nil {
		if errors.Is(err, errUsernameTaken) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sessionTTL := a.config.sessionTTL
	if sessionTTL <= 0 {
		sessionTTL = 720 * time.Hour
	}
	token, err := a.store.createSession(u.ID, sessionTTL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, authLoginResponse{Token: token, User: u})
}

func (a *app) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	var request authLoginRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}

	u, err := a.store.authenticateUser(strings.TrimSpace(request.Username), request.Password)
	if err != nil {
		http.Error(w, "invalid username or password", http.StatusUnauthorized)
		return
	}

	sessionTTL := a.config.sessionTTL
	if sessionTTL <= 0 {
		sessionTTL = 720 * time.Hour
	}
	token, err := a.store.createSession(u.ID, sessionTTL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, authLoginResponse{Token: token, User: u})
}

func (a *app) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerToken(r)
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	_ = a.store.deleteSession(token)
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerToken(r)
	if !ok {
		writeUnauthorized(w)
		return
	}

	u, err := a.store.validateSession(token)
	if err != nil {
		writeUnauthorized(w)
		return
	}

	writeJSON(w, http.StatusOK, u)
}

// --- CORS ---

func corsMiddleware(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// --- session auth ---

func (a *app) requireSessionUser(w http.ResponseWriter, r *http.Request) (user, bool) {
	presentedToken, ok := bearerToken(r)
	if !ok {
		writeUnauthorized(w)
		return user{}, false
	}

	currentUser, err := a.store.validateSession(presentedToken)
	if err != nil {
		writeUnauthorized(w)
		return user{}, false
	}

	return currentUser, true
}

func bearerToken(r *http.Request) (string, bool) {
	authorization := strings.TrimSpace(r.Header.Get(authorizationHeader))
	if authorization == "" {
		return "", false
	}

	scheme, token, ok := strings.Cut(authorization, " ")
	if !ok || !strings.EqualFold(strings.TrimSpace(scheme), bearerScheme) {
		return "", false
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", false
	}

	return token, true
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", bearerScheme)
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}
