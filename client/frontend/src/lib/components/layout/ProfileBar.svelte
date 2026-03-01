<script lang="ts">
  import { Mic, MicOff, Headphones, HeadphoneOff, Settings, PhoneOff } from "lucide-svelte";
  import { isMuted, isDeafened, setMuted, setDeafened } from "../../stores/voice.svelte.ts";
  import { activeVoiceChannelId, leaveChannel } from "../../stores/voice.svelte.ts";
  import { ProfileService } from "../../wails";
  import Avatar from "../common/Avatar.svelte";
  import MyProfilePopover from "../user/MyProfilePopover.svelte";

  interface Props {
    onSettingsClick: () => void;
  }

  let { onSettingsClick }: Props = $props();

  let displayName = $state("Loading...");
  let avatarUrl = $state<string | null>(null);
  let publicKey = $state("");
  let showPopover = $state(false);

  $effect(() => {
    ProfileService.GetProfile().then((p) => {
      displayName = p.displayName || "User";
    }).catch(() => {
      displayName = "User";
    });

    ProfileService.GetAvatar().then((url) => {
      avatarUrl = url || null;
    }).catch(() => {});

    ProfileService.GetPublicKey().then((key) => {
      publicKey = key;
    }).catch(() => {});
  });
</script>

<div class="profile-bar">
  <button class="profile-area" onclick={() => (showPopover = !showPopover)} type="button">
    <Avatar name={displayName} src={avatarUrl} size={36} rounded="circle" />
    <span class="display-name">{displayName}</span>
  </button>
  <div class="spacer"></div>

  <button class="bar-btn" onclick={onSettingsClick} title="Settings" type="button">
    <Settings size={16} />
  </button>

  <button
    class="bar-btn"
    class:active={isDeafened()}
    onclick={() => setDeafened(!isDeafened())}
    title={isDeafened() ? "Undeafen" : "Deafen"}
    type="button"
  >
    {#if isDeafened()}
      <HeadphoneOff size={16} />
    {:else}
      <Headphones size={16} />
    {/if}
  </button>

  <button
    class="bar-btn"
    class:active={isMuted()}
    onclick={() => setMuted(!isMuted())}
    title={isMuted() ? "Unmute" : "Mute"}
    type="button"
  >
    {#if isMuted()}
      <MicOff size={16} />
    {:else}
      <Mic size={16} />
    {/if}
  </button>

  {#if activeVoiceChannelId()}
    <button
      class="bar-btn end-btn"
      onclick={() => leaveChannel()}
      title="Disconnect"
      type="button"
    >
      <PhoneOff size={16} />
    </button>
  {/if}
</div>

{#if showPopover}
  <MyProfilePopover
    {displayName}
    {publicKey}
    {avatarUrl}
    onClose={() => (showPopover = false)}
    onEditProfile={onSettingsClick}
  />
{/if}

<style>
  .profile-bar {
    display: flex;
    align-items: center;
    gap: 10px;
    height: 73px;
    padding: 0 12px;
    background: var(--background);
  }

  .profile-area {
    display: flex;
    align-items: center;
    gap: 10px;
    background: none;
    border: none;
    padding: 4px;
    margin: -4px;
    border-radius: 8px;
    cursor: pointer;
    overflow: hidden;
    transition: background 0.15s;
  }

  .profile-area:hover {
    background: rgba(255, 255, 255, 0.06);
  }

  .display-name {
    font-size: 13px;
    font-weight: 500;
    color: var(--foreground);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .spacer {
    flex: 1;
  }

  .bar-btn {
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
    flex-shrink: 0;
    transition: background 0.15s, color 0.15s;
  }

  .bar-btn:hover {
    background: rgba(255, 255, 255, 0.1);
    color: var(--foreground);
  }

  .bar-btn.active {
    color: var(--destructive);
  }

  .end-btn {
    background: rgba(255, 68, 68, 0.13);
  }

  .end-btn:hover {
    background: rgba(255, 68, 68, 0.25);
  }

  .end-btn svg {
    color: var(--destructive);
  }
</style>
