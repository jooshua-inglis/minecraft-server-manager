<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import {
		getServer,
		streamLogs,
		streamStats,
		startServer,
		stopServer,
		execCommand,
		getWhitelist,
		addWhitelist,
		removeWhitelist,
		getOps,
		addOp,
		removeOp,
		getMods,
		addMod,
		getModpack,
		installModpack,
		getBackups,
		createBackup,
		restoreBackup,
		destroyServer,
		type ServerDetail,
		type ServerStats,
		type WhitelistEntry,
		type OpEntry,
		type ModsInfo,
		type ModpackStatus,
		type BackupInfo
	} from '$lib/api';
	import { auth } from '$lib/auth.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import Sparkline from '$lib/components/Sparkline.svelte';
	import PlayerListEditor from '$lib/components/PlayerListEditor.svelte';

	const DETAIL_POLL_MS = 5000;
	const HISTORY_LEN = 60;
	const MAX_LOG_LINES = 500;

	const name = $derived(page.params.name ?? '');
	const unlocked = $derived(!!auth.token);

	let detail = $state<ServerDetail | null>(null);
	let error = $state<string | null>(null);

	let cpuHistory = $state<number[]>([]);
	let memHistory = $state<number[]>([]);
	let memLimit = $state(0);
	let players = $state('-');

	let logLines = $state<string[]>([]);
	let logEl: HTMLPreElement | undefined = $state();

	function mib(bytes: number) {
		return bytes / 1024 / 1024;
	}

	async function refreshDetail() {
		try {
			detail = await getServer(name);
			error = null;
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		}
	}

	function onStats(s: ServerStats) {
		cpuHistory = [...cpuHistory, s.cpu_percent].slice(-HISTORY_LEN);
		memHistory = [...memHistory, mib(s.mem_usage_bytes)].slice(-HISTORY_LEN);
		memLimit = mib(s.mem_limit_bytes);
		players = s.players;
	}

	function onLogLine(line: string) {
		logLines = [...logLines, line].slice(-MAX_LOG_LINES);
		queueMicrotask(() => {
			if (logEl) logEl.scrollTop = logEl.scrollHeight;
		});
	}

	// --- write actions ---

	let lifecycleBusy = $state(false);
	async function toggleLifecycle() {
		if (!detail) return;
		lifecycleBusy = true;
		try {
			if (detail.status === 'running') await stopServer(name);
			else await startServer(name);
			await refreshDetail();
		} catch (e) {
			alert(e instanceof Error ? e.message : String(e));
		} finally {
			lifecycleBusy = false;
		}
	}

	let command = $state('');
	let commandOutput = $state('');
	let commandBusy = $state(false);
	async function runCommand(e: Event) {
		e.preventDefault();
		if (!command.trim()) return;
		commandBusy = true;
		try {
			commandOutput = await execCommand(name, command.trim());
			command = '';
		} catch (e) {
			commandOutput = e instanceof Error ? e.message : String(e);
		} finally {
			commandBusy = false;
		}
	}

	let whitelist = $state<WhitelistEntry[]>([]);
	let ops = $state<OpEntry[]>([]);
	async function loadPlayerLists() {
		try {
			[whitelist, ops] = await Promise.all([getWhitelist(name), getOps(name)]);
		} catch {
			// non-fatal — the main detail card already surfaces load errors
		}
	}

	let mods = $state<ModsInfo>({});
	let modpack = $state<ModpackStatus | null>(null);
	async function loadModsAndModpack() {
		try {
			[mods, modpack] = await Promise.all([getMods(name), getModpack(name)]);
		} catch {
			// non-fatal
		}
	}

	let modSource = $state<'modrinth' | 'curseforge' | 'url'>('modrinth');
	let modRef = $state('');
	let modBusy = $state(false);
	let modError = $state('');
	async function submitMod(e: Event) {
		e.preventDefault();
		if (!modRef.trim()) return;
		modBusy = true;
		modError = '';
		try {
			await addMod(name, modSource, modRef.trim());
			modRef = '';
			await loadModsAndModpack();
		} catch (e) {
			modError = e instanceof Error ? e.message : String(e);
		} finally {
			modBusy = false;
		}
	}

	let modpackSource = $state<'modrinth' | 'curseforge'>('modrinth');
	let modpackRef = $state('');
	let modpackBusy = $state(false);
	let modpackError = $state('');
	async function submitModpack(e: Event) {
		e.preventDefault();
		if (!modpackRef.trim()) return;
		modpackBusy = true;
		modpackError = '';
		try {
			await installModpack(name, modpackSource, modpackRef.trim());
			modpackRef = '';
			await loadModsAndModpack();
			await refreshDetail();
		} catch (e) {
			modpackError = e instanceof Error ? e.message : String(e);
		} finally {
			modpackBusy = false;
		}
	}

	let backups = $state<BackupInfo[]>([]);
	async function loadBackups() {
		try {
			backups = await getBackups(name);
		} catch {
			// non-fatal
		}
	}

	let backupBusy = $state(false);
	async function onBackupClick() {
		backupBusy = true;
		try {
			await createBackup(name);
			await loadBackups();
		} catch (e) {
			alert(e instanceof Error ? e.message : String(e));
		} finally {
			backupBusy = false;
		}
	}

	let restoringId = $state<string | null>(null);
	async function onRestoreClick(id: string) {
		if (!confirm(`Restore backup "${id}"? This replaces the server's current data. The server must be stopped.`))
			return;
		restoringId = id;
		try {
			await restoreBackup(name, id);
			await refreshDetail();
		} catch (e) {
			alert(e instanceof Error ? e.message : String(e));
		} finally {
			restoringId = null;
		}
	}

	let destroying = $state(false);
	async function onDestroyClick() {
		if (!confirm(`Destroy "${name}"? This also deletes its data directory (world, mods, configs).`)) return;
		destroying = true;
		try {
			await destroyServer(name, true);
			await goto('/');
		} catch (e) {
			alert(e instanceof Error ? e.message : String(e));
			destroying = false;
		}
	}

	let detailTimer: ReturnType<typeof setInterval>;
	let stopStats: () => void;
	let stopLogs: () => void;

	onMount(() => {
		refreshDetail();
		loadPlayerLists();
		loadModsAndModpack();
		loadBackups();
		detailTimer = setInterval(refreshDetail, DETAIL_POLL_MS);
		stopStats = streamStats(name, onStats);
		stopLogs = streamLogs(name, onLogLine, '100');
	});
	onDestroy(() => {
		clearInterval(detailTimer);
		stopStats?.();
		stopLogs?.();
	});
</script>

<div class="page">
	<p class="back"><a href="/">&larr; all servers</a></p>

	{#if error && !detail}
		<p class="error">{error}</p>
	{:else if !detail}
		<p>Loading…</p>
	{:else}
		<h1>
			{detail.name}
			<StatusBadge status={detail.status} />
			{#if unlocked}
				<button class="lifecycle" disabled={lifecycleBusy} onclick={toggleLifecycle}>
					{#if lifecycleBusy}…{:else if detail.status === 'running'}Stop{:else}Start{/if}
				</button>
			{/if}
		</h1>

		{#if detail.crash}
			<div class="card crash">
				<h2>Crash</h2>
				<p>
					{#if detail.crash.gave_up}
						Gave up after {detail.crash.restart_count}/{detail.crash.max_retries} restarts
					{:else}
						Restarting (attempt {detail.crash.restart_count}/{detail.crash.max_retries})
					{/if}
					&mdash; {detail.crash.oom_killed ? 'OOM killed' : `exit code ${detail.crash.exit_code}`}
				</p>
				{#if detail.crash.last_log_lines?.length}
					<pre class="mono log">{detail.crash.last_log_lines.join('\n')}</pre>
				{/if}
			</div>
		{/if}

		<div class="card console-card">
			<h2>Console</h2>
			<pre class="mono log live" bind:this={logEl}>{logLines.join('\n')}</pre>
			{#if unlocked}
				<form class="console-form" onsubmit={runCommand}>
					<input class="mono" placeholder="say hello" bind:value={command} />
					<button type="submit" disabled={commandBusy}>Run</button>
				</form>
				{#if commandOutput}<pre class="mono log">{commandOutput}</pre>{/if}
			{/if}
		</div>

		<div class="bento">
			<div class="card span-2">
				<h2>Info</h2>
				<dl>
					<dt>Type</dt>
					<dd>{detail.type}</dd>
					<dt>Version</dt>
					<dd>{detail.version}</dd>
					<dt>Port</dt>
					<dd>{detail.port}</dd>
					<dt>Players</dt>
					<dd>{players}</dd>
					<dt>Data dir</dt>
					<dd class="mono">{detail.data_dir}</dd>
					<dt>Created</dt>
					<dd>{new Date(detail.created_at).toLocaleString()}</dd>
				</dl>
			</div>

			<div class="card alt">
				<h2>Usage</h2>
				<div class="metric">
					<span class="label">CPU</span>
					<Sparkline values={cpuHistory} max={100} formatValue={(v) => `${v.toFixed(1)}%`} />
				</div>
				<div class="metric">
					<span class="label">Memory</span>
					<Sparkline
						values={memHistory}
						max={memLimit || undefined}
						color="var(--good)"
						formatValue={(v) => `${v.toFixed(0)} MiB`}
					/>
				</div>
			</div>

			<PlayerListEditor
				title="Whitelist"
				entries={whitelist}
				{unlocked}
				onAdd={async (p, o) => {
					await addWhitelist(name, p, o);
					await loadPlayerLists();
				}}
				onRemove={async (p) => {
					await removeWhitelist(name, p);
					await loadPlayerLists();
				}}
			/>

			<PlayerListEditor
				title="Ops"
				entries={ops}
				{unlocked}
				onAdd={async (p, o) => {
					await addOp(name, p, o);
					await loadPlayerLists();
				}}
				onRemove={async (p) => {
					await removeOp(name, p);
					await loadPlayerLists();
				}}
			/>

			<div class="card">
				<h2>Mods &amp; plugins</h2>
				{#each [['Modrinth', mods.modrinth_projects], ['CurseForge', mods.curseforge_files], ['Mod URLs', mods.mod_urls], ['Plugin URLs', mods.plugin_urls]] as [label, list] (label)}
					{#if list?.length}
						<p class="muted small">{label}</p>
						<ul class="list plain">
							{#each list as item (item)}<li class="mono">{item}</li>{/each}
						</ul>
					{/if}
				{/each}
				{#if !mods.modrinth_projects?.length && !mods.curseforge_files?.length && !mods.mod_urls?.length && !mods.plugin_urls?.length}
					<p class="muted">None installed.</p>
				{/if}
				{#if unlocked}
					<form class="mod-form" onsubmit={submitMod}>
						<select bind:value={modSource}>
							<option value="modrinth">Modrinth</option>
							<option value="curseforge">CurseForge</option>
							<option value="url">URL</option>
						</select>
						<input placeholder="project slug / file id / url" bind:value={modRef} />
						<button type="submit" disabled={modBusy}>Add</button>
					</form>
					{#if modError}<p class="error">{modError}</p>{/if}
				{/if}
			</div>

			<div class="card alt">
				<h2>Modpack</h2>
				{#if modpack?.ref}
					<p>{modpack.source} &mdash; <span class="mono">{modpack.ref}</span></p>
				{:else}
					<p class="muted">None installed.</p>
				{/if}
				{#if unlocked}
					<form class="mod-form" onsubmit={submitModpack}>
						<select bind:value={modpackSource}>
							<option value="modrinth">Modrinth</option>
							<option value="curseforge">CurseForge</option>
						</select>
						<input placeholder="pack slug / page url" bind:value={modpackRef} />
						<button type="submit" disabled={modpackBusy}>Install</button>
					</form>
					<p class="muted small">Installing recreates the container, so the server restarts if it's running.</p>
					{#if modpackError}<p class="error">{modpackError}</p>{/if}
				{/if}
			</div>

			<div class="card span-2">
				<h2>Backups</h2>
				{#if backups.length === 0}
					<p class="muted">No backups yet.</p>
				{:else}
					<table>
						<thead>
							<tr>
								<th>ID</th>
								<th>Created</th>
								<th>Size</th>
								{#if unlocked}<th></th>{/if}
							</tr>
						</thead>
						<tbody>
							{#each backups as b (b.id)}
								<tr>
									<td class="mono">{b.id}</td>
									<td>{new Date(b.created_at).toLocaleString()}</td>
									<td>{(b.size_bytes / 1024 / 1024).toFixed(1)} MiB</td>
									{#if unlocked}
										<td>
											<button
												class="ghost"
												disabled={restoringId === b.id}
												onclick={() => onRestoreClick(b.id)}
											>
												{restoringId === b.id ? 'Restoring…' : 'Restore'}
											</button>
										</td>
									{/if}
								</tr>
							{/each}
						</tbody>
					</table>
				{/if}
				{#if unlocked}
					<button disabled={backupBusy} onclick={onBackupClick}>{backupBusy ? 'Backing up…' : 'Backup now'}</button>
				{/if}
			</div>
		</div>

		{#if unlocked}
			<div class="card danger-zone">
				<h2>Danger zone</h2>
				<button class="danger" disabled={destroying} onclick={onDestroyClick}>
					{destroying ? 'Destroying…' : 'Destroy server'}
				</button>
			</div>
		{/if}
	{/if}
</div>

<style>
	.back {
		margin-top: 0;
	}
	h1 {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}
	dl {
		display: grid;
		grid-template-columns: auto 1fr;
		gap: 0.35rem 1rem;
		margin: 0;
	}
	dt {
		color: var(--text-dim);
	}
	dd {
		margin: 0;
	}
	.metric {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		margin: 0.75rem 0;
	}
	.metric .label {
		width: 3.5rem;
		color: var(--text-dim);
		font-size: 0.85rem;
	}
	.crash {
		border-color: color-mix(in srgb, var(--bad) 40%, var(--border));
		margin-bottom: 1rem;
	}
	.log {
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: 8px;
		padding: 0.75rem;
		font-size: 0.85rem;
		white-space: pre-wrap;
		word-break: break-word;
		max-height: 320px;
		overflow-y: auto;
		margin: 0.5rem 0 0;
	}
	.live {
		max-height: 420px;
	}
	.muted {
		color: var(--text-dim);
	}
	.muted.small {
		font-size: 0.8rem;
		margin-bottom: 0.25rem;
	}
	.list.plain {
		list-style: none;
		margin: 0 0 0.75rem;
		padding: 0;
		font-size: 0.85rem;
	}
	.list.plain li {
		padding: 0.15rem 0;
	}
	.crash,
	.console-card,
	.danger-zone {
		margin-bottom: 1rem;
	}
	.danger-zone {
		border-color: color-mix(in srgb, var(--bad) 40%, var(--border));
	}

	button,
	.lifecycle {
		font-family:
			system-ui,
			-apple-system,
			sans-serif;
		background: var(--accent);
		color: var(--accent-contrast);
		border: none;
		border-radius: 6px;
		padding: 0.4em 0.9em;
		font-weight: 600;
		cursor: pointer;
		font-size: 0.85rem;
	}
	button.ghost {
		background: none;
		border: 1px solid var(--border);
		color: var(--text-dim);
		font-weight: 500;
	}
	button.danger {
		background: var(--bad);
	}
	button:disabled {
		opacity: 0.6;
		cursor: default;
	}
	.console-form,
	.mod-form {
		display: flex;
		gap: 0.5rem;
		margin-top: 0.5rem;
	}
	.console-form input,
	.mod-form input,
	.mod-form select {
		background: var(--surface-2);
		border: 1px solid var(--border);
		color: var(--text);
		border-radius: 6px;
		padding: 0.4em 0.6em;
		flex: 1;
	}
</style>
