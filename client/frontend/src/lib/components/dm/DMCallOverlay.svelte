<script lang="ts">
  import type { DMConversation } from "../../types";
  import {
    conversations,
    incomingCallId,
    activeCallId,
    acceptCall,
    rejectCall,
    leaveCall,
  } from "../../stores/dms.svelte.ts";

  let incomingConv = $derived(
    incomingCallId() ? conversations().find((c) => c.id === incomingCallId()) : null,
  );

  let activeConv = $derived(
    activeCallId() ? conversations().find((c) => c.id === activeCallId()) : null,
  );

  function getConvName(conv: DMConversation): string {
    if (conv.name) return conv.name;
    if (conv.isGroup) return `Group (${conv.participants.length})`;
    return conv.participants[0]?.userId.slice(0, 8) ?? "Someone";
  }

  let callTimer = $state(0);
  let timerInterval = $state<ReturnType<typeof setInterval> | null>(null);

  $effect(() => {
    if (activeCallId()) {
      callTimer = 0;
      timerInterval = setInterval(() => {
        callTimer++;
      }, 1000);
    }
    return () => {
      if (timerInterval) {
        clearInterval(timerInterval);
        timerInterval = null;
      }
    };
  });

  function formatDuration(seconds: number): string {
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return `${m}:${s.toString().padStart(2, "0")}`;
  }

  function handleAccept(): void {
    if (incomingCallId()) acceptCall(incomingCallId()!);
  }

  function handleReject(): void {
    if (incomingCallId()) rejectCall(incomingCallId()!);
  }

  function handleLeave(): void {
    if (activeCallId()) leaveCall(activeCallId()!);
  }
</script>

{#if incomingConv && incomingCallId()}
  <div class="call-overlay">
    <div class="call-card incoming">
      <div class="call-avatar">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" />
        </svg>
      </div>
      <span class="call-name">{getConvName(incomingConv)}</span>
      <span class="call-status">Incoming call...</span>
      <div class="call-actions">
        <button class="call-btn accept" onclick={handleAccept} type="button">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z" />
          </svg>
        </button>
        <button class="call-btn reject" onclick={handleReject} type="button">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M10.68 13.31a16 16 0 0 0 3.41 2.6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7 2 2 0 0 1 1.72 2v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.42 19.42 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91" />
            <line x1="23" x2="1" y1="1" y2="23" />
          </svg>
        </button>
      </div>
    </div>
  </div>
{/if}

{#if activeConv && activeCallId()}
  <div class="active-call-bar">
    <div class="call-info">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z" />
      </svg>
      <span class="call-label">In call with {getConvName(activeConv)}</span>
      <span class="call-timer">{formatDuration(callTimer)}</span>
    </div>
    <button class="end-call-btn" onclick={handleLeave} type="button">
      Leave
    </button>
  </div>
{/if}

<style>
  .call-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 200;
  }

  .call-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    padding: 32px 48px;
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: 16px;
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.5);
  }

  .call-avatar {
    width: 80px;
    height: 80px;
    border-radius: 9999px;
    background: var(--muted);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted-foreground);
  }

  .call-name {
    font-size: 18px;
    font-weight: 600;
    color: var(--foreground);
  }

  .call-status {
    font-size: 13px;
    color: var(--muted-foreground);
  }

  .call-actions {
    display: flex;
    gap: 16px;
    margin-top: 8px;
  }

  .call-btn {
    width: 48px;
    height: 48px;
    border-radius: 9999px;
    border: none;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .call-btn:hover {
    opacity: 0.9;
  }

  .call-btn.accept {
    background: #22c55e;
    color: #fff;
  }

  .call-btn.reject {
    background: var(--destructive);
    color: #fff;
  }

  .active-call-bar {
    position: fixed;
    top: 0;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 6px 12px 6px 16px;
    background: #22c55e;
    border-radius: 0 0 8px 8px;
    z-index: 150;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  }

  .call-info {
    display: flex;
    align-items: center;
    gap: 8px;
    color: #fff;
  }

  .call-label {
    font-size: 13px;
    font-weight: 500;
  }

  .call-timer {
    font-size: 12px;
    opacity: 0.8;
    font-variant-numeric: tabular-nums;
  }

  .end-call-btn {
    padding: 4px 12px;
    background: rgba(0, 0, 0, 0.2);
    border: none;
    border-radius: 4px;
    color: #fff;
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s;
  }

  .end-call-btn:hover {
    background: rgba(0, 0, 0, 0.35);
  }
</style>
