<script lang="ts">
  import type { DMConversation } from "../../types";
  import { conversations, activeDMId, setActiveDM, loadConversations } from "../../stores/dms.svelte.ts";

  interface Props {
    onCreateDM: () => void;
  }

  let { onCreateDM }: Props = $props();

  $effect(() => {
    loadConversations();
  });

  function getConversationName(conv: DMConversation): string {
    if (conv.name) return conv.name;
    if (conv.isGroup) return `Group (${conv.participants.length})`;
    return conv.participants[0]?.userId.slice(0, 8) ?? "DM";
  }

  function getConversationInitial(conv: DMConversation): string {
    const name = getConversationName(conv);
    return name.charAt(0).toUpperCase();
  }
</script>

<div class="dm-list">
  <div class="dm-header">
    <span class="header-title">Direct Messages</span>
    <button class="new-dm-btn" onclick={onCreateDM} title="New DM" type="button">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M12 5v14" /><path d="M5 12h14" />
      </svg>
    </button>
  </div>

  <div class="dm-scroll">
    {#each conversations() as conv (conv.id)}
      <button
        class="dm-item"
        class:active={conv.id === activeDMId()}
        onclick={() => setActiveDM(conv.id)}
        type="button"
      >
        <div class="dm-avatar" class:group={conv.isGroup}>
          {#if conv.isGroup}
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" /><circle cx="9" cy="7" r="4" />
              <path d="M22 21v-2a4 4 0 0 0-3-3.87" /><path d="M16 3.13a4 4 0 0 1 0 7.75" />
            </svg>
          {:else}
            <span class="dm-initial">{getConversationInitial(conv)}</span>
          {/if}
        </div>
        <div class="dm-info">
          <span class="dm-name">{getConversationName(conv)}</span>
          {#if conv.isGroup}
            <span class="dm-meta">{conv.participants.length} members</span>
          {/if}
        </div>
      </button>
    {/each}

    {#if conversations().length === 0}
      <div class="empty-state">
        <span class="empty-text">No conversations yet</span>
      </div>
    {/if}
  </div>
</div>

<style>
  .dm-list {
    display: flex;
    flex-direction: column;
    height: 100%;
    width: 100%;
  }

  .dm-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 72px;
    padding: 0 16px;
    flex-shrink: 0;
  }

  .header-title {
    font-size: 15px;
    font-weight: 600;
    color: var(--foreground);
  }

  .new-dm-btn {
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

  .new-dm-btn:hover {
    color: var(--foreground);
  }

  .dm-scroll {
    flex: 1;
    overflow-y: auto;
    padding: 0 8px 12px;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .dm-item {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    height: 40px;
    padding: 0 8px;
    background: none;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    text-align: left;
    transition: background 0.1s;
  }

  .dm-item:hover {
    background: var(--muted);
  }

  .dm-item.active {
    background: var(--muted);
  }

  .dm-avatar {
    width: 28px;
    height: 28px;
    border-radius: 9999px;
    background: var(--muted);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted-foreground);
    flex-shrink: 0;
    font-size: 12px;
    font-weight: 600;
  }

  .dm-avatar.group {
    border-radius: 8px;
  }

  .dm-initial {
    color: var(--muted-foreground);
  }

  .dm-info {
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .dm-name {
    font-size: 13px;
    font-weight: 500;
    color: var(--foreground);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .dm-meta {
    font-size: 11px;
    color: var(--muted-foreground);
  }

  .empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
  }

  .empty-text {
    font-size: 13px;
    color: var(--muted-foreground);
  }
</style>
