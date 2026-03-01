<script lang="ts">
  import type { DMMessage, DMConversation as DMConvType } from "../../types";
  import {
    conversations,
    activeDMId,
    dmMessages,
    sendDM,
    loadDMHistory,
    startCall,
  } from "../../stores/dms.svelte.ts";
  import { ProfileService } from "../../wails";
  import MessageInput from "../message/MessageInput.svelte";

  let myPubKey = $state("");
  let chatEl = $state<HTMLDivElement | null>(null);

  let activeConv = $derived(
    conversations().find((c) => c.id === activeDMId()),
  );

  let convName = $derived(() => {
    if (!activeConv) return "Conversation";
    if (activeConv.name) return activeConv.name;
    if (activeConv.isGroup) return `Group (${activeConv.participants.length})`;
    return activeConv.participants[0]?.userId.slice(0, 8) ?? "DM";
  });

  $effect(() => {
    ProfileService.GetPublicKey()
      .then((key) => (myPubKey = key))
      .catch(() => {});
  });

  $effect(() => {
    if (activeDMId()) {
      loadDMHistory(activeDMId()!);
    }
  });

  $effect(() => {
    if (chatEl && dmMessages().length > 0) {
      chatEl.scrollTop = chatEl.scrollHeight;
    }
  });

  function handleSend(content: string): void {
    if (!activeDMId()) return;
    sendDM(activeDMId()!, content);
  }

  function handleCall(): void {
    if (!activeDMId()) return;
    startCall(activeDMId()!);
  }

  function isOwn(msg: DMMessage): boolean {
    return msg.senderPubKey === myPubKey;
  }

  function formatTime(iso: string): string {
    const d = new Date(iso);
    return d.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" }) +
      " \u00B7 " +
      d.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" });
  }
</script>

<div class="dm-conversation">
  <div class="top-bar">
    <div class="conv-info">
      {#if activeConv?.isGroup}
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" /><circle cx="9" cy="7" r="4" />
          <path d="M22 21v-2a4 4 0 0 0-3-3.87" /><path d="M16 3.13a4 4 0 0 1 0 7.75" />
        </svg>
      {:else}
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" />
        </svg>
      {/if}
      <span class="conv-name">{convName()}</span>
      {#if activeConv?.isGroup}
        <span class="member-count">{activeConv.participants.length} members</span>
      {/if}
    </div>
    <div class="top-actions">
      <button class="top-btn" onclick={handleCall} title="Start Call" type="button">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z" />
        </svg>
      </button>
      <button class="top-btn" title="Search" type="button">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="11" cy="11" r="8" /><path d="m21 21-4.3-4.3" />
        </svg>
      </button>
    </div>
  </div>

  <div class="top-divider"></div>

  <div class="e2ee-badge">
    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
      <path d="M7 11V7a5 5 0 0 1 10 0v4" />
    </svg>
    <span>End-to-end encrypted</span>
  </div>

  <div class="chat-area" bind:this={chatEl}>
    {#each dmMessages() as msg (msg.id)}
      {#if isOwn(msg)}
        <div class="msg-row own">
          <div class="msg-own">
            <div class="msg-bubble own-bubble">{msg.content}</div>
            <span class="msg-time">{formatTime(msg.remoteCreatedAt)}</span>
          </div>
        </div>
      {:else}
        <div class="msg-row">
          <div class="msg-other">
            <div class="msg-header">
              <div class="msg-avatar">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" />
                </svg>
              </div>
              <span class="msg-sender">{msg.senderPubKey.slice(0, 8)}</span>
            </div>
            <div class="msg-bubble other-bubble">{msg.content}</div>
            <span class="msg-time">{formatTime(msg.remoteCreatedAt)}</span>
          </div>
        </div>
      {/if}
    {/each}

    {#if dmMessages().length === 0}
      <div class="empty-chat">
        <p class="empty-text">No messages yet. Say hello!</p>
      </div>
    {/if}
  </div>

  <div class="bottom-divider"></div>
  <MessageInput onSend={handleSend} />
</div>

<style>
  .dm-conversation {
    display: flex;
    flex-direction: column;
    height: 100%;
    width: 100%;
  }

  .top-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 72px;
    padding: 0 24px;
    flex-shrink: 0;
  }

  .conv-info {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--muted-foreground);
  }

  .conv-name {
    font-size: 16px;
    font-weight: 600;
    color: var(--foreground);
  }

  .member-count {
    font-size: 13px;
    color: var(--muted-foreground);
  }

  .top-actions {
    display: flex;
    gap: 4px;
  }

  .top-btn {
    width: 32px;
    height: 32px;
    border-radius: 8px;
    background: none;
    border: none;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted-foreground);
    cursor: pointer;
    transition: color 0.15s;
  }

  .top-btn:hover {
    color: var(--foreground);
  }

  .top-divider,
  .bottom-divider {
    height: 1px;
    background: var(--border);
    flex-shrink: 0;
  }

  .e2ee-badge {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    padding: 6px;
    font-size: 11px;
    color: var(--muted-foreground);
    background: rgba(34, 197, 94, 0.06);
    flex-shrink: 0;
  }

  .e2ee-badge svg {
    color: #22c55e;
  }

  .chat-area {
    flex: 1;
    overflow-y: auto;
    padding: 24px 48px;
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .msg-row {
    display: flex;
  }

  .msg-row.own {
    justify-content: flex-end;
  }

  .msg-own,
  .msg-other {
    display: flex;
    flex-direction: column;
    gap: 4px;
    max-width: 70%;
  }

  .msg-own {
    align-items: flex-end;
  }

  .msg-header {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .msg-avatar {
    width: 24px;
    height: 24px;
    border-radius: 6px;
    background: var(--muted);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted-foreground);
  }

  .msg-sender {
    font-size: 13px;
    font-weight: 600;
    color: var(--foreground);
  }

  .msg-bubble {
    padding: 8px 14px;
    font-size: 14px;
    line-height: 1.4;
    word-break: break-word;
  }

  .own-bubble {
    background: var(--primary);
    color: var(--primary-foreground);
    border-radius: 12px 0 12px 12px;
  }

  .other-bubble {
    background: var(--muted);
    color: var(--foreground);
    border-radius: 0 12px 12px 12px;
  }

  .msg-time {
    font-size: 11px;
    color: var(--muted-foreground);
  }

  .empty-chat {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .empty-text {
    color: var(--muted-foreground);
    font-size: 14px;
    margin: 0;
  }
</style>
