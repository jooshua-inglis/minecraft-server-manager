<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import UnlockControl from '$lib/components/UnlockControl.svelte';
	import '../app.css';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>mcm</title>
</svelte:head>

<header class="topbar">
	<a href="/" class="brand">mcm</a>
	<UnlockControl />
</header>

{@render children()}

<style>
	.topbar {
		background: var(--surface-alt);
		border-bottom: 1px solid var(--border);
		padding: 1rem 1.25rem;
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	.brand {
		font-weight: 700;
		font-size: 1.3rem;
		color: var(--text);
		letter-spacing: -0.01em;
	}
	.brand:hover {
		text-decoration: none;
		color: var(--accent);
	}
</style>
