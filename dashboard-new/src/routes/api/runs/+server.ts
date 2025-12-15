import { json } from '@sveltejs/kit';
import { listRunsByScenario } from '$lib/server/db';
import type { RunWithPair } from '$lib/server/db';

// Kembalikan:
// - kalau ?scenario=abc => array runs skenario tsb
// - kalau tanpa scenario => payload: { grouped: { [ui_scenario]: RunWithPair[] }, scenarios: string[] }
export async function GET({ url }) {
  const scenario = url.searchParams.get('scenario');

  if (scenario) {
    const runs = await listRunsByScenario(scenario);
    return json({ runs });
  }

  // ambil semua skenario yang ada dengan cara sederhana:
  const knownScenarios = [
    'baseline','burst','cold_vs_resumed','parallel_streams','header_bloat',
    'uplink_loss','connection_churn','nat_rebinding','mixed_load','stress_test'
  ];

  const grouped: Record<string, RunWithPair[]> = {};
  for (const s of knownScenarios) {
    const runs = await listRunsByScenario(s);
    if (runs.length) grouped[s] = runs;
  }

  return json({ grouped, scenarios: Object.keys(grouped) });
}
