<script lang="ts">
  import { X } from "lucide-svelte";
  import { createDM } from "../../stores/dms.svelte.ts";

  interface Props {
    onClose: () => void;
  }

  let { onClose }: Props = $props();

  let publicKey = $state("");
  let loading = $state(false);
  let error = $state<string | null>(null);

  async function handleCreate(): Promise<void> {
    if (!publicKey.trim()) return;
    loading = true;
    error = null;
    try {
      await createDM([publicKey.trim()]);
      onClose();
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to create DM";
    } finally {
      loading = false;
    }
  }

  function handleKeydown(e: KeyboardEvent): void {
    if (e.key === "Escape") onClose();
    if (e.key === "Enter" && publicKey.trim() && !loading) handleCreate();
  }
</script>

<div class="modal-backdrop" onclick={onClose} onkeydown={handleKeydown} role="presentation">
  <div class="modal" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" aria-label="New Direct Message">
    <button class="close-btn" onclick={onClose} title="Close" type="button">
      <X size={18} />
    </button>

    <h3 class="modal-title">New Direct Message</h3>
    <div class="modal-body">
      <p class="modal-desc">Enter the public key of the person you want to message.</p>
      <div class="field">
        <input
          type="text"
          class="field-input"
          placeholder="Public key (hex)"
          bind:value={publicKey}
          onkeydown={handleKeydown}
          autofocus
        />
      </div>

      {#if error}
        <p class="error">{error}</p>
      {/if}
    </div>
    <div class="modal-footer">
      <button class="btn-secondary" onclick={onClose} type="button">Cancel</button>
      <button
        class="btn-primary"
        onclick={handleCreate}
        disabled={!publicKey.trim() || loading}
        type="button"
      >
        {loading ? "Creating..." : "Start Conversation"}
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
    position: relative;
  }

  .close-btn {
    position: absolute;
    top: 16px;
    right: 16px;
    width: 28px;
    height: 28px;
    border-radius: 6px;
    background: none;
    border: none;
    color: var(--muted-foreground);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: color 0.15s, background 0.15s;
    z-index: 1;
  }

  .close-btn:hover {
    color: var(--foreground);
    background: rgba(255, 255, 255, 0.06);
  }

  .modal-title {
    font-size: 18px;
    font-weight: 600;
    color: var(--foreground);
    margin: 0;
    padding: 24px 24px 0;
  }

  .modal-body {
    padding: 16px 24px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .modal-desc {
    font-size: 14px;
    color: var(--muted-foreground);
    margin: 0;
    line-height: 1.5;
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding: 16px 24px 24px;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 6px;
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
  }

  .btn-primary:hover:not(:disabled) {
    opacity: 0.9;
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-secondary {
    padding: 10px 20px;
    background: transparent;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    color: var(--foreground);
    font-size: 14px;
    cursor: pointer;
    transition: background 0.15s;
  }

  .btn-secondary:hover {
    background: var(--secondary);
  }
</style>
