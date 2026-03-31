<?php

declare(strict_types=1);

namespace OCA\AinasControlplane\Service;

use OCA\AinasControlplane\AppInfo\Application;
use OCP\IAppConfig;

class ControlPlaneConfig {
	public function __construct(
		private readonly IAppConfig $appConfig,
	) {
	}

	public function getBaseUrl(): string {
		$environmentUrl = getenv('AINAS_CONTROL_PLANE_URL');
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

