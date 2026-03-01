<script lang="ts">
  import type { Message } from "../../types";
  import { editMessage } from "../../stores/messages.svelte.ts";
  import { activeServerId } from "../../stores/servers.svelte.ts";

  interface Props {
    message: Message;
    onClose: () => void;
  }

  let { message, onClose }: Props = $props();

  let content = $state(message.content);
  let saving = $state(false);
  let error = $state<string | null>(null);

  let canSave = $derived(content.trim().length > 0 && content.trim() !== message.content && !saving);

  async function handleSave(): Promise<void> {
    if (!canSave || activeServerId() === null) return;
    saving = true;
    error = null;
    try {
      await editMessage(activeServerId()!, message.remoteMessageId, content.trim());
      onClose();
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to edit message";
    } finally {
      saving = false;
    }
  }

  function handleKeydown(e: KeyboardEvent): void {
    if (e.key === "Escape") onClose();
    if (e.key === "Enter" && !e.shiftKey && canSave) {
      e.preventDefault();
      handleSave();
    }
  }
</script>

<div class="modal-backdrop" onclick={onClose} onkeydown={handleKeydown} role="presentation">
  <div class="modal" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" aria-label="Edit Message">
    <h3 class="modal-title">Edit Message</h3>

    <div class="modal-body">
      <textarea
        class="edit-textarea"
        bind:value={content}
        onkeydown={handleKeydown}
        rows="3"
        autofocus
      ></textarea>

      {#if error}
        <p class="error">{error}</p>
      {/if}
    </div>

    <div class="modal-footer">
      <button class="btn-secondary" onclick={onClose} type="button">Cancel</button>
      <button class="btn-primary" onclick={handleSave} disabled={!canSave} type="button">
        {saving ? "Saving..." : "Save"}
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
    width: 480px;
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
    padding: 16px 24px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .edit-textarea {
    width: 100%;
    padding: 10px 12px;
    background: transparent;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    color: var(--foreground);
    font-size: 14px;
    outline: none;
    resize: vertical;
    min-height: 80px;
    line-height: 1.5;
    transition: border-color 0.15s;
  }

  .edit-textarea:focus {
    border-color: var(--foreground);
  }

  .error {
    color: var(--destructive);
    font-size: 13px;
    margin: 0;
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding: 0 24px 24px;
  }

  .btn-primary {
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

  .btn-primary:hover:not(:disabled) { opacity: 0.9; }
  .btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

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

  .btn-secondary:hover { background: var(--secondary); }
</style>
