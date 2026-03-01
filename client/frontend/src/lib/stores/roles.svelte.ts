import type { Role } from "../types";
import { RoleService, on } from "../wails";

let _roles = $state<Role[]>([]);

export function roles() { return _roles; }

on("role:created", (data: unknown) => {
  const role = data as Role;
  _roles = [..._roles, role];
});

on("role:updated", (data: unknown) => {
  const role = data as Role;
  _roles = _roles.map((r) => (r.id === role.id ? role : r));
});

on("role:deleted", (data: unknown) => {
  const d = data as { roleId: string };
  _roles = _roles.filter((r) => r.id !== d.roleId);
});

export async function loadRoles(serverId: number): Promise<void> {
  try {
    _roles = await RoleService.GetRoles(serverId);
  } catch {
    _roles = [];
  }
}

export async function createRole(
  serverId: number,
  name: string,
  color: string,
  permissions: number,
): Promise<void> {
  await RoleService.CreateRole(serverId, name, color, permissions);
}

export async function updateRole(
  serverId: number,
  roleId: string,
  name: string,
  color: string,
  position: number,
  permissions: number,
): Promise<void> {
  await RoleService.UpdateRole(serverId, roleId, name, color, position, permissions);
}

export async function deleteRole(serverId: number, roleId: string): Promise<void> {
  await RoleService.DeleteRole(serverId, roleId);
}

export async function assignRole(serverId: number, publicKey: string, roleId: string): Promise<void> {
  await RoleService.AssignRole(serverId, publicKey, roleId);
}

export async function revokeRole(serverId: number, publicKey: string, roleId: string): Promise<void> {
  await RoleService.RevokeRole(serverId, publicKey, roleId);
}
