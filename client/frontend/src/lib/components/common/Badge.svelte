<script lang="ts">
  interface Props {
    text: string;
    color?: string | null;
    variant?: "outline" | "filled";
  }

  let { text, color = null, variant = "outline" }: Props = $props();
</script>

<span
  class="badge {variant}"
  style={color ? `border-color: ${color}; color: ${color}` : ""}
>
  {text}
</span>

<style>
  .badge {
    font-size: 11px;
    font-weight: 500;
    padding: 2px 8px;
    border-radius: 9999px;
    white-space: nowrap;
  }

  .outline {
    border: 1px solid var(--border);
    color: var(--foreground);
    background: none;
  }

  .filled {
    border: none;
    background: var(--muted);
    color: var(--foreground);
  }
</style>
