<?php

declare(strict_types=1);

namespace OCA\betternasControlplane\Service;

use OCA\betternasControlplane\AppInfo\Application;
use OCP\IAppConfig;

class ControlPlaneConfig {
	public function __construct(
		private readonly IAppConfig $appConfig,
	) {
	}

	public function getBaseUrl(): string {
		$environmentUrl = getenv('betternas_CONTROL_PLANE_URL');
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
}

