// Shared, reactive auth state: the write-action token the operator
// pastes in from `mcm web`'s startup output (see internal/cliapp/web.go
// / RESEARCH.md §7.5). Persisted to localStorage as a per-browser
// convenience only — never sent anywhere but this app's own API.

const STORAGE_KEY = 'mcm-web-token';

function readStored(): string {
	try {
		return localStorage.getItem(STORAGE_KEY) ?? '';
	} catch {
		return '';
	}
}

export const auth = $state({ token: readStored() });

export function setToken(token: string) {
	auth.token = token;
	try {
		if (token) localStorage.setItem(STORAGE_KEY, token);
		else localStorage.removeItem(STORAGE_KEY);
	} catch {
		// a locked-down browser just won't remember it across reloads
	}
}

export function clearToken() {
	setToken('');
}
