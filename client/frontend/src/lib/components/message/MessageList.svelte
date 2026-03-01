<script lang="ts">
  import type { Message } from "../../types";
  import { messages, hasMore, loading, loadHistory, clearMessages } from "../../stores/messages.svelte.ts";
  import { activeChannelId } from "../../stores/channels.svelte.ts";
  import { activeServerId } from "../../stores/servers.svelte.ts";
  import { users } from "../../stores/users.svelte.ts";
  import MessageItem from "./MessageItem.svelte";
  import TypingIndicator from "./TypingIndicator.svelte";

  interface Props {
    myPubKey: string;
    onEdit: (message: Message) => void;
    onDelete: (message: Message) => void;
  }

  let { myPubKey, onEdit, onDelete }: Props = $props();

  let scrollContainer = $state<HTMLDivElement | null>(null);
  let wasAtBottom = $state(true);
  let prevMessageCount = $state(0);

  $effect(() => {
    const channelId = activeChannelId();
    const serverId = activeServerId();
    if (channelId && serverId !== null) {
      clearMessages();
      loadHistory(serverId, channelId);
    }
  });

  $effect(() => {
    const count = messages().length;
    if (count > prevMessageCount && wasAtBottom && scrollContainer) {
      requestAnimationFrame(() => {
        if (scrollContainer) {
          scrollContainer.scrollTop = scrollContainer.scrollHeight;
        }
      });
    }
    prevMessageCount = count;
  });

  function handleScroll(): void {
    if (!scrollContainer) return;
    const { scrollTop, scrollHeight, clientHeight } = scrollContainer;
    wasAtBottom = scrollHeight - scrollTop - clientHeight < 50;

    if (scrollTop < 100 && hasMore() && !loading() && activeServerId() !== null && activeChannelId()) {
      const firstMsg = messages()[0];
      if (firstMsg) {
        const prevScrollHeight = scrollContainer.scrollHeight;
        loadHistory(activeServerId()!, activeChannelId()!, firstMsg.remoteMessageId).then(() => {
          requestAnimationFrame(() => {
            if (scrollContainer) {
              scrollContainer.scrollTop = scrollContainer.scrollHeight - prevScrollHeight;
            }
          });
        });
      }
    }
  }

  function getAuthor(pubKey: string) {
    return users().find((u) => u.publicKey === pubKey);
  }
</script>

<div class="message-list-wrapper">
  <div
    class="message-list"
    bind:this={scrollContainer}
    onscroll={handleScroll}
  >
    {#if loading() && messages().length === 0}
      <div class="loading-state">
        <span>Loading messages...</span>
      </div>
    {:else if messages().length === 0}
      <div class="empty-state">
        <span>No messages yet. Start the conversation!</span>
      </div>
    {:else}
      {#if loading()}
        <div class="loading-more">
          <span>Loading older messages...</span>
        </div>
      {/if}

      {#each messages() as message (message.remoteMessageId)}
        <MessageItem
          {message}
          author={getAuthor(message.authorPubKey)}
          isOwn={message.authorPubKey === myPubKey}
          onEdit={() => onEdit(message)}
          onDelete={() => onDelete(message)}
        />
      {/each}
    {/if}

    <div class="bottom-pad"></div>
  </div>

  <TypingIndicator />
</div>

<style>
  .message-list-wrapper {
    display: flex;
    flex-direction: column;
    flex: 1;
    overflow: hidden;
  }

  .message-list {
    flex: 1;
    overflow-y: auto;
    padding: 24px 48px;
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .loading-state,
  .empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 1;
    color: var(--muted-foreground);
    font-size: 14px;
  }

  .loading-more {
    display: flex;
    justify-content: center;
    padding: 8px 0;
    color: var(--muted-foreground);
    font-size: 12px;
  }

  .bottom-pad {
    flex-shrink: 0;
    height: 1px;
  }
</style>
