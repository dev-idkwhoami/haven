<script lang="ts">
  import { participants } from "../../stores/voice.svelte.ts";
</script>

<div class="participant-list">
  {#each participants() as p (p.publicKey)}
    <div class="participant" class:speaking={p.isSpeaking}>
      <div class="p-avatar" class:muted={p.isMuted}>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" />
        </svg>
      </div>
      <span class="p-name">{p.displayName}</span>
      <div class="p-indicators">
        {#if p.isMuted}
          <svg class="indicator-icon muted-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="2" x2="22" y1="2" y2="22" /><path d="M18.89 13.23A7.12 7.12 0 0 0 19 12v-2" /><path d="M5 10v2a7 7 0 0 0 12 5" /><path d="M15 9.34V5a3 3 0 0 0-5.68-1.33" /><path d="M9 9v3a3 3 0 0 0 5.12 2.12" /><line x1="12" x2="12" y1="19" y2="22" />
          </svg>
        {/if}
        {#if p.isDeafened}
          <svg class="indicator-icon muted-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="2" x2="22" y1="2" y2="22" /><path d="M8.5 16.5a5 5 0 0 1-1.643-2.022" /><path d="M19.198 10.802A2 2 0 0 1 21 14v1a2 2 0 0 1-2 2h-1" /><path d="M3 14h3a2 2 0 0 1 2 2v3a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-7a9 9 0 0 1 9-9 8.981 8.981 0 0 1 5 1.516" />
          </svg>
        {/if}
      </div>
    </div>
  {/each}
</div>

<style>
  .participant-list {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .participant {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 6px;
    border-radius: 4px;
    transition: background 0.1s;
  }

  .participant:hover {
    background: var(--muted);
  }

  .p-avatar {
    width: 24px;
    height: 24px;
    border-radius: 6px;
    background: var(--muted);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted-foreground);
    flex-shrink: 0;
    border: 2px solid transparent;
    transition: border-color 0.2s;
  }

  .participant.speaking .p-avatar {
    border-color: #22c55e;
  }

  .p-avatar.muted {
    opacity: 0.6;
  }

  .p-name {
    font-size: 12px;
    font-weight: 500;
    color: var(--foreground);
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .p-indicators {
    display: flex;
    gap: 2px;
  }

  .indicator-icon {
    flex-shrink: 0;
  }

  .muted-icon {
    color: var(--destructive);
  }
</style>
