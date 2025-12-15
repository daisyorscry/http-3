import type { Scenario } from "../types/benchmark";

// Map UI scenario names to backend scenario identifiers
// Since we removed level-based configs, we now pass scenario names directly
export function getBackendScenario(s: Scenario): string {
  // Direct pass-through - UI scenario names match backend scenario names
  return s;
}
