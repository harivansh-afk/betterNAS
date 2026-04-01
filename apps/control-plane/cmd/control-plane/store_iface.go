package main

import "time"

// store defines the persistence interface for the control-plane.
type store interface {
	// Node management
	registerNode(request nodeRegistrationRequest, registeredAt time.Time) (nodeRegistrationResult, error)
	upsertExports(nodeID string, request nodeExportsRequest) ([]storageExport, error)
	recordHeartbeat(nodeID string, request nodeHeartbeatRequest) error
	listExports() []storageExport
	exportContext(exportID string) (exportContext, bool)
	nodeByID(nodeID string) (nasNode, bool)
	nodeAuthByMachineID(machineID string) (nodeAuthState, bool)
	nodeAuthByID(nodeID string) (nodeAuthState, bool)

	// User auth
	createUser(username string, password string) (user, error)
	authenticateUser(username string, password string) (user, error)
	createSession(userID string, ttl time.Duration) (string, error)
	validateSession(token string) (user, error)
	deleteSession(token string) error
}
