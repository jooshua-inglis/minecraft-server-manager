import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) => filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			adapter: adapter({
				// mcm's Go binary embeds this build via go:embed and serves it
				// itself, so it needs a plain static build under
				// internal/webui/dist rather than a Node/edge adapter output.
				pages: '../internal/webui/dist',
				assets: '../internal/webui/dist',
				fallback: 'index.html',
				strict: false
			})
		})
	],
	server: {
		// So `npm run dev` works against a locally running `mcm web` without
		// CORS wrangling — same convenience the embedded build gets for free
		// by being served from the same origin as the API.
		proxy: {
			'/api': 'http://127.0.0.1:8080'
		}
	}
});
