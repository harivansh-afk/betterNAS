export const CONTROL_PLANE_ROUTES = {
  health: "/health",
  version: "/version",
} as const;

export interface NextcloudBackendStatus {
  configured: boolean;
  baseUrl: string;
  provider: "nextcloud";
}

export interface ControlPlaneHealthResponse {
  service: "control-plane";
  status: "ok";
  timestamp: string;
  uptimeSeconds: number;
  nextcloud: NextcloudBackendStatus;
}

export interface ControlPlaneVersionResponse {
  service: "control-plane";
  version: string;
  apiVersion: "v1";
}
