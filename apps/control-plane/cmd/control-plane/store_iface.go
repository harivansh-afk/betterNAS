package main

import "time"

// store defines the persistence interface for the control-plane.
type store interface {
	// Node management
	registerNode(ownerID string, request nodeRegistrationRequest, registeredAt time.Time) (nodeRegistrationResult, error)
	upsertExports(nodeID string, ownerID string, request nodeExportsRequest) ([]storageExport, error)
	recordHeartbeat(nodeID string, ownerID string, request nodeHeartbeatRequest) error
	listExports(ownerID string) []storageExport
	listNodes(ownerID string) []nasNode
	exportContext(exportID string, ownerID string) (exportContext, bool)
	nodeByID(nodeID string) (nasNode, bool)

	// User auth
	createUser(username string, password string) (user, error)
	authenticateUser(username string, password string) (user, error)
	createSession(userID string, ttl time.Duration) (string, error)
	validateSession(token string) (user, error)
	deleteSession(token string) error
}
