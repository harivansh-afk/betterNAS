export const FOUNDATION_API_ROUTES = {
  registerNode: "/api/v1/nodes/register",
  nodeHeartbeat: "/api/v1/nodes/:nodeId/heartbeat",
  nodeExports: "/api/v1/nodes/:nodeId/exports",
  listExports: "/api/v1/exports",
  issueMountProfile: "/api/v1/mount-profiles/issue",
  issueCloudProfile: "/api/v1/cloud-profiles/issue",
} as const;

export const FOUNDATION_API_HEADERS = {
  authorization: "Authorization",
  nodeToken: "X-BetterNAS-Node-Token",
} as const;

export type NasNodeStatus = "online" | "offline" | "degraded";
export type StorageAccessProtocol = "webdav";
export type AccessMode = "mount" | "cloud";
export type AccessPrincipalType = "user" | "device";
export type MountCredentialMode = "basic-auth";
export type CloudProvider = "nextcloud";

export interface NasNode {
  id: string;
  machineId: string;
  displayName: string;
  agentVersion: string;
  status: NasNodeStatus;
  lastSeenAt: string;
  directAddress: string | null;
  relayAddress: string | null;
}

export interface StorageExport {
  id: string;
  nasNodeId: string;
  label: string;
  path: string;
  mountPath?: string;
  protocols: StorageAccessProtocol[];
  capacityBytes: number | null;
  tags: string[];
}

export interface AccessGrant {
  id: string;
  exportId: string;
  principalType: AccessPrincipalType;
  principalId: string;
  modes: AccessMode[];
  readonly: boolean;
}

export interface MountProfile {
  id: string;
  exportId: string;
  protocol: "webdav";
  displayName: string;
  mountUrl: string;
  readonly: boolean;
  credential: MountCredential;
}

export interface MountCredential {
  mode: MountCredentialMode;
  username: string;
  password: string;
  expiresAt: string;
}

export interface CloudProfile {
  id: string;
  exportId: string;
  provider: CloudProvider;
  baseUrl: string;
  path: string;
}

export interface StorageExportInput {
  label: string;
  path: string;
  mountPath?: string;
  protocols: StorageAccessProtocol[];
  capacityBytes: number | null;
  tags: string[];
}

export interface NodeRegistrationRequest {
  machineId: string;
  displayName: string;
  agentVersion: string;
  directAddress: string | null;
  relayAddress: string | null;
}

export interface NodeExportsRequest {
  exports: StorageExportInput[];
}

export interface NodeHeartbeatRequest {
  nodeId: string;
  status: NasNodeStatus;
  lastSeenAt: string;
}

export interface MountProfileRequest {
  exportId: string;
}

export interface CloudProfileRequest {
  userId: string;
  exportId: string;
  provider: CloudProvider;
}
