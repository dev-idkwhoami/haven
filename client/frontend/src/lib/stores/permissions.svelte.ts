import { ProfileService, on } from "../wails";
import { users } from "./users.svelte.ts";
import { roles } from "./roles.svelte.ts";

// Permission bitfield flags — mirrors shared/permissions.go
export const PERM_MANAGE_SERVER   = 1 << 0;
export const PERM_MANAGE_CHANNELS = 1 << 1;
export const PERM_MANAGE_ROLES    = 1 << 2;
export const PERM_MANAGE_MESSAGES = 1 << 3;
export const PERM_KICK_USERS      = 1 << 4;
export const PERM_BAN_USERS       = 1 << 5;
export const PERM_MANAGE_INVITES  = 1 << 6;
export const PERM_SEND_MESSAGES   = 1 << 7;
export const PERM_ATTACH_FILES    = 1 << 8;
export const PERM_JOIN_VOICE      = 1 << 9;
export const PERM_SPEAK           = 1 << 10;

let _myPubKey = $state<string | null>(null);
let _isOwner = $state(false);

export function myPublicKey() { return _myPubKey; }
export function isOwner() { return _isOwner; }

export function setOwner(val: boolean): void {
  _isOwner = val;
}

export async function loadMyPublicKey(): Promise<void> {
  try {
    _myPubKey = await ProfileService.GetPublicKey();
  } catch {
    _myPubKey = null;
  }
}

/**
 * Computes the effective permission bitfield for the current user
 * by ORing together all permissions from their assigned roles.
 */
export function myPermissions(): number {
  if (!_myPubKey) return 0;
  const me = users().find((u) => u.publicKey === _myPubKey);
  if (!me?.roleIds?.length) return 0;

  const allRoles = roles();
  let effective = 0;
  for (const rid of me.roleIds) {
    const role = allRoles.find((r) => r.id === rid);
    if (role) effective |= role.permissions;
  }
  return effective;
}

export function hasPermission(flag: number): boolean {
  if (_isOwner) return true;
  return (myPermissions() & flag) !== 0;
}

// Listen for hot-reload owner status changes from the server.
// event.owner.changed -> owner:changed (Wails event mapping)
on("owner:changed", (data: unknown) => {
  const d = data as { is_owner: boolean };
  _isOwner = d.is_owner;
});
