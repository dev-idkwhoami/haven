<script lang="ts">
  import type { Message, User } from "../../types";

  interface Props {
    message: Message;
    author: User | undefined;
    isOwn: boolean;
    onEdit?: (message: Message) => void;
    onDelete?: (message: Message) => void;
  }

  let { message, author, isOwn, onEdit, onDelete }: Props = $props();

  let showActions = $state(false);

  let displayName = $derived(isOwn ? "You" : (author?.displayName ?? "Unknown"));

  let formattedTime = $derived(() => {
    const d = new Date(message.remoteCreatedAt);
    const date = d.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
    const time = d.toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit" });
    return `${date} · ${time}`;
  });
</script>

<div
  class="message-row"
  class:own={isOwn}
  onmouseenter={() => (showActions = true)}
  onmouseleave={() => (showActions = false)}
  role="article"
>
  <div class="message-content" class:own={isOwn}>
    <div class="msg-header" class:own={isOwn}>
      {#if !isOwn}
        <div class="avatar">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" />
          </svg>
        </div>
      {/if}
      <span class="author-name">{displayName}</span>
      {#if isOwn}
        <div class="avatar own">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" />
          </svg>
        </div>
      {/if}
    </div>

    <div class="bubble" class:own={isOwn}>
      <span class="text">{message.content}</span>
      {#if message.editedAt}
        <span class="edited">(edited)</span>
      {/if}
    </div>

    <span class="timestamp">{formattedTime()}</span>
  </div>

  {#if showActions && (isOwn || onDelete)}
    <div class="actions" class:own={isOwn}>
      {#if isOwn && onEdit}
        <button class="action-btn" onclick={() => onEdit(message)} title="Edit" type="button">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z" />
          </svg>
        </button>
      {/if}
      {#if onDelete}
        <button class="action-btn destructive" onclick={() => onDelete(message)} title="Delete" type="button">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M3 6h18" /><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6" /><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2" />
          </svg>
        </button>
      {/if}
    </div>
  {/if}
</div>

<style>
  .message-row {
    display: flex;
    align-items: flex-start;
    position: relative;
  }

  .message-row.own {
    justify-content: flex-end;
  }

  .message-content {
    display: flex;
    flex-direction: column;
    gap: 4px;
    max-width: 70%;
  }

  .msg-header {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .msg-header.own {
    justify-content: flex-end;
  }

  .avatar {
    width: 24px;
    height: 24px;
    border-radius: 6px;
    background: var(--muted);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted-foreground);
    flex-shrink: 0;
  }

  .avatar.own {
    background: var(--primary);
    color: var(--primary-foreground);
  }

  .author-name {
    font-size: 12px;
    font-weight: 600;
    color: var(--foreground);
  }

  .bubble {
    padding: 8px 14px;
    background: var(--muted);
    border-radius: 0 12px 12px 12px;
    display: inline-flex;
    flex-wrap: wrap;
    gap: 6px;
    align-items: baseline;
  }

  .bubble.own {
    background: var(--primary);
    border-radius: 12px 0 12px 12px;
  }

  .text {
    font-size: 14px;
    color: var(--foreground);
    line-height: 1.5;
    word-break: break-word;
    white-space: pre-wrap;
  }

  .bubble.own .text {
    color: var(--primary-foreground);
  }

  .edited {
    font-size: 11px;
    color: var(--muted-foreground);
    font-style: italic;
  }

  .bubble.own .edited {
    color: rgba(255, 255, 255, 0.5);
  }

  .timestamp {
    font-size: 11px;
    color: var(--muted-foreground);
  }

  .actions {
    display: flex;
    gap: 4px;
    position: absolute;
    top: 0;
    left: calc(70% + 8px);
  }

  .actions.own {
    left: auto;
    right: calc(70% + 8px);
  }

  .action-btn {
    width: 28px;
    height: 28px;
    border-radius: 6px;
    background: var(--card);
    border: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted-foreground);
    cursor: pointer;
    transition: background 0.15s, color 0.15s;
  }

  .action-btn:hover {
    background: var(--secondary);
    color: var(--foreground);
  }

  .action-btn.destructive:hover {
    color: var(--destructive);
  }
</style>
