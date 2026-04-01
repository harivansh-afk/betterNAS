<?php

declare(strict_types=1);

use OCA\BetterNasControlplane\AppInfo\Application;
use OCP\Util;

Util::addStyle(Application::APP_ID, 'betternascontrolplane');

$snapshot = $_['snapshot'];
$version = $snapshot['version']['version'] ?? 'unreachable';
$status = !empty($snapshot['available']) ? 'Connected' : 'Unavailable';
$error = $snapshot['error'] ?? null;
?>

<div class="betternas-shell">
	<div class="betternas-shell__hero">
		<p class="betternas-shell__eyebrow">betterNAS inside Nextcloud</p>
		<h1 class="betternas-shell__title"><?php p($_['appName']); ?></h1>
		<p class="betternas-shell__copy">
			This shell app stays intentionally thin. It exposes betterNAS entry points inside Nextcloud and delegates business logic to the external control-plane service.
		</p>
	</div>

	<div class="betternas-shell__grid">
		<section class="betternas-shell__card">
			<h2>Control plane</h2>
			<dl>
				<dt>Configured URL</dt>
				<dd><code><?php p($_['controlPlaneUrl']); ?></code></dd>
				<dt>Status</dt>
				<dd><?php p($status); ?></dd>
				<dt>Version</dt>
				<dd><?php p($version); ?></dd>
			</dl>
			<?php if ($error !== null): ?>
				<p class="betternas-shell__error"><?php p($error); ?></p>
			<?php endif; ?>
		</section>

		<section class="betternas-shell__card">
			<h2>Boundary</h2>
			<ul>
				<li>Nextcloud provides file and client primitives.</li>
				<li>betterNAS owns control-plane policy and orchestration.</li>
				<li>The shell app only adapts between the two.</li>
			</ul>
		</section>
	</div>
</div>

