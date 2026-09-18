<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { listServers, createServer, destroyServer, type ServerView } from '$lib/api';
	import { auth } from '$lib/auth.svelte';
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

	let showCreate = $state(false);
	let name = $state('');
	let type = $state('VANILLA');
	let version = $state('LATEST');
	let memory = $state('2G');
	let acceptEula = $state(false);
	let creating = $state(false);
	let createError = $state('');

	async function submitCreate(e: Event) {
		e.preventDefault();
		if (!acceptEula) {
			createError = 'You must accept the Minecraft EULA to create a server.';
			return;
		}
		creating = true;
		createError = '';
		try {
			await createServer({ name, type, version, memory, accept_eula: acceptEula });
			name = '';
			acceptEula = false;
			showCreate = false;
			await refresh();
		} catch (e) {
			createError = e instanceof Error ? e.message : String(e);
		} finally {
			creating = false;
		}
	}

	let destroying = $state<string | null>(null);
	async function onDestroyClick(s: ServerView) {
		if (!confirm(`Destroy "${s.name}"? This also deletes its data directory (world, mods, configs).`)) return;
		destroying = s.name;
		try {
			await destroyServer(s.name, true);
			await refresh();
		} catch (e) {
			alert(e instanceof Error ? e.message : String(e));
		} finally {
			destroying = null;
		}
	}
</script>

<div class="page">
	<div class="header-row">
		<h1>Servers</h1>
		{#if auth.token}
			<button onclick={() => (showCreate = !showCreate)}>{showCreate ? 'Cancel' : '+ New server'}</button>
		{/if}
	</div>

	{#if showCreate}
		<form class="card create-form" onsubmit={submitCreate}>
			<div class="field">
				<label for="name">Name</label>
				<input id="name" required pattern="[a-z0-9][-a-z0-9]*" bind:value={name} placeholder="my-server" />
			</div>
			<div class="field">
				<label for="type">Type</label>
				<input id="type" bind:value={type} />
			</div>
			<div class="field">
				<label for="version">Version</label>
				<input id="version" bind:value={version} />
			</div>
			<div class="field">
				<label for="memory">Memory</label>
				<input id="memory" bind:value={memory} />
			</div>
			<label class="eula">
				<input type="checkbox" bind:checked={acceptEula} />
				I have read and accept the
				<a href="https://www.minecraft.net/en-us/eula" target="_blank" rel="noreferrer">Minecraft EULA</a>
			</label>
			{#if createError}<p class="error">{createError}</p>{/if}
			<button type="submit" disabled={creating}>{creating ? 'Creating…' : 'Create'}</button>
		</form>
	{/if}

	{#if error}
		<p class="error">{error}</p>
	{:else if !loaded}
		<p>Loading…</p>
	{:else if servers.length === 0}
		<p>No servers yet — create one above{auth.token ? '' : ' (unlock write actions first)'}.</p>
	{:else}
		<div class="card">
			<table>
				<thead>
					<tr>
						<th>Name</th>
						<th>Type</th>
						<th>Version</th>
						<th>Port</th>
						<th>Status</th>
						{#if auth.token}<th></th>{/if}
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
							{#if auth.token}
								<td>
									<button class="danger" disabled={destroying === s.name} onclick={() => onDestroyClick(s)}>
										{destroying === s.name ? 'Destroying…' : 'Destroy'}
									</button>
								</td>
							{/if}
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<style>
	.header-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	.create-form {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
		gap: 0.75rem;
		align-items: end;
		margin: 1rem 0 1.5rem;
	}
	.field {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}
	.field label {
		color: var(--text-dim);
		font-size: 0.8rem;
	}
	.field input {
		background: var(--surface-2);
		border: 1px solid var(--border);
		color: var(--text);
		border-radius: 6px;
		padding: 0.45em 0.6em;
	}
	.eula {
		grid-column: 1 / -1;
		display: flex;
		align-items: center;
		gap: 0.5em;
		font-size: 0.9rem;
	}
	.create-form button[type='submit'] {
		grid-column: 1 / -1;
		justify-self: start;
	}
	button {
		background: var(--accent);
		color: var(--accent-contrast);
		border: none;
		border-radius: 8px;
		padding: 0.5em 1em;
		font-weight: 600;
		cursor: pointer;
		font-family:
			system-ui,
			-apple-system,
			sans-serif;
	}
	button.danger {
		background: var(--bad);
	}
	button:disabled {
		opacity: 0.6;
		cursor: default;
	}
</style>
