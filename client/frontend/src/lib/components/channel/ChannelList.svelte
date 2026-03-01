<script lang="ts">
  import { Settings, Glasses } from "lucide-svelte";
  import {
    categories,
    channels,
    activeChannelId,
    setActiveChannel,
  } from "../../stores/channels.svelte.ts";
  import { activeServerInfo, activeServerId, connectionStatus } from "../../stores/servers.svelte.ts";
  import { hasPermission, PERM_MANAGE_CHANNELS, PERM_MANAGE_SERVER } from "../../stores/permissions.svelte.ts";
  import CategoryHeader from "./CategoryHeader.svelte";

  interface Props {
    onCreateCategory: () => void;
    onCreateChannel: () => void;
    onAdminClick?: () => void;
    onIdentityClick?: () => void;
  }

  let { onCreateCategory, onCreateChannel, onAdminClick, onIdentityClick }: Props = $props();

  let isConnected = $derived(
    activeServerId() !== null && connectionStatus()[activeServerId()!] === "connected"
  );

  let canManageChannels = $derived(isConnected && hasPermission(PERM_MANAGE_CHANNELS));
  let canManageServer = $derived(isConnected && hasPermission(PERM_MANAGE_SERVER));

  let collapsedCategories = $state<Set<string>>(new Set());

  function toggleCategory(categoryId: string): void {
    const next = new Set(collapsedCategories);
    if (next.has(categoryId)) {
      next.delete(categoryId);
    } else {
      next.add(categoryId);
    }
    collapsedCategories = next;
  }

  let sortedCategories = $derived(
    [...categories()].sort((a, b) => a.position - b.position),
  );

  function channelsForCategory(categoryId: string) {
    return channels()
      .filter((c) => c.remoteCategoryId === categoryId)
      .sort((a, b) => a.position - b.position);
  }
</script>

<div class="channel-list">
  <div class="header">
    <span class="server-name">{activeServerInfo()?.name ?? "Server"}</span>
    <div class="header-btns">
      {#if canManageServer}
        <button class="header-btn" title="Server settings" onclick={onAdminClick} type="button">
          <Settings size={16} />
        </button>
      {/if}
      <button class="header-btn" title="Server Identity" onclick={onIdentityClick} type="button">
        <Glasses size={16} />
      </button>
    </div>
  </div>

  <div class="divider"></div>

  <div class="channels-scroll">
    {#each sortedCategories as category (category.remoteCategoryId)}
      {@const collapsed = collapsedCategories.has(category.remoteCategoryId)}
      {@const catChannels = channelsForCategory(category.remoteCategoryId)}

      <CategoryHeader
        name={category.name}
        {collapsed}
        onToggle={() => toggleCategory(category.remoteCategoryId)}
      />

      {#if !collapsed}
        {#each catChannels as channel (channel.remoteChannelId)}
          <button
            class="channel-item"
            class:active={activeChannelId() === channel.remoteChannelId}
            onclick={() => setActiveChannel(channel.remoteChannelId)}
            type="button"
          >
            {#if channel.type === "voice"}
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                <path d="M2 10v3" /><path d="M6 6v11" /><path d="M10 3v18" /><path d="M14 8v7" /><path d="M18 5v13" /><path d="M22 10v3" />
              </svg>
            {:else}
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                <line x1="4" x2="20" y1="9" y2="9" /><line x1="4" x2="20" y1="15" y2="15" /><line x1="10" x2="8" y1="3" y2="21" /><line x1="16" x2="14" y1="3" y2="21" />
              </svg>
            {/if}
            <span class="channel-name">{channel.name}</span>
          </button>
        {/each}
      {/if}

      {#if category !== sortedCategories[sortedCategories.length - 1]}
        <div class="cat-spacer"></div>
      {/if}
    {/each}

    <div class="bottom-spacer"></div>
  </div>

  {#if canManageChannels}
  <div class="btn-row">
    <button class="ghost-btn" onclick={onCreateCategory} type="button">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
      </svg>
      Create Category
    </button>
    <button class="ghost-btn" onclick={onCreateChannel} type="button">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
      </svg>
      Create Channel
    </button>
  </div>
  {/if}
</div>

<style>
  .channel-list {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: var(--background);
    border-right: 1px solid var(--border);
    width: 100%;
  }

  .header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 72px;
    padding: 0 16px;
    flex-shrink: 0;
  }

  .server-name {
    font-size: 15px;
    font-weight: 600;
    color: var(--foreground);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .header-btns {
    display: flex;
    gap: 6px;
  }

  .header-btn {
    width: 32px;
    height: 32px;
    border-radius: 8px;
    background: rgba(255, 255, 255, 0.06);
    border: none;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted-foreground);
    cursor: pointer;
    transition: background 0.15s, color 0.15s;
  }

  .header-btn:hover {
    background: rgba(255, 255, 255, 0.1);
    color: var(--foreground);
  }

  .divider {
    height: 1px;
    background: var(--border);
    flex-shrink: 0;
  }

  .channels-scroll {
    flex: 1;
    overflow-y: auto;
    padding: 12px 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .cat-spacer {
    height: 8px;
    flex-shrink: 0;
  }

  .channel-item {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 32px;
    padding: 0 8px 0 24px;
    border-radius: 4px;
    margin: 0 8px;
    background: none;
    border: none;
    color: var(--muted-foreground);
    cursor: pointer;
    transition: background 0.1s, color 0.1s;
    flex-shrink: 0;
  }

  .channel-item:hover {
    background: rgba(255, 255, 255, 0.06);
    color: var(--foreground);
  }

  .channel-item.active {
    background: var(--sidebar-accent);
    color: var(--foreground);
  }

  .channel-name {
    font-size: 14px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .bottom-spacer {
    flex: 1;
  }

  .btn-row {
    display: flex;
    gap: 4px;
    padding: 8px 8px;
    flex-shrink: 0;
  }

  .ghost-btn {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    padding: 8px;
    background: none;
    border: none;
    border-radius: var(--radius);
    color: var(--muted-foreground);
    font-size: 13px;
    cursor: pointer;
    transition: background 0.15s, color 0.15s;
  }

  .ghost-btn:hover {
    background: rgba(255, 255, 255, 0.06);
    color: var(--foreground);
  }
</style>
