<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  export let scenario: string;

  // Baseline
  export let period: number;
  export let jitter: number;

  // Burst
  export let burstRps: number;
  export let burstDurationMs: number;
  export let idleDurationMs: number;

  // Cold vs Resumed
  export let newConnRatioPct: number;
  export let connPoolSize: number;

  // Parallel Streams
  export let concurrentStreams: number;
  export let interRequestGapMs: number;

  // Header Bloat
  export let headerSizeBytes: number;
  export let headerPairs: number;

  // Uplink Loss
  export let uplinkLossPct: number;
  export let uplinkLatencyMs: number;
  export let retryCount: number;

  // Connection Churn
  export let churnPerSec: number;
  export let sessionDurationMs: number;

  // NAT Rebinding
  export let natRebindIntervalMs: number;
  export let migrationProbPct: number;

  // Mixed Load
  export let heavyPct: number;
  export let queueCapacity: number;

  // Stress Test
  export let ramp: string;

  const dispatch = createEventDispatcher<{ update: { key: string; value: number | string } }>();
  const toNum = (e: Event) => Number((e.target as HTMLInputElement).value);
</script>

  <div class="mb-6">
  <div class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">Scenario Settings</div>

  {#if scenario === 'baseline'}
    <div class="grid grid-cols-2 gap-3">
      <div>
        <label for="scn-period" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Period (ms)</label>
        <input id="scn-period" type="number" value={period} on:input={(e) => dispatch('update', { key: 'period', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
      <div>
        <label for="scn-jitter" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Jitter (ms)</label>
        <input id="scn-jitter" type="number" value={jitter} on:input={(e) => dispatch('update', { key: 'jitter', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
    </div>

  {:else if scenario === 'burst'}
    <div class="grid grid-cols-2 gap-3">
      <div>
        <label for="scn-burst-rps" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Burst RPS</label>
        <input id="scn-burst-rps" type="number" value={burstRps} on:input={(e) => dispatch('update', { key: 'burstRps', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
      <div>
        <label for="scn-burst-dur" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Burst Duration (ms)</label>
        <input id="scn-burst-dur" type="number" value={burstDurationMs} on:input={(e) => dispatch('update', { key: 'burstDurationMs', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
      <div>
        <label for="scn-burst-idle" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Idle Between Bursts (ms)</label>
        <input id="scn-burst-idle" type="number" value={idleDurationMs} on:input={(e) => dispatch('update', { key: 'idleDurationMs', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
    </div>

  {:else if scenario === 'cold_vs_resumed'}
    <div class="grid grid-cols-2 gap-3">
      <div>
        <label for="scn-new-ratio" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">New Connections (%)</label>
        <input id="scn-new-ratio" type="number" min="0" max="100" value={newConnRatioPct} on:input={(e) => dispatch('update', { key: 'newConnRatioPct', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
      <div>
        <label for="scn-pool-size" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Pool Size</label>
        <input id="scn-pool-size" type="number" min="1" value={connPoolSize} on:input={(e) => dispatch('update', { key: 'connPoolSize', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
    </div>

  {:else if scenario === 'parallel_streams'}
    <div class="grid grid-cols-2 gap-3">
      <div>
        <label for="scn-concurrent" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Concurrent Streams</label>
        <input id="scn-concurrent" type="number" min="1" value={concurrentStreams} on:input={(e) => dispatch('update', { key: 'concurrentStreams', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
      <div>
        <label for="scn-gap" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Inter-request Gap (ms)</label>
        <input id="scn-gap" type="number" min="0" value={interRequestGapMs} on:input={(e) => dispatch('update', { key: 'interRequestGapMs', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
    </div>

  {:else if scenario === 'header_bloat'}
    <div class="grid grid-cols-2 gap-3">
      <div>
        <label for="scn-header-size" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Header Size (bytes)</label>
        <input id="scn-header-size" type="number" min="0" step="256" value={headerSizeBytes} on:input={(e) => dispatch('update', { key: 'headerSizeBytes', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
      <div>
        <label for="scn-header-pairs" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Header Pairs (k-v)</label>
        <input id="scn-header-pairs" type="number" min="1" value={headerPairs} on:input={(e) => dispatch('update', { key: 'headerPairs', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
    </div>

  {:else if scenario === 'uplink_loss'}
    <div class="grid grid-cols-2 gap-3">
      <div>
        <label for="scn-loss" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Uplink Loss (%)</label>
        <input id="scn-loss" type="number" min="0" max="100" step="0.1" value={uplinkLossPct} on:input={(e) => dispatch('update', { key: 'uplinkLossPct', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
      <div>
        <label for="scn-latency" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Uplink Latency (ms)</label>
        <input id="scn-latency" type="number" min="0" value={uplinkLatencyMs} on:input={(e) => dispatch('update', { key: 'uplinkLatencyMs', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
      <div class="sm:col-span-2">
        <label for="scn-retries" class="block text-sm font-medium text-gray-700 mb-1">Retries</label>
        <input id="scn-retries" type="number" min="0" value={retryCount} on:input={(e) => dispatch('update', { key: 'retryCount', value: toNum(e) })} class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
    </div>

  {:else if scenario === 'connection_churn'}
    <div class="grid grid-cols-2 gap-3">
      <div>
        <label for="scn-churn" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Churn Rate (conn/sec)</label>
        <input id="scn-churn" type="number" min="1" value={churnPerSec} on:input={(e) => dispatch('update', { key: 'churnPerSec', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
      <div>
        <label for="scn-session" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Session Duration (ms)</label>
        <input id="scn-session" type="number" min="100" step="100" value={sessionDurationMs} on:input={(e) => dispatch('update', { key: 'sessionDurationMs', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
    </div>

  {:else if scenario === 'nat_rebinding'}
    <div class="grid grid-cols-2 gap-3">
      <div>
        <label for="scn-rebind" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Rebind Interval (ms)</label>
        <input id="scn-rebind" type="number" min="500" step="100" value={natRebindIntervalMs} on:input={(e) => dispatch('update', { key: 'natRebindIntervalMs', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
      <div>
        <label for="scn-migrate" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Migration Probability (%)</label>
        <input id="scn-migrate" type="number" min="0" max="100" step="1" value={migrationProbPct} on:input={(e) => dispatch('update', { key: 'migrationProbPct', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
    </div>

  {:else if scenario === 'mixed_load'}
    <div class="grid grid-cols-2 gap-3">
      <div>
        <label for="scn-heavy" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Heavy Share (%)</label>
        <input id="scn-heavy" type="number" min="0" max="100" step="1" value={heavyPct} on:input={(e) => dispatch('update', { key: 'heavyPct', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
      <div>
        <label for="scn-queue" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Queue Capacity</label>
        <input id="scn-queue" type="number" min="0" step="10" value={queueCapacity} on:input={(e) => dispatch('update', { key: 'queueCapacity', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      </div>
    </div>

  {:else if scenario === 'stress_test'}
    <div>
      <label for="scn-ramp" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Ramp Stages</label>
      <input id="scn-ramp" type="text" value={ramp} on:input={(e) => dispatch('update', { key: 'ramp', value: (e.target as HTMLInputElement).value })} placeholder="30@1000,30@2000,30@4000" class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
      <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Format: duration@rps,duration@rps</p>
    </div>
  {/if}
</div>
