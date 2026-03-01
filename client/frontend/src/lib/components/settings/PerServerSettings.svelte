<script lang="ts">
  import {
    perServerSettings,
    loadServerSettings,
    updateServerSettings,
  } from "../../stores/settings.svelte.ts";
  import { activeServerId } from "../../stores/servers.svelte.ts";

  let saving = $state(false);

  $effect(() => {
    if (activeServerId() !== null) {
      loadServerSettings(activeServerId()!);
    }
  });

  async function toggleSyncAvatars(): Promise<void> {
    if (activeServerId() === null || !perServerSettings()) return;
    saving = true;
    try {
      await updateServerSettings(activeServerId()!, {
        syncAvatars: !perServerSettings()!.syncAvatars,
      });
    } catch {
      // error
    } finally {
      saving = false;
    }
  }

  async function toggleSyncBios(): Promise<void> {
    if (activeServerId() === null || !perServerSettings()) return;
    saving = true;
    try {
      await updateServerSettings(activeServerId()!, {
        syncBios: !perServerSettings()!.syncBios,
      });
    } catch {
      // error
    } finally {
      saving = false;
    }
  }

  async function toggleSyncStatus(): Promise<void> {
    if (activeServerId() === null || !perServerSettings()) return;
    saving = true;
    try {
      await updateServerSettings(activeServerId()!, {
        syncStatus: !perServerSettings()!.syncStatus,
      });
    } catch {
      // error
    } finally {
      saving = false;
    }
  }
</script>

<div class="per-server-settings">
  <div class="top-bar">
    <span class="section-title">Server Privacy</span>
  </div>
  <div class="top-divider"></div>

  <div class="content-scroll">
    <div class="content-wrap">
      <div class="section">
        <h3 class="section-label">Field Selection</h3>
        <p class="section-desc">Choose what data this server can access about you. Disabling these will prevent the server from receiving these fields.</p>

        {#if perServerSettings()}
          <div class="toggle-row">
            <div class="toggle-text">
              <span class="toggle-label">Share Avatar</span>
              <span class="toggle-desc">Allow this server to see your profile picture.</span>
            </div>
            <button
              class="toggle-switch"
              onclick={toggleSyncAvatars}
              disabled={saving}
              type="button"
            >
              <div class="toggle-knob" class:on={perServerSettings()?.syncAvatars}></div>
            </button>
          </div>

          <div class="toggle-row">
            <div class="toggle-text">
              <span class="toggle-label">Share Bio</span>
              <span class="toggle-desc">Allow this server to see your bio/about text.</span>
            </div>
            <button
              class="toggle-switch"
              onclick={toggleSyncBios}
              disabled={saving}
              type="button"
            >
              <div class="toggle-knob" class:on={perServerSettings()?.syncBios}></div>
            </button>
          </div>

          <div class="toggle-row">
            <div class="toggle-text">
              <span class="toggle-label">Share Online Status</span>
              <span class="toggle-desc">Allow this server to see when you are online.</span>
            </div>
            <button
              class="toggle-switch"
              onclick={toggleSyncStatus}
              disabled={saving}
              type="button"
            >
              <div class="toggle-knob" class:on={perServerSettings()?.syncStatus}></div>
            </button>
          </div>
        {:else}
          <p class="loading-text">Loading settings...</p>
        {/if}
      </div>
    </div>
  </div>
</div>

<style>
  .per-server-settings {
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

  .toggle-row {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .toggle-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
  }

  .toggle-label {
    font-size: 13px;
    font-weight: 500;
    color: var(--foreground);
  }

  .toggle-desc {
    font-size: 12px;
    color: var(--muted-foreground);
  }

  .toggle-switch {
    width: 40px;
    height: 22px;
    border-radius: 11px;
    background: var(--muted);
    padding: 2px;
    cursor: pointer;
    flex-shrink: 0;
    border: none;
  }

  .toggle-switch:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .toggle-knob {
    width: 18px;
    height: 18px;
    border-radius: 9px;
    background: var(--muted-foreground);
    transition: transform 0.15s, background 0.15s;
  }

  .toggle-knob.on {
    transform: translateX(18px);
    background: var(--primary);
  }

  .loading-text {
    font-size: 13px;
    color: var(--muted-foreground);
    margin: 0;
  }
</style>
