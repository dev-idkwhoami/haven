import type { User } from "../types";
import { UserService, on } from "../wails";

let _users = $state<User[]>([]);

export function users() { return _users; }

on("user:updated", (data: unknown) => {
  const user = data as User;
  _users = _users.map((u) => (u.publicKey === user.publicKey ? user : u));
});

on("user:joined", (data: unknown) => {
  const user = data as User;
  if (!_users.some((u) => u.publicKey === user.publicKey)) {
    _users = [..._users, user];
  }
});

on("user:kicked", (data: unknown) => {
  const d = data as { publicKey: string };
  _users = _users.filter((u) => u.publicKey !== d.publicKey);
});

on("user:banned", (data: unknown) => {
  const d = data as { publicKey: string };
  _users = _users.filter((u) => u.publicKey !== d.publicKey);
});

on("user:roleAdded", (data: unknown) => {
  const d = data as { publicKey: string; roleId: string };
  _users = _users.map((u) => {
    if (u.publicKey === d.publicKey) {
      return { ...u, roleIds: [...(u.roleIds ?? []), d.roleId] };
    }
    return u;
  });
});

on("user:roleRemoved", (data: unknown) => {
  const d = data as { publicKey: string; roleId: string };
  _users = _users.map((u) => {
    if (u.publicKey === d.publicKey) {
      return { ...u, roleIds: (u.roleIds ?? []).filter((r) => r !== d.roleId) };
    }
    return u;
  });
});

export async function loadUsers(serverId: number): Promise<void> {
  try {
    _users = await UserService.GetUsers(serverId);
  } catch {
    _users = [];
  }
}

export async function getUser(serverId: number, publicKey: string): Promise<User> {
  return UserService.GetUser(serverId, publicKey);
}

export async function kickUser(serverId: number, publicKey: string, reason: string): Promise<void> {
  await UserService.KickUser(serverId, publicKey, reason);
}

export async function banUser(serverId: number, publicKey: string, reason: string): Promise<void> {
  await UserService.BanUser(serverId, publicKey, reason);
}

export async function unbanUser(serverId: number, publicKey: string): Promise<void> {
  await UserService.UnbanUser(serverId, publicKey);
}

export async function getBans(serverId: number): Promise<User[]> {
  return UserService.GetBans(serverId);
}
