<script lang="ts">
  import {
    isMuted,
    isDeafened,
    activeVoiceChannelId,
    setMuted,
    setDeafened,
    leaveChannel,
  } from "../../stores/voice.svelte.ts";
  import { channels } from "../../stores/channels.svelte.ts";

  let activeVoiceChannel = $derived(
    activeVoiceChannelId()
      ? channels().find((c) => c.remoteChannelId === activeVoiceChannelId())
      : null,
  );
</script>

{#if activeVoiceChannelId()}
  <div class="voice-controls">
    <div class="voice-info">
      <div class="voice-status">
        <div class="status-dot"></div>
        <span class="status-text">Voice Connected</span>
      </div>
      <span class="voice-channel-name">{activeVoiceChannel?.name ?? "Voice Channel"}</span>
    </div>
    <div class="control-buttons">
      <button
        class="ctrl-btn"
        class:active={isMuted()}
        onclick={() => setMuted(!isMuted())}
        title={isMuted() ? "Unmute" : "Mute"}
        type="button"
      >
        {#if isMuted()}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="2" x2="22" y1="2" y2="22" /><path d="M18.89 13.23A7.12 7.12 0 0 0 19 12v-2" /><path d="M5 10v2a7 7 0 0 0 12 5" /><path d="M15 9.34V5a3 3 0 0 0-5.68-1.33" /><path d="M9 9v3a3 3 0 0 0 5.12 2.12" /><line x1="12" x2="12" y1="19" y2="22" />
          </svg>
        {:else}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z" /><path d="M19 10v2a7 7 0 0 1-14 0v-2" /><line x1="12" x2="12" y1="19" y2="22" />
          </svg>
        {/if}
      </button>

      <button
        class="ctrl-btn"
        class:active={isDeafened()}
        onclick={() => setDeafened(!isDeafened())}
        title={isDeafened() ? "Undeafen" : "Deafen"}
        type="button"
      >
        {#if isDeafened()}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="2" x2="22" y1="2" y2="22" /><path d="M8.5 16.5a5 5 0 0 1-1.643-2.022" /><path d="M19.198 10.802A2 2 0 0 1 21 14v1a2 2 0 0 1-2 2h-1" /><path d="M3 14h3a2 2 0 0 1 2 2v3a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-7a9 9 0 0 1 9-9 8.981 8.981 0 0 1 5 1.516" />
          </svg>
        {:else}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M3 14h3a2 2 0 0 1 2 2v3a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-7a9 9 0 0 1 18 0v7a2 2 0 0 1-2 2h-1a2 2 0 0 1-2-2v-3a2 2 0 0 1 2-2h3" />
          </svg>
        {/if}
      </button>

      <button
        class="ctrl-btn disconnect"
        onclick={() => leaveChannel()}
        title="Disconnect"
        type="button"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M10.68 13.31a16 16 0 0 0 3.41 2.6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7 2 2 0 0 1 1.72 2v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.42 19.42 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91" />
          <line x1="23" x2="1" y1="1" y2="23" />
        </svg>
      </button>
    </div>
  </div>
{/if}

<style>
  .voice-controls {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    background: rgba(34, 197, 94, 0.06);
    border-top: 1px solid var(--border);
    flex-shrink: 0;
  }

  .voice-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .voice-status {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 9999px;
    background: #22c55e;
  }

  .status-text {
    font-size: 12px;
    font-weight: 600;
    color: #22c55e;
  }

  .voice-channel-name {
    font-size: 11px;
    color: var(--muted-foreground);
    padding-left: 14px;
  }

  .control-buttons {
    display: flex;
    gap: 4px;
  }

  .ctrl-btn {
    width: 28px;
    height: 28px;
    border-radius: 6px;
    background: rgba(255, 255, 255, 0.06);
    border: none;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted-foreground);
    cursor: pointer;
    transition: background 0.15s, color 0.15s;
  }

  .ctrl-btn:hover {
    background: rgba(255, 255, 255, 0.1);
    color: var(--foreground);
  }

  .ctrl-btn.active {
    color: var(--destructive);
  }

  .ctrl-btn.disconnect {
    background: rgba(255, 68, 68, 0.13);
  }

  .ctrl-btn.disconnect:hover {
    background: rgba(255, 68, 68, 0.25);
  }

  .ctrl-btn.disconnect svg {
    color: var(--destructive);
  }
</style>
