// Client for mcm's web API (internal/webapi), which mirrors the CLI's
// commands as JSON plus SSE streams for live logs and stats. See
// RESEARCH.md §7.7 / plan M12/M13/M14. Write calls go through
// authedFetch, which attaches the bearer token from lib/auth.svelte.ts.

import { auth } from './auth.svelte';

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

function authedFetch(input: string, init: RequestInit = {}): Promise<Response> {
	const headers = new Headers(init.headers);
	if (auth.token) headers.set('Authorization', `Bearer ${auth.token}`);
	return fetch(input, { ...init, headers });
}

function authedJSON(input: string, method: string, body?: unknown): Promise<Response> {
	return authedFetch(input, {
		method,
		headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
		body: body === undefined ? undefined : JSON.stringify(body)
	});
}

/** Checks whether token is a valid write-action token, without side effects on api.ts's own auth state. */
export async function verifyToken(token: string): Promise<boolean> {
	const res = await fetch('/api/auth/verify', {
		method: 'POST',
		headers: { Authorization: `Bearer ${token}` }
	});
	return res.ok;
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

const encName = (name: string) => encodeURIComponent(name);

async function ok(res: Response): Promise<void> {
	if (!res.ok) throw await apiError(res);
}
async function okJSON<T>(res: Response): Promise<T> {
	if (!res.ok) throw await apiError(res);
	return res.json();
}

export interface CreateServerRequest {
	name: string;
	type?: string;
	version?: string;
	memory?: string;
	port?: number;
	accept_eula: boolean;
}

export async function createServer(req: CreateServerRequest): Promise<void> {
	await ok(await authedJSON('/api/servers', 'POST', req));
}

export async function destroyServer(name: string, purge: boolean): Promise<void> {
	await ok(await authedFetch(`/api/servers/${encName(name)}?purge=${purge}`, { method: 'DELETE' }));
}

export async function startServer(name: string): Promise<void> {
	await ok(await authedJSON(`/api/servers/${encName(name)}/start`, 'POST'));
}

export async function stopServer(name: string): Promise<void> {
	await ok(await authedJSON(`/api/servers/${encName(name)}/stop`, 'POST'));
}

export async function execCommand(name: string, command: string): Promise<string> {
	const body = await okJSON<{ output: string }>(
		await authedJSON(`/api/servers/${encName(name)}/exec`, 'POST', { command })
	);
	return body.output;
}

export interface WhitelistEntry {
	uuid: string;
	name: string;
}

export async function getWhitelist(name: string): Promise<WhitelistEntry[]> {
	return (await okJSON<WhitelistEntry[] | null>(await fetch(`/api/servers/${encName(name)}/whitelist`))) ?? [];
}
export async function addWhitelist(name: string, player: string, offline: boolean): Promise<void> {
	await ok(await authedJSON(`/api/servers/${encName(name)}/whitelist`, 'POST', { player, offline }));
}
export async function removeWhitelist(name: string, player: string): Promise<void> {
	await ok(
		await authedFetch(`/api/servers/${encName(name)}/whitelist/${encodeURIComponent(player)}`, {
			method: 'DELETE'
		})
	);
}

export interface OpEntry {
	uuid: string;
	name: string;
	level: number;
	bypassesPlayerLimit: boolean;
}

export async function getOps(name: string): Promise<OpEntry[]> {
	return (await okJSON<OpEntry[] | null>(await fetch(`/api/servers/${encName(name)}/ops`))) ?? [];
}
export async function addOp(name: string, player: string, offline: boolean): Promise<void> {
	await ok(await authedJSON(`/api/servers/${encName(name)}/ops`, 'POST', { player, offline }));
}
export async function removeOp(name: string, player: string): Promise<void> {
	await ok(
		await authedFetch(`/api/servers/${encName(name)}/ops/${encodeURIComponent(player)}`, { method: 'DELETE' })
	);
}

export interface ModsInfo {
	modrinth_projects?: string[];
	curseforge_files?: string[];
	mod_urls?: string[];
	plugin_urls?: string[];
}

export async function getMods(name: string): Promise<ModsInfo> {
	return okJSON<ModsInfo>(await fetch(`/api/servers/${encName(name)}/mods`));
}
export async function addMod(name: string, source: string, ref: string): Promise<void> {
	await ok(await authedJSON(`/api/servers/${encName(name)}/mods`, 'POST', { source, ref }));
}

export interface ModpackStatus {
	source?: string;
	ref?: string;
	type: string;
}

export async function getModpack(name: string): Promise<ModpackStatus> {
	return okJSON<ModpackStatus>(await fetch(`/api/servers/${encName(name)}/modpack`));
}
export async function installModpack(name: string, source: string, ref: string): Promise<void> {
	await ok(await authedJSON(`/api/servers/${encName(name)}/modpack`, 'POST', { source, ref }));
}

export interface BackupInfo {
	id: string;
	size_bytes: number;
	created_at: string;
}

export async function getBackups(name: string): Promise<BackupInfo[]> {
	return (await okJSON<BackupInfo[] | null>(await fetch(`/api/servers/${encName(name)}/backups`))) ?? [];
}
export async function createBackup(name: string): Promise<BackupInfo> {
	return okJSON<BackupInfo>(await authedJSON(`/api/servers/${encName(name)}/backup`, 'POST'));
}
export async function restoreBackup(name: string, backupId: string): Promise<void> {
	await ok(await authedJSON(`/api/servers/${encName(name)}/restore`, 'POST', { backup_id: backupId }));
}
