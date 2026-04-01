import type { NextcloudBackendStatus } from "@betternas/contracts";

export class NextcloudBackendAdapter {
  constructor(private readonly baseUrl: string) {}

  describe(): NextcloudBackendStatus {
    return {
      configured: this.baseUrl.length > 0,
      baseUrl: this.baseUrl,
      provider: "nextcloud"
    };
  }
}

