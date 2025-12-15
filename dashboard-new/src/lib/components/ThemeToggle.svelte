<script lang="ts">
  import { Moon, Sun } from 'lucide-svelte';

  let isDark = $state<boolean>(false);

  function updateTheme(dark: boolean) {
    try {
      const root = document.documentElement;
      if (dark) {
        root.classList.add('dark');
        localStorage.setItem('theme', 'dark');
      } else {
        root.classList.remove('dark');
        localStorage.setItem('theme', 'light');
      }
      isDark = dark;
    } catch (_) {}
  }

  if (typeof window !== 'undefined') {
    const ls = localStorage.getItem('theme');
    const prefers = window.matchMedia('(prefers-color-scheme: dark)').matches;
    isDark = ls ? ls === 'dark' : prefers;
  }
</script>

<button
  type="button"
  class="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-gray-200 dark:border-gray-800 bg-white/80 dark:bg-gray-900/80 backdrop-blur hover:bg-white dark:hover:bg-gray-800 transition-colors"
  title={isDark ? 'Switch to light' : 'Switch to dark'}
  onclick={() => updateTheme(!isDark)}
>
  {#if isDark}
    <Sun class="h-4 w-4 text-gray-300" />
  {:else}
    <Moon class="h-4 w-4 text-gray-700" />
  {/if}
  <span class="sr-only">Toggle theme</span>
</button>

