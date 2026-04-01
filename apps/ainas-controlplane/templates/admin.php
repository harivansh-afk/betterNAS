<?php

declare(strict_types=1);

use OCA\AinasControlplane\AppInfo\Application;
use OCP\Util;

Util::addStyle(Application::APP_ID, 'ainascontrolplane');

$snapshot = $_['snapshot'];
$reachable = !empty($snapshot['available']) ? 'yes' : 'no';
$version = $snapshot['version']['version'] ?? 'unreachable';
?>

<div class="ainas-shell ainas-shell--admin">
	<div class="ainas-shell__hero">
		<p class="ainas-shell__eyebrow">Admin settings</p>
		<h1 class="ainas-shell__title">betterNAS control-plane wiring</h1>
		<p class="ainas-shell__copy">
			The local scaffold wires this app to the control plane through the <code>AINAS_CONTROL_PLANE_URL</code> environment variable in the Nextcloud container.
		</p>
	</div>

	<div class="ainas-shell__grid">
		<section class="ainas-shell__card">
			<h2>Current wiring</h2>
			<dl>
				<dt>Control-plane URL</dt>
				<dd><code><?php p($_['controlPlaneUrl']); ?></code></dd>
				<dt>Reachable</dt>
				<dd><?php p($reachable); ?></dd>
				<dt>Reported version</dt>
				<dd><?php p($version); ?></dd>
			</dl>
		</section>

		<section class="ainas-shell__card">
			<h2>Next step</h2>
			<p>Keep storage policy, sharing logic, and orchestration in the control-plane service. This page should remain a thin integration surface.</p>
		</section>
	</div>
</div>

