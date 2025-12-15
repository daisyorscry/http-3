import { json } from "@sveltejs/kit";
import { listRunsByScenario } from "$lib/server/db";
import type { RunWithPair } from "$lib/server/db";

/* util statistik sederhana */
function mean(nums: number[]) {
  return nums.length ? nums.reduce((a, b) => a + b, 0) / nums.length : 0;
}
function pct(n: number) {
  return isFinite(n) ? n : 0;
}

function perScenarioSummary(runs: RunWithPair[]) {
  // hanya pasangan yang punya keduanya:
  const pairs = runs.filter((r) => r.h2 && r.h3);

  // win counters
  let latencyH2 = 0,
    latencyH3 = 0,
    latencyTie = 0;
  let rpsH2 = 0,
    rpsH3 = 0,
    rpsTie = 0;

  // improvements
  const p50ImprovePct: number[] = []; // (h2 - h3)/h2 (lebih besar = H3 lebih cepat)
  const rpsGainPct: number[] = []; // (h3 - h2)/h2 (lebih besar = H3 lebih kencang)

  const h2p50: number[] = [];
  const h3p50: number[] = [];
  const h2rps: number[] = [];
  const h3rps: number[] = [];

  for (const r of pairs) {
    const h2 = r.h2!,
      h3 = r.h3!;
    // latency p50: kecil lebih baik
    if (h2.p50ms < h3.p50ms) latencyH2++;
    else if (h3.p50ms < h2.p50ms) latencyH3++;
    else latencyTie++;

    // throughput rps: besar lebih baik
    if (h2.rps > h3.rps) rpsH2++;
    else if (h3.rps > h2.rps) rpsH3++;
    else rpsTie++;

    if (h2.p50ms > 0) p50ImprovePct.push((h2.p50ms - h3.p50ms) / h2.p50ms);
    if (h2.rps > 0) rpsGainPct.push((h3.rps - h2.rps) / h2.rps);

    h2p50.push(h2.p50ms);
    h3p50.push(h3.p50ms);
    h2rps.push(h2.rps);
    h3rps.push(h3.rps);
  }

  const nPairs = pairs.length;
  const latencyWinRateH3 = nPairs ? latencyH3 / nPairs : 0;
  const rpsWinRateH3 = nPairs ? rpsH3 / nPairs : 0;

  return {
    counts: { totalRuns: runs.length, comparablePairs: nPairs },
    latency: {
      h2Wins: latencyH2,
      h3Wins: latencyH3,
      ties: latencyTie,
      avgP50_h2_ms: mean(h2p50),
      avgP50_h3_ms: mean(h3p50),
      avgLatencyImprovementPct_vsH2: mean(p50ImprovePct), // + berarti H3 lebih cepat
    },
    throughput: {
      h2Wins: rpsH2,
      h3Wins: rpsH3,
      ties: rpsTie,
      avgRPS_h2: mean(h2rps),
      avgRPS_h3: mean(h3rps),
      avgRpsGainPct_vsH2: mean(rpsGainPct), // + berarti H3 lebih kencang
    },
    winRates: {
      latencyH3: latencyWinRateH3,
      rpsH3: rpsWinRateH3,
    },
    winner: {
      latency:
        latencyH3 > latencyH2 ? "h3" : latencyH2 > latencyH3 ? "h2" : "tie",
      rps: rpsH3 > rpsH2 ? "h3" : rpsH2 > rpsH3 ? "h2" : "tie",
    },
  };
}

export async function GET({ url }) {
  const scenario = url.searchParams.get("scenario");

  if (scenario) {
    const runs = await listRunsByScenario(scenario);
    const earliest = runs[0] ?? null;
    const latest = runs[runs.length - 1] ?? null;
    const stabilityLike = perScenarioSummary(runs);
    return json({
      scenario,
      earliest,
      latest,
      summary: stabilityLike,
    });
  }

  // GLOBAL: kumpulkan semua skenario yang punya data
  const knownScenarios = [
    "baseline",
    "burst",
    "cold_vs_resumed",
    "parallel_streams",
    "header_bloat",
    "uplink_loss",
    "connection_churn",
    "nat_rebinding",
    "mixed_load",
    "stress_test",
  ];

  const perScenario: Record<string, ReturnType<typeof perScenarioSummary>> = {};
  const allPairs: RunWithPair[] = [];

  for (const s of knownScenarios) {
    const runs = await listRunsByScenario(s);
    if (!runs.length) continue;
    perScenario[s] = perScenarioSummary(runs);
    allPairs.push(...runs);
  }

  const overall = perScenarioSummary(allPairs);

  return json({ overall, perScenario, scenarios: Object.keys(perScenario) });
}
