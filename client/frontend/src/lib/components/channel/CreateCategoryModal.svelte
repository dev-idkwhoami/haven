<script lang="ts">
  import { createCategory } from "../../stores/channels.svelte.ts";

  interface Props {
    serverId: number;
    onClose: () => void;
  }

  let { serverId, onClose }: Props = $props();

  let name = $state("");
  let type = $state<"text" | "voice">("text");
  let error = $state<string | null>(null);
  let saving = $state(false);

  let canSave = $derived(name.trim().length > 0 && !saving);

  async function handleCreate(): Promise<void> {
    if (!canSave) return;
    saving = true;
    error = null;
    try {
      await createCategory(serverId, name.trim(), type);
      onClose();
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to create category";
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
  <div class="modal" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" aria-label="Create Category">
    <h3 class="modal-title">Create Category</h3>

    <div class="modal-body">
      <div class="field">
        <label class="field-label" for="categoryName">Category Name</label>
        <input
          id="categoryName"
          type="text"
          class="field-input"
          placeholder="Enter category name"
          bind:value={name}
          onkeydown={handleKeydown}
          autofocus
        />
      </div>

      <div class="segmented-control">
        <button
          class="segment"
          class:active={type === "text"}
          onclick={() => (type = "text")}
          type="button"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <line x1="4" x2="20" y1="9" y2="9" /><line x1="4" x2="20" y1="15" y2="15" /><line x1="10" x2="8" y1="3" y2="21" /><line x1="16" x2="14" y1="3" y2="21" />
          </svg>
          Text
        </button>
        <button
          class="segment"
          class:active={type === "voice"}
          onclick={() => (type = "voice")}
          type="button"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M2 10v3" /><path d="M6 6v11" /><path d="M10 3v18" /><path d="M14 8v7" /><path d="M18 5v13" /><path d="M22 10v3" />
          </svg>
          Voice
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
        {saving ? "Creating..." : "Create Category"}
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

  .field-input {
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

  .field-input:focus {
    border-color: var(--foreground);
  }

  .field-input::placeholder {
    color: var(--muted-foreground);
  }

  .segmented-control {
    display: flex;
    background: var(--muted);
    border-radius: 8px;
    padding: 4px;
    height: 40px;
  }

  .segment {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    border: none;
    border-radius: 6px;
    background: transparent;
    color: var(--muted-foreground);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s, color 0.15s, box-shadow 0.15s;
  }

  .segment.active {
    background: var(--background);
    color: var(--foreground);
    font-weight: 600;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
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
