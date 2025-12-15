<script lang="ts">
  import type { ProtocolOption } from "$lib/types/benchmark";
  import { createEventDispatcher } from "svelte";

  export let selected: "h2" | "h3";
  export let options: ProtocolOption[] = [];
  const dispatch = createEventDispatcher<{ change: { value: "h2" | "h3" } }>();
</script>

<div class="mb-6">
  <div
    id="protocol-label"
    class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3"
  >
    Protocol
  </div>
  <div
    class="grid grid-cols-2 gap-2 md:gap-3"
    role="radiogroup"
    aria-labelledby="protocol-label"
  >
    {#each options as p}
      <label
        for={`proto-${p.value}`}
        class="px-4 py-3 rounded-lg border-2 cursor-pointer select-none transition-colors
               bg-white dark:bg-[#0f1520]
               {selected === p.value
          ? 'border-primary-500 dark:border-primary-400'
          : 'border-gray-200 dark:border-white/10 hover:border-gray-300 dark:hover:border-white/20'}
               focus:outline-none focus:ring-2 focus:ring-primary-500"
      >
        <input
          class="sr-only"
          type="radio"
          name="protocol"
          id={`proto-${p.value}`}
          value={p.value}
          checked={selected === p.value}
          on:change={() => dispatch("change", { value: p.value })}
        />
        <div class="font-semibold text-gray-900 dark:text-gray-100">
          {p.label}
        </div>
        <div
          class="w-full h-1 {p.color} rounded mt-2 opacity-70 dark:opacity-60 transition-opacity"
          aria-hidden="true"
        ></div>
      </label>
    {/each}
  </div>
</div>
