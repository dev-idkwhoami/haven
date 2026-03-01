<script lang="ts">
  import type { Message } from "../../types";
  import { search } from "../../stores/messages.svelte.ts";
  import { activeServerId } from "../../stores/servers.svelte.ts";
  import { activeChannelId } from "../../stores/channels.svelte.ts";

  interface Props {
    onClose: () => void;
    onResultClick?: (message: Message) => void;
  }

  let { onClose, onResultClick }: Props = $props();

  let query = $state("");
  let results = $state<Message[]>([]);
  let searching = $state(false);
  let searchTimeout: ReturnType<typeof setTimeout> | null = null;

  function handleInput(): void {
    if (searchTimeout) clearTimeout(searchTimeout);
    if (!query.trim()) {
      results = [];
      return;
    }
    searchTimeout = setTimeout(doSearch, 300);
  }

  async function doSearch(): Promise<void> {
    if (!query.trim() || activeServerId() === null || !activeChannelId()) return;
    searching = true;
    try {
      results = await search(activeServerId()!, activeChannelId()!, query.trim());
    } catch {
      results = [];
    } finally {
      searching = false;
    }
  }

  function handleKeydown(e: KeyboardEvent): void {
    if (e.key === "Escape") onClose();
    if (e.key === "Enter") {
      if (searchTimeout) clearTimeout(searchTimeout);
      doSearch();
    }
  }
</script>

<div class="search-bar">
  <div class="search-input-row">
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
      <circle cx="11" cy="11" r="8" /><path d="m21 21-4.3-4.3" />
    </svg>
    <input
      type="text"
      class="search-input"
      placeholder="Search messages..."
      bind:value={query}
      oninput={handleInput}
      onkeydown={handleKeydown}
      autofocus
    />
    <button class="close-btn" onclick={onClose} title="Close search" type="button">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <line x1="18" x2="6" y1="6" y2="18" /><line x1="6" x2="18" y1="6" y2="18" />
      </svg>
    </button>
  </div>

  {#if results.length > 0}
    <div class="results">
      {#each results as msg (msg.remoteMessageId)}
        <button
          class="result-item"
          onclick={() => onResultClick?.(msg)}
          type="button"
        >
          <span class="result-content">{msg.content}</span>
          <span class="result-time">
            {new Date(msg.remoteCreatedAt).toLocaleDateString()}
          </span>
        </button>
      {/each}
    </div>
  {:else if query.trim() && !searching}
    <div class="no-results">No messages found</div>
  {/if}

  {#if searching}
    <div class="no-results">Searching...</div>
  {/if}
</div>

<style>
  .search-bar {
    display: flex;
    flex-direction: column;
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: 8px;
    overflow: hidden;
    max-width: 400px;
    width: 100%;
  }

  .search-input-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    color: var(--muted-foreground);
  }

  .search-input {
    flex: 1;
    background: none;
    border: none;
    color: var(--foreground);
    font-size: 14px;
    outline: none;
  }

  .search-input::placeholder {
    color: var(--muted-foreground);
  }

  .close-btn {
    width: 24px;
    height: 24px;
    background: none;
    border: none;
    color: var(--muted-foreground);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 4px;
    transition: color 0.15s;
  }

  .close-btn:hover {
    color: var(--foreground);
  }

  .results {
    display: flex;
    flex-direction: column;
    border-top: 1px solid var(--border);
    max-height: 300px;
    overflow-y: auto;
  }

  .result-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 8px 12px;
    background: none;
    border: none;
    cursor: pointer;
    text-align: left;
    transition: background 0.1s;
  }

  .result-item:hover {
    background: var(--secondary);
  }

  .result-content {
    font-size: 13px;
    color: var(--foreground);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    flex: 1;
  }

  .result-time {
    font-size: 11px;
    color: var(--muted-foreground);
    flex-shrink: 0;
  }

  .no-results {
    padding: 12px;
    text-align: center;
    font-size: 13px;
    color: var(--muted-foreground);
    border-top: 1px solid var(--border);
  }
</style>
