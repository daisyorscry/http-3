<script lang="ts">
  import ThemeToggle from '$lib/components/ThemeToggle.svelte';
  import { Gauge, Home, Clock, FileText } from 'lucide-svelte';
  import { page } from '$app/stores';

  const links = [
    { href: '/', label: 'Home', icon: Home },
    { href: '/history', label: 'History', icon: Clock },
    { href: '/docs', label: 'Docs', icon: FileText }
  ];

  function isActive(href: string, pathname: string) {
    if (href === '/') return pathname === '/';
    return pathname.startsWith(href);
  }
</script>

<header class="sticky top-0 z-40 border-b border-gray-200/70 dark:border-gray-800/70 bg-white/70 dark:bg-gray-950/60 backdrop-blur">
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
    <a href="/" class="flex items-center gap-2 group">
      <div class="h-8 w-8 rounded-lg bg-primary-600/10 text-primary-600 grid place-items-center group-hover:bg-primary-600/20 transition-colors">
        <Gauge class="h-4 w-4" />
      </div>
      <div class="leading-tight">
        <div class="font-semibold text-gray-900 dark:text-gray-100">Bench Dashboard</div>
        <div class="text-xs text-gray-500 dark:text-gray-400">gRPC H2 vs H3</div>
      </div>
    </a>

    <nav class="hidden md:flex items-center gap-1">
      {#if $page}
        {#each links as l}
          {@const active = isActive(l.href, $page.url.pathname)}
          <a href={l.href}
             class="inline-flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors hover:bg-gray-100 dark:hover:bg-gray-800/80 {active ? 'bg-gray-100 text-gray-900 dark:bg-gray-800/80 dark:text-gray-100' : 'text-gray-600 dark:text-gray-400'}">
            <svelte:component this={l.icon} class="h-4 w-4" />
            {l.label}
          </a>
        {/each}
      {/if}
    </nav>

    <div class="flex items-center gap-2">
      <ThemeToggle />
    </div>
  </div>
  <!-- mobile nav -->
  <div class="md:hidden border-t border-gray-200 dark:border-gray-800 px-4 py-2 flex gap-2 overflow-x-auto">
    {#if $page}
      {#each links as l}
        {@const active = isActive(l.href, $page.url.pathname)}
        <a href={l.href}
           class="inline-flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors whitespace-nowrap hover:bg-gray-100 dark:hover:bg-gray-800/80 {active ? 'bg-gray-100 text-gray-900 dark:bg-gray-800/80 dark:text-gray-100' : 'text-gray-600 dark:text-gray-400'}">
          <svelte:component this={l.icon} class="h-4 w-4" />
          {l.label}
        </a>
      {/each}
    {/if}
  </div>
</header>

