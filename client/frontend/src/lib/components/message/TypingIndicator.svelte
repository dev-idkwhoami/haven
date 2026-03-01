<script lang="ts">
  import { typingUsers } from "../../stores/messages.svelte.ts";
  import { users } from "../../stores/users.svelte.ts";

  let displayText = $derived(() => {
    if (typingUsers().length === 0) return "";
    const names = typingUsers().map((pubKey) => {
      const user = users().find((u) => u.publicKey === pubKey);
      return user?.displayName ?? "Someone";
    });
    if (names.length === 1) return `${names[0]} is typing...`;
    if (names.length === 2) return `${names[0]} and ${names[1]} are typing...`;
    return `${names[0]} and ${names.length - 1} others are typing...`;
  });
</script>

{#if typingUsers().length > 0}
  <div class="typing-indicator">
    <span class="typing-text">{displayText()}</span>
  </div>
{/if}

<style>
  .typing-indicator {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 0 48px;
    height: 24px;
    flex-shrink: 0;
  }

  .typing-text {
    font-size: 12px;
    font-style: italic;
    color: var(--muted-foreground);
  }
</style>
