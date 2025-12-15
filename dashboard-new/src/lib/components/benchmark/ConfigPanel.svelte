<script lang="ts">
  import ScenarioSelector from "$lib/components/ScenarioSelector.svelte";
  import { Zap, Loader2 } from "lucide-svelte";
  import type { Scenario, ScenarioOption } from "$lib/types/benchmark";
  import { createEventDispatcher } from "svelte";

  export let scenario: Scenario;
  export let scenarios: ScenarioOption[];
  export let startServers: boolean = false;
  export let running: boolean;
  export let progress: string;

  const dispatch = createEventDispatcher<{
    update: { key: string; value: unknown };
    runCompare: void;
  }>();

  function run() {
    dispatch("runCompare");
  }

  const allowed = new Set<Scenario>([
    "baseline",
    "burst",
    "cold_vs_resumed",
    "parallel_streams",
    "header_bloat",
    "uplink_loss",
    "connection_churn",
    "nat_rebinding",
    "mixed_load",
    "stress_test"
  ] as const);
  $: limitedScenarios = (scenarios ?? []).filter((s) => allowed.has(s.value as Scenario));
  // If current scenario is not allowed, reset to baseline
  $: if (scenario && !allowed.has(scenario)) {
    dispatch("update", { key: "scenario", value: "baseline" });
  }
</script>

<div class="card p-6">
  <h2 class="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-6">Configuration</h2>

  <ScenarioSelector
    items={limitedScenarios}
    selected={scenario}
    on:select={(e) =>
      dispatch("update", { key: "scenario", value: e.detail.value })}
  />

  <div
    class="mb-6 p-4 rounded-lg border-2 border-dashed border-primary-300 dark:border-white/15 bg-white dark:bg-[#0f1520]"
  >
    <div class="flex items-center justify-center space-x-4">
      <div class="text-center">
        <div class="text-lg font-bold text-blue-600 dark:text-blue-300">HTTP/2</div>
        <div class="text-xs text-blue-500 dark:text-blue-400">:8444</div>
      </div>
      <div class="text-2xl font-bold text-primary-600 dark:text-primary-400">VS</div>
      <div class="text-center">
        <div class="text-lg font-bold text-purple-600 dark:text-purple-300">HTTP/3</div>
        <div class="text-xs text-purple-500 dark:text-purple-400">:8443</div>
      </div>
    </div>
    <div class="text-center mt-2 text-xs text-gray-600 dark:text-gray-400">
      Both protocols will be tested
    </div>
  </div>

  <div class="mb-6 p-4 rounded-lg bg-gray-50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
    <div class="text-sm text-gray-700 dark:text-gray-300">
      <div class="font-medium mb-2">ℹ️ Fixed Configuration</div>
      <p class="text-xs text-gray-600 dark:text-gray-400">
        All benchmark parameters (clients, duration, RPS, etc.) are pre-configured in each scenario's Go client and cannot be modified from the dashboard.
      </p>
    </div>
  </div>

  <div class="mb-6">
    <label class="inline-flex items-center gap-2 text-sm cursor-pointer select-none">
      <input
        type="checkbox"
        checked={startServers}
        on:change={(e) => dispatch('update', { key: 'startServers', value: (e.target as HTMLInputElement).checked })}
        class="h-4 w-4 rounded border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-primary-600 focus:ring-primary-500"
      />
      <span class="text-gray-700 dark:text-gray-300">Start servers automatically (H2 & H3)</span>
    </label>
    <div class="text-xs mt-1 text-gray-500 dark:text-gray-400">Disable if servers are already running on ports 8444/8443</div>
  </div>

  <button
    type="button"
    on:click={run}
    disabled={running}
    class="w-full bg-primary-600 hover:bg-primary-700 disabled:bg-gray-400 text-white font-semibold py-3 px-4 rounded-lg transition-colors flex items-center justify-center"
  >
    {#if running}
      <Loader2 class="mr-2 h-5 w-5 animate-spin" />
      Running Comparison...
    {:else}
      <Zap class="mr-2 h-5 w-5" />
      Run H2 vs H3 Comparison
    {/if}
  </button>

  {#if progress}
    <div class="mt-4 p-3 bg-blue-50 dark:bg-blue-950/40 border border-blue-200 dark:border-blue-900 rounded-lg">
      <p class="text-sm text-blue-800 dark:text-blue-300">{progress}</p>
    </div>
  {/if}
</div>
