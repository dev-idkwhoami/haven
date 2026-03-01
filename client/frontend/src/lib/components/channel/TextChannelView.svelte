<script lang="ts">
  import type { Message } from "../../types";
  import { activeChannelId, channels } from "../../stores/channels.svelte.ts";
  import { activeServerId } from "../../stores/servers.svelte.ts";
  import { sendMessage, deleteMessage } from "../../stores/messages.svelte.ts";
  import { ProfileService } from "../../wails";
  import MessageList from "../message/MessageList.svelte";
  import MessageInput from "../message/MessageInput.svelte";
  import EditMessageModal from "../message/EditMessageModal.svelte";
  import SearchBar from "../message/SearchBar.svelte";
  import UserList from "../user/UserList.svelte";

  let myPubKey = $state("");
  let editingMessage = $state<Message | null>(null);
  let showSearch = $state(false);
  let showMembers = $state(true);

  let activeChannel = $derived(
    channels().find((c) => c.remoteChannelId === activeChannelId()),
  );

  $effect(() => {
    ProfileService.GetPublicKey()
      .then((key) => (myPubKey = key))
      .catch(() => {});
  });

  function handleSend(content: string): void {
    if (activeServerId() === null || !activeChannelId()) return;
    sendMessage(activeServerId()!, activeChannelId()!, content);
  }

  function handleEdit(message: Message): void {
    editingMessage = message;
  }

  async function handleDelete(message: Message): Promise<void> {
    if (activeServerId() === null) return;
    try {
      await deleteMessage(activeServerId()!, message.remoteMessageId);
    } catch {
      // ignore
    }
  }
</script>

<div class="text-channel-view">
  <div class="channel-main">
    <div class="top-bar">
      <div class="channel-info">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <line x1="4" x2="20" y1="9" y2="9" /><line x1="4" x2="20" y1="15" y2="15" /><line x1="10" x2="8" y1="3" y2="21" /><line x1="16" x2="14" y1="3" y2="21" />
        </svg>
        <span class="channel-name">{activeChannel?.name ?? "Channel"}</span>
      </div>
      <div class="top-bar-actions">
        <button
          class="top-btn"
          class:active={showSearch}
          onclick={() => (showSearch = !showSearch)}
          title="Search"
          type="button"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="11" cy="11" r="8" /><path d="m21 21-4.3-4.3" />
          </svg>
        </button>
        <button
          class="top-btn"
          class:active={showMembers}
          onclick={() => (showMembers = !showMembers)}
          title="Toggle members"
          type="button"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" /><circle cx="9" cy="7" r="4" />
            <path d="M22 21v-2a4 4 0 0 0-3-3.87" /><path d="M16 3.13a4 4 0 0 1 0 7.75" />
          </svg>
        </button>
      </div>
    </div>

    <div class="top-divider"></div>

    {#if showSearch}
      <div class="search-wrapper">
        <SearchBar onClose={() => (showSearch = false)} />
      </div>
    {/if}

    <MessageList
      {myPubKey}
      onEdit={handleEdit}
      onDelete={handleDelete}
    />

    <div class="bottom-divider"></div>

    <MessageInput onSend={handleSend} />
  </div>

  {#if showMembers}
    <UserList />
  {/if}
</div>

{#if editingMessage}
  <EditMessageModal
    message={editingMessage}
    onClose={() => (editingMessage = null)}
  />
{/if}

<style>
  .text-channel-view {
    display: flex;
    height: 100%;
    width: 100%;
  }

  .channel-main {
    display: flex;
    flex-direction: column;
    flex: 1;
    overflow: hidden;
  }

  .top-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
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

  .top-bar-actions {
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
    transition: background 0.15s, color 0.15s;
  }

  .top-btn:hover,
  .top-btn.active {
    color: var(--foreground);
  }

  .top-divider,
  .bottom-divider {
    height: 1px;
    background: var(--border);
    flex-shrink: 0;
  }

  .search-wrapper {
    display: flex;
    justify-content: flex-end;
    padding: 8px 24px;
    flex-shrink: 0;
  }
</style>
