const API_URL = process.env.NEXT_PUBLIC_BETTERNAS_API_URL || "";

export interface NasNode {
  id: string;
  machineId: string;
  displayName: string;
  agentVersion: string;
  status: "online" | "offline" | "degraded";
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
  protocols: string[];
  capacityBytes: number | null;
  tags: string[];
}

export interface MountCredential {
  mode: "basic-auth";
  username: string;
  password: string;
  expiresAt: string;
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

export interface User {
  id: string;
  username: string;
  createdAt: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("betternas_session");
}

export function setToken(token: string): void {
  localStorage.setItem("betternas_session", token);
}

export function clearToken(): void {
  localStorage.removeItem("betternas_session");
}

export function isAuthenticated(): boolean {
  return getToken() !== null;
}

async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const headers: Record<string, string> = {};
  const token = getToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }
  if (options?.body) {
    headers["Content-Type"] = "application/json";
  }

  const response = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: {
      ...headers,
      ...Object.fromEntries(new Headers(options?.headers).entries()),
    },
  });

  if (!response.ok) {
    const text = await response.text();
    throw new ApiError(text || response.statusText, response.status);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}

export async function register(
  username: string,
  password: string,
): Promise<AuthResponse> {
  const res = await fetch(`${API_URL}/api/v1/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });

  if (!res.ok) {
    const text = await res.text();
    throw new ApiError(text || res.statusText, res.status);
  }

  const data = (await res.json()) as AuthResponse;
  setToken(data.token);
  return data;
}

export async function login(
  username: string,
  password: string,
): Promise<AuthResponse> {
  const res = await fetch(`${API_URL}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });

  if (!res.ok) {
    const text = await res.text();
    throw new ApiError(text || res.statusText, res.status);
  }

  const data = (await res.json()) as AuthResponse;
  setToken(data.token);
  return data;
}

export async function logout(): Promise<void> {
  try {
    await apiFetch("/api/v1/auth/logout", { method: "POST" });
  } finally {
    clearToken();
  }
}

export async function getMe(): Promise<User> {
  return apiFetch<User>("/api/v1/auth/me");
}

export async function listNodes(): Promise<NasNode[]> {
  return apiFetch<NasNode[]>("/api/v1/nodes");
}

export async function listExports(): Promise<StorageExport[]> {
  return apiFetch<StorageExport[]>("/api/v1/exports");
}

export async function issueMountProfile(
  exportId: string,
): Promise<MountProfile> {
  return apiFetch<MountProfile>("/api/v1/mount-profiles/issue", {
    method: "POST",
    body: JSON.stringify({ exportId }),
  });
}
