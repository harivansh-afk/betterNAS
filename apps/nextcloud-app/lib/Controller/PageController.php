<?php

declare(strict_types=1);

namespace OCA\BetterNasControlplane\Controller;

use OCA\BetterNasControlplane\AppInfo\Application;
use OCA\BetterNasControlplane\Service\ControlPlaneClient;
use OCA\BetterNasControlplane\Service\ControlPlaneConfig;
use OCP\AppFramework\Controller;
use OCP\AppFramework\Http\Attribute\FrontpageRoute;
use OCP\AppFramework\Http\Attribute\NoAdminRequired;
use OCP\AppFramework\Http\Attribute\NoCSRFRequired;
use OCP\AppFramework\Http\Attribute\OpenAPI;
use OCP\AppFramework\Http\TemplateResponse;
use OCP\IRequest;

class PageController extends Controller {
	public function __construct(
		IRequest $request,
		private readonly ControlPlaneClient $controlPlaneClient,
		private readonly ControlPlaneConfig $controlPlaneConfig,
	) {
		parent::__construct(Application::APP_ID, $request);
	}

	#[NoCSRFRequired]
	#[NoAdminRequired]
	#[OpenAPI(OpenAPI::SCOPE_IGNORE)]
	#[FrontpageRoute(verb: 'GET', url: '/')]
	public function index(): TemplateResponse {
		return new TemplateResponse(
			Application::APP_ID,
			'index',
			[
				'appName' => 'betterNAS Control Plane',
				'controlPlaneUrl' => $this->controlPlaneConfig->getBaseUrl(),
				'snapshot' => $this->controlPlaneClient->fetchSnapshot(),
			],
		);
	}

	#[NoCSRFRequired]
	#[NoAdminRequired]
	#[OpenAPI(OpenAPI::SCOPE_IGNORE)]
	#[FrontpageRoute(verb: 'GET', url: '/exports/{exportId}')]
	public function showExport(string $exportId): TemplateResponse {
		return new TemplateResponse(
			Application::APP_ID,
			'export',
			[
				'appName' => 'betterNAS Control Plane',
				'controlPlaneUrl' => $this->controlPlaneConfig->getBaseUrl(),
				'exportId' => $exportId,
				'export' => $this->controlPlaneClient->fetchExport($exportId),
			],
		);
	}
}
