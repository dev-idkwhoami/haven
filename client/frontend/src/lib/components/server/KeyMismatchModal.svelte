<script lang="ts">
  import { trustAndAuth, rejectTrust } from "../../stores/servers.svelte.ts";

  interface Props {
    serverName: string;
    serverPubKey: string;
    storedPubKey: string;
    onClose: () => void;
  }

  let { serverName, serverPubKey, storedPubKey, onClose }: Props = $props();

  let loading = $state(false);
  let error = $state<string | null>(null);

  function formatFingerprint(hex: string): string {
    return hex
      .replace(/(.{4})/g, "$1 ")
      .trim()
      .toUpperCase();
  }

  async function handleRetrust(): Promise<void> {
    loading = true;
    error = null;
    try {
      await trustAndAuth("");
      onClose();
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to re-trust server";
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
  <div class="modal" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" aria-label="Server Key Changed">
    <h3 class="modal-title">Server Key Changed</h3>

    <div class="modal-body">
      <div class="warning-banner">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z" /><line x1="12" x2="12" y1="9" y2="13" /><line x1="12" x2="12.01" y1="17" y2="17" />
        </svg>
        <span>
          The identity key for <strong>{serverName}</strong> has changed since you last connected.
          This could indicate a man-in-the-middle attack or a legitimate server migration.
        </span>
      </div>

      <div class="key-compare">
        <div class="key-section">
          <span class="key-label">Previously stored key:</span>
          <code class="fingerprint-code">{formatFingerprint(storedPubKey)}</code>
        </div>
        <div class="key-section">
          <span class="key-label">Current server key:</span>
          <code class="fingerprint-code">{formatFingerprint(serverPubKey)}</code>
        </div>
      </div>

      {#if error}
        <p class="error">{error}</p>
      {/if}
    </div>

    <div class="modal-footer">
      <button class="btn-primary" onclick={handleCancel} type="button" autofocus>
        Cancel
      </button>
      <button
        class="btn-destructive"
        onclick={handleRetrust}
        disabled={loading}
        type="button"
      >
        {loading ? "Re-trusting..." : "Re-trust"}
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
    color: var(--destructive);
    margin: 0;
    padding: 24px 24px 0;
  }

  .modal-body {
    padding: 16px 24px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding: 16px 24px 24px;
  }

  .warning-banner {
    display: flex;
    gap: 12px;
    padding: 12px;
    background: rgba(255, 102, 105, 0.1);
    border: 1px solid rgba(255, 102, 105, 0.3);
    border-radius: var(--radius);
    color: var(--destructive);
    font-size: 13px;
    line-height: 1.5;
    align-items: flex-start;
  }

  .warning-banner svg {
    flex-shrink: 0;
    margin-top: 1px;
  }

  .warning-banner strong {
    color: var(--foreground);
  }

  .key-compare {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .key-section {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .key-label {
    font-size: 12px;
    font-weight: 500;
    color: var(--muted-foreground);
  }

  .fingerprint-code {
    font-size: 11px;
    color: var(--foreground);
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
    background: var(--muted);
    padding: 8px;
    border-radius: 4px;
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

  .btn-primary:hover {
    opacity: 0.9;
  }

  .btn-destructive {
    padding: 10px 20px;
    background: rgba(255, 102, 105, 0.15);
    border: 1px solid rgba(255, 102, 105, 0.3);
    border-radius: var(--radius);
    color: var(--destructive);
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s;
  }

  .btn-destructive:hover:not(:disabled) {
    background: rgba(255, 102, 105, 0.25);
  }

  .btn-destructive:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
