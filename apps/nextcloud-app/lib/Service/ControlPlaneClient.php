<?php

declare(strict_types=1);

namespace OCA\BetterNasControlplane\Service;

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
			$healthResponse = $this->requestObject($baseUrl . '/health');
			$versionResponse = $this->requestObject($baseUrl . '/version');

			return [
				'available' => $healthResponse['statusCode'] === 200,
				'url' => $baseUrl,
				'health' => $healthResponse['body'],
				'version' => $versionResponse['body'],
			];
		} catch (\Throwable $exception) {
			$this->logger->warning('Failed to reach betterNAS control plane', [
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
	 * @return array<string, mixed>|null
	 */
	public function fetchExport(string $exportId): ?array {
		$baseUrl = $this->controlPlaneConfig->getBaseUrl();

		try {
			$exportsResponse = $this->requestList($baseUrl . '/api/v1/exports', true);
		} catch (\Throwable $exception) {
			$this->logger->warning('Failed to fetch betterNAS exports', [
				'exception' => $exception,
				'url' => $baseUrl,
				'exportId' => $exportId,
			]);

			return null;
		}

		foreach ($exportsResponse['body'] as $export) {
			if (!is_array($export)) {
				continue;
			}
			if (($export['id'] ?? null) === $exportId) {
				return $export;
			}
		}

		return null;
	}

	/**
	 * @return array{statusCode: int, body: array<string, mixed>}
	 */
	private function requestObject(string $url, bool $authenticated = false): array {
		$response = $this->request($url, $authenticated);

		return [
			'statusCode' => $response->getStatusCode(),
			'body' => $this->decodeObjectBody($response),
		];
	}

	/**
	 * @return array{statusCode: int, body: array<int, array<string, mixed>>}
	 */
	private function requestList(string $url, bool $authenticated = false): array {
		$response = $this->request($url, $authenticated);

		return [
			'statusCode' => $response->getStatusCode(),
			'body' => $this->decodeListBody($response),
		];
	}

	private function request(string $url, bool $authenticated = false): IResponse {
		$headers = [
			'Accept' => 'application/json',
		];
		if ($authenticated) {
			$apiToken = $this->controlPlaneConfig->getApiToken();
			if ($apiToken === '') {
				throw new \RuntimeException('Missing betterNAS control plane API token');
			}

			$headers['Authorization'] = 'Bearer ' . $apiToken;
		}

		$client = $this->clientService->newClient();
		return $client->get($url, [
			'headers' => $headers,
			'http_errors' => false,
			'timeout' => 2,
			'nextcloud' => [
				'allow_local_address' => true,
			],
		]);
	}

	/**
	 * @return array<string, mixed>
	 */
	private function decodeObjectBody(IResponse $response): array {
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

	/**
	 * @return array<int, array<string, mixed>>
	 */
	private function decodeListBody(IResponse $response): array {
		$body = $response->getBody();
		if ($body === '') {
			return [];
		}

		$decoded = json_decode($body, true, 512, JSON_THROW_ON_ERROR);
		if (!is_array($decoded)) {
			return [];
		}

		$exports = [];
		foreach ($decoded as $export) {
			if (!is_array($export)) {
				continue;
			}
			$exports[] = $export;
		}

		return $exports;
	}
}
