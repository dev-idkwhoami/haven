<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    text: string;
    position?: "top" | "bottom" | "left" | "right";
    children: Snippet;
  }

  let { text, position = "top", children }: Props = $props();
  let visible = $state(false);
</script>

<div
  class="tooltip-wrapper"
  onmouseenter={() => (visible = true)}
  onmouseleave={() => (visible = false)}
  role="group"
>
  {@render children()}
  {#if visible}
    <div class="tooltip {position}">
      {text}
    </div>
  {/if}
</div>

<style>
  .tooltip-wrapper {
    position: relative;
    display: inline-flex;
  }

  .tooltip {
    position: absolute;
    padding: 4px 8px;
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: 6px;
    font-size: 12px;
    color: var(--foreground);
    white-space: nowrap;
    pointer-events: none;
    z-index: 50;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  }

  .top {
    bottom: calc(100% + 6px);
    left: 50%;
    transform: translateX(-50%);
  }

  .bottom {
    top: calc(100% + 6px);
    left: 50%;
    transform: translateX(-50%);
  }

  .left {
    right: calc(100% + 6px);
    top: 50%;
    transform: translateY(-50%);
  }

  .right {
    left: calc(100% + 6px);
    top: 50%;
    transform: translateY(-50%);
  }
</style>
