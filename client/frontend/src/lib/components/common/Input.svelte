<script lang="ts">
  interface Props {
    value: string;
    label?: string;
    placeholder?: string;
    error?: string | null;
    type?: string;
    disabled?: boolean;
    oninput?: (e: Event) => void;
  }

  let {
    value = $bindable(),
    label,
    placeholder = "",
    error = null,
    type = "text",
    disabled = false,
    oninput,
  }: Props = $props();
</script>

<div class="input-group">
  {#if label}
    <label class="input-label">{label}</label>
  {/if}
  <input
    class="input"
    class:has-error={error}
    {type}
    {placeholder}
    {disabled}
    bind:value
    {oninput}
  />
  {#if error}
    <span class="input-error">{error}</span>
  {/if}
</div>

<style>
  .input-group {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .input-label {
    font-size: 13px;
    font-weight: 500;
    color: var(--foreground);
  }

  .input {
    height: 36px;
    padding: 0 12px;
    background: var(--muted);
    border: 1px solid transparent;
    border-radius: 6px;
    color: var(--foreground);
    font-size: 14px;
    font-family: inherit;
    outline: none;
    transition: border-color 0.15s;
  }

  .input::placeholder {
    color: var(--muted-foreground);
  }

  .input:focus {
    border-color: var(--ring);
  }

  .input.has-error {
    border-color: var(--destructive);
  }

  .input:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .input-error {
    font-size: 12px;
    color: var(--destructive);
  }
</style>
