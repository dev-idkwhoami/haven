<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    variant?: "primary" | "secondary" | "danger" | "ghost";
    size?: "sm" | "md";
    disabled?: boolean;
    onclick?: () => void;
    type?: "button" | "submit";
    children: Snippet;
  }

  let {
    variant = "primary",
    size = "md",
    disabled = false,
    onclick,
    type = "button",
    children,
  }: Props = $props();
</script>

<button
  class="btn {variant} {size}"
  {disabled}
  {onclick}
  {type}
>
  {@render children()}
</button>

<style>
  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    border: none;
    border-radius: var(--radius);
    font-family: inherit;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s, background 0.15s;
    flex-shrink: 0;
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .md {
    height: 36px;
    padding: 0 16px;
    font-size: 13px;
  }

  .sm {
    height: 28px;
    padding: 0 10px;
    font-size: 12px;
  }

  .primary {
    background: var(--primary);
    color: var(--primary-foreground);
  }

  .primary:hover:not(:disabled) {
    opacity: 0.9;
  }

  .secondary {
    background: var(--secondary);
    color: var(--foreground);
  }

  .secondary:hover:not(:disabled) {
    background: var(--muted);
  }

  .danger {
    background: rgba(255, 68, 68, 0.13);
    color: var(--destructive);
  }

  .danger:hover:not(:disabled) {
    background: rgba(255, 68, 68, 0.25);
  }

  .ghost {
    background: none;
    color: var(--muted-foreground);
  }

  .ghost:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.06);
    color: var(--foreground);
  }
</style>
