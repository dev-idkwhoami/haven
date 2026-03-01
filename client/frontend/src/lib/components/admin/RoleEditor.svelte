<script lang="ts">
  import type { Role } from "../../types";
  import { activeServerId } from "../../stores/servers.svelte.ts";
  import { roles, createRole, updateRole, deleteRole } from "../../stores/roles.svelte.ts";

  let selectedRoleId = $state<string | null>(null);
  let editName = $state("");
  let editColor = $state("#ffffff");
  let editPermissions = $state(0);
  let saving = $state(false);
  let creating = $state(false);

  let sortedRoles = $derived(
    [...roles()].sort((a, b) => a.position - b.position),
  );

  let selectedRole = $derived(
    selectedRoleId ? roles().find((r) => r.id === selectedRoleId) : null,
  );

  $effect(() => {
    if (selectedRole) {
      editName = selectedRole.name;
      editColor = selectedRole.color ?? "#ffffff";
      editPermissions = selectedRole.permissions;
    }
  });

  const permissionLabels: [number, string][] = [
    [1, "Administrator"],
    [2, "Manage Channels"],
    [4, "Manage Roles"],
    [8, "Kick Members"],
    [16, "Ban Members"],
    [32, "Manage Messages"],
    [64, "Manage Server"],
    [128, "Send Messages"],
    [256, "Read Messages"],
    [512, "Connect Voice"],
    [1024, "Speak Voice"],
    [2048, "Create Invites"],
  ];

  function hasPermission(perm: number): boolean {
    return (editPermissions & perm) !== 0;
  }

  function togglePermission(perm: number): void {
    if (hasPermission(perm)) {
      editPermissions &= ~perm;
    } else {
      editPermissions |= perm;
    }
  }

  async function handleSave(): Promise<void> {
    if (activeServerId() === null || !selectedRole || !editName.trim()) return;
    saving = true;
    try {
      await updateRole(
        activeServerId()!,
        selectedRole.id,
        editName.trim(),
        editColor,
        selectedRole.position,
        editPermissions,
      );
    } catch {
      // error
    } finally {
      saving = false;
    }
  }

  async function handleCreate(): Promise<void> {
    if (activeServerId() === null || creating) return;
    creating = true;
    try {
      await createRole(activeServerId()!, "New Role", "#888888", 0);
    } catch {
      // error
    } finally {
      creating = false;
    }
  }

  async function handleDelete(): Promise<void> {
    if (activeServerId() === null || !selectedRoleId) return;
    try {
      await deleteRole(activeServerId()!, selectedRoleId);
      selectedRoleId = null;
    } catch {
      // error
    }
  }
</script>

<div class="role-editor">
  <div class="top-bar">
    <span class="section-title">Roles</span>
  </div>
  <div class="top-divider"></div>

  <div class="editor-layout">
    <div class="role-list-panel">
      <button class="create-role-btn" onclick={handleCreate} disabled={creating} type="button">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 5v14" /><path d="M5 12h14" />
        </svg>
        {creating ? "Creating..." : "Create Role"}
      </button>

      <div class="role-items">
        {#each sortedRoles as role (role.id)}
          <button
            class="role-item"
            class:active={role.id === selectedRoleId}
            onclick={() => (selectedRoleId = role.id)}
            type="button"
          >
            <div class="role-dot" style={role.color ? `background: ${role.color}` : ""}></div>
            <span class="role-name">{role.name}</span>
            {#if role.isDefault}
              <span class="default-badge">Default</span>
            {/if}
          </button>
        {/each}
      </div>
    </div>

    <div class="role-detail-panel">
      {#if selectedRole}
        <div class="detail-section">
          <div class="field-group">
            <label class="field-label">Role Name</label>
            <input class="field-input" bind:value={editName} placeholder="Role name" />
          </div>

          <div class="field-group">
            <label class="field-label">Color</label>
            <div class="color-row">
              <input class="color-picker" type="color" bind:value={editColor} />
              <input class="field-input color-hex" bind:value={editColor} placeholder="#ffffff" />
            </div>
          </div>
        </div>

        <div class="divider"></div>

        <div class="detail-section">
          <h4 class="perm-title">Permissions</h4>
          <div class="perm-list">
            {#each permissionLabels as [perm, label]}
              <label class="perm-row">
                <input
                  type="checkbox"
                  class="perm-checkbox"
                  checked={hasPermission(perm)}
                  onchange={() => togglePermission(perm)}
                />
                <span class="perm-label">{label}</span>
              </label>
            {/each}
          </div>
        </div>

        <div class="divider"></div>

        <div class="action-row">
          {#if !selectedRole.isDefault}
            <button class="delete-btn" onclick={handleDelete} type="button">Delete Role</button>
          {/if}
          <div class="spacer"></div>
          <button class="save-btn" onclick={handleSave} disabled={saving} type="button">
            {saving ? "Saving..." : "Save Changes"}
          </button>
        </div>
      {:else}
        <div class="no-selection">
          <p class="no-text">Select a role to edit</p>
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .role-editor {
    display: flex;
    flex-direction: column;
    height: 100%;
    width: 100%;
  }

  .top-bar {
    display: flex;
    align-items: center;
    height: 72px;
    padding: 0 24px;
    flex-shrink: 0;
  }

  .section-title {
    font-size: 16px;
    font-weight: 600;
    color: var(--foreground);
  }

  .top-divider, .divider {
    height: 1px;
    background: var(--border);
    flex-shrink: 0;
  }

  .editor-layout {
    display: flex;
    flex: 1;
    overflow: hidden;
  }

  .role-list-panel {
    width: 240px;
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    padding: 12px 8px;
    gap: 8px;
    flex-shrink: 0;
    overflow-y: auto;
  }

  .create-role-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 32px;
    padding: 0 12px;
    background: none;
    border: 1px dashed var(--border);
    border-radius: 6px;
    color: var(--muted-foreground);
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: border-color 0.15s, color 0.15s;
  }

  .create-role-btn:hover:not(:disabled) {
    border-color: var(--foreground);
    color: var(--foreground);
  }

  .create-role-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .role-items {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .role-item {
    display: flex;
    align-items: center;
    gap: 8px;
    height: 32px;
    padding: 0 8px;
    background: none;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    text-align: left;
    transition: background 0.1s;
    width: 100%;
  }

  .role-item:hover {
    background: var(--muted);
  }

  .role-item.active {
    background: var(--muted);
  }

  .role-dot {
    width: 10px;
    height: 10px;
    border-radius: 9999px;
    background: var(--muted-foreground);
    flex-shrink: 0;
  }

  .role-name {
    font-size: 13px;
    font-weight: 500;
    color: var(--foreground);
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .default-badge {
    font-size: 10px;
    color: var(--muted-foreground);
    padding: 1px 6px;
    background: var(--muted);
    border-radius: 4px;
  }

  .role-detail-panel {
    flex: 1;
    overflow-y: auto;
    padding: 24px 32px;
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .detail-section {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .field-group {
    display: flex;
    flex-direction: column;
    gap: 4px;
    max-width: 400px;
  }

  .field-label {
    font-size: 13px;
    font-weight: 500;
    color: var(--foreground);
  }

  .field-input {
    height: 36px;
    padding: 0 12px;
    background: var(--muted);
    border: 1px solid transparent;
    border-radius: 6px;
    color: var(--foreground);
    font-size: 14px;
    font-family: inherit;
    outline: none;
  }

  .field-input:focus {
    border-color: var(--ring);
  }

  .color-row {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .color-picker {
    width: 36px;
    height: 36px;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    padding: 0;
    background: none;
  }

  .color-hex {
    width: 120px;
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
    font-size: 13px;
  }

  .perm-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--foreground);
    margin: 0;
  }

  .perm-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .perm-row {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
  }

  .perm-checkbox {
    width: 16px;
    height: 16px;
    accent-color: var(--primary);
    cursor: pointer;
  }

  .perm-label {
    font-size: 13px;
    color: var(--foreground);
  }

  .action-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .spacer {
    flex: 1;
  }

  .delete-btn {
    padding: 8px 16px;
    background: rgba(255, 68, 68, 0.13);
    border: none;
    border-radius: var(--radius);
    color: var(--destructive);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s;
  }

  .delete-btn:hover {
    background: rgba(255, 68, 68, 0.25);
  }

  .save-btn {
    padding: 8px 16px;
    background: var(--primary);
    border: none;
    border-radius: var(--radius);
    color: var(--primary-foreground);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .save-btn:hover:not(:disabled) {
    opacity: 0.9;
  }

  .save-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .no-selection {
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 1;
  }

  .no-text {
    color: var(--muted-foreground);
    font-size: 14px;
    margin: 0;
  }
</style>
