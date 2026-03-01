<script lang="ts">
  import type { User } from "../../types";

  interface Props {
    user: User;
    roleColor?: string | null;
    onclick?: () => void;
  }

  let { user, roleColor = null, onclick }: Props = $props();

  let statusColor = $derived(() => {
    switch (user.status) {
      case "online": return "#22c55e";
      case "idle": return "#eab308";
      case "dnd": return "#ef4444";
      default: return "#737373";
    }
  });
</script>

<button class="user-card" onclick={onclick} type="button">
  <div class="avatar">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" />
    </svg>
    <div class="status-dot" style="background: {statusColor()}"></div>
  </div>
  <span class="name" style={roleColor ? `color: ${roleColor}` : ""}>{user.displayName}</span>
</button>

<style>
  .user-card {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 12px;
    background: none;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    width: 100%;
    transition: background 0.1s;
  }

  .user-card:hover {
    background: rgba(255, 255, 255, 0.06);
  }

  .avatar {
    position: relative;
    width: 28px;
    height: 28px;
    border-radius: 6px;
    background: var(--muted);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted-foreground);
    flex-shrink: 0;
  }

  .status-dot {
    position: absolute;
    bottom: -1px;
    right: -1px;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    border: 2px solid var(--background);
  }

  .name {
    font-size: 13px;
    font-weight: 500;
    color: var(--foreground);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
