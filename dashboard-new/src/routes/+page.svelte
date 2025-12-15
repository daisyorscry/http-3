<script lang="ts">
  import ConfigPanel from "$lib/components/benchmark/ConfigPanel.svelte";
  import ResultsPanel from "$lib/components/benchmark/ResultsPanel.svelte";
  import LiveLogs from "$lib/components/LiveLogs.svelte";

  import {
    BarChart3,
    Zap,
    Power,
    Layers,
    FileText,
    UploadCloud,
    Shuffle,
    Route,
    SlidersHorizontal,
    Flame,
  } from "lucide-svelte";

  import type {
    BenchmarkResult,
    ComparisonResult,
    Scenario,
    ScenarioOption,
  } from "$lib/types/benchmark";
  import { getBackendScenario } from "$lib/utils/scenario";

  // state (Svelte 5 runes)
  let scenario = $state<Scenario>("baseline");


  let running = $state<boolean>(false);
  let progress = $state<string>("");
  let result = $state<BenchmarkResult | null>(null);
  let comparisonResult = $state<ComparisonResult | null>(null);
  let logH2 = $state<string>("");
  let logH3 = $state<string>("");
  let cmdH2 = $state<string>("");
  let cmdH3 = $state<string>("");
  let phaseH2 = $state<string>("");
  let phaseH3 = $state<string>("");
  let startServers = $state<boolean>(false);

  const scenarios: ScenarioOption[] = [
    {
      value: "baseline",
      label: "Low Traffic Baseline",
      subtitle: "Light periodic traffic with jitter",
      Icon: BarChart3,
    },
    {
      value: "burst",
      label: "Burst Traffic (Autocomplete)",
      subtitle: "Short spikes on top of steady base",
      Icon: Zap,
    },
    {
      value: "cold_vs_resumed",
      label: "Cold-Start vs Resumed",
      subtitle: "New connections vs pooled keep-alive",
      Icon: Power,
    },
    {
      value: "parallel_streams",
      label: "Parallel Requests (N Streams)",
      subtitle: "Evaluate multiplexing & head-of-line",
      Icon: Layers,
    },
    {
      value: "header_bloat",
      label: "Large Metadata Overhead",
      subtitle: "Stress HPACK/QPACK efficiency",
      Icon: FileText,
    },
    {
      value: "uplink_loss",
      label: "Small Uploads under Loss",
      subtitle: "Unstable uplink with retries",
      Icon: UploadCloud,
    },
    {
      value: "connection_churn",
      label: "High Connection Churn",
      subtitle: "Short-lived IoT-like sessions",
      Icon: Shuffle,
    },
    {
      value: "nat_rebinding",
      label: "NAT Rebinding / Migration",
      subtitle: "IP migration & path changes",
      Icon: Route,
    },
    {
      value: "mixed_load",
      label: "Mixed Load + Queueing",
      subtitle: "Heterogeneous traffic with queue",
      Icon: SlidersHorizontal,
    },
    {
      value: "stress_test",
      label: "High Traffic Stress Test",
      subtitle: "Aggressive ramp & saturation",
      Icon: Flame,
    },
  ];

  async function runComparison(): Promise<void> {
    running = true;
    progress = 'Starting comparison…';
    result = null; comparisonResult = null;
    logH2 = ''; logH3 = ''; cmdH2 = ''; cmdH3 = ''; phaseH2 = ''; phaseH3 = '';

    // Build SSE URL with query params
    const params = new URLSearchParams({
      scenario: getBackendScenario(scenario),
      uiScenario: String(scenario),
    });

    if (startServers) params.set('startServers', '1');
    const url = `/api/benchmark/compare?${params.toString()}`;
    const es = new EventSource(url);
    let fellBack = false;

    es.addEventListener('info', (e: MessageEvent) => {
      const data = JSON.parse(e.data);
      cmdH2 = data.cmdH2 || '';
      cmdH3 = data.cmdH3 || '';
      progress = `Running ${data.uiScenario || ''} (H2 then H3)…`;
    });
    es.addEventListener('phase', (e: MessageEvent) => {
      const d = JSON.parse(e.data);
      if (d.protocol === 'h2') phaseH2 = d.state || '';
      if (d.protocol === 'h3') phaseH3 = d.state || '';
    });
    es.addEventListener('log', (e: MessageEvent) => {
      const d = JSON.parse(e.data);
      if (d.protocol === 'h2') logH2 += String(d.line || '');
      if (d.protocol === 'h3') logH3 += String(d.line || '');
    });
    es.addEventListener('result', (e: MessageEvent) => {
      const payload = JSON.parse(e.data);
      comparisonResult = payload;
      progress = 'Comparison completed!';
    });
    es.addEventListener('end', () => { es.close(); running = false; });
    es.onerror = async () => {
      try { es.close(); } catch {}
      if (!fellBack) {
        fellBack = true;
        // Fallback to non-streaming endpoint (keeps UX working if SSE route not available)
        try {
          progress = 'Stream unavailable, running comparison (fallback)…';
          const response = await fetch('/api/benchmark/compare', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              scenario: getBackendScenario(scenario),
              uiScenario: scenario
            })
          });
          const data = await response.json();
          if ((data as any)?.error) progress = `Error: ${(data as any).error}`;
          else { comparisonResult = data; progress = 'Comparison completed!'; }
        } catch (e) {
          progress = 'Error: stream + fallback failed';
        } finally {
          running = false;
        }
      } else {
        running = false;
        if (!comparisonResult) progress = 'Error: stream closed';
      }
    };
  }

  function onUpdate(e: CustomEvent<{ key: string; value: unknown }>) {
    const { key, value } = e.detail;

    if (key === "scenario") scenario = value as any;
    else if (key === "startServers") startServers = Boolean(value);
  }
</script>

<div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
  <div class="lg:col-span-1">
    <ConfigPanel
      {scenario}
      {scenarios}
      {startServers}
      {running}
      {progress}
      on:update={onUpdate}
      on:runCompare={runComparison}
    />
  </div>

  <div class="lg:col-span-2 space-y-6">
    <LiveLogs {cmdH2} {cmdH3} logH2={logH2} logH3={logH3} {phaseH2} {phaseH3} />
    <ResultsPanel {result} {comparisonResult} />
  </div>
</div>
