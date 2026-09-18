<script lang="ts">
	let {
		title,
		entries,
		unlocked,
		onAdd,
		onRemove
	}: {
		title: string;
		entries: { name: string }[];
		unlocked: boolean;
		onAdd: (player: string, offline: boolean) => Promise<void>;
		onRemove: (player: string) => Promise<void>;
	} = $props();

	let player = $state('');
	let offline = $state(false);
	let busy = $state(false);
	let error = $state('');

	async function add(e: Event) {
		e.preventDefault();
		if (!player.trim()) return;
		busy = true;
		error = '';
		try {
			await onAdd(player.trim(), offline);
			player = '';
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			busy = false;
		}
	}

	async function remove(name: string) {
		busy = true;
		error = '';
		try {
			await onRemove(name);
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			busy = false;
		}
	}
</script>

<div class="card">
	<h2>{title}</h2>
	{#if entries.length === 0}
		<p class="muted">Empty.</p>
	{:else}
		<ul class="list">
			{#each entries as entry (entry.name)}
				<li>
					<span>{entry.name}</span>
					{#if unlocked}
						<button class="ghost" disabled={busy} onclick={() => remove(entry.name)}>Remove</button>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}

	{#if unlocked}
		<form class="add-form" onsubmit={add}>
			<input placeholder="player name" bind:value={player} />
			<label><input type="checkbox" bind:checked={offline} /> offline</label>
			<button type="submit" disabled={busy}>Add</button>
		</form>
	{/if}
	{#if error}<p class="error">{error}</p>{/if}
</div>

<style>
	.muted {
		color: var(--text-dim);
	}
	.list {
		list-style: none;
		margin: 0 0 0.75rem;
		padding: 0;
	}
	.list li {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.35rem 0;
		border-bottom: 1px solid var(--border);
	}
	.add-form {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		align-items: center;
	}
	.add-form input:not([type]) {
		background: var(--surface-2);
		border: 1px solid var(--border);
		color: var(--text);
		border-radius: 6px;
		padding: 0.4em 0.6em;
		flex: 1 1 8rem;
		min-width: 0;
	}
	.add-form label {
		font-size: 0.85rem;
		color: var(--text-dim);
		white-space: nowrap;
	}
	button {
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
	}
	button.ghost {
		background: none;
		border: 1px solid var(--border);
		color: var(--text-dim);
		font-weight: 500;
	}
	button:disabled {
		opacity: 0.6;
		cursor: default;
	}
</style>
