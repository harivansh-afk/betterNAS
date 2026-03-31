<?php

declare(strict_types=1);

namespace OCA\AinasControlplane\Controller;

use OCA\AinasControlplane\AppInfo\Application;
use OCA\AinasControlplane\Service\ControlPlaneClient;
use OCP\AppFramework\Http\Attribute\ApiRoute;
use OCP\AppFramework\Http\Attribute\NoAdminRequired;
use OCP\AppFramework\Http\DataResponse;
use OCP\AppFramework\OCSController;
use OCP\IRequest;

class ApiController extends OCSController {
	public function __construct(
		IRequest $request,
		private readonly ControlPlaneClient $controlPlaneClient,
	) {
		parent::__construct(Application::APP_ID, $request);
	}

	#[NoAdminRequired]
	#[ApiRoute(verb: 'GET', url: '/api/status')]
	public function status(): DataResponse {
		return new DataResponse($this->controlPlaneClient->fetchSnapshot());
	}
}

