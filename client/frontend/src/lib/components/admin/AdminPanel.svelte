<script lang="ts">
  import ServerSettings from "./ServerSettings.svelte";
  import RoleEditor from "./RoleEditor.svelte";
  import AuditLog from "./AuditLog.svelte";
  import InviteManager from "./InviteManager.svelte";
  import AccessRequestManager from "./AccessRequestManager.svelte";

  interface Props {
    onClose: () => void;
  }

  let { onClose }: Props = $props();

  type AdminSection = "overview" | "roles" | "bans" | "audit" | "invites" | "access_requests";
  let activeSection = $state<AdminSection>("overview");

  interface NavItem {
    id: AdminSection;
    label: string;
    category: string;
  }

  const navItems: NavItem[] = [
    { id: "overview", label: "Overview", category: "GENERAL" },
    { id: "roles", label: "Roles", category: "GENERAL" },
    { id: "audit", label: "Audit Log", category: "MODERATION" },
    { id: "access_requests", label: "Access Requests", category: "MODERATION" },
    { id: "invites", label: "Invites", category: "COMMUNITY" },
  ];

  let categories = $derived(() => {
    const cats: { name: string; items: NavItem[] }[] = [];
    for (const item of navItems) {
      let cat = cats.find((c) => c.name === item.category);
      if (!cat) {
        cat = { name: item.category, items: [] };
        cats.push(cat);
      }
      cat.items.push(item);
    }
    return cats;
  });
</script>

<div class="admin-panel">
  <div class="admin-sidebar">
    <div class="sidebar-header">
      <button class="back-btn" onclick={onClose} type="button">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="m15 18-6-6 6-6" />
        </svg>
      </button>
      <span class="sidebar-title">Server Administration</span>
    </div>
    <div class="sidebar-nav">
      {#each categories() as cat (cat.name)}
        <span class="cat-label">{cat.name}</span>
        {#each cat.items as item (item.id)}
          <button
            class="nav-item"
            class:active={activeSection === item.id}
            onclick={() => (activeSection = item.id)}
            type="button"
          >
            {item.label}
          </button>
        {/each}
        <div class="cat-spacer"></div>
      {/each}
    </div>
  </div>

  <div class="admin-content">
    {#if activeSection === "overview"}
      <ServerSettings {onClose} />
    {:else if activeSection === "roles"}
      <RoleEditor />
    {:else if activeSection === "audit"}
      <AuditLog />
    {:else if activeSection === "access_requests"}
      <AccessRequestManager />
    {:else if activeSection === "invites"}
      <InviteManager />
    {/if}
  </div>
</div>

<style>
  .admin-panel {
    display: flex;
    height: 100%;
    width: 100%;
  }

  .admin-sidebar {
    width: 240px;
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    background: var(--background);
    flex-shrink: 0;
  }

  .sidebar-header {
    display: flex;
    align-items: center;
    gap: 10px;
    height: 72px;
    padding: 0 16px;
    flex-shrink: 0;
  }

  .back-btn {
    width: 32px;
    height: 32px;
    border-radius: 8px;
    background: rgba(255, 255, 255, 0.06);
    border: none;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted-foreground);
    cursor: pointer;
    flex-shrink: 0;
    transition: color 0.15s;
  }

  .back-btn:hover {
    color: var(--foreground);
  }

  .sidebar-title {
    font-size: 15px;
    font-weight: 600;
    color: var(--foreground);
  }

  .sidebar-nav {
    flex: 1;
    overflow-y: auto;
    padding: 12px 8px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .cat-label {
    font-size: 11px;
    font-weight: 600;
    color: var(--muted-foreground);
    letter-spacing: 0.5px;
    padding: 0 12px;
  }

  .cat-spacer {
    height: 8px;
  }

  .nav-item {
    display: flex;
    align-items: center;
    height: 32px;
    padding: 0 12px;
    background: none;
    border: none;
    border-radius: 4px;
    color: var(--foreground);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    text-align: left;
    transition: background 0.1s;
    width: 100%;
  }

  .nav-item:hover {
    background: var(--muted);
  }

  .nav-item.active {
    background: var(--muted);
  }

  .admin-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background: var(--background);
  }
</style>
