<script lang="ts">
  import { relayServers, removeRelay } from "../../stores/servers.svelte.ts";
</script>

<div class="relay-servers">
  <div class="top-bar">
    <span class="section-title">Relay Servers</span>
  </div>
  <div class="top-divider"></div>

  <div class="content-scroll">
    <div class="content-wrap">
      <div class="section">
        <h3 class="section-label">DM Relay Servers</h3>
        <p class="section-desc">
          Relay servers are used to forward encrypted DM messages. They cannot read your messages.
        </p>

        {#if relayServers().length === 0}
          <p class="empty-text">No relay servers configured</p>
        {/if}

        {#each relayServers() as relay (relay.id)}
          <div class="relay-row">
            <div class="relay-info">
              <span class="relay-name">{relay.name || relay.address}</span>
              <span class="relay-address">{relay.address}</span>
            </div>
            <div class="relay-status" class:connected={relay.connected}>
              {relay.connected ? "Connected" : "Disconnected"}
            </div>
            <button class="remove-btn" onclick={() => removeRelay(relay.id)} type="button">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M18 6 6 18" /><path d="m6 6 12 12" />
              </svg>
            </button>
          </div>
        {/each}
      </div>
    </div>
  </div>
</div>

<style>
  .relay-servers {
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
    gap: 32px;
    width: 600px;
  }

  .section {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .section-label {
    font-size: 14px;
    font-weight: 600;
    color: var(--foreground);
    margin: 0;
  }

  .section-desc {
    font-size: 13px;
    color: var(--muted-foreground);
    margin: 0;
    line-height: 1.4;
  }

  .empty-text {
    font-size: 13px;
    color: var(--muted-foreground);
  }

  .relay-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 12px;
    border-radius: 6px;
    transition: background 0.1s;
  }

  .relay-row:hover {
    background: var(--muted);
  }

  .relay-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
  }

  .relay-name {
    font-size: 13px;
    font-weight: 500;
    color: var(--foreground);
  }

  .relay-address {
    font-size: 12px;
    color: var(--muted-foreground);
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
  }

  .relay-status {
    font-size: 11px;
    font-weight: 500;
    color: var(--muted-foreground);
    padding: 2px 8px;
    border-radius: 4px;
    background: var(--muted);
  }

  .relay-status.connected {
    color: #22c55e;
    background: rgba(34, 197, 94, 0.1);
  }

  .remove-btn {
    width: 28px;
    height: 28px;
    border-radius: 6px;
    background: none;
    border: none;
    color: var(--muted-foreground);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: color 0.15s;
  }

  .remove-btn:hover {
    color: var(--destructive);
  }
</style>
