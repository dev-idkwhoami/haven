<script lang="ts">
  interface Props {
    name: string;
    iconHash?: string;
    isActive: boolean;
    connected: boolean;
    reconnecting?: boolean;
    onclick: () => void;
  }

  let { name, iconHash, isActive, connected, reconnecting = false, onclick }: Props = $props();

  let initials = $derived(
    name
      .split(/\s+/)
      .map((w) => w[0])
      .join("")
      .slice(0, 3)
      .toUpperCase(),
  );
</script>

<button
  class="server-icon"
  class:active={isActive}
  onclick={onclick}
  title={name}
  type="button"
>
  <div class="icon-circle" class:active={isActive}>
    <span class="initials">{initials}</span>
  </div>
  <div class="status-dot" class:connected class:reconnecting></div>
  {#if isActive}
    <div class="active-pill"></div>
  {/if}
</button>

<style>
  .server-icon {
    position: relative;
    width: 48px;
    height: 48px;
    background: none;
    border: none;
    cursor: pointer;
    padding: 0;
    flex-shrink: 0;
  }

  .icon-circle {
    width: 48px;
    height: 48px;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.06);
    display: flex;
    align-items: center;
    justify-content: center;
    transition: border-radius 0.2s, background 0.15s;
  }

  .icon-circle:hover {
    background: rgba(255, 255, 255, 0.1);
    border-radius: 16px;
  }

  .icon-circle.active {
    background: var(--primary);
    border-radius: 16px;
  }

  .initials {
    font-size: 14px;
    font-weight: 600;
    color: var(--muted-foreground);
    user-select: none;
  }

  .icon-circle.active .initials {
    color: var(--primary-foreground);
  }

  .status-dot {
    position: absolute;
    bottom: 0;
    right: 0;
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background: #737373;
    border: 2px solid var(--background);
  }

  .status-dot.connected {
    background: #22c55e;
  }

  .status-dot.reconnecting {
    background: #f59e0b;
  }

  .active-pill {
    position: absolute;
    left: -8px;
    top: 50%;
    transform: translateY(-50%);
    width: 4px;
    height: 32px;
    border-radius: 0 4px 4px 0;
    background: var(--foreground);
  }
</style>
