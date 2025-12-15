<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { Zap } from 'lucide-svelte';

  export let mode: 'single' | 'compare' = 'single';
  const dispatch = createEventDispatcher<{ change: { mode: 'single' | 'compare' } }>();

  function setMode(m: 'single' | 'compare') {
    if (mode !== m) dispatch('change', { mode: m });
  }
</script>

<div class="card p-2 inline-flex">
  <button
    onclick={() => setMode('single')}
    class="px-6 py-2 rounded-lg font-medium transition-all {mode === 'single' ? 'bg-primary-600 text-white' : 'text-gray-600 hover:text-gray-900 dark:text-gray-300 dark:hover:text-white'}"
    aria-pressed={mode === 'single'}
  >
    Single Test
  </button>
  <button
    onclick={() => setMode('compare')}
    class="px-6 py-2 rounded-lg font-medium transition-all {mode === 'compare' ? 'bg-primary-600 text-white' : 'text-gray-600 hover:text-gray-900 dark:text-gray-300 dark:hover:text-white'}"
    aria-pressed={mode === 'compare'}
  >
    <span class="inline-flex items-center">
      <Zap class="mr-2 w-4" />
      H2 vs H3 Comparison
    </span>
  </button>
</div>
