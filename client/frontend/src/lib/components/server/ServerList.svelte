<script lang="ts">
  import { MessageCircle, CirclePlus } from "lucide-svelte";
  import { servers, activeServerId, setActiveServer, connectionStatus } from "../../stores/servers.svelte.ts";
  import ServerIcon from "./ServerIcon.svelte";

  interface Props {
    onDMClick: () => void;
    onAddServerClick: () => void;
  }

  let { onDMClick, onAddServerClick }: Props = $props();
</script>

<div class="server-list">
  <button class="dm-btn" onclick={onDMClick} title="Direct Messages" type="button">
    <MessageCircle size={24} />
  </button>

  <div class="divider-spacer"></div>
  <div class="divider"></div>

  <div class="servers-scroll">
    {#each servers().filter((s) => !s.isRelayOnly) as server (server.id)}
      <ServerIcon
        name={server.name || server.address}
        iconHash={server.iconHash}
        isActive={activeServerId() === server.id}
        connected={connectionStatus()[server.id] === "connected"}
        reconnecting={connectionStatus()[server.id] === "reconnecting"}
        onclick={() => setActiveServer(server.id)}
      />
    {/each}
  </div>

  <button class="add-btn" onclick={onAddServerClick} title="Add Server" type="button">
    <CirclePlus size={24} />
  </button>

</div>

<style>
  .server-list {
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 72px;
    padding: 12px;
    gap: 8px;
    background: var(--background);
    height: 100%;
    flex-shrink: 0;
  }

  .dm-btn {
    width: 48px;
    height: 48px;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.06);
    border: none;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted-foreground);
    cursor: pointer;
    flex-shrink: 0;
    transition: background 0.15s;
  }

  .dm-btn:hover {
    background: rgba(255, 255, 255, 0.1);
  }

  .divider-spacer {
    height: 4px;
    flex-shrink: 0;
  }

  .divider {
    width: 32px;
    height: 1px;
    background: var(--border);
    flex-shrink: 0;
  }

  .servers-scroll {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    overflow-y: auto;
    overflow-x: hidden;
    flex: 1;
    width: 100%;
    scrollbar-width: none;
  }

  .servers-scroll::-webkit-scrollbar {
    display: none;
  }

  .add-btn {
    width: 48px;
    height: 48px;
    border-radius: 12px;
    background: transparent;
    border: none;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted-foreground);
    cursor: pointer;
    flex-shrink: 0;
    transition: color 0.15s;
  }

  .add-btn:hover {
    color: var(--foreground);
  }

</style>
