<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { page } from '$app/state';
	import { getServer, streamLogs, streamStats, type ServerDetail, type ServerStats } from '$lib/api';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import Sparkline from '$lib/components/Sparkline.svelte';

	const DETAIL_POLL_MS = 5000;
	const HISTORY_LEN = 60;
	const MAX_LOG_LINES = 500;

	const name = $derived(page.params.name ?? '');

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

	let detailTimer: ReturnType<typeof setInterval>;
	let stopStats: () => void;
	let stopLogs: () => void;

	onMount(() => {
		refreshDetail();
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
		<h1>{detail.name} <StatusBadge status={detail.status} /></h1>

		<div class="grid">
			<div class="card">
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

			<div class="card">
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
		</div>

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

		<div class="card">
			<h2>Console</h2>
			<pre class="mono log live" bind:this={logEl}>{logLines.join('\n')}</pre>
		</div>
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
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
		gap: 1rem;
		margin-bottom: 1rem;
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
</style>
