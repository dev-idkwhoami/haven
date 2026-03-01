<script lang="ts">
  import type { Message } from "../../types";
  import { activeChannelId, channels } from "../../stores/channels.svelte.ts";
  import { activeServerId } from "../../stores/servers.svelte.ts";
  import { participants, joinChannel, activeVoiceChannelId } from "../../stores/voice.svelte.ts";
  import { sendMessage } from "../../stores/messages.svelte.ts";
  import { ProfileService } from "../../wails";
  import MessageList from "../message/MessageList.svelte";
  import MessageInput from "../message/MessageInput.svelte";

  let myPubKey = $state("");

  let activeChannel = $derived(
    channels().find((c) => c.remoteChannelId === activeChannelId()),
  );

  let isInThisChannel = $derived(activeVoiceChannelId() === activeChannelId());

  $effect(() => {
    ProfileService.GetPublicKey()
      .then((key) => (myPubKey = key))
      .catch(() => {});
  });

  function handleJoin(): void {
    if (activeServerId() === null || !activeChannelId()) return;
    joinChannel(activeServerId()!, activeChannelId()!);
  }

  function handleSend(content: string): void {
    if (activeServerId() === null || !activeChannelId()) return;
    sendMessage(activeServerId()!, activeChannelId()!, content);
  }
</script>

<div class="voice-channel-view">
  <div class="top-bar">
    <div class="channel-info">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <path d="M2 10v3" /><path d="M6 6v11" /><path d="M10 3v18" /><path d="M14 8v7" /><path d="M18 5v13" /><path d="M22 10v3" />
      </svg>
      <span class="channel-name">{activeChannel?.name ?? "Voice Channel"}</span>
    </div>
  </div>

  <div class="top-divider"></div>

  <div class="voice-content">
    <div class="participants-section">
      {#if participants().length > 0}
        <div class="participant-grid">
          {#each participants() as participant (participant.publicKey)}
            <div class="participant-card" class:speaking={participant.isSpeaking}>
              <div class="participant-avatar" class:muted={participant.isMuted}>
                <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" />
                </svg>
              </div>
              <span class="participant-name">{participant.displayName}</span>
              <div class="participant-status">
                {#if participant.isMuted}
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <line x1="2" x2="22" y1="2" y2="22" /><path d="M18.89 13.23A7.12 7.12 0 0 0 19 12v-2" /><path d="M5 10v2a7 7 0 0 0 12 5" /><path d="M15 9.34V5a3 3 0 0 0-5.68-1.33" /><path d="M9 9v3a3 3 0 0 0 5.12 2.12" /><line x1="12" x2="12" y1="19" y2="22" />
                  </svg>
                {/if}
                {#if participant.isDeafened}
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <line x1="2" x2="22" y1="2" y2="22" /><path d="M8.5 16.5a5 5 0 0 1-1.643-2.022" /><path d="M19.198 10.802A2 2 0 0 1 21 14v1a2 2 0 0 1-2 2h-1" /><path d="M3 14h3a2 2 0 0 1 2 2v3a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-7a9 9 0 0 1 9-9 8.981 8.981 0 0 1 5 1.516" />
                  </svg>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {:else if !isInThisChannel}
        <div class="join-prompt">
          <p class="join-text">Join the voice channel to start talking</p>
          <button class="join-btn" onclick={handleJoin} type="button">
            Join Voice
          </button>
        </div>
      {:else}
        <div class="waiting-text">
          <p>Waiting for others to join...</p>
        </div>
      {/if}
    </div>

    <div class="mid-divider"></div>

    <div class="chat-section">
      <MessageList
        {myPubKey}
        onEdit={() => {}}
        onDelete={() => {}}
      />
    </div>
  </div>

  <div class="bottom-divider"></div>
  <MessageInput onSend={handleSend} />
</div>

<style>
  .voice-channel-view {
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

  .channel-info {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--muted-foreground);
  }

  .channel-name {
    font-size: 16px;
    font-weight: 600;
    color: var(--foreground);
  }

  .top-divider,
  .mid-divider,
  .bottom-divider {
    height: 1px;
    background: var(--border);
    flex-shrink: 0;
  }

  .voice-content {
    display: flex;
    flex-direction: column;
    flex: 1;
    overflow: hidden;
  }

  .participants-section {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 32px 48px;
    flex-shrink: 0;
  }

  .participant-grid {
    display: flex;
    gap: 48px;
    justify-content: center;
    flex-wrap: wrap;
  }

  .participant-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    width: 140px;
  }

  .participant-avatar {
    width: 80px;
    height: 80px;
    border-radius: 16px;
    background: var(--muted);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted-foreground);
    border: 3px solid transparent;
    transition: border-color 0.2s;
  }

  .participant-card.speaking .participant-avatar {
    border-color: #22c55e;
  }

  .participant-avatar.muted {
    opacity: 0.6;
  }

  .participant-name {
    font-size: 13px;
    font-weight: 500;
    color: var(--foreground);
    text-align: center;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 100%;
  }

  .participant-status {
    display: flex;
    gap: 4px;
    color: var(--destructive);
  }

  .join-prompt,
  .waiting-text {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    color: var(--muted-foreground);
    font-size: 14px;
  }

  .join-prompt p,
  .waiting-text p {
    margin: 0;
  }

  .join-btn {
    padding: 10px 24px;
    background: #22c55e;
    color: #fff;
    border: none;
    border-radius: var(--radius);
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .join-btn:hover {
    opacity: 0.9;
  }

  .chat-section {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }
</style>
