<script lang="ts">
  import { Check, Copy, KeyRound, Pencil } from "lucide-svelte";
  import Avatar from "../common/Avatar.svelte";
  import { UserService } from "../../wails";
  import { myPublicKey } from "../../stores/permissions.svelte.ts";
  import { users } from "../../stores/users.svelte.ts";

  interface Props {
    displayName: string;
    publicKey: string;
    avatarUrl: string | null;
    onClose: () => void;
    onEditProfile: () => void;
  }

  let { displayName, publicKey, avatarUrl, onClose, onEditProfile }: Props = $props();

  const statusOptions = [
    { value: "online", label: "Online", color: "#22c55e" },
    { value: "idle", label: "Away", color: "#eab308" },
    { value: "offline", label: "Invisible", color: "var(--muted-foreground)" },
  ] as const;

  // Resolve current status from users store
  let me = $derived(users().find((u) => u.publicKey === myPublicKey()));
  let initialStatus = $derived(me?.status ?? "online");
  let statusOverride = $state<string | null>(null);
  let currentStatus = $derived(statusOverride ?? initialStatus);
  let copied = $state(false);

  let currentStatusInfo = $derived(
    statusOptions.find((o) => o.value === currentStatus) ?? statusOptions[0],
  );

  let truncatedKey = $derived(
    publicKey.length > 16
      ? publicKey.slice(0, 8) + "\u2026" + publicKey.slice(-8)
      : publicKey,
  );

  function handleStatusClick(status: string) {
    statusOverride = status;
    UserService.SetStatusAll(status).catch(() => {});
  }

  function handleCopy() {
    navigator.clipboard.writeText(publicKey).then(() => {
      copied = true;
      setTimeout(() => {
        copied = false;
      }, 2000);
    }).catch(() => {});
  }

  function handleEditProfile() {
    onEditProfile();
    onClose();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") {
      onClose();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="popover-backdrop" onclick={onClose} role="presentation">
  <div class="popover" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()} role="dialog" tabindex="-1">
    <!-- Purple accent banner -->
    <div class="accent-banner"></div>

    <!-- User section -->
    <div class="user-section">
      <div class="avatar-ring" style="border-color: {currentStatusInfo.color};">
        <Avatar name={displayName} src={avatarUrl} size={50} rounded="circle" />
      </div>
      <div class="user-info">
        <span class="display-name">{displayName}</span>
        <div class="status-row">
          <span class="status-dot" style="background: {currentStatusInfo.color};"></span>
          <span class="status-text">{currentStatusInfo.label}</span>
        </div>
      </div>
    </div>

    <div class="divider"></div>

    <!-- Public Key section -->
    <div class="section">
      <span class="section-label">Public Key</span>
      <div class="key-row">
        <div class="key-box">
          <KeyRound size={14} />
          <span class="key-text">{truncatedKey}</span>
        </div>
        <button class="copy-btn" onclick={handleCopy} type="button" title="Copy public key">
          {#if copied}
            <Check size={16} />
          {:else}
            <Copy size={16} />
          {/if}
        </button>
      </div>
      {#if copied}
        <span class="copied-feedback">Copied!</span>
      {/if}
    </div>

    <div class="divider"></div>

    <!-- Status section -->
    <div class="section">
      <span class="section-label">Set Status</span>
      <div class="status-list">
        {#each statusOptions as option (option.value)}
          <button
            class="status-option"
            class:selected={currentStatus === option.value}
            onclick={() => handleStatusClick(option.value)}
            type="button"
          >
            <span class="status-dot" style="background: {option.color};"></span>
            <span class="status-option-label">{option.label}</span>
            {#if currentStatus === option.value}
              <span class="check-icon">
                <Check size={16} />
              </span>
            {/if}
          </button>
        {/each}
      </div>
    </div>

    <div class="divider"></div>

    <!-- Actions section -->
    <div class="section">
      <button class="action-row" onclick={handleEditProfile} type="button">
        <Pencil size={16} />
        <span>Edit Profile</span>
      </button>
    </div>
  </div>
</div>

<style>
  .popover-backdrop {
    position: fixed;
    inset: 0;
    z-index: 100;
  }

  .popover {
    position: fixed;
    bottom: 81px;
    left: 12px;
    width: 320px;
    background: var(--popover);
    border: 1px solid var(--border);
    border-radius: 12px;
    display: flex;
    flex-direction: column;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
    overflow: hidden;
  }

  .accent-banner {
    height: 8px;
    background: #7c3aed;
    border-radius: 12px 12px 0 0;
    flex-shrink: 0;
  }

  .user-section {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px 16px 0;
  }

  .avatar-ring {
    width: 56px;
    height: 56px;
    border-radius: 9999px;
    border: 3px solid;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .user-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
    overflow: hidden;
  }

  .display-name {
    font-size: 16px;
    font-weight: 600;
    color: var(--foreground);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .status-row {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 9999px;
    flex-shrink: 0;
  }

  .status-text {
    font-size: 12px;
    color: var(--muted-foreground);
  }

  .divider {
    height: 1px;
    background: var(--border);
    margin: 12px 16px;
  }

  .section {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 0 16px;
  }

  .section:last-child {
    padding-bottom: 12px;
  }

  .section-label {
    font-size: 11px;
    font-weight: 500;
    color: var(--muted-foreground);
    letter-spacing: 0.5px;
  }

  .key-row {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .key-box {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 10px;
    background: var(--muted);
    border-radius: 8px;
    color: var(--muted-foreground);
    overflow: hidden;
  }

  .key-text {
    font-size: 12px;
    font-family: "JetBrains Mono", "Fira Code", monospace;
    color: var(--foreground);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .copy-btn {
    width: 36px;
    height: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--muted);
    border: none;
    border-radius: 8px;
    color: var(--muted-foreground);
    cursor: pointer;
    flex-shrink: 0;
    transition: background 0.15s;
  }

  .copy-btn:hover {
    background: var(--accent);
    color: var(--foreground);
  }

  .copied-feedback {
    font-size: 11px;
    color: #22c55e;
  }

  .status-list {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .status-option {
    display: flex;
    align-items: center;
    gap: 10px;
    height: 36px;
    padding: 0 10px;
    border: none;
    border-radius: 8px;
    background: transparent;
    color: var(--foreground);
    font-size: 13px;
    cursor: pointer;
    transition: background 0.15s;
    width: 100%;
  }

  .status-option:hover {
    background: var(--muted);
  }

  .status-option.selected {
    background: var(--muted);
  }

  .status-option-label {
    flex: 1;
    text-align: left;
  }

  .check-icon {
    display: flex;
    align-items: center;
    color: var(--foreground);
  }

  .action-row {
    display: flex;
    align-items: center;
    gap: 10px;
    height: 36px;
    padding: 0 10px;
    border: none;
    border-radius: 8px;
    background: transparent;
    color: var(--foreground);
    font-size: 13px;
    cursor: pointer;
    transition: background 0.15s;
    width: 100%;
  }

  .action-row:hover {
    background: var(--muted);
  }
</style>
