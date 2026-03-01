import type { ServerEntry, ServerInfo, ServerHello, ConnectionStatus } from "../types";
import { ServerService, on } from "../wails";
import { setOwner } from "./permissions.svelte.ts";

let _servers = $state<ServerEntry[]>([]);
let _relayServers = $state<ServerEntry[]>([]);
let _activeServerId = $state<number | null>(null);
let _connectionStatus = $state<Record<number, ConnectionStatus>>({});
let _activeServerInfo = $state<ServerInfo | null>(null);
let _showAccessRequest = $state(false);

export function servers() { return _servers; }
export function relayServers() { return _relayServers; }
export function activeServerId() { return _activeServerId; }
export function connectionStatus() { return _connectionStatus; }
export function activeServerInfo() { return _activeServerInfo; }
export function showAccessRequest() { return _showAccessRequest; }
export function clearAccessRequest() { _showAccessRequest = false; }

on("server:connected", (data: unknown) => {
  const entry = data as ServerEntry;
  _connectionStatus[entry.id] = "connected";
  setOwner(entry.isOwner ?? false);
  const exists = _servers.some((s) => s.id === entry.id);
  if (!exists) {
    _servers = [..._servers, entry];
  } else {
    _servers = _servers.map((s) => (s.id === entry.id ? entry : s));
  }
});

on("server:disconnected", (data: unknown) => {
  const d = data as { serverID: number };
  _connectionStatus[d.serverID] = "disconnected";
});

on("server:reconnecting", (data: unknown) => {
  const d = data as { serverID: number; attempt: number };
  _connectionStatus[d.serverID] = "reconnecting";
});

on("server:updated", (data: unknown) => {
  const info = data as ServerInfo;
  _activeServerInfo = info;
});

export async function loadServers(): Promise<void> {
  try {
    _servers = await ServerService.GetServers();
    _relayServers = await ServerService.GetRelayServers();
  } catch {
    // Not available yet
  }
}

export async function connect(address: string): Promise<ServerHello> {
  return ServerService.Connect(address);
}

export async function trustAndAuth(accessToken: string): Promise<void> {
  await ServerService.TrustAndAuth(accessToken);
}

export async function rejectTrust(): Promise<void> {
  await ServerService.RejectTrust();
}

export async function submitAccessRequest(message: string): Promise<void> {
  await ServerService.SubmitAccessRequest(message);
}

export async function cancelAccessRequest(): Promise<void> {
  await ServerService.CancelAccessRequest();
}

export async function disconnect(serverId: number): Promise<void> {
  await ServerService.Disconnect(serverId);
}

export async function reconnect(serverId: number): Promise<void> {
  try {
    await ServerService.Reconnect(serverId);
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    if (msg.includes("waiting_room")) {
      _showAccessRequest = true;
      return;
    }
    throw e;
  }
}

export async function leaveServer(serverId: number, mode: string = "ghost"): Promise<void> {
  await ServerService.LeaveServer(serverId, mode);
  _servers = _servers.filter((s) => s.id !== serverId);
  if (_activeServerId === serverId) {
    _activeServerId = _servers.length > 0 ? _servers[0].id : null;
  }
}

export async function removeRelay(serverId: number): Promise<void> {
  await ServerService.RemoveRelay(serverId);
  _relayServers = _relayServers.filter((s) => s.id !== serverId);
}

export function setActiveServer(serverId: number | null): void {
  _activeServerId = serverId;
  _activeServerInfo = null;
  // Tell backend which server the user is viewing, so reconnect can be more aggressive.
  ServerService.SetFocusedServer(serverId ?? 0).catch(() => {});
  if (serverId !== null) {
    ServerService.GetServerInfo(serverId)
      .then((info) => {
        if (_activeServerId === serverId) {
          _activeServerInfo = info;
        }
      })
      .catch(() => {});
  }
}
