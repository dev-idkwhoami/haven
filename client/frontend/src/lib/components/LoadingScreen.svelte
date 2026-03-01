<script lang="ts">
  import { loadingMsg, progress } from "../stores/app.svelte.ts";
</script>

<div class="loading-screen">
  <div class="logo">
    <svg width="80" height="80" viewBox="0 0 80 80" fill="none">
      <path
        d="M24 16v48M56 16v48M24 40h32"
        stroke="#2563eb"
        stroke-width="4"
        stroke-linecap="round"
      />
    </svg>
  </div>

  <div class="progress-track">
    <div class="progress-fill" style="width: {progress()}%"></div>
  </div>

  <span class="loading-text">{loadingMsg()}</span>
</div>

<style>
  .loading-screen {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    width: 100%;
    gap: 24px;
    background: var(--background);
  }

  .logo {
    width: 200px;
    height: 200px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .progress-track {
    width: 280px;
    height: 6px;
    background: var(--secondary);
    border-radius: 9999px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: var(--primary);
    border-radius: 9999px;
    transition: width 0.3s ease;
  }

  .loading-text {
    color: var(--muted-foreground);
    font-family: var(--font-sans);
    font-size: 14px;
    letter-spacing: 2px;
    text-transform: capitalize;
  }
</style>
