<script lang="ts">
  import type { AccessRequest } from "../../types";
  import { activeServerId } from "../../stores/servers.svelte.ts";
  import { AdminService, on } from "../../wails";

  let requests = $state<AccessRequest[]>([]);
  let loading = $state(false);

  $effect(() => {
    if (activeServerId() !== null) {
      loadRequests();
    }
  });

  $effect(() => {
    const unsubNew = on("access_request:new", (data: unknown) => {
      const req = data as AccessRequest;
      if (!requests.some(r => r.id === req.id)) {
        requests = [req, ...requests];
      }
    });
    const unsubApproved = on("event.access_request.approved", (data: unknown) => {
      const d = data as { id: string };
      requests = requests.filter(r => r.id !== d.id);
    });
    const unsubRejected = on("event.access_request.rejected", (data: unknown) => {
      const d = data as { id: string };
      requests = requests.filter(r => r.id !== d.id);
    });
    return () => {
      unsubNew();
      unsubApproved();
      unsubRejected();
    };
  });

  async function loadRequests(): Promise<void> {
    if (activeServerId() === null) return;
    loading = true;
    try {
      requests = await AdminService.GetAccessRequests(activeServerId()!);
    } catch {
      requests = [];
    } finally {
      loading = false;
    }
  }

  async function handleApprove(id: string): Promise<void> {
    if (activeServerId() === null) return;
    try {
      await AdminService.ApproveAccessRequest(activeServerId()!, id);
      requests = requests.filter(r => r.id !== id);
    } catch {
      // error
    }
  }

  async function handleReject(id: string): Promise<void> {
    if (activeServerId() === null) return;
    try {
      await AdminService.RejectAccessRequest(activeServerId()!, id);
      requests = requests.filter(r => r.id !== id);
    } catch {
      // error
    }
  }

  function formatFingerprint(hex: string): string {
    return hex.substring(0, 16).toUpperCase();
  }

  function formatTime(iso: string): string {
    const d = new Date(iso);
    return d.toLocaleDateString(undefined, { month: "short", day: "numeric" }) +
      " " +
      d.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" });
  }
</script>

<div class="request-manager">
  <div class="top-bar">
    <span class="section-title">Access Requests</span>
  </div>
  <div class="top-divider"></div>

  <div class="content-scroll">
    <div class="content-wrap">
      {#if requests.length === 0 && !loading}
        <div class="empty-state">
          <p class="empty-text">No pending access requests</p>
          <p class="empty-sub">When users request access to this server, they'll appear here.</p>
        </div>
      {/if}

      {#each requests as req (req.id)}
        <div class="request-row">
          <div class="request-info">
            <div class="request-header">
              <span class="request-name">{req.displayName}</span>
              {#if req.isOnline}
                <span class="online-dot" title="Currently waiting"></span>
              {/if}
            </div>
            <code class="request-key">{formatFingerprint(req.pubKey)}</code>
            {#if req.message}
              <p class="request-message">"{req.message}"</p>
            {/if}
            <span class="request-time">{formatTime(req.createdAt)}</span>
          </div>
          <div class="request-actions">
            <button class="approve-btn" onclick={() => handleApprove(req.id)} type="button">
              Approve
            </button>
            <button class="reject-btn" onclick={() => handleReject(req.id)} type="button">
              Reject
            </button>
          </div>
        </div>
      {/each}
    </div>
  </div>
</div>

<style>
  .request-manager {
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

  .content-scroll {
    flex: 1;
    overflow-y: auto;
    padding: 32px 48px;
  }

  .content-wrap {
    display: flex;
    flex-direction: column;
    gap: 8px;
    width: 600px;
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 24px 0;
  }

  .empty-text {
    font-size: 14px;
    font-weight: 500;
    color: var(--foreground);
    margin: 0;
  }

  .empty-sub {
    font-size: 13px;
    color: var(--muted-foreground);
    margin: 0;
  }

  .request-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px;
    border-radius: 8px;
    transition: background 0.1s;
  }

  .request-row:hover {
    background: var(--muted);
  }

  .request-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
    flex: 1;
    min-width: 0;
  }

  .request-header {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .request-name {
    font-size: 14px;
    font-weight: 600;
    color: var(--foreground);
  }

  .online-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #4ade80;
    flex-shrink: 0;
  }

  .request-key {
    font-size: 11px;
    color: var(--muted-foreground);
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
  }

  .request-message {
    font-size: 13px;
    color: var(--muted-foreground);
    font-style: italic;
    margin: 0;
  }

  .request-time {
    font-size: 11px;
    color: var(--muted-foreground);
  }

  .request-actions {
    display: flex;
    gap: 8px;
    flex-shrink: 0;
  }

  .approve-btn {
    padding: 6px 16px;
    background: var(--primary);
    border: none;
    border-radius: var(--radius);
    color: var(--primary-foreground);
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .approve-btn:hover {
    opacity: 0.9;
  }

  .reject-btn {
    padding: 6px 16px;
    background: rgba(255, 68, 68, 0.13);
    border: none;
    border-radius: var(--radius);
    color: var(--destructive);
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s;
  }

  .reject-btn:hover {
    background: rgba(255, 68, 68, 0.25);
  }
</style>
