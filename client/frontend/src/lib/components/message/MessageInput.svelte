<script lang="ts">
  import { FileService } from "../../wails";

  interface Props {
    onSend: (content: string) => void;
    onAttach?: () => void;
    disabled?: boolean;
  }

  let { onSend, onAttach, disabled = false }: Props = $props();

  let content = $state("");
  let inputEl = $state<HTMLTextAreaElement | null>(null);

  function handleKeydown(e: KeyboardEvent): void {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      send();
    }
  }

  function send(): void {
    const trimmed = content.trim();
    if (!trimmed || disabled) return;
    onSend(trimmed);
    content = "";
    if (inputEl) {
      inputEl.style.height = "auto";
    }
  }

  async function handleAttach(): Promise<void> {
    if (onAttach) {
      onAttach();
    } else {
      try {
        await FileService.PickFile();
      } catch {
        // cancelled
      }
    }
  }

  function handleInput(): void {
    if (inputEl) {
      inputEl.style.height = "auto";
      inputEl.style.height = Math.min(inputEl.scrollHeight, 120) + "px";
    }
  }
</script>

<div class="input-bar">
  <div class="input-row">
    <button class="icon-btn" onclick={handleAttach} title="Attach file" type="button">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <path d="m21.44 11.05-9.19 9.19a6 6 0 0 1-8.49-8.49l8.57-8.57A4 4 0 1 1 18 8.84l-8.59 8.57a2 2 0 0 1-2.83-2.83l8.49-8.48" />
      </svg>
    </button>

    <textarea
      bind:this={inputEl}
      bind:value={content}
      class="text-input"
      placeholder="Type a message..."
      rows="1"
      onkeydown={handleKeydown}
      oninput={handleInput}
      {disabled}
    ></textarea>

    <button
      class="send-btn"
      onclick={send}
      disabled={!content.trim() || disabled}
      title="Send message"
      type="button"
    >
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <path d="M14.536 21.686a.5.5 0 0 0 .937-.024l6.5-19a.496.496 0 0 0-.635-.635l-19 6.5a.5.5 0 0 0-.024.937l7.93 3.18a2 2 0 0 1 1.112 1.11z" />
        <path d="m21.854 2.147-10.94 10.939" />
      </svg>
    </button>
  </div>
</div>

<style>
  .input-bar {
    display: flex;
    align-items: center;
    justify-content: center;
    height: auto;
    min-height: 73px;
    padding: 16px 24px;
  }

  .input-row {
    display: flex;
    align-items: flex-end;
    gap: 8px;
    background: var(--muted);
    border-radius: 8px;
    padding: 0 4px;
    width: 100%;
    max-width: 864px;
    min-height: 40px;
  }

  .icon-btn {
    width: 32px;
    height: 32px;
    border-radius: 6px;
    background: none;
    border: none;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted-foreground);
    cursor: pointer;
    flex-shrink: 0;
    margin-bottom: 4px;
    transition: color 0.15s;
  }

  .icon-btn:hover {
    color: var(--foreground);
  }

  .text-input {
    flex: 1;
    background: none;
    border: none;
    color: var(--foreground);
    font-size: 14px;
    padding: 10px 0;
    outline: none;
    resize: none;
    line-height: 1.4;
    max-height: 120px;
    overflow-y: auto;
  }

  .text-input::placeholder {
    color: var(--muted-foreground);
  }

  .send-btn {
    width: 32px;
    height: 32px;
    border-radius: 6px;
    background: var(--primary);
    border: none;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--primary-foreground);
    cursor: pointer;
    flex-shrink: 0;
    margin-bottom: 4px;
    transition: opacity 0.15s;
  }

  .send-btn:hover:not(:disabled) {
    opacity: 0.9;
  }

  .send-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
</style>
