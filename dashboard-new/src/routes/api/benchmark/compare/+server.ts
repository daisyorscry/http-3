import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { spawn } from 'child_process';
import { parse } from 'csv-parse/sync';
import path from 'path';
import fs from 'fs/promises';
import os from 'os';
import { insertRunWithResults } from '$lib/server/db';

/* =========================
   Types
========================= */

// Scenario mapping to binary names (must match SCENARIO_MAP in +server.ts)
const SCENARIO_MAP: Record<string, string> = {
  'baseline': 'bench-client',
  'burst': 'bench-burst',
  'cold_vs_resumed': 'bench-coldstart',
  'parallel_streams': 'bench-parallel',
  'header_bloat': 'bench-header-bloat',
  'uplink_loss': 'bench-uplink',
  'connection_churn': 'bench-churn',
  'nat_rebinding': 'bench-migration',
  'mixed_load': 'bench-mixed',
  'stress_test': 'bench-stress'
};

interface BenchmarkRequest {
  // scenario = scenario name (e.g., "baseline", "burst", "stress_test")
  scenario: string;
  // uiScenario = same as scenario (kept for backward compatibility)
  uiScenario?: string;
}

interface CSVRecord {
  latency_ns: string;
  ts_unix_ns: string;
  ok: string;
}

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

interface ProtocolResult {
  summary: BenchmarkSummary;
  protocol: 'HTTP/2' | 'HTTP/3';
}

interface ComparisonMetrics {
  latencyWinner: 'h2' | 'h3' | 'tie';
  throughputWinner: 'h2' | 'h3' | 'tie';
  p50Diff: number;
  p99Diff: number;
  rpsDiff: number;
  latencyImprovement: number;
}

interface ComparisonResult {
  h2: ProtocolResult;
  h3: ProtocolResult;
  comparison: ComparisonMetrics;
}

/* =========================
   POST (non-stream) — run compare once
========================= */

export const POST: RequestHandler = async ({ request }) => {
  const requestId = Date.now().toString(36);

  try {
    const body: BenchmarkRequest = await request.json();

    console.log(`\n${'='.repeat(60)}`);
    console.log(`[${new Date().toISOString()}] [Compare-${requestId}] ====== NEW COMPARISON REQUEST ======`);
    console.log(`[Compare-${requestId}] Request body:`, JSON.stringify(body, null, 2));
    console.log(`${'='.repeat(60)}\n`);

    const validScenarios = Object.keys(SCENARIO_MAP);
    if (!validScenarios.includes(body.scenario)) {
      console.log(`[Compare-${requestId}] Invalid scenario: ${body.scenario}`);
      return json({ error: `Invalid scenario. Valid: ${validScenarios.join(', ')}` }, { status: 400 });
    }

    console.log(`[Compare-${requestId}] >>> Phase 1: Running HTTP/2 benchmark...`);
    const h2Start = Date.now();
    const h2Result = await runSingleBenchmark(body, false);
    console.log(`[Compare-${requestId}] <<< HTTP/2 completed in ${Date.now() - h2Start}ms`);

    console.log(`[Compare-${requestId}] >>> Phase 2: Running HTTP/3 benchmark...`);
    const h3Start = Date.now();
    const h3Result = await runSingleBenchmark(body, true);
    console.log(`[Compare-${requestId}] <<< HTTP/3 completed in ${Date.now() - h3Start}ms`);

    const comparison = calculateComparison(h2Result, h3Result);

    console.log(`\n[Compare-${requestId}] ====== COMPARISON RESULTS ======`);
    console.log(`[Compare-${requestId}] Latency winner: ${comparison.latencyWinner.toUpperCase()}`);
    console.log(`[Compare-${requestId}] Throughput winner: ${comparison.throughputWinner.toUpperCase()}`);
    console.log(`[Compare-${requestId}] P50 diff: ${comparison.p50Diff.toFixed(2)}%`);
    console.log(`[Compare-${requestId}] P99 diff: ${comparison.p99Diff.toFixed(2)}%`);
    console.log(`[Compare-${requestId}] RPS diff: ${comparison.rpsDiff.toFixed(2)}%`);
    console.log(`[Compare-${requestId}] ================================\n`);

    const result: ComparisonResult = {
      h2: { summary: h2Result, protocol: 'HTTP/2' },
      h3: { summary: h3Result, protocol: 'HTTP/3' },
      comparison
    };

    try {
      await insertRunWithResults({
        uiScenario: body.uiScenario || body.scenario,
        backendScenario: body.scenario,
        config: {}, // All configs are now fixed in Go clients
        h2: h2Result,
        h3: h3Result
      });
      console.log(`[Compare-${requestId}] Results persisted to DB`);
    } catch (e) {
      console.warn(`[Compare-${requestId}] Persist failed:`, e);
    }

    console.log(`[Compare-${requestId}] ====== REQUEST COMPLETE ======\n`);
    return json({ success: true, ...result });
  } catch (error: any) {
    console.error(`[Compare-${requestId}] FATAL ERROR:`, error);
    return json({ error: error.message || 'Comparison failed' }, { status: 500 });
  }
};

/* =========================
   Logging helpers
========================= */

function logInfo(tag: string, message: string, data?: Record<string, unknown>) {
  const timestamp = new Date().toISOString();
  if (data) {
    console.log(`[${timestamp}] [${tag}] ${message}`, JSON.stringify(data));
  } else {
    console.log(`[${timestamp}] [${tag}] ${message}`);
  }
}

function logError(tag: string, message: string, error?: unknown) {
  const timestamp = new Date().toISOString();
  console.error(`[${timestamp}] [${tag}] ERROR: ${message}`, error);
}

/* =========================
   Core runner — uses built binary based on scenario
========================= */

async function runSingleBenchmark(
  body: BenchmarkRequest,
  useH3: boolean
): Promise<BenchmarkSummary> {
  const protocol = useH3 ? 'H3' : 'H2';
  const h2Addr = process.env.H2_ADDR || 'https://localhost:8444';
  const h3Addr = process.env.H3_ADDR || 'https://localhost:8443';
  const addr = useH3 ? h3Addr : h2Addr;

  const tmpDir = os.tmpdir();
  const csvPath = path.join(tmpDir, `bench-${useH3 ? 'h3' : 'h2'}-${Date.now()}.csv`);

  // Get binary name from mapping
  const binaryName = SCENARIO_MAP[body.scenario];
  if (!binaryName) {
    throw new Error(`Unknown scenario: ${body.scenario}`);
  }

  const args = [
    '--addr', addr,
    '--h3=' + useH3,
    '--csv', csvPath,
    '--quiet'
  ];

  // Add mode flag for cold_vs_resumed scenario
  if (body.scenario === 'cold_vs_resumed') {
    args.push('--mode', 'cold');
  }

  // Log benchmark start
  logInfo(`Benchmark-${protocol}`, '>>> Starting benchmark', {
    scenario: body.scenario,
    binaryName,
    protocol,
    addr,
    command: `${binaryName} ${args.join(' ')}`
  });

  const startTime = Date.now();
  await runBenchmarkBinary(binaryName, args, protocol);
  const elapsed = Date.now() - startTime;

  logInfo(`Benchmark-${protocol}`, `<<< Benchmark completed in ${elapsed}ms`);

  try {
    const csvContent = await fs.readFile(csvPath, 'utf-8');
    const records = parse(csvContent, { columns: true, skip_empty_lines: true }) as CSVRecord[];
    const summary = calculateSummary(records);

    logInfo(`Benchmark-${protocol}`, 'Results summary', {
      samples: summary.Samples,
      okRate: `${summary.OKRatePct.toFixed(2)}%`,
      rps: summary.RPS.toFixed(2),
      p50ms: summary.P50ms.toFixed(3),
      p99ms: summary.P99ms.toFixed(3)
    });

    await fs.unlink(csvPath).catch(() => {});
    return summary;
  } catch (err) {
    logError(`Benchmark-${protocol}`, 'CSV parse error', err);
    throw new Error(`Failed to parse benchmark results for ${protocol}`);
  }
}

function runBenchmarkBinary(binaryName: string, args: string[], protocol: string): Promise<void> {
  return new Promise((resolve, reject) => {
    logInfo(`Process-${protocol}`, `Spawning: ${binaryName} ${args.join(' ')}`);

    const proc = spawn(binaryName, args, {
      env: { ...process.env, GOMAXPROCS: '8' }
    });

    let stdout = '';
    let stderr = '';

    proc.stdout?.on('data', (d) => {
      stdout += d.toString();
      // Log stdout in real-time for debugging
      const lines = d.toString().trim().split('\n');
      lines.forEach((line: string) => {
        if (line.trim()) {
          logInfo(`Client-${protocol}`, `[stdout] ${line}`);
        }
      });
    });

    proc.stderr?.on('data', (d) => {
      stderr += d.toString();
      const lines = d.toString().trim().split('\n');
      lines.forEach((line: string) => {
        if (line.trim()) {
          logInfo(`Client-${protocol}`, `[stderr] ${line}`);
        }
      });
    });

    proc.on('close', (code) => {
      if (code !== 0) {
        logError(`Process-${protocol}`, `Process exited with code ${code}`, stderr);
        reject(new Error(`${binaryName} exit ${code}: ${stderr}`));
      } else {
        logInfo(`Process-${protocol}`, `Process exited successfully`);
        resolve();
      }
    });

    proc.on('error', (err) => {
      logError(`Process-${protocol}`, 'Process error', err);
      reject(err);
    });
  });
}

/* =========================
   CSV → Summary
========================= */

function calculateSummary(records: CSVRecord[]): BenchmarkSummary {
  if (records.length === 0) throw new Error('No benchmark data received');

  const latencies: number[] = records.map((r) => parseFloat(r.latency_ns) / 1e6);
  const timestamps: number[] = records.map((r) => parseInt(r.ts_unix_ns, 10));
  const okCount = records.filter((r) => r.ok === 'true').length;

  latencies.sort((a, b) => a - b);

  const percentile = (p: number): number => {
    const pos = p * (latencies.length - 1);
    const i = Math.floor(pos);
    const f = pos - i;
    if (i + 1 < latencies.length) return latencies[i] + f * (latencies[i + 1] - latencies[i]);
    return latencies[i];
  };

  let minTS = timestamps[0];
  let maxTS = timestamps[0];
  for (let i = 1; i < timestamps.length; i++) {
    if (timestamps[i] < minTS) minTS = timestamps[i];
    if (timestamps[i] > maxTS) maxTS = timestamps[i];
  }
  const durationS = (maxTS - minTS) / 1e9;

  let sum = 0;
  for (let i = 0; i < latencies.length; i++) sum += latencies[i];
  const meanMs = sum / latencies.length;

  return {
    Samples: records.length,
    OKRatePct: (okCount / records.length) * 100,
    RPS: records.length / (durationS || 1),
    DurationS: durationS,
    P50ms: round6(percentile(0.5)),
    P90ms: round6(percentile(0.9)),
    P95ms: round6(percentile(0.95)),
    P99ms: round6(percentile(0.99)),
    Meanms: round6(meanMs),
    Minms: round6(latencies[0]),
    Maxms: round6(latencies[latencies.length - 1])
  };
}

function round6(x: number): number {
  return Math.round(x * 1e6) / 1e6;
}

function calculateComparison(h2: BenchmarkSummary, h3: BenchmarkSummary): ComparisonMetrics {
  const p50Diff = ((h2.P50ms - h3.P50ms) / h2.P50ms) * 100;
  const p99Diff = ((h2.P99ms - h3.P99ms) / h2.P99ms) * 100;
  const rpsDiff = ((h3.RPS - h2.RPS) / h2.RPS) * 100;

  const latencyWinner: 'h2' | 'h3' | 'tie' = h3.P50ms < h2.P50ms ? 'h3' : h2.P50ms < h3.P50ms ? 'h2' : 'tie';
  const throughputWinner: 'h2' | 'h3' | 'tie' = h3.RPS > h2.RPS ? 'h3' : h2.RPS > h3.RPS ? 'h2' : 'tie';

  const latencyImprovement = (p50Diff + p99Diff) / 2;

  return {
    latencyWinner,
    throughputWinner,
    p50Diff: round6(p50Diff),
    p99Diff: round6(p99Diff),
    rpsDiff: round6(rpsDiff),
    latencyImprovement: round6(latencyImprovement)
  };
}

/* =========================
   SSE helpers
========================= */

function sseEvent(name: string, data: unknown): string {
  return `event: ${name}\n` + `data: ${JSON.stringify(data)}\n\n`;
}

/* =========================
   GET (SSE) — run compare with live logs
========================= */

export const GET: RequestHandler = async ({ url }) => {
  const scenario = url.searchParams.get('scenario') || 'baseline';
  const uiScenario = url.searchParams.get('uiScenario') || scenario;
  const startServers = /^(1|true|yes)$/i.test(url.searchParams.get('startServers') || '');

  const validScenarios = Object.keys(SCENARIO_MAP);
  if (!validScenarios.includes(scenario)) {
    return new Response(`Invalid scenario. Valid: ${validScenarios.join(', ')}`, { status: 400 });
  }

  const stream = new ReadableStream<Uint8Array>({
    start: async (controller) => {
      const encoder = new TextEncoder();
      let closed = false;
      const procs = new Set<import('child_process').ChildProcess>();

      const send = (name: string, data: unknown) => {
        if (closed) return;
        try {
          controller.enqueue(encoder.encode(sseEvent(name, data)));
        } catch {
          // ignore once closed
        }
      };

      const attachProc = (
        proc: import('child_process').ChildProcess,
        meta: { protocol?: 'h2' | 'h3'; actor?: 'server' | 'client' }
      ) => {
        procs.add(proc);
        const onStdout = (d: Buffer) => send('log', { ...meta, stream: 'stdout', line: d.toString() });
        const onStderr = (d: Buffer) => send('log', { ...meta, stream: 'stderr', line: d.toString() });
        const onClose = (code: number | null) => {
          send('phase', { ...meta, state: 'exit', code });
          proc.stdout?.off('data', onStdout);
          proc.stderr?.off('data', onStderr);
          proc.off('close', onClose);
          procs.delete(proc);
        };
        proc.stdout?.on('data', onStdout);
        proc.stderr?.on('data', onStderr);
        proc.once('close', onClose);
      };

      const killProc = (p: import('child_process').ChildProcess) => {
        try { p.kill('SIGINT'); } catch {}
      };

      const cleanup = () => {
        if (closed) return;
        closed = true;
        for (const p of Array.from(procs)) {
          try {
            p.stdout?.removeAllListeners('data');
            p.stderr?.removeAllListeners('data');
            p.removeAllListeners('close');
            killProc(p);
          } catch {}
          procs.delete(p);
        }
        try { controller.close(); } catch {}
      };

      try {
        // Returns { binaryName, args } based on scenario
        // All configs are now FIXED in Go clients
        const buildCommand = (useH3: boolean, csvPath: string) => {
          const addr = useH3
            ? (process.env.H3_ADDR || 'https://localhost:8443')
            : (process.env.H2_ADDR || 'https://localhost:8444');

          const binaryName = SCENARIO_MAP[scenario];
          const args = [
            '--addr', addr,
            '--h3=' + useH3,
            '--csv', csvPath
          ];

          // Add mode flag for cold_vs_resumed scenario
          if (scenario === 'cold_vs_resumed') {
            args.push('--mode', 'cold');
          }

          return { binaryName, args };
        };

        const tmpDir = os.tmpdir();
        const csvH2 = path.join(tmpDir, `bench-h2-${Date.now()}.csv`);
        const csvH3 = path.join(tmpDir, `bench-h3-${Date.now() + 1}.csv`);
        const cmdH2 = buildCommand(false, csvH2);
        const cmdH3 = buildCommand(true, csvH3);

        send('info', {
          uiScenario,
          scenario,
          cmdH2: cmdH2.binaryName + ' ' + cmdH2.args.join(' '),
          cmdH3: cmdH3.binaryName + ' ' + cmdH3.args.join(' ')
        });

        // Opsional: startServers=true → start server binaries (bukan go run)
        let srvH2: import('child_process').ChildProcess | null = null;
        let srvH3: import('child_process').ChildProcess | null = null;

        if (startServers) {
          srvH2 = spawn('bench-server-h2', ['--addr', ':8444', '--cert', 'cert/dev.crt', '--key', 'cert/dev.key'], { env: { ...process.env } });
          srvH3 = spawn('bench-server-h3', ['--addr', ':8443', '--cert', 'cert/dev.crt', '--key', 'cert/dev.key'], { env: { ...process.env } });
          attachProc(srvH2, { protocol: 'h2', actor: 'server' });
          attachProc(srvH3, { protocol: 'h3', actor: 'server' });
          send('phase', { protocol: 'h2', actor: 'server', state: 'start' });
          send('phase', { protocol: 'h3', actor: 'server', state: 'start' });
          await new Promise((r) => setTimeout(r, 500));
        }

        const runOne = async (useH3: boolean, cmd: { binaryName: string; args: string[] }) =>
          new Promise<void>((resolve, reject) => {
            const proc = spawn(cmd.binaryName, cmd.args, { env: { ...process.env, GOMAXPROCS: '8' } });
            attachProc(proc, { protocol: useH3 ? 'h3' : 'h2', actor: 'client' });
            send('phase', { protocol: useH3 ? 'h3' : 'h2', state: 'start' });
            proc.once('close', (code) => code !== 0 ? reject(new Error(`${cmd.binaryName} ${useH3 ? 'h3' : 'h2'} exited ${code}`)) : resolve());
            proc.once('error', (err) => reject(err));
          });

        await runOne(false, cmdH2);
        await runOne(true, cmdH3);

        const readAndParse = async (p: string) => {
          const content = await fs.readFile(p, 'utf-8');
          const records = parse(content, { columns: true, skip_empty_lines: true }) as CSVRecord[];
          await fs.unlink(p).catch(() => {});
          return calculateSummary(records);
        };

        const h2 = await readAndParse(csvH2);
        const h3 = await readAndParse(csvH3);

        const p50Diff = ((h2.P50ms - h3.P50ms) / h2.P50ms) * 100;
        const p99Diff = ((h2.P99ms - h3.P99ms) / h2.P99ms) * 100;
        const rpsDiff = ((h3.RPS - h2.RPS) / h2.RPS) * 100;
        const latencyWinner = h3.P50ms < h2.P50ms ? 'h3' : h2.P50ms < h3.P50ms ? 'h2' : 'tie';
        const throughputWinner = h3.RPS > h2.RPS ? 'h3' : h2.RPS > h3.RPS ? 'h2' : 'tie';
        const latencyImprovement = (p50Diff + p99Diff) / 2;

        // Persist results to database
        try {
          await insertRunWithResults({
            uiScenario: uiScenario,
            backendScenario: scenario,
            config: {}, // All configs are now fixed in Go clients
            h2: h2,
            h3: h3
          });
          console.log(`[SSE] Results persisted to DB for scenario: ${uiScenario}`);
        } catch (e) {
          console.warn(`[SSE] Persist failed:`, e);
        }

        send('result', {
          h2: { summary: h2, protocol: 'HTTP/2' },
          h3: { summary: h3, protocol: 'HTTP/3' },
          comparison: {
            latencyWinner,
            throughputWinner,
            p50Diff: Math.round(p50Diff * 1e6) / 1e6,
            p99Diff: Math.round(p99Diff * 1e6) / 1e6,
            rpsDiff: Math.round(rpsDiff * 1e6) / 1e6,
            latencyImprovement: Math.round(latencyImprovement * 1e6) / 1e6
          }
        });

        if (startServers) {
          try { srvH2 && killProc(srvH2); } catch {}
          try { srvH3 && killProc(srvH3); } catch {}
          send('phase', { protocol: 'h2', actor: 'server', state: 'exit' });
          send('phase', { protocol: 'h3', actor: 'server', state: 'exit' });
        }

        send('end', { ok: true });
        cleanup();
      } catch (err: any) {
        send('error', { message: err?.message || String(err) });
        cleanup();
      }
    },
    cancel: () => {
      // client disconnect → cleanup akan dijalankan via flag di atas
    }
  });

  return new Response(stream, {
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache, no-transform',
      'Connection': 'keep-alive'
    }
  });
};
