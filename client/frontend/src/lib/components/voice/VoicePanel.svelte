<script lang="ts">
  import { participants, activeVoiceChannelId } from "../../stores/voice.svelte.ts";
  import { channels } from "../../stores/channels.svelte.ts";
  import ParticipantList from "./ParticipantList.svelte";

  let activeVoiceChannel = $derived(
    activeVoiceChannelId()
      ? channels().find((c) => c.remoteChannelId === activeVoiceChannelId())
      : null,
  );
</script>

{#if activeVoiceChannelId()}
  <div class="voice-panel">
    <div class="panel-header">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <path d="M2 10v3" /><path d="M6 6v11" /><path d="M10 3v18" /><path d="M14 8v7" /><path d="M18 5v13" /><path d="M22 10v3" />
      </svg>
      <span class="panel-title">{activeVoiceChannel?.name ?? "Voice Channel"}</span>
      <span class="participant-count">{participants().length}</span>
    </div>

    <ParticipantList />
  </div>
{/if}

<style>
  .voice-panel {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 8px 12px;
    background: var(--background);
    border-top: 1px solid var(--border);
    flex-shrink: 0;
  }

  .panel-header {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--muted-foreground);
  }

  .panel-title {
    font-size: 12px;
    font-weight: 600;
    color: var(--foreground);
  }

  .participant-count {
    font-size: 11px;
    color: var(--muted-foreground);
    margin-left: auto;
  }
</style>
