<?php

declare(strict_types=1);

use OCA\BetterNasControlplane\AppInfo\Application;
use OCP\Util;

Util::addStyle(Application::APP_ID, 'betternascontrolplane');

$export = $_['export'];
$exportId = $_['exportId'];
?>

<div class="betternas-shell">
	<div class="betternas-shell__hero">
		<p class="betternas-shell__eyebrow">betterNAS export</p>
		<h1 class="betternas-shell__title">Export <?php p($exportId); ?></h1>
		<p class="betternas-shell__copy">
			This Nextcloud route is export-specific so cloud profiles can land on a concrete betterNAS surface without inventing new API shapes.
		</p>
	</div>

	<div class="betternas-shell__grid">
		<section class="betternas-shell__card">
			<h2>Control plane</h2>
			<dl>
				<dt>Configured URL</dt>
				<dd><code><?php p($_['controlPlaneUrl']); ?></code></dd>
				<dt>Export ID</dt>
				<dd><code><?php p($exportId); ?></code></dd>
				<?php if (is_array($export)): ?>
					<dt>Label</dt>
					<dd><?php p((string)($export['label'] ?? '')); ?></dd>
					<dt>Path</dt>
					<dd><code><?php p((string)($export['path'] ?? '')); ?></code></dd>
					<dt>Protocols</dt>
					<dd><?php p(implode(', ', array_map('strval', (array)($export['protocols'] ?? [])))); ?></dd>
				<?php else: ?>
					<dt>Status</dt>
					<dd>Export unavailable</dd>
				<?php endif; ?>
			</dl>
		</section>

		<section class="betternas-shell__card">
			<h2>Boundary</h2>
			<ul>
				<li>Control-plane registry decides which export this page represents.</li>
				<li>Nextcloud stays a thin cloud-facing adapter.</li>
				<li>Mount-mode still flows directly to the NAS WebDAV endpoint.</li>
			</ul>
		</section>
	</div>
</div>
