import type { ComponentType } from "svelte";

export interface BenchmarkSummary {
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

export interface BenchmarkResult {
  success: boolean;
  summary: BenchmarkSummary;
  stdout?: string;
  stderr?: string;
  error?: string;
}

export interface ComparisonMetrics {
  latencyWinner: "h2" | "h3" | "tie";
  throughputWinner: "h2" | "h3" | "tie";
  p50Diff: number;
  p99Diff: number;
  rpsDiff: number;
  latencyImprovement: number;
}

export interface ProtocolResult {
  summary: BenchmarkSummary;
  protocol: "HTTP/2" | "HTTP/3";
}

export interface ComparisonResult {
  success: boolean;
  h2: ProtocolResult;
  h3: ProtocolResult;
  comparison: ComparisonMetrics;
  error?: string;
}

export type Scenario =
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

export type Protocol = "h2" | "h3";

export interface ProtocolOption {
  value: Protocol;
  label: string;
  color: string;
}

export interface ScenarioOption {
  value: Scenario;
  label: string;
  subtitle: string;
  Icon: ComponentType;
}


  export interface ProtocolOption {
    value: 'h2' | 'h3';
    label: string;
    color: string;
  }