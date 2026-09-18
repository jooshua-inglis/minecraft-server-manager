// Pure client-side app: it's served as static files embedded in the mcm
// binary (see vite.config.ts's adapter-static config), with all data
// coming from mcm's own /api/* endpoints at runtime, not from SvelteKit's
// server-side rendering.
export const ssr = false;
