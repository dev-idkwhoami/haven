<script lang="ts">
  import { activeServerId, activeServerInfo } from "../../stores/servers.svelte.ts";
  import { channels } from "../../stores/channels.svelte.ts";
  import { AdminService, FileService } from "../../wails";

  interface Props {
    onClose: () => void;
  }

  let { onClose }: Props = $props();

  let serverName = $state(activeServerInfo()?.name ?? "");
  let serverDesc = $state(activeServerInfo()?.description ?? "");
  let saving = $state(false);
  let error = $state<string | null>(null);

  let textChannels = $derived(
    channels().filter((c) => c.type === "text"),
  );

  $effect(() => {
    if (activeServerInfo()) {
      serverName = activeServerInfo()!.name;
      serverDesc = activeServerInfo()!.description ?? "";
    }
  });

  async function handleSave(): Promise<void> {
    if (activeServerId() === null || !serverName.trim()) return;
    saving = true;
    error = null;
    try {
      await AdminService.UpdateServer(activeServerId()!, serverName.trim(), serverDesc.trim());
    } catch {
      error = "Failed to save settings";
    } finally {
      saving = false;
    }
  }

  async function handleIconUpload(): Promise<void> {
    if (activeServerId() === null) return;
    try {
      const path = await FileService.PickFile();
      if (path) {
        await AdminService.SetServerIcon(activeServerId()!, path);
      }
    } catch {
      // cancelled or error
    }
  }
</script>

<div class="server-settings">
  <div class="top-bar">
    <span class="section-title">Overview</span>
  </div>
  <div class="top-divider"></div>

  <div class="content-scroll">
    <div class="content-wrap">
      <div class="section">
        <h3 class="section-label">Server Info</h3>
        <p class="section-desc">Customize your server's identity. The server name and icon are visible to all members.</p>

        <div class="server-row">
          <button class="icon-upload" onclick={handleIconUpload} type="button">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z" /><circle cx="12" cy="13" r="3" />
            </svg>
            <span class="upload-text">Upload</span>
          </button>
          <div class="fields-col">
            <div class="field-group">
              <label class="field-label">Server Name</label>
              <input class="field-input" bind:value={serverName} placeholder="Server name" />
            </div>
            <div class="field-group">
              <label class="field-label">Description</label>
              <textarea
                class="field-textarea"
                bind:value={serverDesc}
                placeholder="A few words about this server..."
                rows="3"
              ></textarea>
            </div>
          </div>
        </div>
      </div>

      <div class="divider"></div>

      <div class="section">
        <h3 class="section-label">Notifications</h3>
        <div class="toggle-row">
          <div class="toggle-text">
            <span class="toggle-label">Default Notification Settings</span>
            <span class="toggle-desc">Notify members about all new messages by default.</span>
          </div>
          <div class="toggle-switch">
            <div class="toggle-knob on"></div>
          </div>
        </div>
        <div class="toggle-row">
          <div class="toggle-text">
            <span class="toggle-label">Suppress @everyone and @here</span>
            <span class="toggle-desc">Prevent server-wide mentions from triggering notifications.</span>
          </div>
          <div class="toggle-switch">
            <div class="toggle-knob"></div>
          </div>
        </div>
      </div>

      <div class="divider"></div>

      <div class="section">
        <h3 class="section-label">System Channel</h3>
        <div class="field-group">
          <label class="field-label">Welcome Messages Channel</label>
          <div class="select-field">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <line x1="4" x2="20" y1="9" y2="9" /><line x1="4" x2="20" y1="15" y2="15" /><line x1="10" x2="8" y1="3" y2="21" /><line x1="16" x2="14" y1="3" y2="21" />
            </svg>
            <span class="select-text">{textChannels[0]?.name ?? "general"}</span>
            <div class="select-spacer"></div>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="m6 9 6 6 6-6" />
            </svg>
          </div>
        </div>
        <div class="field-group">
          <label class="field-label">System Messages Channel</label>
          <div class="select-field">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <line x1="4" x2="20" y1="9" y2="9" /><line x1="4" x2="20" y1="15" y2="15" /><line x1="10" x2="8" y1="3" y2="21" /><line x1="16" x2="14" y1="3" y2="21" />
            </svg>
            <span class="select-text">{textChannels[1]?.name ?? "announcements"}</span>
            <div class="select-spacer"></div>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="m6 9 6 6 6-6" />
            </svg>
          </div>
        </div>
      </div>

      <div class="divider"></div>

      <div class="section">
        <h3 class="section-label danger-label">Danger Zone</h3>
        <p class="section-desc">This action will reset all server settings and channels to their defaults. This cannot be undone.</p>
        <div class="btn-row">
          <button class="danger-btn" type="button">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" /><path d="M3 3v5h5" />
            </svg>
            Reset Server
          </button>
        </div>
      </div>

      {#if error}
        <p class="error-text">{error}</p>
      {/if}

      <div class="save-row">
        <button class="cancel-btn" onclick={onClose} type="button">Cancel</button>
        <button class="save-btn" onclick={handleSave} disabled={saving} type="button">
          {saving ? "Saving..." : "Save Changes"}
        </button>
      </div>
    </div>
  </div>
</div>

<style>
  .server-settings {
    display: flex;
    flex-direction: column;
    height: 100%;
    width: 100%;
  }

  .top-bar {
    display: flex;
    align-items: center;
    height: 72px;
    padding: 0 24px;
    flex-shrink: 0;
  }

  .section-title {
    font-size: 16px;
    font-weight: 600;
    color: var(--foreground);
  }

  .top-divider {
    height: 1px;
    background: var(--border);
    flex-shrink: 0;
  }

  .content-scroll {
    flex: 1;
    overflow-y: auto;
    padding: 32px 48px;
  }

  .content-wrap {
    display: flex;
    flex-direction: column;
    gap: 32px;
    width: 600px;
  }

  .section {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .section-label {
    font-size: 14px;
    font-weight: 600;
    color: var(--foreground);
    margin: 0;
  }

  .danger-label {
    color: var(--destructive);
  }

  .section-desc {
    font-size: 13px;
    color: var(--muted-foreground);
    margin: 0;
    line-height: 1.4;
  }

  .divider {
    height: 1px;
    background: var(--border);
  }

  .server-row {
    display: flex;
    gap: 16px;
  }

  .icon-upload {
    width: 80px;
    height: 80px;
    border-radius: 12px;
    background: var(--muted);
    border: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 4px;
    color: var(--muted-foreground);
    cursor: pointer;
    flex-shrink: 0;
    transition: border-color 0.15s;
  }

  .icon-upload:hover {
    border-color: var(--ring);
  }

  .upload-text {
    font-size: 10px;
  }

  .fields-col {
    display: flex;
    flex-direction: column;
    gap: 12px;
    flex: 1;
  }

  .field-group {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .field-label {
    font-size: 13px;
    font-weight: 500;
    color: var(--foreground);
  }

  .field-input {
    height: 36px;
    padding: 0 12px;
    background: var(--muted);
    border: 1px solid transparent;
    border-radius: 6px;
    color: var(--foreground);
    font-size: 14px;
    font-family: inherit;
    outline: none;
  }

  .field-input:focus {
    border-color: var(--ring);
  }

  .field-input::placeholder,
  .field-textarea::placeholder {
    color: var(--muted-foreground);
  }

  .field-textarea {
    padding: 8px 12px;
    background: var(--muted);
    border: 1px solid transparent;
    border-radius: 6px;
    color: var(--foreground);
    font-size: 14px;
    font-family: inherit;
    outline: none;
    resize: vertical;
  }

  .field-textarea:focus {
    border-color: var(--ring);
  }

  .toggle-row {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .toggle-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
  }

  .toggle-label {
    font-size: 13px;
    font-weight: 500;
    color: var(--foreground);
  }

  .toggle-desc {
    font-size: 12px;
    color: var(--muted-foreground);
  }

  .toggle-switch {
    width: 40px;
    height: 22px;
    border-radius: 11px;
    background: var(--muted);
    padding: 2px;
    cursor: pointer;
    flex-shrink: 0;
  }

  .toggle-knob {
    width: 18px;
    height: 18px;
    border-radius: 9px;
    background: var(--muted-foreground);
    transition: transform 0.15s, background 0.15s;
  }

  .toggle-knob.on {
    transform: translateX(18px);
    background: var(--primary);
  }

  .select-field {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 36px;
    padding: 0 12px;
    background: var(--muted);
    border-radius: 6px;
    color: var(--muted-foreground);
    cursor: pointer;
  }

  .select-text {
    color: var(--foreground);
    font-size: 14px;
  }

  .select-spacer {
    flex: 1;
  }

  .danger-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 8px 16px;
    background: rgba(255, 68, 68, 0.13);
    border: none;
    border-radius: var(--radius);
    color: var(--destructive);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s;
  }

  .danger-btn:hover {
    background: rgba(255, 68, 68, 0.25);
  }

  .btn-row {
    display: flex;
    gap: 8px;
  }

  .error-text {
    font-size: 13px;
    color: var(--destructive);
    margin: 0;
  }

  .save-row {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }

  .cancel-btn {
    padding: 8px 16px;
    background: var(--secondary);
    border: none;
    border-radius: var(--radius);
    color: var(--foreground);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
  }

  .cancel-btn:hover {
    background: var(--muted);
  }

  .save-btn {
    padding: 8px 16px;
    background: var(--primary);
    border: none;
    border-radius: var(--radius);
    color: var(--primary-foreground);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .save-btn:hover {
    opacity: 0.9;
  }

  .save-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
