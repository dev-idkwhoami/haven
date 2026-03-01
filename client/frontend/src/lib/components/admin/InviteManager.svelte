<script lang="ts">
  import type { Invite } from "../../types";
  import { activeServerId } from "../../stores/servers.svelte.ts";
  import { AdminService } from "../../wails";

  let invites = $state<Invite[]>([]);
  let loading = $state(false);
  let maxUses = $state(0);
  let expiresInHours = $state(24);
  let creating = $state(false);

  $effect(() => {
    if (activeServerId() !== null) {
      loadInvites();
    }
  });

  async function loadInvites(): Promise<void> {
    if (activeServerId() === null) return;
    loading = true;
    try {
      invites = await AdminService.GetInvites(activeServerId()!);
    } catch {
      invites = [];
    } finally {
      loading = false;
    }
  }

  async function handleCreate(): Promise<void> {
    if (activeServerId() === null || creating) return;
    creating = true;
    try {
      const invite = await AdminService.CreateInvite(activeServerId()!, maxUses, expiresInHours);
      invites = [invite, ...invites];
      maxUses = 0;
      expiresInHours = 24;
    } catch {
      // error
    } finally {
      creating = false;
    }
  }

  async function handleRevoke(inviteId: string): Promise<void> {
    if (activeServerId() === null) return;
    try {
      await AdminService.RevokeInvite(activeServerId()!, inviteId);
      invites = invites.filter((i) => i.id !== inviteId);
    } catch {
      // error
    }
  }

  function formatExpiry(expiresAt: string | null): string {
    if (!expiresAt) return "Never";
    const d = new Date(expiresAt);
    if (d < new Date()) return "Expired";
    return d.toLocaleDateString(undefined, { month: "short", day: "numeric" }) +
      " " +
      d.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" });
  }
</script>

<div class="invite-manager">
  <div class="top-bar">
    <span class="section-title">Invites</span>
  </div>
  <div class="top-divider"></div>

  <div class="content-scroll">
    <div class="content-wrap">
      <div class="create-section">
        <h3 class="label">Create Invite</h3>
        <div class="create-row">
          <div class="field-group">
            <label class="field-label">Max Uses (0 = unlimited)</label>
            <input class="field-input" type="number" bind:value={maxUses} min="0" />
          </div>
          <div class="field-group">
            <label class="field-label">Expires In (hours)</label>
            <input class="field-input" type="number" bind:value={expiresInHours} min="0" />
          </div>
          <button class="create-btn" onclick={handleCreate} disabled={creating} type="button">
            {creating ? "Creating..." : "Create"}
          </button>
        </div>
      </div>

      <div class="divider"></div>

      <div class="list-section">
        <h3 class="label">Active Invites</h3>
        {#if invites.length === 0 && !loading}
          <p class="empty-text">No active invites</p>
        {/if}

        {#each invites as invite (invite.id)}
          <div class="invite-row">
            <div class="invite-info">
              <code class="invite-code">{invite.code}</code>
              <div class="invite-meta">
                <span>Uses: {invite.useCount}{invite.maxUses > 0 ? ` / ${invite.maxUses}` : ""}</span>
                <span>Expires: {formatExpiry(invite.expiresAt)}</span>
              </div>
            </div>
            <button class="revoke-btn" onclick={() => handleRevoke(invite.id)} type="button">
              Revoke
            </button>
          </div>
        {/each}
      </div>
    </div>
  </div>
</div>

<style>
  .invite-manager {
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

  .content-scroll {
    flex: 1;
    overflow-y: auto;
    padding: 32px 48px;
  }

  .content-wrap {
    display: flex;
    flex-direction: column;
    gap: 32px;
    width: 600px;
  }

  .create-section,
  .list-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .label {
    font-size: 14px;
    font-weight: 600;
    color: var(--foreground);
    margin: 0;
  }

  .create-row {
    display: flex;
    gap: 12px;
    align-items: flex-end;
  }

  .field-group {
    display: flex;
    flex-direction: column;
    gap: 4px;
    flex: 1;
  }

  .field-label {
    font-size: 12px;
    font-weight: 500;
    color: var(--muted-foreground);
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

  .create-btn {
    height: 36px;
    padding: 0 20px;
    background: var(--primary);
    border: none;
    border-radius: var(--radius);
    color: var(--primary-foreground);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    white-space: nowrap;
    transition: opacity 0.15s;
  }

  .create-btn:hover:not(:disabled) {
    opacity: 0.9;
  }

  .create-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .empty-text {
    font-size: 13px;
    color: var(--muted-foreground);
    margin: 0;
  }

  .invite-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 12px;
    border-radius: 6px;
    transition: background 0.1s;
  }

  .invite-row:hover {
    background: var(--muted);
  }

  .invite-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .invite-code {
    font-size: 14px;
    font-weight: 600;
    color: var(--foreground);
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
  }

  .invite-meta {
    display: flex;
    gap: 12px;
    font-size: 12px;
    color: var(--muted-foreground);
  }

  .revoke-btn {
    padding: 4px 12px;
    background: rgba(255, 68, 68, 0.13);
    border: none;
    border-radius: 4px;
    color: var(--destructive);
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s;
  }

  .revoke-btn:hover {
    background: rgba(255, 68, 68, 0.25);
  }
</style>
