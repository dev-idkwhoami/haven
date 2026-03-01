<script lang="ts">
  import { phase, init } from "./lib/stores/app.svelte.ts";
  import { servers, loadServers } from "./lib/stores/servers.svelte.ts";
  import { loadMyPublicKey } from "./lib/stores/permissions.svelte.ts";
  import LoadingScreen from "./lib/components/LoadingScreen.svelte";
  import ProfileSetup from "./lib/components/ProfileSetup.svelte";
  import NoServerState from "./lib/components/NoServerState.svelte";
  import MainLayout from "./lib/components/layout/MainLayout.svelte";

  let hasServers = $derived(servers().length > 0);

  $effect(() => {
    init();
    loadServers();
    loadMyPublicKey();
  });
</script>

{#if phase() === "loading"}
  <LoadingScreen />
{:else if phase() === "setup"}
  <ProfileSetup />
{:else if phase() === "ready" && !hasServers}
  <NoServerState />
{:else if phase() === "ready" && hasServers}
  <MainLayout />
{/if}
