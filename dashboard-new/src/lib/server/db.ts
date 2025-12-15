// server-only guard
if (!import.meta.env.SSR) throw new Error('db.ts must run on server only');

import * as fs from 'node:fs';
import * as path from 'node:path';

type Summary = {
  Samples: number; OKRatePct: number; RPS: number; DurationS: number;
  P50ms: number; P90ms: number; P95ms: number; P99ms: number;
  Meanms: number; Minms: number; Maxms: number;
};

export type RunRow = {
  id: number;
  ui_scenario: string;
  backend_scenario: string;
  clients: number | null;
  payload: number | null;
  duration: number | null;
  rps: number | null;
  period: number | null;
  jitter: number | null;
  ramp: string | null;
  created_at: string;
};

export type ResultRow = {
  id: number;
  run_id: number;
  protocol: 'h2' | 'h3';
  samples: number;
  ok_rate: number;
  rps: number;
  duration_s: number;
  p50ms: number;
  p90ms: number;
  p95ms: number;
  p99ms: number;
  meanms: number;
  minms: number;
  maxms: number;
};

export type RunWithPair = RunRow & { h2?: ResultRow; h3?: ResultRow };

let mikroOk = true;
let ormPromise: Promise<any> | null = null;
let initialized = false;

function getDbPath() {
  const dataDir = path.resolve(process.cwd(), 'data');
  fs.mkdirSync(dataDir, { recursive: true });
  return path.join(dataDir, 'bench.db');
}

async function getConn() {
  if (!mikroOk) return null;
  try {
    if (!ormPromise) {
      const dbPath = getDbPath();
      const [{ MikroORM }, { BetterSqliteDriver }] = await Promise.all([
        import('@mikro-orm/core'),
        import('@mikro-orm/better-sqlite'),
      ]);

      ormPromise = MikroORM.init({
        driver: BetterSqliteDriver as any,
        dbName: dbPath,
        debug: false,
        allowGlobalContext: true,

        entities: [],
        discovery: { warnWhenNoEntities: false },
      });
    }

    const orm = await ormPromise;
    const conn = orm.em.getConnection();

    if (!initialized) {
      initialized = true;
      await conn.execute('PRAGMA foreign_keys = ON');
      await conn.execute('PRAGMA journal_mode = WAL');

      await conn.execute(`CREATE TABLE IF NOT EXISTS runs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        ui_scenario TEXT NOT NULL,
        backend_scenario TEXT NOT NULL,
        clients INTEGER,
        payload INTEGER,
        duration INTEGER,
        rps INTEGER,
        period INTEGER,
        jitter INTEGER,
        ramp TEXT,
        created_at DATETIME DEFAULT (datetime('now'))
      )`);

      await conn.execute(`CREATE TABLE IF NOT EXISTS results (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        run_id INTEGER NOT NULL,
        protocol TEXT NOT NULL CHECK(protocol IN ('h2','h3')),
        samples INTEGER,
        ok_rate REAL,
        rps REAL,
        duration_s REAL,
        p50ms REAL,
        p90ms REAL,
        p95ms REAL,
        p99ms REAL,
        meanms REAL,
        minms REAL,
        maxms REAL,
        FOREIGN KEY(run_id) REFERENCES runs(id) ON DELETE CASCADE
      )`);

      await conn.execute('CREATE INDEX IF NOT EXISTS idx_results_run ON results(run_id)');
      await conn.execute('CREATE INDEX IF NOT EXISTS idx_runs_scenario ON runs(ui_scenario, created_at)');
    }

    return conn;
  } catch (e) {
    mikroOk = false;
    console.error('[DB] SQLite init failed:', e);
    return null;
  }
}

export async function insertRunWithResults(params: {
  uiScenario: string;
  backendScenario: string;
  config: { clients?: number; payload?: number; duration?: number; rps?: number; period?: number; jitter?: number; ramp?: string };
  h2?: Summary;
  h3?: Summary;
}): Promise<number | null> {
  const conn = await getConn();
  if (!conn) return null;

  await conn.execute(
    'INSERT INTO runs(ui_scenario, backend_scenario, clients, payload, duration, rps, period, jitter, ramp) VALUES (?,?,?,?,?,?,?,?,?)',
    [
      params.uiScenario,
      params.backendScenario,
      params.config.clients ?? null,
      params.config.payload ?? null,
      params.config.duration ?? null,
      params.config.rps ?? null,
      params.config.period ?? null,
      params.config.jitter ?? null,
      params.config.ramp ?? null,
    ]
  );

  const row = (await conn.execute('SELECT last_insert_rowid() AS id')) as Array<{ id: number }>;
  const runId = row?.[0]?.id;
  if (!runId) return null;

  const push = async (proto: 'h2' | 'h3', s?: Summary) => {
    if (!s) return;
    await conn.execute(
      'INSERT INTO results(run_id, protocol, samples, ok_rate, rps, duration_s, p50ms, p90ms, p95ms, p99ms, meanms, minms, maxms) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)',
      [runId, proto, s.Samples, s.OKRatePct, s.RPS, s.DurationS, s.P50ms, s.P90ms, s.P95ms, s.P99ms, s.Meanms, s.Minms, s.Maxms]
    );
  };
  await push('h2', params.h2);
  await push('h3', params.h3);
  return runId ?? null;
}

export async function listRunsByScenario(uiScenario: string): Promise<RunWithPair[]> {
  const conn = await getConn();
  if (!conn) return [];

  const runs = (await conn.execute(
    'SELECT * FROM runs WHERE ui_scenario = ? ORDER BY created_at ASC, id ASC',
    [uiScenario]
  )) as RunRow[];

  const out: RunWithPair[] = [];
  for (const r of runs) {
    const pair: RunWithPair = { ...(r as any) };
    const rows = (await conn.execute('SELECT * FROM results WHERE run_id = ?', [r.id])) as ResultRow[];
    for (const row of rows) {
      if ((row as any).protocol === 'h2') pair.h2 = row;
      else if ((row as any).protocol === 'h3') pair.h3 = row;
    }
    out.push(pair);
  }
  return out;
}

export async function listAllScenarios(): Promise<string[]> {
  const conn = await getConn();
  if (!conn) return [];
  const rows = (await conn.execute('SELECT DISTINCT ui_scenario as s FROM runs ORDER BY ui_scenario ASC')) as any[];
  return rows.map((r) => r.s as string);
}

export async function listAllRunsGrouped(): Promise<Record<string, RunWithPair[]>> {
  const scenarios = await listAllScenarios();
  const grouped: Record<string, RunWithPair[]> = {};
  for (const s of scenarios) {
    grouped[s] = await listRunsByScenario(s);
  }
  return grouped;
}

function mean(arr: number[]): number {
  if (arr.length === 0) return 0;
  let s = 0;
  for (const x of arr) s += x;
  return s / arr.length;
}
function stddev(arr: number[]): number {
  if (arr.length <= 1) return 0;
  const m = mean(arr);
  let s = 0;
  for (const x of arr) {
    const d = x - m;
    s += d * d;
  }
  return Math.sqrt(s / (arr.length - 1));
}

export async function scenarioStability(uiScenario: string) {
  const runs = await listRunsByScenario(uiScenario);
  if (runs.length === 0) {
    return { runs: [], earliest: null, latest: null, stability: null } as const;
  }
  const earliest = runs[0];
  const latest = runs[runs.length - 1];

  const h2P50 = runs.filter(r => r.h2).map(r => r.h2!.p50ms);
  const h3P50 = runs.filter(r => r.h3).map(r => r.h3!.p50ms);
  const h2RPS = runs.filter(r => r.h2).map(r => r.h2!.rps);
  const h3RPS = runs.filter(r => r.h3).map(r => r.h3!.rps);

  const h2P50Std = stddev(h2P50);
  const h3P50Std = stddev(h3P50);
  const h2P50Mean = mean(h2P50) || 1e-9;
  const h3P50Mean = mean(h3P50) || 1e-9;
  const h2P50CV = h2P50Std / h2P50Mean;
  const h3P50CV = h3P50Std / h3P50Mean;

  const h2RPSStd = stddev(h2RPS);
  const h3RPSStd = stddev(h3RPS);
  const h2RPSMean = mean(h2RPS) || 1e-9;
  const h3RPSMean = mean(h3RPS) || 1e-9;
  const h2RPSCV = h2RPSStd / h2RPSMean;
  const h3RPSCV = h3RPSStd / h3RPSMean;

  const h2Score = (h2P50CV + h2RPSCV) / 2;
  const h3Score = (h3P50CV + h3RPSCV) / 2;

  const stability = {
    h2: { p50CV: h2P50CV, rpsCV: h2RPSCV, score: h2Score },
    h3: { p50CV: h3P50CV, rpsCV: h3RPSCV, score: h3Score },
    winner: h2Score < h3Score ? 'h2' : h3Score < h2Score ? 'h3' : 'tie',
  } as const;

  return { runs, earliest, latest, stability } as const;
}

export async function closeDb() {
  if (!ormPromise) return;
  const orm = await ormPromise;
  await orm.close(true);
}
