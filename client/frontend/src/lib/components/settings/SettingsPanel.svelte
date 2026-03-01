<script lang="ts">
  import AppSettingsView from "./AppSettings.svelte";
  import PerServerSettings from "./PerServerSettings.svelte";
  import RelayServers from "./RelayServers.svelte";

  interface Props {
    onClose: () => void;
    initialSection?: string;
  }

  let { onClose, initialSection }: Props = $props();

  type SettingsSection = "profile" | "connections" | "voice" | "audio" | "appearance" | "notifications" | "relays" | "privacy";
  let activeSection = $state<SettingsSection>((initialSection as SettingsSection) || "profile");

  interface NavItem {
    id: SettingsSection;
    label: string;
    category: string;
  }

  const navItems: NavItem[] = [
    { id: "profile", label: "Profile", category: "ACCOUNT" },
    { id: "connections", label: "Connections", category: "ACCOUNT" },
    { id: "voice", label: "Voice", category: "AUDIO & VIDEO" },
    { id: "audio", label: "Audio Settings", category: "AUDIO & VIDEO" },
    { id: "appearance", label: "Appearance", category: "APP" },
    { id: "notifications", label: "Notifications", category: "APP" },
    { id: "relays", label: "Relay Servers", category: "ADVANCED" },
    { id: "privacy", label: "Server Privacy", category: "ADVANCED" },
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

<div class="settings-panel">
  <div class="settings-sidebar">
    <div class="sidebar-header">
      <button class="back-btn" onclick={onClose} type="button">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="m15 18-6-6 6-6" />
        </svg>
      </button>
      <span class="sidebar-title">Settings</span>
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

  <div class="settings-content">
    {#if activeSection === "profile"}
      <AppSettingsView {onClose} />
    {:else if activeSection === "connections"}
      <div class="placeholder">
        <div class="top-bar">
          <span class="section-title">Connections</span>
        </div>
        <div class="top-divider"></div>
        <div class="placeholder-content">
          <p class="placeholder-text">Coming soon</p>
        </div>
      </div>
    {:else if activeSection === "relays"}
      <RelayServers />
    {:else if activeSection === "privacy"}
      <PerServerSettings />
    {:else}
      <div class="placeholder">
        <div class="top-bar">
          <span class="section-title">{navItems.find((n) => n.id === activeSection)?.label ?? ""}</span>
        </div>
        <div class="top-divider"></div>
        <div class="placeholder-content">
          <p class="placeholder-text">Coming soon</p>
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .settings-panel {
    display: flex;
    height: 100%;
    width: 100%;
  }

  .settings-sidebar {
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

  .settings-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background: var(--background);
  }

  .placeholder {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .top-bar {
    display: flex;
    align-items: center;
    height: 72px;
    padding: 0 24px;
    flex-shrink: 0;
  }

  .section-title {
    font-size: 16px;
    font-weight: 600;
    color: var(--foreground);
  }

  .top-divider {
    height: 1px;
    background: var(--border);
  }

  .placeholder-content {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .placeholder-text {
    font-size: 14px;
    color: var(--muted-foreground);
    margin: 0;
  }
</style>
