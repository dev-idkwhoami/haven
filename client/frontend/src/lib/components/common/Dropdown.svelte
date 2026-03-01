<script lang="ts">
  interface DropdownOption {
    value: string;
    label: string;
    icon?: string;
  }

  interface Props {
    options: DropdownOption[];
    value: string;
    label?: string;
    placeholder?: string;
    onchange: (value: string) => void;
  }

  let { options, value, label, placeholder = "Select...", onchange }: Props = $props();
  let open = $state(false);

  let selectedLabel = $derived(
    options.find((o) => o.value === value)?.label ?? placeholder,
  );

  function select(opt: DropdownOption): void {
    onchange(opt.value);
    open = false;
  }
</script>

<div class="dropdown">
  {#if label}
    <span class="dropdown-label">{label}</span>
  {/if}
  <button class="dropdown-trigger" onclick={() => (open = !open)} type="button">
    <span class="trigger-text">{selectedLabel}</span>
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="m6 9 6 6 6-6" />
    </svg>
  </button>

  {#if open}
    <div class="dropdown-backdrop" onclick={() => (open = false)} role="presentation"></div>
    <div class="dropdown-menu">
      {#each options as opt (opt.value)}
        <button
          class="dropdown-item"
          class:selected={opt.value === value}
          onclick={() => select(opt)}
          type="button"
        >
          {opt.label}
        </button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .dropdown {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .dropdown-label {
    font-size: 13px;
    font-weight: 500;
    color: var(--foreground);
  }

  .dropdown-trigger {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 36px;
    padding: 0 12px;
    background: var(--muted);
    border: none;
    border-radius: 6px;
    color: var(--foreground);
    font-size: 14px;
    font-family: inherit;
    cursor: pointer;
    transition: background 0.15s;
  }

  .dropdown-trigger:hover {
    background: var(--secondary);
  }

  .trigger-text {
    flex: 1;
    text-align: left;
  }

  .dropdown-trigger svg {
    color: var(--muted-foreground);
    flex-shrink: 0;
  }

  .dropdown-backdrop {
    position: fixed;
    inset: 0;
    z-index: 40;
  }

  .dropdown-menu {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    margin-top: 4px;
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 4px;
    z-index: 41;
    max-height: 200px;
    overflow-y: auto;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  }

  .dropdown-item {
    display: flex;
    align-items: center;
    width: 100%;
    height: 32px;
    padding: 0 8px;
    background: none;
    border: none;
    border-radius: 4px;
    color: var(--foreground);
    font-size: 13px;
    font-family: inherit;
    cursor: pointer;
    text-align: left;
    transition: background 0.1s;
  }

  .dropdown-item:hover {
    background: var(--muted);
  }

  .dropdown-item.selected {
    color: var(--primary);
  }
</style>
