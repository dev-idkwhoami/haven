<script lang="ts">
  import type { User, Role } from "../../types";

  interface Props {
    user: User;
    roles: Role[];
    onClose: () => void;
    onKick?: () => void;
    onBan?: () => void;
  }

  let { user, roles, onClose, onKick, onBan }: Props = $props();

  let userRoles = $derived(
    roles.filter((r) => user.roleIds?.includes(r.id)),
  );

  let truncatedKey = $derived(
    user.publicKey.length > 16
      ? user.publicKey.slice(0, 8) + "..." + user.publicKey.slice(-8)
      : user.publicKey,
  );
</script>

<div class="popover-backdrop" onclick={onClose} role="presentation">
  <div class="popover" onclick={(e) => e.stopPropagation()} role="dialog">
    <div class="popover-header">
      <div class="avatar-large">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" />
        </svg>
      </div>
      <div class="name-section">
        <span class="display-name">{user.displayName}</span>
        <code class="pubkey">{truncatedKey}</code>
      </div>
    </div>

    {#if user.bio}
      <div class="bio-section">
        <span class="bio">{user.bio}</span>
      </div>
    {/if}

    {#if userRoles.length > 0}
      <div class="roles-section">
        <span class="section-label">Roles</span>
        <div class="roles-list">
          {#each userRoles as role (role.id)}
            <span
              class="role-badge"
              style={role.color ? `border-color: ${role.color}; color: ${role.color}` : ""}
            >
              {role.name}
            </span>
          {/each}
        </div>
      </div>
    {/if}

    {#if onKick || onBan}
      <div class="actions-section">
        {#if onKick}
          <button class="action-btn" onclick={onKick} type="button">Kick</button>
        {/if}
        {#if onBan}
          <button class="action-btn destructive" onclick={onBan} type="button">Ban</button>
        {/if}
      </div>
    {/if}
  </div>
</div>

<style>
  .popover-backdrop {
    position: fixed;
    inset: 0;
    z-index: 90;
  }

  .popover {
    position: absolute;
    right: 260px;
    top: 100px;
    width: 280px;
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  }

  .popover-header {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .avatar-large {
    width: 48px;
    height: 48px;
    border-radius: 12px;
    background: var(--muted);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted-foreground);
    flex-shrink: 0;
  }

  .name-section {
    display: flex;
    flex-direction: column;
    gap: 2px;
    overflow: hidden;
  }

  .display-name {
    font-size: 16px;
    font-weight: 600;
    color: var(--foreground);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .pubkey {
    font-size: 11px;
    color: var(--muted-foreground);
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
  }

  .bio-section {
    padding-top: 4px;
    border-top: 1px solid var(--border);
  }

  .bio {
    font-size: 13px;
    color: var(--muted-foreground);
    line-height: 1.4;
  }

  .roles-section {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .section-label {
    font-size: 11px;
    font-weight: 600;
    color: var(--muted-foreground);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .roles-list {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }

  .role-badge {
    font-size: 11px;
    font-weight: 500;
    padding: 2px 8px;
    border-radius: 9999px;
    border: 1px solid var(--border);
    color: var(--foreground);
  }

  .actions-section {
    display: flex;
    gap: 8px;
    padding-top: 4px;
    border-top: 1px solid var(--border);
  }

  .action-btn {
    flex: 1;
    padding: 6px;
    background: var(--secondary);
    border: none;
    border-radius: 6px;
    color: var(--foreground);
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s;
  }

  .action-btn:hover {
    background: var(--muted);
  }

  .action-btn.destructive {
    color: var(--destructive);
  }

  .action-btn.destructive:hover {
    background: rgba(255, 102, 105, 0.15);
  }
</style>
