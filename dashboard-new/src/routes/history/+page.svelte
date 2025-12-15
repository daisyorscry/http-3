<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import ScenarioSelector from "$lib/components/ScenarioSelector.svelte";
  import {
    BarChart3,
    TrendingUp,
    Flame,
    Power,
    Layers,
    FileText,
    UploadCloud,
    Shuffle,
    Route,
    SlidersHorizontal,
  } from "lucide-svelte";
  import type { ComponentType } from "svelte";

  // Chart.js: lazy import
  let Chart: any | null = null;

  type Scenario =
    | "baseline"
    | "burst"
    | "cold_vs_resumed"
    | "parallel_streams"
    | "header_bloat"
    | "uplink_loss"
    | "connection_churn"
    | "nat_rebinding"
    | "mixed_load"
    | "stress_test";

  interface ScenarioOption {
    value: Scenario;
    label: string;
    subtitle: string;
    Icon: ComponentType;
  }

  interface StatRow {
    samples: number;
    success_rate: number;
    rps: number;
    duration_s: number;
    p50ms: number;
    p90ms: number;
    p99ms: number;
    meanms: number;
    minms: number;
    maxms: number;
  }

  interface RunWithPair {
    created_at: string;
    h2?: StatRow;
    h3?: StatRow;
  }

  type Paired = { created_at: string; h2: StatRow; h3: StatRow };

  const scenarios: ScenarioOption[] = [
    {
      value: "baseline",
      label: "Low Traffic Baseline",
      subtitle: "Light periodic traffic",
      Icon: BarChart3,
    },
    {
      value: "burst",
      label: "Burst Traffic",
      subtitle: "Short spikes",
      Icon: TrendingUp,
    },
    {
      value: "cold_vs_resumed",
      label: "Cold vs Resumed",
      subtitle: "Connection reuse",
      Icon: Power,
    },
    {
      value: "parallel_streams",
      label: "Parallel Streams",
      subtitle: "Multiplexing test",
      Icon: Layers,
    },
    {
      value: "header_bloat",
      label: "Header Bloat",
      subtitle: "Large metadata stress",
      Icon: FileText,
    },
    {
      value: "uplink_loss",
      label: "Uplink Loss",
      subtitle: "Lossy network",
      Icon: UploadCloud,
    },
    {
      value: "connection_churn",
      label: "Connection Churn",
      subtitle: "Short-lived sessions",
      Icon: Shuffle,
    },
    {
      value: "nat_rebinding",
      label: "NAT Rebinding",
      subtitle: "IP/path changes",
      Icon: Route,
    },
    {
      value: "mixed_load",
      label: "Mixed Load",
      subtitle: "Heterogeneous traffic",
      Icon: SlidersHorizontal,
    },
    {
      value: "stress_test",
      label: "Stress Test",
      subtitle: "Heavy load",
      Icon: Flame,
    },
  ];

  const pct = (n: number) => `${n.toFixed(2)}%`;
  function shortSummary(deltaPct: number) {
    if (!isFinite(deltaPct)) return "Setara";
    if (deltaPct > 0) return `H3 lebih baik ${pct(deltaPct)}`;
    if (deltaPct < 0) return `H3 lebih buruk ${pct(Math.abs(deltaPct))}`;
    return "Setara";
  }

  let scenario: Scenario = "baseline";
  let grouped: Record<string, RunWithPair[]> = {};

  // canvases & chart instances
  let rpsCanvas: HTMLCanvasElement;
  let p50Canvas: HTMLCanvasElement;
  let p90Canvas: HTMLCanvasElement;
  let p99Canvas: HTMLCanvasElement;
  let rpsChart: any = null;
  let p50Chart: any = null;
  let p90Chart: any = null;
  let p99Chart: any = null;

  async function loadAll() {
    const res = await fetch("/api/runs", {
      headers: { Accept: "application/json" },
    });
    const data = await res.json();
    grouped = data.grouped || {};
  }

  onMount(loadAll);
  onDestroy(() => {
    rpsChart?.destroy?.();
    p50Chart?.destroy?.();
    p90Chart?.destroy?.();
    p99Chart?.destroy?.();
  });

  // type guard agar TS tahu h2 & h3 pasti ada
  function hasBoth(r: RunWithPair): r is Paired {
    return !!(r.h2 && r.h3);
  }

  // derived: pasangan yang valid untuk scenario terpilih
  let paired: Paired[] = [];
  $: {
    const rows = grouped?.[scenario] ?? [];
    paired = rows.filter(hasBoth);
  }

  // build / rebuild charts saat data berubah
  $: if (paired.length) {
    (async () => {
      if (!Chart) {
        const mod = await import("chart.js/auto");
        Chart = mod.default;
      }

      const labels = paired.map((r) => new Date(r.created_at).toLocaleString());

      const h2rps = paired.map((r) => r.h2.rps);
      const h3rps = paired.map((r) => r.h3.rps);
      const h2p50 = paired.map((r) => r.h2.p50ms);
      const h3p50 = paired.map((r) => r.h3.p50ms);
      const h2p90 = paired.map((r) => r.h2.p90ms);
      const h3p90 = paired.map((r) => r.h3.p90ms);
      const h2p99 = paired.map((r) => r.h2.p99ms);
      const h3p99 = paired.map((r) => r.h3.p99ms);

      rpsChart?.destroy?.();
      p50Chart?.destroy?.();
      p90Chart?.destroy?.();
      p99Chart?.destroy?.();

      const common: any = {
        responsive: true,
        animation: false,
        maintainAspectRatio: false,
        interaction: { mode: "index", intersect: false },
        plugins: {
          legend: { display: true, position: "top" },
          tooltip: { enabled: true },
        },
        scales: {
          x: { ticks: { autoSkip: true, maxTicksLimit: 8 } },
          y: { beginAtZero: false },
        },
      };

      rpsChart = new Chart(rpsCanvas.getContext("2d")!, {
        type: "line",
        data: {
          labels,
          datasets: [
            { label: "H2 RPS", data: h2rps, tension: 0.2, pointRadius: 0 },
            { label: "H3 RPS", data: h3rps, tension: 0.2, pointRadius: 0 },
          ],
        },
        options: common,
      });

      p50Chart = new Chart(p50Canvas.getContext("2d")!, {
        type: "line",
        data: {
          labels,
          datasets: [
            { label: "H2 p50 (ms)", data: h2p50, tension: 0.2, pointRadius: 0 },
            { label: "H3 p50 (ms)", data: h3p50, tension: 0.2, pointRadius: 0 },
          ],
        },
        options: common,
      });

      p90Chart = new Chart(p90Canvas.getContext("2d")!, {
        type: "line",
        data: {
          labels,
          datasets: [
            { label: "H2 p90 (ms)", data: h2p90, tension: 0.2, pointRadius: 0 },
            { label: "H3 p90 (ms)", data: h3p90, tension: 0.2, pointRadius: 0 },
          ],
        },
        options: common,
      });

      p99Chart = new Chart(p99Canvas.getContext("2d")!, {
        type: "line",
        data: {
          labels,
          datasets: [
            { label: "H2 p99 (ms)", data: h2p99, tension: 0.2, pointRadius: 0 },
            { label: "H3 p99 (ms)", data: h3p99, tension: 0.2, pointRadius: 0 },
          ],
        },
        options: common,
      });
    })();
  } else {
    // jika tidak ada data, pastikan chart dibersihkan
    rpsChart?.destroy?.();
    p50Chart?.destroy?.();
    p90Chart?.destroy?.();
    p99Chart?.destroy?.();
    rpsChart = p50Chart = p90Chart = p99Chart = null;
  }

  // ringkasan rata-rata delta (%)
  function avg(arr: number[]) {
    return arr.length ? arr.reduce((a, b) => a + b, 0) / arr.length : 0;
  }
  $: dRps = avg(paired.map((r) => ((r.h3.rps - r.h2.rps) / r.h2.rps) * 100));
  $: dP50 = avg(
    paired.map((r) => ((r.h2.p50ms - r.h3.p50ms) / r.h2.p50ms) * 100)
  );
  $: dP90 = avg(
    paired.map((r) => ((r.h2.p90ms - r.h3.p90ms) / r.h2.p90ms) * 100)
  );
  $: dP99 = avg(
    paired.map((r) => ((r.h2.p99ms - r.h3.p99ms) / r.h2.p99ms) * 100)
  );

  const fmt = (n: number, d = 2) => (Number.isFinite(n) ? n.toFixed(d) : "-");
</script>

<ScenarioSelector
  items={scenarios}
  selected={scenario}
  on:select={(e) => (scenario = e.detail.value as Scenario)}
/>

{#if grouped[scenario]?.length}
  <!-- CHARTS: tetap ada, responsif, tinggi nyaman -->
  <div class="grid grid-cols-4 xl:grid-cols-1 gap-6 mb-6">
    <div class="card p-4 h-56 sm:h-64 lg:h-72">
      <div class="font-semibold mb-2">Trend: RPS</div>
      <div class="relative w-full h-[calc(100%-0.5rem)]">
        <canvas bind:this={rpsCanvas} class="absolute inset-0 w-full h-full"
        ></canvas>
      </div>
    </div>
    <!-- <div class="card p-3 text-sm">
      <div class="muted">RPS</div>
      <div class="font-semibold">{shortSummary(dRps)}</div>
    </div> -->
    <div class="card p-4 h-56 sm:h-64 lg:h-72">
      <div class="font-semibold mb-2">Trend: p50 (ms)</div>
      <div class="relative w-full h-[calc(100%-0.5rem)]">
        <canvas bind:this={p50Canvas} class="absolute inset-0 w-full h-full"
        ></canvas>
      </div>
    </div>
    <!-- <div class="card p-3 text-sm">
      <div class="muted">p50</div>
      <div class="font-semibold">{shortSummary(dP50)}</div>
    </div> -->
    <div class="card p-4 h-56 sm:h-64 lg:h-72">
      <div class="font-semibold mb-2">Trend: p90 (ms)</div>
      <div class="relative w-full h-[calc(100%-0.5rem)]">
        <canvas bind:this={p90Canvas} class="absolute inset-0 w-full h-full"
        ></canvas>
      </div>
    </div>
    <!-- <div class="card p-3 text-sm">
      <div class="muted">p90</div>
      <div class="font-semibold">{shortSummary(dP90)}</div>
    </div> -->
    <div class="card p-4 h-56 sm:h-64 lg:h-72">
      <div class="font-semibold mb-2">Trend: p99 (ms)</div>
      <div class="relative w-full h-[calc(100%-0.5rem)]">
        <canvas bind:this={p99Canvas} class="absolute inset-0 w-full h-full"
        ></canvas>
      </div>
    </div>
    <!-- <div class="card p-3 text-sm">
      <div class="muted">p99</div>
      <div class="font-semibold">{shortSummary(dP99)}</div>
    </div> -->
  </div>

  <!-- DESKTOP: TABLE LEBAR + SCROLL -->
  <div class="hidden lg:block">
    <div class="card overflow-hidden">
      <div class="overflow-x-auto">
        <table class="min-w-[1100px] w-full text-xs">
          <thead
            class="bg-gray-100 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800 sticky top-0 z-10"
          >
            <tr class="text-left text-gray-700 dark:text-gray-300">
              <th class="py-2 px-3 w-[180px]">Time</th>
              <th class="py-2 px-3 text-right">H2 RPS</th>
              <th class="py-2 px-3 text-right">H3 RPS</th>
              <th class="py-2 px-3">ΔRPS</th>
              <th class="py-2 px-3 text-right">H2 p50</th>
              <th class="py-2 px-3 text-right">H3 p50</th>
              <th class="py-2 px-3">Δp50</th>
              <th class="py-2 px-3 text-right">H2 p90</th>
              <th class="py-2 px-3 text-right">H3 p90</th>
              <th class="py-2 px-3">Δp90</th>
              <th class="py-2 px-3 text-right">H2 p99</th>
              <th class="py-2 px-3 text-right">H3 p99</th>
              <th class="py-2 px-3">Δp99</th>
              <th class="py-2 px-3 text-right">Mean</th>
              <th class="py-2 px-3 text-right">Min</th>
              <th class="py-2 px-3 text-right">Max</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-200 dark:divide-gray-800">
            {#each paired as r}
              {@const dR = ((r.h3.rps - r.h2.rps) / r.h2.rps) * 100}
              {@const d50 = ((r.h2.p50ms - r.h3.p50ms) / r.h2.p50ms) * 100}
              {@const d90 = ((r.h2.p90ms - r.h3.p90ms) / r.h2.p90ms) * 100}
              {@const d99 = ((r.h2.p99ms - r.h3.p99ms) / r.h2.p99ms) * 100}
              <tr class="hover:bg-gray-50 dark:hover:bg-gray-800/60">
                <td class="py-2 px-3 whitespace-nowrap"
                  >{new Date(r.created_at).toLocaleString()}</td
                >
                <td class="py-2 px-3 text-right tabular-nums"
                  >{r.h2.rps.toFixed(0)}</td
                >
                <td class="py-2 px-3 text-right tabular-nums"
                  >{r.h3.rps.toFixed(0)}</td
                >
                <td class="py-2 px-3">{shortSummary(dR)}</td>
                <td class="py-2 px-3 text-right tabular-nums"
                  >{fmt(r.h2.p50ms)}</td
                >
                <td class="py-2 px-3 text-right tabular-nums"
                  >{fmt(r.h3.p50ms)}</td
                >
                <td class="py-2 px-3">{shortSummary(d50)}</td>
                <td class="py-2 px-3 text-right tabular-nums"
                  >{fmt(r.h2.p90ms)}</td
                >
                <td class="py-2 px-3 text-right tabular-nums"
                  >{fmt(r.h3.p90ms)}</td
                >
                <td class="py-2 px-3">{shortSummary(d90)}</td>
                <td class="py-2 px-3 text-right tabular-nums"
                  >{fmt(r.h2.p99ms)}</td
                >
                <td class="py-2 px-3 text-right tabular-nums"
                  >{fmt(r.h3.p99ms)}</td
                >
                <td class="py-2 px-3">{shortSummary(d99)}</td>
                <td class="py-2 px-3 text-right tabular-nums"
                  >{fmt(r.h2.meanms)} / {fmt(r.h3.meanms)}</td
                >
                <td class="py-2 px-3 text-right tabular-nums"
                  >{fmt(r.h2.minms)} / {fmt(r.h3.minms)}</td
                >
                <td class="py-2 px-3 text-right tabular-nums"
                  >{fmt(r.h2.maxms)} / {fmt(r.h3.maxms)}</td
                >
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  </div>

  <!-- MOBILE/TABLET KECIL: CARDS -->
  <div class="lg:hidden">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      {#each paired as r}
        {@const dR = ((r.h3.rps - r.h2.rps) / r.h2.rps) * 100}
        {@const d50 = ((r.h2.p50ms - r.h3.p50ms) / r.h2.p50ms) * 100}
        {@const d90 = ((r.h2.p90ms - r.h3.p90ms) / r.h2.p90ms) * 100}
        {@const d99 = ((r.h2.p99ms - r.h3.p99ms) / r.h2.p99ms) * 100}
        <div class="card p-4">
          <div class="flex items-center justify-between gap-3">
            <div class="font-semibold text-sm">
              {new Date(r.created_at).toLocaleString()}
            </div>
            <div class="text-xs muted">{shortSummary(dR)}</div>
          </div>

          <div class="mt-3 grid grid-cols-2 gap-3 text-xs">
            <div class="surface p-2">
              <div class="muted">RPS</div>
              <div class="mt-1 font-semibold tabular-nums">
                {r.h2.rps.toFixed(0)} / {r.h3.rps.toFixed(0)}
              </div>
            </div>
            <div class="surface p-2">
              <div class="muted">p50 (ms)</div>
              <div class="mt-1 font-semibold tabular-nums">
                {fmt(r.h2.p50ms)} / {fmt(r.h3.p50ms)}
              </div>
              <div class="text-[11px] muted">{shortSummary(d50)}</div>
            </div>
            <div class="surface p-2">
              <div class="muted">p90 (ms)</div>
              <div class="mt-1 font-semibold tabular-nums">
                {fmt(r.h2.p90ms)} / {fmt(r.h3.p90ms)}
              </div>
              <div class="text-[11px] muted">{shortSummary(d90)}</div>
            </div>
            <div class="surface p-2">
              <div class="muted">p99 (ms)</div>
              <div class="mt-1 font-semibold tabular-nums">
                {fmt(r.h2.p99ms)} / {fmt(r.h3.p99ms)}
              </div>
              <div class="text-[11px] muted">{shortSummary(d99)}</div>
            </div>
          </div>

          <div class="mt-3 grid grid-cols-3 gap-3 text-xs">
            <div class="surface p-2">
              <div class="muted">Mean</div>
              <div class="mt-1 font-semibold tabular-nums">
                {fmt(r.h2.meanms)} / {fmt(r.h3.meanms)}
              </div>
            </div>
            <div class="surface p-2">
              <div class="muted">Min</div>
              <div class="mt-1 font-semibold tabular-nums">
                {fmt(r.h2.minms)} / {fmt(r.h3.minms)}
              </div>
            </div>
            <div class="surface p-2">
              <div class="muted">Max</div>
              <div class="mt-1 font-semibold tabular-nums">
                {fmt(r.h2.maxms)} / {fmt(r.h3.maxms)}
              </div>
            </div>
          </div>
        </div>
      {/each}
    </div>
  </div>
{/if}
