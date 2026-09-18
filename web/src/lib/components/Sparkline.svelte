<script lang="ts">
	// A minimal inline-SVG line chart for a rolling window of samples —
	// deliberately dependency-free so the embedded UI bundle stays small.
	let {
		values,
		max,
		color = 'var(--accent)',
		formatValue = (v: number) => v.toFixed(0)
	}: {
		values: number[];
		max?: number;
		color?: string;
		formatValue?: (v: number) => string;
	} = $props();

	const width = 240;
	const height = 60;
	const pad = 4;

	const scaleMax = $derived(max ?? Math.max(1, ...values));

	const points = $derived(
		values
			.map((v, i) => {
				const x = values.length > 1 ? (i / (values.length - 1)) * (width - pad * 2) + pad : width / 2;
				const y = height - pad - (Math.min(v, scaleMax) / scaleMax) * (height - pad * 2);
				return `${x.toFixed(1)},${y.toFixed(1)}`;
			})
			.join(' ')
	);

	const latest = $derived(values.length ? values[values.length - 1] : 0);
</script>

<div class="sparkline">
	<svg viewBox="0 0 {width} {height}" preserveAspectRatio="none" role="img" aria-label="usage over time">
		<polyline fill="none" stroke={color} stroke-width="2" points={points} />
	</svg>
	<span class="value">{formatValue(latest)}</span>
</div>

<style>
	.sparkline {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}
	svg {
		width: 100%;
		max-width: 240px;
		height: 40px;
		flex-shrink: 0;
	}
	.value {
		font-variant-numeric: tabular-nums;
		font-weight: 600;
		min-width: 4.5em;
		text-align: right;
	}
</style>
