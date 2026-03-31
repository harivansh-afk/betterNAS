<?php

declare(strict_types=1);

namespace OCA\AinasControlplane\Service;

use OCP\Http\Client\IClientService;
use OCP\Http\Client\IResponse;
use Psr\Log\LoggerInterface;

class ControlPlaneClient {
	public function __construct(
		private readonly IClientService $clientService,
		private readonly ControlPlaneConfig $controlPlaneConfig,
		private readonly LoggerInterface $logger,
	) {
	}

	/**
	 * @return array<string, mixed>
	 */
	public function fetchSnapshot(): array {
		$baseUrl = $this->controlPlaneConfig->getBaseUrl();

		try {
			$healthResponse = $this->request($baseUrl . '/health');
			$versionResponse = $this->request($baseUrl . '/version');

			return [
				'available' => $healthResponse['statusCode'] === 200,
				'url' => $baseUrl,
				'health' => $healthResponse['body'],
				'version' => $versionResponse['body'],
			];
		} catch (\Throwable $exception) {
			$this->logger->warning('Failed to reach aiNAS control plane', [
				'exception' => $exception,
				'url' => $baseUrl,
			]);

			return [
				'available' => false,
				'url' => $baseUrl,
				'error' => $exception->getMessage(),
			];
		}
	}

	/**
	 * @return array{statusCode: int, body: array<string, mixed>}
	 */
	private function request(string $url): array {
		$client = $this->clientService->newClient();
		$response = $client->get($url, [
			'headers' => [
				'Accept' => 'application/json',
			],
			'http_errors' => false,
			'timeout' => 2,
			'nextcloud' => [
				'allow_local_address' => true,
			],
		]);

		return [
			'statusCode' => $response->getStatusCode(),
			'body' => $this->decodeBody($response),
		];
	}

	/**
	 * @return array<string, mixed>
	 */
	private function decodeBody(IResponse $response): array {
		$body = $response->getBody();
		if ($body === '') {
			return [];
		}

		$decoded = json_decode($body, true, 512, JSON_THROW_ON_ERROR);
		if (!is_array($decoded)) {
			return [];
		}

		return $decoded;
	}
}

