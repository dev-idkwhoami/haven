<script lang="ts">
  interface Props {
    name: string;
    src?: string | null;
    size?: number;
    rounded?: "square" | "circle";
  }

  let { name, src = null, size = 36, rounded = "square" }: Props = $props();

  let initial = $derived(name ? name.charAt(0).toUpperCase() : "?");
  let borderRadius = $derived(rounded === "circle" ? "9999px" : `${Math.round(size * 0.22)}px`);
  let fontSize = $derived(Math.round(size * 0.4));
</script>

<div
  class="avatar"
  style="width: {size}px; height: {size}px; border-radius: {borderRadius}; font-size: {fontSize}px;"
>
  {#if src}
    <img {src} alt={name} class="avatar-img" style="border-radius: {borderRadius};" />
  {:else}
    <span class="initial">{initial}</span>
  {/if}
</div>

<style>
  .avatar {
    background: var(--muted);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    overflow: hidden;
  }

  .avatar-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .initial {
    font-weight: 600;
    color: var(--muted-foreground);
    user-select: none;
  }
</style>
