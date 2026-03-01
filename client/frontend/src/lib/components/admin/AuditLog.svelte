<script lang="ts">
  import type { AuditLogEntry } from "../../types";
  import { activeServerId } from "../../stores/servers.svelte.ts";
  import { users } from "../../stores/users.svelte.ts";
  import { AdminService } from "../../wails";

  let entries = $state<AuditLogEntry[]>([]);
  let loading = $state(false);
  let hasMore = $state(true);

  $effect(() => {
    if (activeServerId() !== null) {
      loadEntries();
    }
  });

  async function loadEntries(): Promise<void> {
    if (activeServerId() === null) return;
    loading = true;
    try {
      entries = await AdminService.GetAuditLog(activeServerId()!, "", 50);
      hasMore = entries.length >= 50;
    } catch {
      entries = [];
    } finally {
      loading = false;
    }
  }

  async function loadMore(): Promise<void> {
    if (activeServerId() === null || !hasMore || loading) return;
    const lastId = entries[entries.length - 1]?.id ?? "";
    loading = true;
    try {
      const batch = await AdminService.GetAuditLog(activeServerId()!, lastId, 50);
      entries = [...entries, ...batch];
      hasMore = batch.length >= 50;
    } catch {
      // ignore
    } finally {
      loading = false;
    }
  }

  function getActorName(pubKey: string): string {
    const user = users().find((u) => u.publicKey === pubKey);
    return user?.displayName ?? pubKey.slice(0, 8) + "...";
  }

  function formatTime(iso: string): string {
    const d = new Date(iso);
    return d.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" }) +
      " " +
      d.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" });
  }

  function formatAction(action: string): string {
    return action
      .replace(/([A-Z])/g, " $1")
      .replace(/^./, (s) => s.toUpperCase())
      .trim();
  }
</script>

<div class="audit-log">
  <div class="top-bar">
    <span class="section-title">Audit Log</span>
  </div>
  <div class="top-divider"></div>

  <div class="log-scroll">
    {#if entries.length === 0 && !loading}
      <div class="empty-state">
        <p class="empty-text">No audit log entries</p>
      </div>
    {/if}

    {#each entries as entry (entry.id)}
      <div class="log-entry">
        <div class="entry-header">
          <span class="actor-name">{getActorName(entry.actorPubKey)}</span>
          <span class="entry-action">{formatAction(entry.action)}</span>
        </div>
        <div class="entry-details">
          <span class="target-info">{entry.targetType}: {entry.targetId.slice(0, 8)}</span>
          {#if entry.details}
            <span class="detail-text">{entry.details}</span>
          {/if}
        </div>
        <span class="entry-time">{formatTime(entry.createdAt)}</span>
      </div>
    {/each}

    {#if hasMore && entries.length > 0}
      <button class="load-more-btn" onclick={loadMore} disabled={loading} type="button">
        {loading ? "Loading..." : "Load More"}
      </button>
    {/if}
  </div>
</div>

<style>
  .audit-log {
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

  .top-divider {
    height: 1px;
    background: var(--border);
    flex-shrink: 0;
  }

  .log-scroll {
    flex: 1;
    overflow-y: auto;
    padding: 16px 24px;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .log-entry {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 12px 16px;
    border-radius: 6px;
    transition: background 0.1s;
  }

  .log-entry:hover {
    background: var(--muted);
  }

  .entry-header {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .actor-name {
    font-size: 13px;
    font-weight: 600;
    color: var(--foreground);
  }

  .entry-action {
    font-size: 13px;
    color: var(--muted-foreground);
  }

  .entry-details {
    display: flex;
    gap: 8px;
    font-size: 12px;
    color: var(--muted-foreground);
  }

  .target-info {
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
    font-size: 11px;
    padding: 1px 6px;
    background: var(--muted);
    border-radius: 4px;
  }

  .detail-text {
    font-size: 12px;
  }

  .entry-time {
    font-size: 11px;
    color: var(--muted-foreground);
  }

  .empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 48px;
  }

  .empty-text {
    color: var(--muted-foreground);
    font-size: 14px;
    margin: 0;
  }

  .load-more-btn {
    align-self: center;
    padding: 8px 24px;
    background: var(--secondary);
    border: none;
    border-radius: var(--radius);
    color: var(--foreground);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    margin-top: 8px;
    transition: background 0.15s;
  }

  .load-more-btn:hover:not(:disabled) {
    background: var(--muted);
  }

  .load-more-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
