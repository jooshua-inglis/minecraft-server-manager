<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import '../app.css';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>mcm</title>
</svelte:head>

<header class="topbar">
	<a href="/" class="brand">mcm</a>
</header>

{@render children()}

<style>
	.topbar {
		border-bottom: 1px solid var(--border);
		padding: 0.9rem 1rem;
	}
	.brand {
		font-weight: 700;
		font-size: 1.1rem;
		color: var(--text);
		letter-spacing: -0.02em;
	}
	.brand:hover {
		text-decoration: none;
		color: var(--accent);
	}
</style>
