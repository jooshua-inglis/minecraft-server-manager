<script lang="ts">
	let { status }: { status: string } = $props();

	const tone = $derived(
		status === 'running'
			? 'good'
			: status === 'restarting'
				? 'warn'
				: status === 'exited' || status === 'dead'
					? 'bad'
					: 'neutral'
	);
</script>

<span class="badge {tone}">{status}</span>

<style>
	.badge {
		display: inline-block;
		padding: 0.15em 0.6em;
		border-radius: 999px;
		font-size: 0.85em;
		font-weight: 600;
		text-transform: lowercase;
		border: 1px solid transparent;
	}
	.good {
		background: color-mix(in srgb, var(--good) 18%, transparent);
		color: var(--good);
		border-color: color-mix(in srgb, var(--good) 35%, transparent);
	}
	.warn {
		background: color-mix(in srgb, var(--warn) 18%, transparent);
		color: var(--warn);
		border-color: color-mix(in srgb, var(--warn) 35%, transparent);
	}
	.bad {
		background: color-mix(in srgb, var(--bad) 18%, transparent);
		color: var(--bad);
		border-color: color-mix(in srgb, var(--bad) 35%, transparent);
	}
	.neutral {
		background: var(--surface-2);
		color: var(--text-dim);
		border-color: var(--border);
	}
</style>
