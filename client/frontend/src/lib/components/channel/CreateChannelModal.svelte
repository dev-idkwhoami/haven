<script lang="ts">
  import { categories, createChannel } from "../../stores/channels.svelte.ts";

  interface Props {
    serverId: number;
    onClose: () => void;
  }

  let { serverId, onClose }: Props = $props();

  let name = $state("");
  let selectedCategoryId = $state("");
  let isPrivate = $state(false);
  let error = $state<string | null>(null);
  let saving = $state(false);

  let sortedCategories = $derived(
    [...categories()].sort((a, b) => a.position - b.position),
  );

  $effect(() => {
    if (sortedCategories.length > 0 && !selectedCategoryId) {
      selectedCategoryId = sortedCategories[0].remoteCategoryId;
    }
  });

  let canSave = $derived(name.trim().length > 0 && selectedCategoryId && !saving);

  async function handleCreate(): Promise<void> {
    if (!canSave) return;
    saving = true;
    error = null;
    try {
      await createChannel(serverId, selectedCategoryId, name.trim(), "text");
      onClose();
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to create channel";
    } finally {
      saving = false;
    }
  }

  function handleKeydown(e: KeyboardEvent): void {
    if (e.key === "Escape") onClose();
    if (e.key === "Enter" && canSave) handleCreate();
  }
</script>

<div class="modal-backdrop" onclick={onClose} onkeydown={handleKeydown} role="presentation">
  <div class="modal" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" aria-label="Create Channel">
    <h3 class="modal-title">Create Channel</h3>

    <div class="modal-body">
      <div class="field">
        <label class="field-label" for="channelName">Channel Name</label>
        <input
          id="channelName"
          type="text"
          class="field-input"
          placeholder="Enter channel name"
          bind:value={name}
          onkeydown={handleKeydown}
          autofocus
        />
      </div>

      <div class="field">
        <label class="field-label" for="category">Category</label>
        <select id="category" class="field-select" bind:value={selectedCategoryId}>
          {#each sortedCategories as cat (cat.remoteCategoryId)}
            <option value={cat.remoteCategoryId}>{cat.name}</option>
          {/each}
        </select>
      </div>

      <div class="toggle-row">
        <div class="toggle-info">
          <span class="toggle-title">Private Channel</span>
          <span class="toggle-desc">Only selected members can view this channel</span>
        </div>
        <button
          class="toggle-switch"
          class:active={isPrivate}
          onclick={() => (isPrivate = !isPrivate)}
          type="button"
          role="switch"
          aria-checked={isPrivate}
        >
          <div class="toggle-thumb"></div>
        </button>
      </div>

      {#if error}
        <p class="error">{error}</p>
      {/if}
    </div>

    <div class="modal-footer">
      <button
        class="btn-primary"
        onclick={handleCreate}
        disabled={!canSave}
        type="button"
      >
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
        </svg>
        {saving ? "Creating..." : "Create Channel"}
      </button>
    </div>
  </div>
</div>

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.65);
    backdrop-filter: blur(8px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }

  .modal {
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: 12px;
    width: 420px;
    max-width: 90vw;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .modal-title {
    font-size: 18px;
    font-weight: 600;
    color: var(--foreground);
    margin: 0;
    padding: 24px 24px 0;
  }

  .modal-body {
    padding: 20px 24px;
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    padding: 0 24px 24px;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .field-label {
    font-size: 13px;
    font-weight: 500;
    color: var(--foreground);
  }

  .field-input,
  .field-select {
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
  .field-select:focus {
    border-color: var(--foreground);
  }

  .field-input::placeholder {
    color: var(--muted-foreground);
  }

  .field-select {
    cursor: pointer;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='%23a3a3a3' stroke-width='2'%3E%3Cpath d='m6 9 6 6 6-6'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 12px center;
    padding-right: 36px;
  }

  .field-select option {
    background: var(--card);
    color: var(--foreground);
  }

  .toggle-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-top: 12px;
  }

  .toggle-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .toggle-title {
    font-size: 14px;
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
    border: none;
    cursor: pointer;
    position: relative;
    transition: background 0.2s;
    flex-shrink: 0;
  }

  .toggle-switch.active {
    background: var(--primary);
  }

  .toggle-thumb {
    width: 18px;
    height: 18px;
    border-radius: 50%;
    background: var(--foreground);
    position: absolute;
    top: 2px;
    left: 2px;
    transition: transform 0.2s;
  }

  .toggle-switch.active .toggle-thumb {
    transform: translateX(18px);
    background: var(--primary-foreground);
  }

  .error {
    color: var(--destructive);
    font-size: 13px;
    margin: 0;
  }

  .btn-primary {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 20px;
    background: var(--primary);
    color: var(--primary-foreground);
    border: none;
    border-radius: var(--radius);
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s;
    width: 100%;
    justify-content: center;
  }

  .btn-primary:hover:not(:disabled) {
    opacity: 0.9;
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
