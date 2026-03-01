<script lang="ts">
  import { X } from "lucide-svelte";
  import type { ServerHello, TrustStatus } from "../../types";
  import { connect, trustAndAuth, rejectTrust, submitAccessRequest, cancelAccessRequest } from "../../stores/servers.svelte.ts";
  import { on } from "../../wails";

  type Step = "address" | "trust" | "credentials" | "connecting" | "waiting_room";

  interface Props {
    onClose: () => void;
    initialStep?: Step;
  }

  let { onClose, initialStep = "address" }: Props = $props();

  let step = $state<Step>(initialStep);
  let address = $state("");
  let hello = $state<ServerHello | null>(null);
  let accessToken = $state("");
  let error = $state<string | null>(null);
  let loading = $state(false);
  let requestMessage = $state("");
  let waitingForDecision = $state(false);
  let accessResult = $state<"approved" | "rejected" | "timeout" | null>(null);

  let trustStatus = $derived<TrustStatus | null>(hello?.trustStatus ?? null);
  let accessMode = $derived(hello?.accessMode ?? "open");
  let needsCredentials = $derived(accessMode === "invite" || accessMode === "password");

  function isWaitingRoomError(e: unknown): boolean {
    const msg = e instanceof Error ? e.message : String(e);
    return msg.includes("waiting_room");
  }

  async function handleConnect(): Promise<void> {
    if (!address.trim()) return;
    loading = true;
    error = null;
    try {
      hello = await connect(address.trim());
      if (hello.trustStatus === "new" || hello.trustStatus === "mismatch") {
        step = "trust";
      } else if (needsCredentials) {
        step = "credentials";
      } else {
        step = "connecting";
        await trustAndAuth("");
        onClose();
      }
    } catch (e) {
      if (isWaitingRoomError(e)) {
        step = "waiting_room";
      } else {
        error = e instanceof Error ? e.message : "Connection failed";
      }
    } finally {
      loading = false;
    }
  }

  async function handleTrust(): Promise<void> {
    loading = true;
    error = null;
    try {
      if (needsCredentials) {
        step = "credentials";
      } else {
        await trustAndAuth("");
        onClose();
      }
    } catch (e) {
      if (isWaitingRoomError(e)) {
        step = "waiting_room";
      } else {
        error = e instanceof Error ? e.message : "Trust failed";
      }
    } finally {
      loading = false;
    }
  }

  async function handleReject(): Promise<void> {
    try {
      await rejectTrust();
    } catch {
      // ignore
    }
    onClose();
  }

  async function handleAuth(): Promise<void> {
    loading = true;
    error = null;
    try {
      await trustAndAuth(accessToken.trim());
      onClose();
    } catch (e) {
      if (isWaitingRoomError(e)) {
        step = "waiting_room";
      } else {
        error = e instanceof Error ? e.message : "Authentication failed";
      }
    } finally {
      loading = false;
    }
  }

  function handleKeydown(e: KeyboardEvent): void {
    if (e.key === "Escape") {
      if (step !== "address") {
        handleReject();
      } else {
        onClose();
      }
    }
    if (e.key === "Enter") {
      if (step === "address" && address.trim() && !loading) handleConnect();
      if (step === "credentials" && accessToken.trim() && !loading) handleAuth();
    }
  }

  function formatFingerprint(hex: string): string {
    return hex
      .replace(/(.{4})/g, "$1 ")
      .trim()
      .toUpperCase();
  }

  async function handleSubmitAccessRequest(): Promise<void> {
    loading = true;
    error = null;
    try {
      await submitAccessRequest(requestMessage.trim());
      waitingForDecision = true;
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to submit request";
    } finally {
      loading = false;
    }
  }

  async function handleCancelAccessRequest(): Promise<void> {
    try {
      await cancelAccessRequest();
    } catch {
      // ignore
    }
    onClose();
  }

  $effect(() => {
    const unsubApproved = on("access_request:approved", () => {
      accessResult = "approved";
      waitingForDecision = false;
    });
    const unsubRejected = on("access_request:rejected", () => {
      accessResult = "rejected";
      waitingForDecision = false;
    });
    const unsubTimeout = on("access_request:timeout", () => {
      accessResult = "timeout";
      waitingForDecision = false;
    });
    return () => {
      unsubApproved();
      unsubRejected();
      unsubTimeout();
    };
  });
</script>

<div class="modal-backdrop" onclick={() => step === "address" ? onClose() : handleReject()} onkeydown={handleKeydown} role="presentation">
  <div class="modal" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" aria-label="Join Server">
    <button
      class="close-btn"
      onclick={() => step === "address" ? onClose() : handleReject()}
      title="Close"
      type="button"
    >
      <X size={18} />
    </button>

    {#if step === "address"}
      <h3 class="modal-title">Join a Server</h3>
      <div class="modal-body">
        <div class="field">
          <input
            type="text"
            class="field-input"
            placeholder="server.example.com"
            bind:value={address}
            onkeydown={handleKeydown}
            autofocus
          />
        </div>

        {#if error}
          <p class="error">{error}</p>
        {/if}
      </div>
      <div class="modal-footer">
        <button
          class="btn-primary"
          onclick={handleConnect}
          disabled={!address.trim() || loading}
          type="button"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4" /><polyline points="10 17 15 12 10 7" /><line x1="15" x2="3" y1="12" y2="12" />
          </svg>
          {loading ? "Connecting..." : "Connect"}
        </button>
      </div>

    {:else if step === "trust" && hello}
      {#if trustStatus === "new"}
        <h3 class="modal-title">Trust This Server?</h3>
        <div class="modal-body">
          <p class="modal-desc">
            This is your first time connecting to <strong>{hello.serverName || address}</strong>.
            Verify the server fingerprint:
          </p>
          <div class="fingerprint-box">
            <code>{formatFingerprint(hello.serverPubKey)}</code>
          </div>
          {#if error}
            <p class="error">{error}</p>
          {/if}
        </div>
        <div class="modal-footer">
          <button class="btn-secondary" onclick={handleReject} type="button">Cancel</button>
          <button
            class="btn-primary"
            onclick={handleTrust}
            disabled={loading}
            type="button"
          >
            {loading ? "Trusting..." : "Trust"}
          </button>
        </div>

      {:else if trustStatus === "mismatch"}
        <h3 class="modal-title warning-title">Server Key Changed</h3>
        <div class="modal-body">
          <div class="warning-banner">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z" /><line x1="12" x2="12" y1="9" y2="13" /><line x1="12" x2="12.01" y1="17" y2="17" />
            </svg>
            <span>The server's identity key has changed. This could indicate a man-in-the-middle attack.</span>
          </div>

          <div class="key-compare">
            <div class="key-section">
              <span class="key-label">Previously stored key:</span>
              <code class="fingerprint-code">{formatFingerprint(hello.storedPubKey)}</code>
            </div>
            <div class="key-section">
              <span class="key-label">Current server key:</span>
              <code class="fingerprint-code">{formatFingerprint(hello.serverPubKey)}</code>
            </div>
          </div>

          {#if error}
            <p class="error">{error}</p>
          {/if}
        </div>
        <div class="modal-footer">
          <button
            class="btn-primary"
            onclick={handleReject}
            type="button"
            autofocus
          >Cancel</button>
          <button
            class="btn-destructive"
            onclick={handleTrust}
            disabled={loading}
            type="button"
          >
            {loading ? "Re-trusting..." : "Re-trust"}
          </button>
        </div>
      {/if}

    {:else if step === "credentials" && hello}
      <h3 class="modal-title">
        {accessMode === "invite" ? "Enter Invite Code" : "Enter Password"}
      </h3>
      <div class="modal-body">
        <p class="modal-desc">
          <strong>{hello.serverName || address}</strong> requires
          {accessMode === "invite" ? "an invite code" : "a password"} to join.
        </p>
        <div class="field">
          <input
            type={accessMode === "password" ? "password" : "text"}
            class="field-input"
            placeholder={accessMode === "invite" ? "Enter invite code" : "Enter password"}
            bind:value={accessToken}
            onkeydown={handleKeydown}
            autofocus
          />
        </div>
        {#if error}
          <p class="error">{error}</p>
        {/if}
      </div>
      <div class="modal-footer">
        <button class="btn-secondary" onclick={handleReject} type="button">Cancel</button>
        <button
          class="btn-primary"
          onclick={handleAuth}
          disabled={!accessToken.trim() || loading}
          type="button"
        >
          {loading ? "Authenticating..." : "Join"}
        </button>
      </div>

    {:else if step === "waiting_room"}
      <h3 class="modal-title">Access Required</h3>
      <div class="modal-body">
        {#if accessResult === "approved"}
          <p class="modal-desc success-text">Access granted! You can now connect to the server.</p>
          <div class="modal-footer">
            <button class="btn-primary" onclick={onClose} type="button">Close</button>
          </div>
        {:else if accessResult === "rejected"}
          <p class="modal-desc">Your access request was denied by an administrator.</p>
          <div class="modal-footer">
            <button class="btn-secondary" onclick={onClose} type="button">Close</button>
          </div>
        {:else if accessResult === "timeout"}
          <p class="modal-desc">Your access request timed out. Please try again later.</p>
          <div class="modal-footer">
            <button class="btn-secondary" onclick={onClose} type="button">Close</button>
          </div>
        {:else if waitingForDecision}
          <p class="modal-desc">
            Your request has been submitted. Waiting for an administrator to review...
          </p>
          <div class="waiting-spinner">
            <div class="spinner"></div>
            <span>Waiting for approval...</span>
          </div>
          <div class="modal-footer">
            <button class="btn-secondary" onclick={handleCancelAccessRequest} type="button">Cancel</button>
          </div>
        {:else}
          <p class="modal-desc">
            <strong>{hello?.serverName || address}</strong> requires approval to join.
            You can send a message to the administrators with your request.
          </p>
          <div class="field">
            <textarea
              class="field-input field-textarea"
              placeholder="Why do you want to join? (optional)"
              bind:value={requestMessage}
              rows="3"
            ></textarea>
          </div>
          {#if error}
            <p class="error">{error}</p>
          {/if}
        {/if}
      </div>
      {#if !accessResult && !waitingForDecision}
        <div class="modal-footer">
          <button class="btn-secondary" onclick={handleCancelAccessRequest} type="button">Cancel</button>
          <button
            class="btn-primary"
            onclick={handleSubmitAccessRequest}
            disabled={loading}
            type="button"
          >
            {loading ? "Submitting..." : "Request Access"}
          </button>
        </div>
      {/if}

    {:else if step === "connecting"}
      <h3 class="modal-title">Connecting...</h3>
      <div class="modal-body">
        <p class="modal-desc">Establishing connection to {address}...</p>
      </div>
    {/if}
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

  .warning-title {
    color: var(--destructive);
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

  .field-textarea {
    resize: vertical;
    min-height: 60px;
    padding: 10px 12px;
    font-family: inherit;
    line-height: 1.5;
  }

  .waiting-spinner {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px;
    color: var(--muted-foreground);
    font-size: 14px;
  }

  .spinner {
    width: 20px;
    height: 20px;
    border: 2px solid var(--border);
    border-top-color: var(--primary);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .success-text {
    color: #4ade80;
  }
</style>
