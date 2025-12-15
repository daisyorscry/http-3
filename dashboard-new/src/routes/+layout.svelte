<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
  import AppHeader from '$lib/components/AppHeader.svelte';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="min-h-screen">
  <AppHeader />
  <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
    {@render children?.()}
  </main>
</div>
