<script lang="ts">
  import { FileService, ProfileService } from "../wails";

  let displayName = $state("");
  let bio = $state("");
  let avatarPreview = $state<string | null>(null);
  let avatarPath = $state<string | null>(null);
  let pickingFile = $state(false);
  let loadingAvatar = $state(false);
  let saving = $state(false);
  let error = $state<string | null>(null);

  let canSave = $derived(displayName.trim().length > 0 && !saving && !loadingAvatar);

  async function pickAvatar(): Promise<void> {
    if (pickingFile || loadingAvatar) return;
    try {
      error = null;
      pickingFile = true;
      const path = await FileService.PickFile();
      pickingFile = false;
      if (!path) return;

      // Instant preview — just reads the file, no resize.
      const rawUrl = await ProfileService.ReadFileAsDataURL(path);
      if (rawUrl) {
        avatarPreview = rawUrl;
        avatarPath = path;
      }

      // Process (resize) in the background — replaces preview when done.
      loadingAvatar = true;
      await new Promise((r) => setTimeout(r, 0));
      const processedUrl = await ProfileService.PreviewAvatar(path);
      if (processedUrl) {
        avatarPreview = processedUrl;
      }
    } catch (e) {
      error = e instanceof Error ? e.message : "Unsupported image format";
    } finally {
      pickingFile = false;
      loadingAvatar = false;
    }
  }

  async function save(): Promise<void> {
    if (!canSave) return;
    saving = true;
    error = null;
    try {
      await ProfileService.UpdateProfile(displayName.trim(), bio.trim(), avatarPath ?? "");
      // The app:stateChanged event will transition phase to "ready"
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to save profile";
    } finally {
      saving = false;
    }
  }

  function handleKeydown(e: KeyboardEvent): void {
    if (e.key === "Enter" && !e.shiftKey && canSave) {
      e.preventDefault();
      save();
    }
  }
</script>

<div class="setup-screen">
  <div class="form-container">
    <div class="header">
      <h1 class="heading">Welcome to Haven</h1>
      <p class="subtitle">Create your profile</p>
    </div>

    <button class="avatar-section" onclick={pickAvatar} type="button">
      <div class="avatar-circle" class:loading={loadingAvatar}>
        {#if avatarPreview}
          <img src={avatarPreview} alt="Avatar" class="avatar-img" />
        {:else}
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z" />
            <circle cx="12" cy="13" r="4" />
          </svg>
        {/if}
      </div>
      <span class="avatar-label">{loadingAvatar ? "Processing..." : avatarPreview ? "Change Avatar" : "Upload Avatar"}</span>
      <span class="avatar-hint">PNG, JPG, GIF, or WebP</span>
    </button>

    <div class="fields">
      <div class="field">
        <label class="field-label" for="displayName">Display Name</label>
        <input
          id="displayName"
          type="text"
          class="field-input"
          placeholder="Enter your display name"
          bind:value={displayName}
          onkeydown={handleKeydown}
        />
      </div>

      <div class="field">
        <label class="field-label" for="bio">Bio</label>
        <textarea
          id="bio"
          class="field-textarea"
          placeholder="Say something about you (optional)"
          bind:value={bio}
          rows="3"
        ></textarea>
      </div>
    </div>

    {#if error}
      <p class="error">{error}</p>
    {/if}

    <button class="save-btn" onclick={save} disabled={!canSave} type="button">
      {saving ? "Saving..." : "Save Profile"}
    </button>
  </div>
</div>

<style>
  .setup-screen {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    width: 100%;
    background: var(--background);
    overflow-y: auto;
    padding: 48px 0;
  }

  .form-container {
    display: flex;
    flex-direction: column;
    gap: 32px;
    width: 400px;
  }

  .header {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
  }

  .heading {
    font-size: 28px;
    font-weight: 600;
    color: var(--foreground);
    text-align: center;
    margin: 0;
  }

  .subtitle {
    font-size: 16px;
    color: var(--muted-foreground);
    text-align: center;
    margin: 0;
  }

  .avatar-section {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    background: none;
    border: none;
    cursor: pointer;
    padding: 0;
  }

  .avatar-circle {
    width: 96px;
    height: 96px;
    border-radius: 9999px;
    background: var(--muted);
    border: 2px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted-foreground);
    transition: border-color 0.15s;
  }

  .avatar-section:hover .avatar-circle {
    border-color: var(--foreground);
  }

  .avatar-circle.loading {
    animation: avatar-pulse 1.2s ease-in-out infinite;
    pointer-events: none;
  }

  @keyframes avatar-pulse {
    0%, 100% {
      border-color: var(--primary);
      opacity: 1;
    }
    50% {
      border-color: var(--primary);
      opacity: 0.5;
    }
  }

  .avatar-img {
    width: 100%;
    height: 100%;
    border-radius: 9999px;
    object-fit: cover;
  }

  .avatar-label {
    font-size: 14px;
    font-weight: 500;
    color: var(--muted-foreground);
  }

  .avatar-hint {
    font-size: 12px;
    color: var(--muted-foreground);
    opacity: 0.6;
  }

  .fields {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .field-label {
    font-size: 14px;
    font-weight: 500;
    color: var(--foreground);
  }

  .field-input,
  .field-textarea {
    width: 100%;
    padding: 10px 12px;
    background: transparent;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    color: var(--foreground);
    font-size: 14px;
    outline: none;
    transition: border-color 0.15s;
  }

  .field-input:focus,
  .field-textarea:focus {
    border-color: var(--foreground);
  }

  .field-input::placeholder,
  .field-textarea::placeholder {
    color: var(--muted-foreground);
  }

  .field-textarea {
    resize: vertical;
    min-height: 80px;
  }

  .error {
    color: var(--destructive);
    font-size: 13px;
    text-align: center;
    margin: 0;
  }

  .save-btn {
    width: 100%;
    padding: 12px;
    background: var(--primary);
    color: var(--primary-foreground);
    border: none;
    border-radius: var(--radius);
    font-size: 14px;
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
