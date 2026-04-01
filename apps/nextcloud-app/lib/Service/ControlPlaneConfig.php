<?php

declare(strict_types=1);

namespace OCA\BetterNasControlplane\Service;

use OCA\BetterNasControlplane\AppInfo\Application;
use OCP\IAppConfig;

class ControlPlaneConfig {
	public function __construct(
		private readonly IAppConfig $appConfig,
	) {
	}

	public function getBaseUrl(): string {
		$environmentUrl = getenv('BETTERNAS_CONTROL_PLANE_URL');
		if (is_string($environmentUrl) && $environmentUrl !== '') {
			return rtrim($environmentUrl, '/');
		}

		$configuredUrl = $this->appConfig->getValueString(
			Application::APP_ID,
			'control_plane_url',
			'http://control-plane:3000',
		);

		return rtrim($configuredUrl, '/');
	}

	public function getApiToken(): string {
		$environmentToken = getenv('BETTERNAS_CONTROL_PLANE_API_TOKEN');
		if (is_string($environmentToken) && $environmentToken !== '') {
			return $environmentToken;
		}

		return $this->appConfig->getValueString(
			Application::APP_ID,
			'control_plane_api_token',
			'',
		);
	}
}
