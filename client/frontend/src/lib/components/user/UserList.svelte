<script lang="ts">
  import type { User, Role } from "../../types";
  import { users } from "../../stores/users.svelte.ts";
  import { roles } from "../../stores/roles.svelte.ts";
  import UserCard from "./UserCard.svelte";
  import UserProfilePopover from "./UserProfilePopover.svelte";

  let selectedUser = $state<User | null>(null);

  let sortedRoles = $derived(
    [...roles()].sort((a, b) => a.position - b.position),
  );

  let onlineUsers = $derived(
    users().filter((u) => u.status && u.status !== "offline"),
  );

  let offlineUsers = $derived(
    users().filter((u) => !u.status || u.status === "offline"),
  );

  function getHighestRoleColor(user: User): string | null {
    if (!user.roleIds || user.roleIds.length === 0) return null;
    for (const role of sortedRoles) {
      if (user.roleIds.includes(role.id) && role.color) {
        return role.color;
      }
    }
    return null;
  }

  function getUserRoleForGroup(user: User): Role | null {
    if (!user.roleIds || user.roleIds.length === 0) return null;
    for (const role of sortedRoles) {
      if (user.roleIds.includes(role.id) && !role.isDefault) {
        return role;
      }
    }
    return null;
  }

  interface RoleGroup {
    role: Role | null;
    label: string;
    users: User[];
  }

  let groupedOnlineUsers = $derived(() => {
    const groups: RoleGroup[] = [];
    const ungrouped: User[] = [];

    for (const user of onlineUsers) {
      const role = getUserRoleForGroup(user);
      if (role) {
        let group = groups.find((g) => g.role?.id === role.id);
        if (!group) {
          group = { role, label: role.name, users: [] };
          groups.push(group);
        }
        group.users.push(user);
      } else {
        ungrouped.push(user);
      }
    }

    groups.sort((a, b) => (a.role?.position ?? 999) - (b.role?.position ?? 999));
    if (ungrouped.length > 0) {
      groups.push({ role: null, label: "Online", users: ungrouped });
    }
    return groups;
  });
</script>

<div class="user-list">
  <div class="user-list-header">
    <span class="header-label">Members — {users().length}</span>
  </div>

  <div class="user-list-scroll">
    {#each groupedOnlineUsers() as group (group.label)}
      <div class="role-group">
        <span class="group-label" style={group.role?.color ? `color: ${group.role.color}` : ""}>
          {group.label} — {group.users.length}
        </span>
        {#each group.users as user (user.publicKey)}
          <UserCard
            {user}
            roleColor={getHighestRoleColor(user)}
            onclick={() => (selectedUser = user)}
          />
        {/each}
      </div>
    {/each}

    {#if offlineUsers.length > 0}
      <div class="role-group">
        <span class="group-label">Offline — {offlineUsers.length}</span>
        {#each offlineUsers as user (user.publicKey)}
          <UserCard
            {user}
            roleColor={getHighestRoleColor(user)}
            onclick={() => (selectedUser = user)}
          />
        {/each}
      </div>
    {/if}
  </div>
</div>

{#if selectedUser}
  <UserProfilePopover
    user={selectedUser}
    roles={roles()}
    onClose={() => (selectedUser = null)}
  />
{/if}

<style>
  .user-list {
    display: flex;
    flex-direction: column;
    width: 240px;
    height: 100%;
    background: var(--background);
    border-left: 1px solid var(--border);
    flex-shrink: 0;
  }

  .user-list-header {
    display: flex;
    align-items: center;
    height: 72px;
    padding: 0 16px;
    flex-shrink: 0;
  }

  .header-label {
    font-size: 13px;
    font-weight: 600;
    color: var(--muted-foreground);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .user-list-scroll {
    flex: 1;
    overflow-y: auto;
    padding: 0 8px 12px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .role-group {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .group-label {
    font-size: 11px;
    font-weight: 600;
    color: var(--muted-foreground);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    padding: 8px 12px 4px;
  }
</style>
