// Client for mcm's read-only web API (internal/webapi), which mirrors
// `mcm list`/`status`/`logs` as JSON plus SSE streams for live logs and
// stats. See RESEARCH.md §7.7 / plan M12/M13.

export interface ServerView {
	name: string;
	type: string;
	version: string;
	port: number;
	status: string;
}

export interface CrashInfo {
	restart_count: number;
	max_retries: number;
	exit_code: number;
	oom_killed: boolean;
	gave_up: boolean;
	last_log_lines?: string[];
}

export interface ServerDetail {
	name: string;
	type: string;
	version: string;
	port: number;
	rcon_port: number;
	data_dir: string;
	created_at: string;
	container?: string;
	status: string;
	started_at?: string;
	last_error?: string;
	crash?: CrashInfo;
}

export interface ServerStats {
	name: string;
	status: string;
	cpu_percent: number;
	mem_usage_bytes: number;
	mem_limit_bytes: number;
	players: string;
}

async function apiError(res: Response): Promise<Error> {
	try {
		const body = await res.json();
		if (typeof body?.error === 'string') return new Error(body.error);
	} catch {
		// body wasn't JSON — fall through to the generic message below.
	}
	return new Error(`${res.status} ${res.statusText}`);
}

export async function listServers(): Promise<ServerView[]> {
	const res = await fetch('/api/servers');
	if (!res.ok) throw await apiError(res);
	return (await res.json()) ?? [];
}

export async function getServer(name: string): Promise<ServerDetail> {
	const res = await fetch(`/api/servers/${encodeURIComponent(name)}`);
	if (!res.ok) throw await apiError(res);
	return res.json();
}

export async function getLogs(name: string, tail = '200'): Promise<string[]> {
	const res = await fetch(`/api/servers/${encodeURIComponent(name)}/logs?tail=${encodeURIComponent(tail)}`);
	if (!res.ok) throw await apiError(res);
	const body = await res.json();
	return body.lines ?? [];
}

/** Subscribes to a server's live stats stream; call the returned function to unsubscribe. */
export function streamStats(name: string, onData: (s: ServerStats) => void): () => void {
	const es = new EventSource(`/api/servers/${encodeURIComponent(name)}/stats/stream`);
	es.onmessage = (ev) => {
		try {
			onData(JSON.parse(ev.data));
		} catch {
			// ignore a malformed frame rather than tearing down the stream
		}
	};
	return () => es.close();
}

/** Subscribes to a server's live log tail; call the returned function to unsubscribe. */
export function streamLogs(name: string, onLine: (line: string) => void, tail = '0'): () => void {
	const es = new EventSource(
		`/api/servers/${encodeURIComponent(name)}/logs?follow=true&tail=${encodeURIComponent(tail)}`
	);
	es.onmessage = (ev) => onLine(ev.data);
	return () => es.close();
}
