import { createServer, type IncomingMessage, type ServerResponse } from "node:http";
import {
  CONTROL_PLANE_ROUTES,
  type ControlPlaneHealthResponse,
  type ControlPlaneVersionResponse
} from "@betternas/contracts";

import type { ControlPlaneConfig } from "./config.js";
import { NextcloudBackendAdapter } from "./adapters/nextcloud-backend.js";

export function createApp(config: ControlPlaneConfig) {
  const startedAt = Date.now();
  const nextcloudBackend = new NextcloudBackendAdapter(config.nextcloudBaseUrl);

  return createServer((request, response) => {
    if (request.method !== "GET" || !request.url) {
      writeJson(response, 405, { error: "Method not allowed" });
      return;
    }

    const url = new URL(request.url, "http://localhost");

    if (url.pathname === CONTROL_PLANE_ROUTES.health) {
      const payload: ControlPlaneHealthResponse = {
        service: "control-plane",
        status: "ok",
        timestamp: new Date().toISOString(),
        uptimeSeconds: Math.floor((Date.now() - startedAt) / 1000),
        nextcloud: nextcloudBackend.describe()
      };

      writeJson(response, 200, payload);
      return;
    }

    if (url.pathname === CONTROL_PLANE_ROUTES.version) {
      const payload: ControlPlaneVersionResponse = {
        service: "control-plane",
        version: config.version,
        apiVersion: "v1"
      };

      writeJson(response, 200, payload);
      return;
    }

    writeJson(response, 404, {
      error: "Not found"
    });
  });
}

function writeJson(response: ServerResponse<IncomingMessage>, statusCode: number, payload: unknown) {
  response.writeHead(statusCode, {
    "content-type": "application/json; charset=utf-8"
  });
  response.end(JSON.stringify(payload));
}

