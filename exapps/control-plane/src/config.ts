export interface ControlPlaneConfig {
  port: number;
  version: string;
  nextcloudBaseUrl: string;
}

export function loadConfig(env: NodeJS.ProcessEnv = process.env): ControlPlaneConfig {
  const portValue = env.PORT ?? "3000";
  const port = Number.parseInt(portValue, 10);

  if (Number.isNaN(port)) {
    throw new Error(`Invalid PORT value: ${portValue}`);
  }

  return {
    port,
    version: env.AINAS_VERSION ?? "0.1.0-dev",
    nextcloudBaseUrl: normalizeBaseUrl(env.NEXTCLOUD_BASE_URL ?? "http://nextcloud")
  };
}

function normalizeBaseUrl(url: string): string {
  return url.replace(/\/+$/, "");
}

