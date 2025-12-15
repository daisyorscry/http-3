<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  export let clients: number;
  export let payload: number;
  export let duration: number;
  export let rps: number;

  const dispatch = createEventDispatcher<{ update: { key: 'clients' | 'payload' | 'duration' | 'rps'; value: number } }>();
  const toNum = (e: Event) => Number((e.target as HTMLInputElement).value);
</script>

<div class="space-y-4 mb-6">
  <div>
    <label for="gen-clients" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Clients</label>
    <input id="gen-clients" type="number" value={clients} on:input={(e) => dispatch('update', { key: 'clients', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
  </div>
  <div>
    <label for="gen-payload" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Payload (bytes)</label>
    <input id="gen-payload" type="number" value={payload} on:input={(e) => dispatch('update', { key: 'payload', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
  </div>
  <div>
    <label for="gen-duration" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Duration (seconds)</label>
    <input id="gen-duration" type="number" value={duration} on:input={(e) => dispatch('update', { key: 'duration', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
  </div>
  <div>
    <label for="gen-rps" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">RPS (Requests/sec)</label>
    <input id="gen-rps" type="number" value={rps} on:input={(e) => dispatch('update', { key: 'rps', value: toNum(e) })} class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-white/10 bg-white dark:bg-[#0f1520] text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent" />
    <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Used for constant-rate scenarios</p>
  </div>
</div>
