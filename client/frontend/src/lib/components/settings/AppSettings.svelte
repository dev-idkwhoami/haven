<script lang="ts">
  import {
    appSettings,
    loadAppSettings,
    updateAppSettings,
  } from "../../stores/settings.svelte.ts";
  import { ProfileService, FileService } from "../../wails";

  interface Props {
    onClose: () => void;
  }

  let { onClose }: Props = $props();

  let displayName = $state("");
  let bio = $state("");
  let fingerprint = $state("");
  let saving = $state(false);

  $effect(() => {
    loadAppSettings();
    ProfileService.GetProfile()
      .then((p) => {
        displayName = p.displayName;
        bio = p.bio ?? "";
      })
      .catch(() => {});
    ProfileService.GetPublicKey()
      .then((key) => {
        fingerprint = key;
      })
      .catch(() => {});
  });

  async function handleSaveProfile(): Promise<void> {
    if (!displayName.trim()) return;
    saving = true;
    try {
      await ProfileService.UpdateProfile(displayName.trim(), bio.trim(), "");
    } catch {
      // error
    } finally {
      saving = false;
    }
  }

  async function handleAvatarUpload(): Promise<void> {
    try {
      const path = await FileService.PickFile();
      if (path) {
        await ProfileService.SetAvatar(path);
      }
    } catch {
      // cancelled
    }
  }

  async function handleExportKey(): Promise<void> {
    try {
      const path = await FileService.PickFile();
      if (path) {
        await ProfileService.ExportIdentity(path);
      }
    } catch {
      // cancelled
    }
  }

  function formatFingerprint(key: string): string {
    if (!key) return "";
    const hex = key.slice(0, 48);
    return hex.match(/.{1,2}/g)?.join(":") ?? hex;
  }
</script>

<div class="app-settings">
  <div class="top-bar">
    <span class="section-title">Profile</span>
  </div>
  <div class="top-divider"></div>

  <div class="content-scroll">
    <div class="content-wrap">
      <div class="section">
        <h3 class="section-label">Profile</h3>
        <p class="section-desc">Set your display name and avatar. These are visible to other members on servers you join.</p>

        <div class="profile-row">
          <button class="avatar-upload" onclick={handleAvatarUpload} type="button">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z" /><circle cx="12" cy="13" r="3" />
            </svg>
            <span class="upload-text">Upload</span>
          </button>
          <div class="fields-col">
            <div class="field-group">
              <label class="field-label">Display Name</label>
              <input class="field-input" bind:value={displayName} placeholder="Display name" />
            </div>
            <div class="field-group">
              <label class="field-label">About</label>
              <textarea
                class="field-textarea"
                bind:value={bio}
                placeholder="Any info, a song lyric, whatever."
                rows="3"
              ></textarea>
            </div>
          </div>
        </div>
      </div>

      <div class="divider"></div>

      <div class="section">
        <h3 class="section-label">Identity</h3>
        <div class="identity-row">
          <div class="identity-text">
            <span class="identity-label">Public Key Fingerprint</span>
            <code class="identity-value">{formatFingerprint(fingerprint)}</code>
          </div>
        </div>
      </div>

      <div class="divider"></div>

      <div class="section">
        <h3 class="section-label">Key Management</h3>
        <p class="section-desc">Export your private key for backup. Keep it safe — anyone with your private key can impersonate you.</p>
        <div class="btn-row">
          <button class="export-btn" onclick={handleExportKey} type="button">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" /><path d="M3 3v5h5" />
            </svg>
            Export Private Key
          </button>
        </div>
      </div>

      <div class="save-row">
        <button class="cancel-btn" onclick={onClose} type="button">Cancel</button>
        <button class="save-btn" onclick={handleSaveProfile} disabled={saving} type="button">
          {saving ? "Saving..." : "Save Changes"}
        </button>
      </div>
    </div>
  </div>
</div>

<style>
  .app-settings {
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

  .top-divider, .divider {
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

  .section-desc {
    font-size: 13px;
    color: var(--muted-foreground);
    margin: 0;
    line-height: 1.4;
  }

  .profile-row {
    display: flex;
    gap: 16px;
  }

  .avatar-upload {
    width: 80px;
    height: 80px;
    border-radius: 9999px;
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

  .avatar-upload:hover {
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

  .identity-row {
    display: flex;
    align-items: center;
  }

  .identity-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .identity-label {
    font-size: 13px;
    font-weight: 500;
    color: var(--foreground);
  }

  .identity-value {
    font-size: 12px;
    color: var(--muted-foreground);
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
  }

  .btn-row {
    display: flex;
    gap: 8px;
  }

  .export-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
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

  .export-btn:hover {
    opacity: 0.9;
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

  .save-btn:hover:not(:disabled) {
    opacity: 0.9;
  }

  .save-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
