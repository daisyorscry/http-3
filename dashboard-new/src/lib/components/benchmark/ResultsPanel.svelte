<script lang="ts">
  import ComparisonResults from "$lib/components/ComparisonResults.svelte";
  import type { BenchmarkResult, ComparisonResult } from "$lib/types/benchmark";

  export let result: BenchmarkResult | null = null;
  export let comparisonResult: ComparisonResult | null = null;

  // helpers angka
  const nf0 = (n: number) => (Number.isFinite(n) ? n.toFixed(0) : "-");
  const nf1 = (n: number) => (Number.isFinite(n) ? n.toFixed(1) : "-");
  const nf2 = (n: number) => (Number.isFinite(n) ? n.toFixed(2) : "-");
  const nf3 = (n: number) => (Number.isFinite(n) ? n.toFixed(3) : "-");

  // warna adaptif utk OK Rate
  function toneOk(okPct: number) {
    if (!Number.isFinite(okPct)) return "text-gray-700 dark:text-gray-200";
    if (okPct >= 99.5) return "text-emerald-700 dark:text-emerald-300";
    if (okPct >= 97) return "text-green-700 dark:text-green-300";
    if (okPct >= 90) return "text-yellow-700 dark:text-yellow-300";
    return "text-red-700 dark:text-red-300";
  }
  function barWidth(val: number, max: number) {
    if (!Number.isFinite(val) || !Number.isFinite(max) || max <= 0) return "0%";
    return `${Math.min(100, (val / max) * 100)}%`;
  }

  // akses opsional meta tanpa menambah tipe baru
  const meta: any = (result as any)?.meta;
</script>

<div class="card p-6">
  {#if comparisonResult}
    <div class="flex items-center justify-between mb-6">
      <h2 class="text-xl font-semibold">Comparison Results</h2>
    </div>
    <ComparisonResults data={comparisonResult} />
  {:else if result}
    <!-- Header -->
    <div class="mb-6">
      <h2 class="text-xl font-semibold">Results</h2>
      <p class="text-sm muted">
        Ringkasan satu run berdasarkan CSV yang sudah diparse.
      </p>
    </div>

    <!-- KPI -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
      <div class="surface p-4">
        <div class="text-xs muted">Samples</div>
        <div class="mt-1 text-2xl font-bold tabular-nums">
          {result.summary.Samples.toLocaleString()}
        </div>
      </div>

      <div class="surface p-4">
        <div class="text-xs muted">OK Rate</div>
        <div
          class={"mt-1 text-2xl font-bold tabular-nums " +
            toneOk(result.summary.OKRatePct)}
        >
          {nf2(result.summary.OKRatePct)}%
        </div>
        <div
          class="mt-2 h-2 rounded bg-gray-200 dark:bg-white/10 overflow-hidden"
        >
          <div
            class="h-full bg-emerald-500 dark:bg-emerald-400"
            style={"width:" + barWidth(result.summary.OKRatePct, 100)}
          ></div>
        </div>
      </div>

      <div class="surface p-4">
        <div class="text-xs muted">RPS</div>
        <div class="mt-1 text-2xl font-bold tabular-nums">
          {nf0(result.summary.RPS)}
        </div>
      </div>

      <div class="surface p-4">
        <div class="text-xs muted">Duration</div>
        <div class="mt-1 text-2xl font-bold tabular-nums">
          {nf1(result.summary.DurationS)}s
        </div>
      </div>
    </div>

    <!-- Percentiles + Details + Meta -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-8">
      <div class="surface p-4">
        <div class="flex items-center justify-between">
          <h3 class="font-semibold">Latency Percentiles</h3>
          <span class="text-xs muted">ms</span>
        </div>

        <div class="mt-3 grid grid-cols-2 sm:grid-cols-5 gap-3">
          <div
            class="p-3 rounded-lg border border-gray-200 dark:border-white/10"
          >
            <div class="text-[11px] muted">P50</div>
            <div class="text-lg font-semibold tabular-nums">
              {nf3(result.summary.P50ms)}
            </div>
          </div>
          <div
            class="p-3 rounded-lg border border-gray-200 dark:border-white/10"
          >
            <div class="text-[11px] muted">P90</div>
            <div class="text-lg font-semibold tabular-nums">
              {nf3(result.summary.P90ms)}
            </div>
          </div>
          <div
            class="p-3 rounded-lg border border-gray-200 dark:border-white/10"
          >
            <div class="text-[11px] muted">P95</div>
            <div class="text-lg font-semibold tabular-nums">
              {nf3(result.summary.P95ms)}
            </div>
          </div>
          <div
            class="p-3 rounded-lg border border-gray-200 dark:border-white/10"
          >
            <div class="text-[11px] muted">P99</div>
            <div class="text-lg font-semibold tabular-nums">
              {nf3(result.summary.P99ms)}
            </div>
          </div>
          <div
            class="p-3 rounded-lg border border-gray-200 dark:border-white/10"
          >
            <div class="text-[11px] muted">Mean</div>
            <div class="text-lg font-semibold tabular-nums">
              {nf3(result.summary.Meanms)}
            </div>
          </div>
        </div>

        <!-- visual relatif ke P99 -->
        <div class="mt-4 space-y-2">
          <div class="text-[11px] muted">Relative to P99</div>
          <div class="h-2 rounded bg-gray-200 dark:bg-white/10 overflow-hidden">
            <div class="h-full bg-blue-500" style={"width:" + barWidth(result.summary.P50ms, result.summary.P99ms)}></div>
          </div>
          <div class="h-2 rounded bg-gray-200 dark:bg-white/10 overflow-hidden">
            <div class="h-full bg-indigo-500" style={"width:" + barWidth(result.summary.P90ms, result.summary.P99ms)}></div>
          </div>
          <div class="h-2 rounded bg-gray-200 dark:bg-white/10 overflow-hidden">
            <div class="h-full bg-purple-500" style={"width:" + barWidth(result.summary.P95ms, result.summary.P99ms)}></div>
          </div>
          <div class="h-2 rounded bg-gray-200 dark:bg-white/10 overflow-hidden">
            <div class="h-full bg-fuchsia-500" style={"width:" + barWidth(result.summary.P99ms, result.summary.P99ms)}></div>
          </div>
        </div>
      </div>

      <div class="surface p-4">
        <h3 class="font-semibold">Latency Distribution</h3>
        <p class="mt-2 text-sm muted">
          CDF & throughput sudah dihitung saat parsing. Visual detail tersedia
          di halaman chart utama.
        </p>

        <div class="mt-3 grid grid-cols-2 gap-3 text-sm">
          <div
            class="p-3 rounded-lg border border-gray-200 dark:border-white/10"
          >
            <div class="text-[11px] muted">Min</div>
            <div class="text-lg font-semibold tabular-nums">
              {nf3(result.summary.Minms)} ms
            </div>
          </div>
          <div
            class="p-3 rounded-lg border border-gray-200 dark:border-white/10"
          >
            <div class="text-[11px] muted">Max</div>
            <div class="text-lg font-semibold tabular-nums">
              {nf3(result.summary.Maxms)} ms
            </div>
          </div>
        </div>
      </div>

      <!-- Meta (opsional) -->
      {#if meta}
        <div class="surface p-4">
          <h3 class="font-semibold">Meta</h3>
          <div class="mt-3 grid grid-cols-2 gap-3 text-sm">
            <div
              class="p-3 rounded-lg border border-gray-200 dark:border-white/10"
            >
              <div class="text-[11px] muted">Start</div>
              <div class="font-semibold">
                {#if meta?.Start}{new Date(
                    meta.Start
                  ).toLocaleString()}{:else}-{/if}
              </div>
            </div>
            <div
              class="p-3 rounded-lg border border-gray-200 dark:border-white/10"
            >
              <div class="text-[11px] muted">End</div>
              <div class="font-semibold">
                {#if meta?.End}{new Date(
                    meta.End
                  ).toLocaleString()}{:else}-{/if}
              </div>
            </div>
            <div
              class="p-3 rounded-lg border border-gray-200 dark:border-white/10"
            >
              <div class="text-[11px] muted">Scenario</div>
              <div class="font-semibold">{meta?.Scenario ?? "-"}</div>
            </div>
            <div
              class="p-3 rounded-lg border border-gray-200 dark:border-white/10"
            >
              <div class="text-[11px] muted">Payload</div>
              <div class="font-semibold tabular-nums">
                {Number.isFinite(meta?.Payload) ? meta.Payload : "-"} B
              </div>
            </div>
          </div>
        </div>
      {/if}
    </div>

    <!-- Raw Summary -->
    <div class="surface p-4">
      <h3 class="font-semibold mb-3">Raw Summary</h3>
      <div class="overflow-x-auto">
        <table class="min-w-[800px] w-full text-xs">
          <thead
            class="bg-gray-100 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800"
          >
            <tr class="text-left">
              <th class="py-2 px-3">Metric</th>
              <th class="py-2 px-3">Value</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-200 dark:divide-gray-800">
            <tr
              ><td class="py-2 px-3">Samples</td><td
                class="py-2 px-3 tabular-nums"
                >{result.summary.Samples.toLocaleString()}</td
              ></tr
            >
            <tr
              ><td class="py-2 px-3">OK Rate</td><td
                class="py-2 px-3 tabular-nums"
                >{nf2(result.summary.OKRatePct)}%</td
              ></tr
            >
            <tr
              ><td class="py-2 px-3">RPS</td><td class="py-2 px-3 tabular-nums"
                >{nf0(result.summary.RPS)}</td
              ></tr
            >
            <tr
              ><td class="py-2 px-3">Duration</td><td
                class="py-2 px-3 tabular-nums"
                >{nf1(result.summary.DurationS)}s</td
              ></tr
            >
            <tr
              ><td class="py-2 px-3">P50</td><td class="py-2 px-3 tabular-nums"
                >{nf3(result.summary.P50ms)} ms</td
              ></tr
            >
            <tr
              ><td class="py-2 px-3">P90</td><td class="py-2 px-3 tabular-nums"
                >{nf3(result.summary.P90ms)} ms</td
              ></tr
            >
            <tr
              ><td class="py-2 px-3">P95</td><td class="py-2 px-3 tabular-nums"
                >{nf3(result.summary.P95ms)} ms</td
              ></tr
            >
            <tr
              ><td class="py-2 px-3">P99</td><td class="py-2 px-3 tabular-nums"
                >{nf3(result.summary.P99ms)} ms</td
              ></tr
            >
            <tr
              ><td class="py-2 px-3">Mean</td><td class="py-2 px-3 tabular-nums"
                >{nf3(result.summary.Meanms)} ms</td
              ></tr
            >
            <tr
              ><td class="py-2 px-3">Min</td><td class="py-2 px-3 tabular-nums"
                >{nf3(result.summary.Minms)} ms</td
              ></tr
            >
            <tr
              ><td class="py-2 px-3">Max</td><td class="py-2 px-3 tabular-nums"
                >{nf3(result.summary.Maxms)} ms</td
              ></tr
            >
          </tbody>
        </table>
      </div>
    </div>
  {:else}
    <div class="p-10">
      <div class="text-center">
        <div class="mx-auto h-16 w-16 text-gray-400 dark:text-gray-500 mb-4">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
            <path d="M3 3v18h18" stroke-width="2" />
          </svg>
        </div>
        <h3 class="mt-2 text-sm font-medium">No results yet</h3>
        <p class="mt-1 text-sm muted">Run a benchmark to see results</p>
      </div>
    </div>
  {/if}
</div>
