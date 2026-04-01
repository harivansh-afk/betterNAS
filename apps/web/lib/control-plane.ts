import { readFile } from "node:fs/promises";
import path from "node:path";
import { cache } from "react";

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

export interface ControlPlaneConfig {
  baseUrl: string | null;
  clientToken: string | null;
}

export class ControlPlaneConfigurationError extends Error {
  constructor() {
    super(
      "Control-plane configuration is missing. Set BETTERNAS_CONTROL_PLANE_URL and BETTERNAS_CONTROL_PLANE_CLIENT_TOKEN, or provide them through .env.agent.",
    );
  }
}

export class ControlPlaneRequestError extends Error {
  constructor(message: string) {
    super(message);
  }
}

const readAgentEnvFile = cache(async (): Promise<Record<string, string>> => {
  const candidatePaths = [
    path.resolve(/* turbopackIgnore: true */ process.cwd(), ".env.agent"),
    path.resolve(/* turbopackIgnore: true */ process.cwd(), "../../.env.agent"),
  ];

  for (const candidatePath of candidatePaths) {
    try {
      const raw = await readFile(candidatePath, "utf8");
      return parseEnvLikeFile(raw);
    } catch (error) {
      const nodeError = error as NodeJS.ErrnoException;
      if (nodeError.code === "ENOENT") {
        continue;
      }

      throw error;
    }
  }

  return {};
});

function parseEnvLikeFile(raw: string): Record<string, string> {
  return raw.split(/\r?\n/).reduce<Record<string, string>>((env, line) => {
    const trimmedLine = line.trim();
    if (trimmedLine === "" || trimmedLine.startsWith("#")) {
      return env;
    }

    const separatorIndex = trimmedLine.indexOf("=");
    if (separatorIndex === -1) {
      return env;
    }

    const key = trimmedLine.slice(0, separatorIndex).trim();
    const value = trimmedLine.slice(separatorIndex + 1).trim();
    env[key] = unwrapEnvValue(value);
    return env;
  }, {});
}

function unwrapEnvValue(value: string): string {
  if (
    (value.startsWith('"') && value.endsWith('"')) ||
    (value.startsWith("'") && value.endsWith("'"))
  ) {
    return value.slice(1, -1);
  }

  return value;
}

export async function getControlPlaneConfig(): Promise<ControlPlaneConfig> {
  const agentEnv = await readAgentEnvFile();
  const cloneName = firstDefinedValue(
    process.env.BETTERNAS_CLONE_NAME,
    agentEnv.BETTERNAS_CLONE_NAME,
  );

  const clientToken = firstDefinedValue(
    process.env.BETTERNAS_CONTROL_PLANE_CLIENT_TOKEN,
    agentEnv.BETTERNAS_CONTROL_PLANE_CLIENT_TOKEN,
    cloneName === null ? undefined : `${cloneName}-local-client-token`,
  );

  const directBaseUrl = firstDefinedValue(
    process.env.BETTERNAS_CONTROL_PLANE_URL,
    agentEnv.BETTERNAS_CONTROL_PLANE_URL,
  );
  if (directBaseUrl !== null) {
    return {
      baseUrl: trimTrailingSlash(directBaseUrl),
      clientToken,
    };
  }

  const controlPlanePort = firstDefinedValue(
    process.env.BETTERNAS_CONTROL_PLANE_PORT,
    agentEnv.BETTERNAS_CONTROL_PLANE_PORT,
  );

  return {
    baseUrl:
      controlPlanePort === null
        ? null
        : trimTrailingSlash(`http://localhost:${controlPlanePort}`),
    clientToken,
  };
}

export async function listExports(): Promise<StorageExport[]> {
  return controlPlaneRequest<StorageExport[]>("/api/v1/exports");
}

export async function issueMountProfile(
  exportId: string,
): Promise<MountProfile> {
  return controlPlaneRequest<MountProfile>("/api/v1/mount-profiles/issue", {
    method: "POST",
    body: JSON.stringify({ exportId }),
  });
}

async function controlPlaneRequest<T>(
  requestPath: string,
  init?: RequestInit,
): Promise<T> {
  const config = await getControlPlaneConfig();
  if (config.baseUrl === null || config.clientToken === null) {
    throw new ControlPlaneConfigurationError();
  }

  const headers = new Headers(init?.headers);
  headers.set("Authorization", `Bearer ${config.clientToken}`);
  if (init?.body !== undefined) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`${config.baseUrl}${requestPath}`, {
    ...init,
    headers,
    cache: "no-store",
  });

  if (!response.ok) {
    const responseBody = await response.text();
    throw new ControlPlaneRequestError(
      `Control-plane request failed for ${requestPath} with status ${response.status}: ${responseBody || response.statusText}`,
    );
  }

  return (await response.json()) as T;
}

function firstDefinedValue(
  ...values: Array<string | undefined>
): string | null {
  for (const value of values) {
    const trimmedValue = value?.trim();
    if (trimmedValue) {
      return trimmedValue;
    }
  }

  return null;
}

function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, "");
}
