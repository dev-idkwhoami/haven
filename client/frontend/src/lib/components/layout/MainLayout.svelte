<script lang="ts">
  import { WifiOff } from "lucide-svelte";
  import ServerList from "../server/ServerList.svelte";
  import ChannelList from "../channel/ChannelList.svelte";
  import ProfileBar from "./ProfileBar.svelte";
  import JoinServerModal from "../server/JoinServerModal.svelte";
  import CreateChannelModal from "../channel/CreateChannelModal.svelte";
  import CreateCategoryModal from "../channel/CreateCategoryModal.svelte";
  import CreateDMModal from "../dm/CreateDMModal.svelte";
  import TextChannelView from "../channel/TextChannelView.svelte";
  import VoiceChannelView from "../channel/VoiceChannelView.svelte";
  import VoiceControls from "../voice/VoiceControls.svelte";
  import VoicePanel from "../voice/VoicePanel.svelte";
  import DMList from "../dm/DMList.svelte";
  import DMConversation from "../dm/DMConversation.svelte";
  import DMCallOverlay from "../dm/DMCallOverlay.svelte";
  import AdminPanel from "../admin/AdminPanel.svelte";
  import SettingsPanel from "../settings/SettingsPanel.svelte";
  import { activeServerId, setActiveServer, connectionStatus, reconnect, showAccessRequest, clearAccessRequest } from "../../stores/servers.svelte.ts";
  import { activeChannelId, channels, loadChannels } from "../../stores/channels.svelte.ts";
  import { loadUsers } from "../../stores/users.svelte.ts";
  import { loadRoles } from "../../stores/roles.svelte.ts";
  import { activeDMId } from "../../stores/dms.svelte.ts";
  import { activeVoiceChannelId } from "../../stores/voice.svelte.ts";
  import { incomingCallId } from "../../stores/dms.svelte.ts";

  type ViewMode = "server" | "dms" | "admin" | "settings";
  let viewMode = $state<ViewMode>("server");

  let showJoinServer = $state(false);
  let showCreateChannel = $state(false);
  let showCreateCategory = $state(false);
  let showCreateDM = $state(false);
  let settingsInitialSection = $state<string | undefined>(undefined);

  let activeChannel = $derived(
    channels().find((c) => c.remoteChannelId === activeChannelId()),
  );

  let isFullscreenPanel = $derived(viewMode === "admin" || viewMode === "settings");

  let isServerConnected = $derived(
    activeServerId() !== null && connectionStatus()[activeServerId()!] === "connected"
  );

  let isServerReconnecting = $derived(
    activeServerId() !== null && connectionStatus()[activeServerId()!] === "reconnecting"
  );

  $effect(() => {
    if (activeServerId() !== null) {
      loadChannels(activeServerId()!);
      loadUsers(activeServerId()!);
      loadRoles(activeServerId()!);
    }
  });

  // Fix 12: When a server is clicked from DM view, switch to server view
  let prevServerId: number | null = null;
  $effect(() => {
    const sid = activeServerId();
    if (sid !== null && sid !== prevServerId && viewMode === "dms") {
      viewMode = "server";
    }
    prevServerId = sid;
  });
</script>

<div class="main-layout">
  {#if !isFullscreenPanel}
    <ServerList
      onDMClick={() => { setActiveServer(null); viewMode = "dms"; }}
      onAddServerClick={() => (showJoinServer = true)}
    />
  {/if}

  {#if isFullscreenPanel}
    <div class="fullscreen-panel">
      {#if viewMode === "admin"}
        <AdminPanel onClose={() => (viewMode = "server")} />
      {:else if viewMode === "settings"}
        <SettingsPanel
          onClose={() => { settingsInitialSection = undefined; viewMode = "server"; }}
          initialSection={settingsInitialSection}
        />
      {/if}
    </div>
  {:else}
    <div class="middle-panel">
      <div class="top-wrap">
        {#if viewMode === "dms"}
          <DMList onCreateDM={() => (showCreateDM = true)} />
        {:else if viewMode === "server" && activeServerId() !== null && !isServerConnected}
          <div class="offline-middle">
            <span class="offline-title">Server Offline</span>
          </div>
        {:else}
          <ChannelList
            onCreateCategory={() => (showCreateCategory = true)}
            onCreateChannel={() => (showCreateChannel = true)}
            onAdminClick={() => (viewMode = "admin")}
            onIdentityClick={() => { settingsInitialSection = "privacy"; viewMode = "settings"; }}
          />
        {/if}
      </div>
      {#if activeVoiceChannelId()}
        <VoicePanel />
        <VoiceControls />
      {/if}
      <div class="prof-divider"></div>
      <ProfileBar
        onSettingsClick={() => { settingsInitialSection = undefined; viewMode = "settings"; }}
      />
    </div>

    <div class="content-area">
      {#if viewMode === "server" && activeServerId() !== null && !isServerConnected}
        <div class="offline-state">
          <WifiOff size={48} strokeWidth={1.5} />
          {#if isServerReconnecting}
            <h2 class="offline-heading">Reconnecting...</h2>
            <p class="offline-desc">Attempting to restore connection</p>
          {:else}
            <h2 class="offline-heading">Server Offline</h2>
            <p class="offline-desc">Unable to connect to this server</p>
            <button class="reconnect-btn" onclick={() => reconnect(activeServerId()!)} type="button">
              Reconnect
            </button>
          {/if}
        </div>
      {:else if viewMode === "dms"}
        {#if activeDMId()}
          <DMConversation />
        {:else}
          <div class="no-channel">
            <h2 class="no-channel-text">Select a conversation to start chatting</h2>
          </div>
        {/if}
      {:else if activeChannel?.type === "text"}
        <TextChannelView />
      {:else if activeChannel?.type === "voice"}
        <VoiceChannelView />
      {:else}
        <div class="no-channel">
          <h2 class="no-channel-text">Select any channel to start chatting</h2>
        </div>
      {/if}
    </div>
  {/if}
</div>

{#if showJoinServer || showAccessRequest()}
  <JoinServerModal onClose={() => { showJoinServer = false; clearAccessRequest(); }} initialStep={showAccessRequest() ? "waiting_room" : "address"} />
{/if}

{#if showCreateChannel && activeServerId() !== null}
  <CreateChannelModal
    serverId={activeServerId()!}
    onClose={() => (showCreateChannel = false)}
  />
{/if}

{#if showCreateCategory && activeServerId() !== null}
  <CreateCategoryModal
    serverId={activeServerId()!}
    onClose={() => (showCreateCategory = false)}
  />
{/if}

{#if showCreateDM}
  <CreateDMModal onClose={() => (showCreateDM = false)} />
{/if}

{#if incomingCallId()}
  <DMCallOverlay />
{/if}

<style>
  .main-layout {
    display: flex;
    height: 100%;
    width: 100%;
  }

  .middle-panel {
    display: flex;
    flex-direction: column;
    width: 312px;
    height: 100%;
    background: var(--background);
    border-left: 1px solid var(--border);
    border-right: 1px solid var(--border);
    flex-shrink: 0;
  }

  .fullscreen-panel {
    display: flex;
    flex: 1;
    height: 100%;
    background: var(--background);
    overflow: hidden;
  }

  .top-wrap {
    flex: 1;
    overflow: hidden;
    display: flex;
  }

  .prof-divider {
    height: 1px;
    background: var(--border);
    flex-shrink: 0;
  }

  .content-area {
    flex: 1;
    display: flex;
    flex-direction: column;
    background: var(--background);
    overflow: hidden;
  }

  .no-channel {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
  }

  .no-channel-text {
    font-size: 20px;
    font-weight: 600;
    color: var(--foreground);
    text-align: center;
    max-width: 400px;
    margin: 0;
  }

  .offline-middle {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    width: 100%;
    color: var(--muted-foreground);
  }

  .offline-title {
    font-size: 14px;
    font-weight: 500;
  }

  .offline-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    gap: 12px;
    color: var(--muted-foreground);
  }

  .offline-heading {
    font-size: 20px;
    font-weight: 600;
    color: var(--foreground);
    margin: 0;
  }

  .offline-desc {
    font-size: 14px;
    margin: 0;
  }

  .reconnect-btn {
    padding: 10px 24px;
    background: var(--primary);
    color: var(--primary-foreground);
    border: none;
    border-radius: var(--radius);
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s;
    margin-top: 8px;
  }

  .reconnect-btn:hover {
    opacity: 0.9;
  }
</style>
