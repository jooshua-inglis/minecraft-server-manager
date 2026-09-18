<script lang="ts">
	import { auth, setToken, clearToken } from '$lib/auth.svelte';
	import { verifyToken } from '$lib/api';

	let editing = $state(false);
	let input = $state('');
	let error = $state('');
	let checking = $state(false);

	function startEditing() {
		input = auth.token;
		error = '';
		editing = true;
	}

	async function save() {
		if (!input.trim()) {
			clearToken();
			editing = false;
			return;
		}
		checking = true;
		error = '';
		try {
			if (!(await verifyToken(input.trim()))) {
				error = 'Invalid token';
				return;
			}
			setToken(input.trim());
			editing = false;
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			checking = false;
		}
	}

	function cancel() {
		editing = false;
		error = '';
	}
</script>

{#if editing}
	<form
		class="unlock-form"
		onsubmit={(e) => {
			e.preventDefault();
			save();
		}}
	>
		<input type="password" placeholder="paste web token" bind:value={input} />
		<button type="submit" disabled={checking}>{checking ? '…' : 'Save'}</button>
		<button type="button" class="ghost" onclick={cancel}>Cancel</button>
		{#if error}<span class="error">{error}</span>{/if}
	</form>
{:else if auth.token}
	<button class="unlock-toggle unlocked" onclick={startEditing} title="Write actions unlocked">
		🔓 unlocked
	</button>
{:else}
	<button class="unlock-toggle" onclick={startEditing} title="Paste the web token to unlock write actions">
		🔒 locked
	</button>
{/if}

<style>
	.unlock-toggle {
		background: none;
		border: 1px solid var(--border);
		color: var(--text-dim);
		border-radius: 999px;
		padding: 0.3em 0.8em;
		font-size: 0.85em;
		cursor: pointer;
		font-family:
			system-ui,
			-apple-system,
			sans-serif;
	}
	.unlock-toggle.unlocked {
		color: var(--good);
		border-color: color-mix(in srgb, var(--good) 40%, var(--border));
	}
	.unlock-form {
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}
	.unlock-form input {
		background: var(--surface-2);
		border: 1px solid var(--border);
		color: var(--text);
		border-radius: 6px;
		padding: 0.3em 0.6em;
		font-size: 0.85em;
		width: 12rem;
	}
	.unlock-form button {
		font-family:
			system-ui,
			-apple-system,
			sans-serif;
		font-size: 0.85em;
		padding: 0.3em 0.7em;
		border-radius: 6px;
		border: 1px solid var(--border);
		background: var(--accent);
		color: var(--accent-contrast);
		font-weight: 600;
		cursor: pointer;
	}
	.unlock-form button.ghost {
		background: none;
		color: var(--text-dim);
	}
	.error {
		color: var(--bad);
		font-size: 0.8em;
	}
</style>
