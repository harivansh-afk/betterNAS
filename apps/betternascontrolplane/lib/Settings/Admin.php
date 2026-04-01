<?php

declare(strict_types=1);

namespace OCA\BetterNasControlplane\Settings;

use OCA\BetterNasControlplane\AppInfo\Application;
use OCA\BetterNasControlplane\Service\ControlPlaneClient;
use OCA\BetterNasControlplane\Service\ControlPlaneConfig;
use OCP\AppFramework\Http\TemplateResponse;
use OCP\Settings\ISettings;

class Admin implements ISettings {
	public function __construct(
		private readonly ControlPlaneConfig $controlPlaneConfig,
		private readonly ControlPlaneClient $controlPlaneClient,
	) {
	}

	public function getForm(): TemplateResponse {
		return new TemplateResponse(
			Application::APP_ID,
			'admin',
			[
				'controlPlaneUrl' => $this->controlPlaneConfig->getBaseUrl(),
				'snapshot' => $this->controlPlaneClient->fetchSnapshot(),
			],
		);
	}

	public function getSection(): string {
		return Application::APP_ID;
	}

	public function getPriority(): int {
		return 50;
	}
}

