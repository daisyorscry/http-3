<script lang="ts">
  import { Trophy, Circle, Handshake } from "lucide-svelte";

  // === tipe lokal (sesuai struktur kamu di luar) ===
  interface BenchmarkSummary {
    Samples: number;
    OKRatePct: number;
    RPS: number;
    DurationS: number;
    P50ms: number;
    P90ms: number;
    P95ms: number;
    P99ms: number;
    Meanms: number;
    Minms: number;
    Maxms: number;
  }

  interface ComparisonMetrics {
    latencyWinner: "h2" | "h3" | "tie";
    throughputWinner: "h2" | "h3" | "tie";
    p50Diff: number; // ((h2.p50 - h3.p50)/h2.p50)*100 → positif = H3 lebih cepat
    p99Diff: number; // ((h2.p99 - h3.p99)/h2.p99)*100 → positif = H3 lebih cepat
    rpsDiff: number; // ((h3.rps - h2.rps)/h2.rps)*100 → positif = H3 lebih tinggi
    latencyImprovement: number; // ringkasan improv latency (avg p50/p99 diff)
  }

  interface ProtocolResult {
    summary: BenchmarkSummary;
    protocol: "HTTP/2" | "HTTP/3";
  }

  interface ComparisonData {
    h2: ProtocolResult;
    h3: ProtocolResult;
    comparison: ComparisonMetrics;
  }

  let { data } = $props<{ data: ComparisonData }>();

  // ==== helpers ====
  function getWinnerSurface(
    winner: "h2" | "h3" | "tie",
    current: "h2" | "h3"
  ): string {
    if (winner === "tie") return "border-gray-200 dark:border-white/10";
    return winner === current
      ? "border-emerald-500 dark:border-emerald-400"
      : "border-gray-200 dark:border-white/10";
  }

  function showWinnerBadge(
    winner: "h2" | "h3" | "tie",
    current: "h2" | "h3"
  ): boolean {
    return winner !== "tie" && winner === current;
  }

  const nf0 = (n: number) => (Number.isFinite(n) ? n.toFixed(0) : "-");
  const nf1 = (n: number) => (Number.isFinite(n) ? n.toFixed(1) : "-");
  const nf2 = (n: number) => (Number.isFinite(n) ? n.toFixed(2) : "-");
  const nf3 = (n: number) => (Number.isFinite(n) ? n.toFixed(3) : "-");

  function diffTonePosNeg(v: number) {
    if (!Number.isFinite(v) || v === 0)
      return "text-gray-700 dark:text-gray-200";
    return v > 0
      ? "text-emerald-600 dark:text-emerald-400"
      : "text-rose-600 dark:text-rose-400";
  }

  // label per metric (benar secara definisi diff)
  function pLabel(v: number) {
    if (!Number.isFinite(v) || v === 0) return "Tie";
    return v > 0 ? "H3 faster" : "H2 faster";
  }
  function rpsLabel(v: number) {
    if (!Number.isFinite(v) || v === 0) return "Tie";
    return v > 0 ? "H3 higher" : "H2 higher";
  }
</script>

<div class="space-y-6">
  <!-- Winner Banner -->
  <div class="card p-6">
    <div class="text-center">
      <h3 class="text-2xl font-bold">Comparison Results</h3>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 max-w-3xl mx-auto mt-4">
        <div class="surface p-4">
          <div class="text-sm muted mb-1">Latency Winner</div>
          <div
            class="text-3xl font-bold flex items-center justify-center gap-2"
          >
            {#if data.comparison.latencyWinner === "h2"}
              <Circle class="w-6 h-6 fill-blue-500 text-blue-500" />
              HTTP/2
            {:else if data.comparison.latencyWinner === "h3"}
              <Circle class="w-6 h-6 fill-purple-500 text-purple-500" />
              HTTP/3
            {:else}
              <Handshake class="w-6 h-6 text-gray-500 dark:text-gray-400" />
              Tie
            {/if}
          </div>
          <div
            class="text-sm mt-1 {diffTonePosNeg(
              data.comparison.latencyImprovement
            )}"
          >
            {data.comparison.latencyImprovement > 0 ? "+" : ""}{nf2(
              data.comparison.latencyImprovement
            )}% improvement
          </div>
        </div>

        <div class="surface p-4">
          <div class="text-sm muted mb-1">Throughput Winner</div>
          <div
            class="text-3xl font-bold flex items-center justify-center gap-2"
          >
            {#if data.comparison.throughputWinner === "h2"}
              <Circle class="w-6 h-6 fill-blue-500 text-blue-500" />
              HTTP/2
            {:else if data.comparison.throughputWinner === "h3"}
              <Circle class="w-6 h-6 fill-purple-500 text-purple-500" />
              HTTP/3
            {:else}
              <Handshake class="w-6 h-6 text-gray-500 dark:text-gray-400" />
              Tie
            {/if}
          </div>
          <div class="text-sm mt-1 {diffTonePosNeg(data.comparison.rpsDiff)}">
            {data.comparison.rpsDiff > 0 ? "+" : ""}{nf2(
              data.comparison.rpsDiff
            )}% difference
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- Side-by-Side (latency winner highlight di border) -->
  <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
    <!-- HTTP/2 -->
    <div
      class="card border-2 {getWinnerSurface(
        data.comparison.latencyWinner,
        'h2'
      )}"
    >
      <div class="p-6">
        <div class="flex items-center justify-between mb-4">
          <h3
            class="text-xl font-bold text-blue-600 dark:text-blue-400 flex items-center gap-2"
          >
            <Circle class="w-5 h-5 fill-blue-500 text-blue-500" />
            HTTP/2
          </h3>
          {#if showWinnerBadge(data.comparison.latencyWinner, "h2")}
            <Trophy class="w-6 h-6 text-amber-500" />
          {/if}
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div class="surface p-3">
            <div class="text-xs muted">Samples</div>
            <div class="mt-1 font-bold tabular-nums">
              {data.h2.summary.Samples.toLocaleString()}
            </div>
          </div>
          <div class="surface p-3">
            <div class="text-xs muted">OK Rate</div>
            <div class="mt-1 font-bold tabular-nums">
              {nf2(data.h2.summary.OKRatePct)}%
            </div>
          </div>
          <div class="surface p-3">
            <div class="text-xs muted">RPS</div>
            <div class="mt-1 font-bold tabular-nums">
              {nf0(data.h2.summary.RPS)}
            </div>
          </div>
          <div class="surface p-3">
            <div class="text-xs muted">Duration</div>
            <div class="mt-1 font-bold tabular-nums">
              {nf1(data.h2.summary.DurationS)}s
            </div>
          </div>
        </div>

        <div class="mt-5 pt-5 border-t border-gray-200 dark:border-white/10">
          <h4 class="text-sm font-semibold mb-3">Latency Percentiles</h4>
          <div class="grid grid-cols-2 sm:grid-cols-5 gap-2 text-sm">
            <div class="surface p-2">
              <div class="text-[11px] muted">P50</div>
              <div class="font-medium tabular-nums">
                {nf3(data.h2.summary.P50ms)}ms
              </div>
            </div>
            <div class="surface p-2">
              <div class="text-[11px] muted">P90</div>
              <div class="font-medium tabular-nums">
                {nf3(data.h2.summary.P90ms)}ms
              </div>
            </div>
            <div class="surface p-2">
              <div class="text-[11px] muted">P95</div>
              <div class="font-medium tabular-nums">
                {nf3(data.h2.summary.P95ms)}ms
              </div>
            </div>
            <div class="surface p-2">
              <div class="text-[11px] muted">P99</div>
              <div class="font-medium tabular-nums">
                {nf3(data.h2.summary.P99ms)}ms
              </div>
            </div>
            <div class="surface p-2">
              <div class="text-[11px] muted">Mean</div>
              <div class="font-medium tabular-nums">
                {nf3(data.h2.summary.Meanms)}ms
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- HTTP/3 -->
    <div
      class="card border-2 {getWinnerSurface(
        data.comparison.latencyWinner,
        'h3'
      )}"
    >
      <div class="p-6">
        <div class="flex items-center justify-between mb-4">
          <h3
            class="text-xl font-bold text-purple-600 dark:text-purple-400 flex items-center gap-2"
          >
            <Circle class="w-5 h-5 fill-purple-500 text-purple-500" />
            HTTP/3
          </h3>
          {#if showWinnerBadge(data.comparison.latencyWinner, "h3")}
            <Trophy class="w-6 h-6 text-amber-500" />
          {/if}
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div class="surface p-3">
            <div class="text-xs muted">Samples</div>
            <div class="mt-1 font-bold tabular-nums">
              {data.h3.summary.Samples.toLocaleString()}
            </div>
          </div>
          <div class="surface p-3">
            <div class="text-xs muted">OK Rate</div>
            <div class="mt-1 font-bold tabular-nums">
              {nf2(data.h3.summary.OKRatePct)}%
            </div>
          </div>
          <div class="surface p-3">
            <div class="text-xs muted">RPS</div>
            <div class="mt-1 font-bold tabular-nums">
              {nf0(data.h3.summary.RPS)}
            </div>
          </div>
          <div class="surface p-3">
            <div class="text-xs muted">Duration</div>
            <div class="mt-1 font-bold tabular-nums">
              {nf1(data.h3.summary.DurationS)}s
            </div>
          </div>
        </div>

        <div class="mt-5 pt-5 border-t border-gray-200 dark:border-white/10">
          <h4 class="text-sm font-semibold mb-3">Latency Percentiles</h4>
          <div class="grid grid-cols-2 sm:grid-cols-5 gap-2 text-sm">
            <div class="surface p-2">
              <div class="text-[11px] muted">P50</div>
              <div class="font-medium tabular-nums">
                {nf3(data.h3.summary.P50ms)}ms
              </div>
            </div>
            <div class="surface p-2">
              <div class="text-[11px] muted">P90</div>
              <div class="font-medium tabular-nums">
                {nf3(data.h3.summary.P90ms)}ms
              </div>
            </div>
            <div class="surface p-2">
              <div class="text-[11px] muted">P95</div>
              <div class="font-medium tabular-nums">
                {nf3(data.h3.summary.P95ms)}ms
              </div>
            </div>
            <div class="surface p-2">
              <div class="text-[11px] muted">P99</div>
              <div class="font-medium tabular-nums">
                {nf3(data.h3.summary.P99ms)}ms
              </div>
            </div>
            <div class="surface p-2">
              <div class="text-[11px] muted">Mean</div>
              <div class="font-medium tabular-nums">
                {nf3(data.h3.summary.Meanms)}ms
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- Detailed Comparison Metrics -->
  <div class="card p-6">
    <h3 class="text-lg font-semibold mb-4">Detailed Comparison</h3>
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <div class="surface p-4 text-center">
        <div class="text-xs muted mb-1">P50 Difference</div>
        <div
          class={"text-2xl font-bold " +
            diffTonePosNeg(data.comparison.p50Diff)}
        >
          {data.comparison.p50Diff > 0 ? "+" : ""}{nf2(
            data.comparison.p50Diff
          )}%
        </div>
        <div class="text-xs muted mt-1">{pLabel(data.comparison.p50Diff)}</div>
      </div>

      <div class="surface p-4 text-center">
        <div class="text-xs muted mb-1">P99 Difference</div>
        <div
          class={"text-2xl font-bold " +
            diffTonePosNeg(data.comparison.p99Diff)}
        >
          {data.comparison.p99Diff > 0 ? "+" : ""}{nf2(
            data.comparison.p99Diff
          )}%
        </div>
        <div class="text-xs muted mt-1">{pLabel(data.comparison.p99Diff)}</div>
      </div>

      <div class="surface p-4 text-center">
        <div class="text-xs muted mb-1">RPS Difference</div>
        <div
          class={"text-2xl font-bold " +
            diffTonePosNeg(data.comparison.rpsDiff)}
        >
          {data.comparison.rpsDiff > 0 ? "+" : ""}{nf2(
            data.comparison.rpsDiff
          )}%
        </div>
        <div class="text-xs muted mt-1">
          {rpsLabel(data.comparison.rpsDiff)}
        </div>
      </div>
    </div>
  </div>
</div>
