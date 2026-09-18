<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { listServers, type ServerView } from '$lib/api';
	import StatusBadge from '$lib/components/StatusBadge.svelte';

	const POLL_MS = 3000;

	let servers = $state<ServerView[]>([]);
	let error = $state<string | null>(null);
	let loaded = $state(false);

	async function refresh() {
		try {
			servers = await listServers();
			error = null;
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loaded = true;
		}
	}

	let timer: ReturnType<typeof setInterval>;
	onMount(() => {
		refresh();
		timer = setInterval(refresh, POLL_MS);
	});
	onDestroy(() => clearInterval(timer));
</script>

<div class="page">
	<h1>Servers</h1>

	{#if error}
		<p class="error">{error}</p>
	{:else if !loaded}
		<p>Loading…</p>
	{:else if servers.length === 0}
		<p>No servers yet — create one with <code>mcm create &lt;name&gt;</code>.</p>
	{:else}
		<table>
			<thead>
				<tr>
					<th>Name</th>
					<th>Type</th>
					<th>Version</th>
					<th>Port</th>
					<th>Status</th>
				</tr>
			</thead>
			<tbody>
				{#each servers as s (s.name)}
					<tr>
						<td><a href="/servers/{encodeURIComponent(s.name)}">{s.name}</a></td>
						<td>{s.type}</td>
						<td>{s.version}</td>
						<td>{s.port}</td>
						<td><StatusBadge status={s.status} /></td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</div>
