<script lang="ts">
  import { createEventDispatcher, onMount } from "svelte";
  import { ChevronDown, Check } from "lucide-svelte";
  import type { ScenarioOption } from "$lib/types/benchmark"; // ← import tipe yang sama

  export let items: ScenarioOption[] = [];
  export let selected: ScenarioOption["value"];

  const dispatch = createEventDispatcher<{
    select: { value: ScenarioOption["value"] };
  }>();

  let open = false;
  let rootEl: HTMLDivElement;
  let activeIndex = -1;

  $: selectedItem = items.find((i) => i.value === selected) ?? items[0];
  $: if (open)
    activeIndex = Math.max(
      0,
      items.findIndex((i) => i.value === selected)
    );

  function toggle() {
    open = !open;
  }
  function choose(v: ScenarioOption["value"]) {
    dispatch("select", { value: v });
    open = false;
  }

  function onKeydown(e: KeyboardEvent) {
    if (!open) {
      if (e.key === "ArrowDown" || e.key === "Enter" || e.key === " ") {
        open = true;
        e.preventDefault();
      }
      return;
    }
    if (e.key === "Escape") open = false;
    else if (e.key === "ArrowDown") {
      activeIndex = (activeIndex + 1) % items.length;
      e.preventDefault();
    } else if (e.key === "ArrowUp") {
      activeIndex = (activeIndex + items.length - 1) % items.length;
      e.preventDefault();
    } else if (e.key === "Enter" || e.key === " ") {
      if (activeIndex >= 0) choose(items[activeIndex].value);
      e.preventDefault();
    }
  }

  onMount(() => {
    const handler = (e: MouseEvent) => {
      if (!rootEl?.contains(e.target as Node)) open = false;
    };
    window.addEventListener("click", handler);
    return () => window.removeEventListener("click", handler);
  });
</script>

<div class="mb-6" bind:this={rootEl}>
  <div class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
    Traffic Scenario
  </div>
  <div class="relative">
    <button
      type="button"
      class="w-full px-4 py-3 rounded-lg border-2 border-gray-200 dark:border-white/10 hover:border-gray-300 dark:hover:border-white/20 bg-white dark:bg-[#0f1520] text-left flex items-center justify-between gap-3 focus:outline-none focus:ring-2 focus:ring-primary-500"
      aria-haspopup="listbox"
      aria-expanded={open}
      on:click|stopPropagation={toggle}
      on:keydown={onKeydown}
    >
      {#if selectedItem}
        <div class="flex items-start gap-3">
          <svelte:component
            this={selectedItem.Icon}
            class="mt-0.5 h-5 w-5 text-gray-700 dark:text-gray-300"
          />
          <div>
            <div class="font-semibold text-gray-900 dark:text-gray-100">
              {selectedItem.label}
            </div>
            <div class="text-xs text-gray-600 dark:text-gray-400 -mt-0.5">
              {selectedItem.subtitle}
            </div>
          </div>
        </div>
      {/if}
      <ChevronDown
        class="h-4 w-4 text-gray-500 dark:text-gray-400 flex-shrink-0"
        aria-hidden="true"
      />
    </button>

    {#if open}
      <div
        class="absolute z-20 mt-2 w-full rounded-lg border border-gray-200 dark:border-white/10 bg-white dark:bg-[#0f1520] shadow-lg overflow-hidden"
        role="listbox"
      >
        {#each items as s, i}
          <button
            type="button"
            role="option"
            aria-selected={selected === s.value}
            class="w-full text-left px-4 py-2.5 flex items-start gap-3 hover:bg-gray-50 dark:hover:bg-white/5 {selected ===
            s.value
              ? 'bg-primary-50 dark:bg-primary-950/20'
              : ''} {activeIndex === i ? 'bg-gray-50 dark:bg-white/5' : ''}"
            on:click={() => choose(s.value)}
          >
            <svelte:component
              this={s.Icon}
              class="mt-0.5 h-4 w-4 text-gray-700 dark:text-gray-300"
            />
            <div class="flex-1">
              <div class="text-sm font-medium text-gray-900 dark:text-gray-100">
                {s.label}
              </div>
              <div class="text-xs text-gray-600 dark:text-gray-400">
                {s.subtitle}
              </div>
            </div>
            {#if selected === s.value}
              <Check class="h-4 w-4 text-primary-600 mt-0.5" />
            {/if}
          </button>
        {/each}
      </div>
    {/if}
  </div>

  <p class="sr-only" aria-live="polite">
    Selected scenario: {selectedItem?.label}
  </p>
</div>
