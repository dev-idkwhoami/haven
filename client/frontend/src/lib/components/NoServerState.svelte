<script lang="ts">
  import JoinServerModal from "./server/JoinServerModal.svelte";

  let showModal = $state(false);
</script>

<div class="no-server-layout">
  <div class="left-panel">
    <div class="server-sidebar">
      <button class="dm-btn" disabled title="Join a server to use Direct Messages" type="button">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M7.9 20A9 9 0 1 0 4 16.1L2 22Z" />
        </svg>
      </button>
      <div class="sidebar-divider"></div>
      <div class="server-empty-hint">
        <span>No Servers Connected</span>
      </div>
    </div>
    <div class="channel-panel">
      <div class="channel-panel-empty"></div>
    </div>
  </div>

  <div class="content-area">
    <h2 class="content-heading">Chat or talk to others by adding your first server</h2>
    <button class="add-server-btn" onclick={() => (showModal = true)} type="button">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <line x1="12" y1="5" x2="12" y2="19" />
        <line x1="5" y1="12" x2="19" y2="12" />
      </svg>
      Add Server
    </button>
  </div>
</div>

{#if showModal}
  <JoinServerModal onClose={() => (showModal = false)} />
{/if}

<style>
  .no-server-layout {
    display: flex;
    height: 100%;
    width: 100%;
  }

  .left-panel {
    display: flex;
    flex-direction: row;
    height: 100%;
    border-right: 1px solid var(--border);
  }

  .server-sidebar {
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 72px;
    padding: 12px;
    gap: 8px;
    background: var(--background);
  }

  .dm-btn {
    width: 48px;
    height: 48px;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.06);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted-foreground);
    cursor: pointer;
    transition: background 0.15s;
  }

  .dm-btn:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.1);
  }

  .dm-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .sidebar-divider {
    width: 32px;
    height: 1px;
    background: var(--border);
    margin: 4px 0;
  }

  .server-empty-hint {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .server-empty-hint span {
    writing-mode: vertical-rl;
    text-orientation: mixed;
    color: var(--muted-foreground);
    font-size: 11px;
    opacity: 0.5;
  }

  .channel-panel {
    width: 240px;
    background: var(--background);
    border-left: 1px solid var(--border);
  }

  .channel-panel-empty {
    height: 100%;
  }

  .content-area {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 20px;
    background: var(--background);
  }

  .content-heading {
    font-size: 20px;
    font-weight: 600;
    color: var(--foreground);
    text-align: center;
    max-width: 400px;
    margin: 0;
  }

  .add-server-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 20px;
    background: var(--primary);
    color: var(--primary-foreground);
    border: none;
    border-radius: var(--radius);
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .add-server-btn:hover {
    opacity: 0.9;
  }

</style>
