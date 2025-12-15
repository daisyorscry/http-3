<script lang="ts">
  export let ready = true;
</script>

<header
  class="sticky top-0 z-40 bg-white/80 backdrop-blur shadow-sm border-b border-gray-200"
>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
    <div class="flex items-center justify-between">
      <div>

        <p class="mt-1 text-sm text-gray-500">
          Performance testing tool for gRPC protocols  HTTP/3 vs HTTP/2 Benchmark
        </p>
      </div>
      <div class="flex items-center space-x-2">
        {#if ready}
          <span
            class="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800"
          >
            <span class="w-2 h-2 bg-green-500 rounded-full mr-2"></span>
            Ready
          </span>
        {:else}
          <span
            class="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800"
          >
            <span class="w-2 h-2 bg-yellow-500 rounded-full mr-2"></span>
            Warming up
          </span>
        {/if}
      </div>
    </div>
  </div>
</header>
