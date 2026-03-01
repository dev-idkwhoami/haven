<script lang="ts">
  interface Props {
    name: string;
    collapsed: boolean;
    onToggle: () => void;
    onContextMenu?: (e: MouseEvent) => void;
  }

  let { name, collapsed, onToggle, onContextMenu }: Props = $props();
</script>

<button
  class="category-header"
  onclick={onToggle}
  oncontextmenu={onContextMenu}
  type="button"
>
  <svg
    class="chevron"
    class:collapsed
    width="14"
    height="14"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="2"
    stroke-linecap="round"
    stroke-linejoin="round"
  >
    <path d="m6 9 6 6 6-6" />
  </svg>
  <span class="label">{name.toUpperCase()}</span>
  <div class="spacer"></div>
</button>

<style>
  .category-header {
    display: flex;
    align-items: center;
    gap: 4px;
    height: 32px;
    padding: 0 8px 0 4px;
    width: 100%;
    background: none;
    border: none;
    cursor: pointer;
    color: var(--muted-foreground);
    transition: color 0.15s;
  }

  .category-header:hover {
    color: var(--foreground);
  }

  .chevron {
    flex-shrink: 0;
    transition: transform 0.15s;
  }

  .chevron.collapsed {
    transform: rotate(-90deg);
  }

  .label {
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.5px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .spacer {
    flex: 1;
  }
</style>
