package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type storeState struct {
	NextNodeOrdinal     int                          `json:"nextNodeOrdinal"`
	NextExportOrdinal   int                          `json:"nextExportOrdinal"`
	NodeIDByMachineID   map[string]string            `json:"nodeIdByMachineId"`
	NodesByID           map[string]nasNode           `json:"nodesById"`
	NodeTokenHashByID   map[string]string            `json:"nodeTokenHashById"`
	ExportIDsByNodePath map[string]map[string]string `json:"exportIdsByNodePath"`
	ExportsByID         map[string]storageExport     `json:"exportsById"`
}

type memoryStore struct {
	mu        sync.RWMutex
	statePath string
	state     storeState
}

type nodeRegistrationResult struct {
	Node            nasNode
	IssuedNodeToken string
}

type nodeAuthState struct {
	NodeID    string
	TokenHash string
}

func newMemoryStore(statePath string) (*memoryStore, error) {
	store := &memoryStore{
		statePath: statePath,
		state:     newDefaultStoreState(),
	}

	if statePath == "" {
		return store, nil
	}

	loadedState, err := loadStoreState(statePath)
	if err != nil {
		return nil, err
	}

	store.state = loadedState
	return store, nil
}

func newDefaultStoreState() storeState {
	return storeState{
		NextNodeOrdinal:     1,
		NextExportOrdinal:   1,
		NodeIDByMachineID:   make(map[string]string),
		NodesByID:           make(map[string]nasNode),
		NodeTokenHashByID:   make(map[string]string),
		ExportIDsByNodePath: make(map[string]map[string]string),
		ExportsByID:         make(map[string]storageExport),
	}
}

func loadStoreState(statePath string) (storeState, error) {
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return newDefaultStoreState(), nil
		}

		return storeState{}, fmt.Errorf("read control-plane state %s: %w", statePath, err)
	}

	var state storeState
	if err := json.Unmarshal(data, &state); err != nil {
		return storeState{}, fmt.Errorf("decode control-plane state %s: %w", statePath, err)
	}

	return normalizeStoreState(state), nil
}

func normalizeStoreState(state storeState) storeState {
	if state.NextNodeOrdinal < 1 {
		state.NextNodeOrdinal = len(state.NodesByID) + 1
	}
	if state.NextExportOrdinal < 1 {
		state.NextExportOrdinal = len(state.ExportsByID) + 1
	}
	if state.NodeIDByMachineID == nil {
		state.NodeIDByMachineID = make(map[string]string)
	}
	if state.NodesByID == nil {
		state.NodesByID = make(map[string]nasNode)
	}
	if state.NodeTokenHashByID == nil {
		state.NodeTokenHashByID = make(map[string]string)
	}
	if state.ExportIDsByNodePath == nil {
		state.ExportIDsByNodePath = make(map[string]map[string]string)
	}
	if state.ExportsByID == nil {
		state.ExportsByID = make(map[string]storageExport)
	}

	return cloneStoreState(state)
}

func cloneStoreState(state storeState) storeState {
	cloned := storeState{
		NextNodeOrdinal:     state.NextNodeOrdinal,
		NextExportOrdinal:   state.NextExportOrdinal,
		NodeIDByMachineID:   make(map[string]string, len(state.NodeIDByMachineID)),
		NodesByID:           make(map[string]nasNode, len(state.NodesByID)),
		NodeTokenHashByID:   make(map[string]string, len(state.NodeTokenHashByID)),
		ExportIDsByNodePath: make(map[string]map[string]string, len(state.ExportIDsByNodePath)),
		ExportsByID:         make(map[string]storageExport, len(state.ExportsByID)),
	}

	for machineID, nodeID := range state.NodeIDByMachineID {
		cloned.NodeIDByMachineID[machineID] = nodeID
	}

	for nodeID, node := range state.NodesByID {
		cloned.NodesByID[nodeID] = copyNasNode(node)
	}

	for nodeID, tokenHash := range state.NodeTokenHashByID {
		cloned.NodeTokenHashByID[nodeID] = tokenHash
	}

	for nodeID, exportIDsByPath := range state.ExportIDsByNodePath {
		clonedExportIDsByPath := make(map[string]string, len(exportIDsByPath))
		for exportPath, exportID := range exportIDsByPath {
			clonedExportIDsByPath[exportPath] = exportID
		}
		cloned.ExportIDsByNodePath[nodeID] = clonedExportIDsByPath
	}

	for exportID, export := range state.ExportsByID {
		cloned.ExportsByID[exportID] = copyStorageExport(export)
	}

	return cloned
}

func (s *memoryStore) registerNode(request nodeRegistrationRequest, registeredAt time.Time) (nodeRegistrationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	nextState := cloneStoreState(s.state)
	result, err := registerNodeInState(&nextState, request, registeredAt)
	if err != nil {
		return nodeRegistrationResult{}, err
	}
	if err := s.persistLocked(nextState); err != nil {
		return nodeRegistrationResult{}, err
	}

	s.state = nextState
	return result, nil
}

func registerNodeInState(state *storeState, request nodeRegistrationRequest, registeredAt time.Time) (nodeRegistrationResult, error) {
	nodeID, ok := state.NodeIDByMachineID[request.MachineID]
	if !ok {
		nodeID = nextNodeID(state)
		state.NodeIDByMachineID[request.MachineID] = nodeID
	}

	issuedNodeToken := ""
	if stringsTrimmedEmpty(state.NodeTokenHashByID[nodeID]) {
		nodeToken, err := newOpaqueToken()
		if err != nil {
			return nodeRegistrationResult{}, err
		}
		state.NodeTokenHashByID[nodeID] = hashOpaqueToken(nodeToken)
		issuedNodeToken = nodeToken
	}

	node := nasNode{
		ID:            nodeID,
		MachineID:     request.MachineID,
		DisplayName:   request.DisplayName,
		AgentVersion:  request.AgentVersion,
		Status:        "online",
		LastSeenAt:    registeredAt.UTC().Format(time.RFC3339),
		DirectAddress: copyStringPointer(request.DirectAddress),
		RelayAddress:  copyStringPointer(request.RelayAddress),
	}

	exportIDsByPath, ok := state.ExportIDsByNodePath[nodeID]
	if !ok {
		exportIDsByPath = make(map[string]string)
		state.ExportIDsByNodePath[nodeID] = exportIDsByPath
	}

	keepPaths := make(map[string]struct{}, len(request.Exports))
	for _, export := range request.Exports {
		exportID, ok := exportIDsByPath[export.Path]
		if !ok {
			exportID = nextExportID(state)
			exportIDsByPath[export.Path] = exportID
		}

		state.ExportsByID[exportID] = storageExport{
			ID:            exportID,
			NasNodeID:     nodeID,
			Label:         export.Label,
			Path:          export.Path,
			MountPath:     export.MountPath,
			Protocols:     copyStringSlice(export.Protocols),
			CapacityBytes: copyInt64Pointer(export.CapacityBytes),
			Tags:          copyStringSlice(export.Tags),
		}
		keepPaths[export.Path] = struct{}{}
	}

	for exportPath, exportID := range exportIDsByPath {
		if _, ok := keepPaths[exportPath]; ok {
			continue
		}

		delete(exportIDsByPath, exportPath)
		delete(state.ExportsByID, exportID)
	}

	state.NodesByID[nodeID] = node
	return nodeRegistrationResult{
		Node:            node,
		IssuedNodeToken: issuedNodeToken,
	}, nil
}

func (s *memoryStore) recordHeartbeat(nodeID string, request nodeHeartbeatRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	nextState := cloneStoreState(s.state)
	if err := recordHeartbeatInState(&nextState, nodeID, request); err != nil {
		return err
	}
	if err := s.persistLocked(nextState); err != nil {
		return err
	}

	s.state = nextState
	return nil
}

func recordHeartbeatInState(state *storeState, nodeID string, request nodeHeartbeatRequest) error {
	node, ok := state.NodesByID[nodeID]
	if !ok {
		return errNodeNotFound
	}

	node.Status = request.Status
	node.LastSeenAt = request.LastSeenAt
	state.NodesByID[nodeID] = node

	return nil
}

func (s *memoryStore) listExports() []storageExport {
	s.mu.RLock()
	defer s.mu.RUnlock()

	exports := make([]storageExport, 0, len(s.state.ExportsByID))
	for _, export := range s.state.ExportsByID {
		exports = append(exports, copyStorageExport(export))
	}

	sort.Slice(exports, func(i, j int) bool {
		return exports[i].ID < exports[j].ID
	})

	return exports
}

func (s *memoryStore) exportContext(exportID string) (exportContext, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	export, ok := s.state.ExportsByID[exportID]
	if !ok {
		return exportContext{}, false
	}

	node, ok := s.state.NodesByID[export.NasNodeID]
	if !ok {
		return exportContext{}, false
	}

	return exportContext{
		export: copyStorageExport(export),
		node:   copyNasNode(node),
	}, true
}

func (s *memoryStore) nodeByID(nodeID string) (nasNode, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	node, ok := s.state.NodesByID[nodeID]
	if !ok {
		return nasNode{}, false
	}

	return copyNasNode(node), true
}

func (s *memoryStore) nodeAuthByMachineID(machineID string) (nodeAuthState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodeID, ok := s.state.NodeIDByMachineID[machineID]
	if !ok {
		return nodeAuthState{}, false
	}

	return nodeAuthState{
		NodeID:    nodeID,
		TokenHash: s.state.NodeTokenHashByID[nodeID],
	}, true
}

func (s *memoryStore) nodeAuthByID(nodeID string) (nodeAuthState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.state.NodesByID[nodeID]; !ok {
		return nodeAuthState{}, false
	}

	return nodeAuthState{
		NodeID:    nodeID,
		TokenHash: s.state.NodeTokenHashByID[nodeID],
	}, true
}

func (s *memoryStore) persistLocked(state storeState) error {
	if s.statePath == "" {
		return nil
	}

	return saveStoreState(s.statePath, state)
}

func saveStoreState(statePath string, state storeState) error {
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode control-plane state %s: %w", statePath, err)
	}
	payload = append(payload, '\n')

	stateDir := filepath.Dir(statePath)
	if err := os.MkdirAll(stateDir, 0o750); err != nil {
		return fmt.Errorf("create control-plane state directory %s: %w", stateDir, err)
	}

	tempFile, err := os.CreateTemp(stateDir, ".control-plane-state-*.tmp")
	if err != nil {
		return fmt.Errorf("create control-plane state temp file in %s: %w", stateDir, err)
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
		return fmt.Errorf("chmod control-plane state temp file %s: %w", tempFilePath, err)
	}
	if _, err := tempFile.Write(payload); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("write control-plane state temp file %s: %w", tempFilePath, err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close control-plane state temp file %s: %w", tempFilePath, err)
	}
	if err := os.Rename(tempFilePath, statePath); err != nil {
		return fmt.Errorf("replace control-plane state %s: %w", statePath, err)
	}

	cleanupTempFile = false
	return nil
}

func nextNodeID(state *storeState) string {
	ordinal := state.NextNodeOrdinal
	state.NextNodeOrdinal++

	if ordinal == 1 {
		return "dev-node"
	}

	return fmt.Sprintf("dev-node-%d", ordinal)
}

func nextExportID(state *storeState) string {
	ordinal := state.NextExportOrdinal
	state.NextExportOrdinal++

	if ordinal == 1 {
		return "dev-export"
	}

	return fmt.Sprintf("dev-export-%d", ordinal)
}

func copyNasNode(node nasNode) nasNode {
	return nasNode{
		ID:            node.ID,
		MachineID:     node.MachineID,
		DisplayName:   node.DisplayName,
		AgentVersion:  node.AgentVersion,
		Status:        node.Status,
		LastSeenAt:    node.LastSeenAt,
		DirectAddress: copyStringPointer(node.DirectAddress),
		RelayAddress:  copyStringPointer(node.RelayAddress),
	}
}

func copyStorageExport(export storageExport) storageExport {
	return storageExport{
		ID:            export.ID,
		NasNodeID:     export.NasNodeID,
		Label:         export.Label,
		Path:          export.Path,
		MountPath:     export.MountPath,
		Protocols:     copyStringSlice(export.Protocols),
		CapacityBytes: copyInt64Pointer(export.CapacityBytes),
		Tags:          copyStringSlice(export.Tags),
	}
}

func newOpaqueToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate node token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashOpaqueToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func stringsTrimmedEmpty(value string) bool {
	return len(value) == 0
}
