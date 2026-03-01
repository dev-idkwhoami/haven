<script lang="ts">
  import { trustAndAuth, rejectTrust } from "../../stores/servers.svelte.ts";

  interface Props {
    serverName: string;
    serverPubKey: string;
    onClose: () => void;
  }

  let { serverName, serverPubKey, onClose }: Props = $props();

  let loading = $state(false);
  let error = $state<string | null>(null);

  function formatFingerprint(hex: string): string {
    return hex
      .replace(/(.{4})/g, "$1 ")
      .trim()
      .toUpperCase();
  }

  async function handleTrust(): Promise<void> {
    loading = true;
    error = null;
    try {
      await trustAndAuth("");
      onClose();
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to trust server";
    } finally {
      loading = false;
    }
  }

  async function handleCancel(): Promise<void> {
    try {
      await rejectTrust();
    } catch {
      // ignore
    }
    onClose();
  }

  function handleKeydown(e: KeyboardEvent): void {
    if (e.key === "Escape") handleCancel();
  }
</script>

<div class="modal-backdrop" onclick={handleCancel} onkeydown={handleKeydown} role="presentation">
  <div class="modal" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" aria-label="Trust Server">
    <h3 class="modal-title">Trust This Server?</h3>

    <div class="modal-body">
      <p class="modal-desc">
        This is your first time connecting to <strong>{serverName}</strong>.
        Verify the server fingerprint with the server operator:
      </p>
      <div class="fingerprint-box">
        <code>{formatFingerprint(serverPubKey)}</code>
      </div>
      {#if error}
        <p class="error">{error}</p>
      {/if}
    </div>

    <div class="modal-footer">
      <button class="btn-secondary" onclick={handleCancel} type="button">Cancel</button>
      <button class="btn-primary" onclick={handleTrust} disabled={loading} type="button">
        {loading ? "Trusting..." : "Trust"}
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

  .modal-desc strong {
    color: var(--foreground);
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding: 16px 24px 24px;
  }

  .fingerprint-box {
    padding: 12px;
    background: var(--muted);
    border-radius: var(--radius);
    overflow-x: auto;
  }

  .fingerprint-box code {
    font-size: 12px;
    color: var(--foreground);
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
    word-break: break-all;
  }

  .error {
    color: var(--destructive);
    font-size: 13px;
    margin: 0;
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
