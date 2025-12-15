import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { spawn } from 'child_process';
import { parse } from 'csv-parse/sync';
import path from 'path';
import fs from 'fs/promises';
import os from 'os';

interface BenchmarkRequest {
	scenario: string;
	protocol: string;
}

interface BenchmarkResult {
	summary: {
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
	};
	rawData?: any[];
}

// Scenario mapping to binary names (pre-built in Docker)
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

export const POST: RequestHandler = async ({ request }) => {
	try {
		const body: BenchmarkRequest = await request.json();

		// Validasi
		const validScenarios = Object.keys(SCENARIO_MAP);
		if (!validScenarios.includes(body.scenario)) {
			return json({ error: `Invalid scenario. Valid: ${validScenarios.join(', ')}` }, { status: 400 });
		}

		if (!['h2', 'h3'].includes(body.protocol)) {
			return json({ error: 'Invalid protocol' }, { status: 400 });
		}

		// Build command
		const h3Flag = body.protocol === 'h3';
		const h2Addr = process.env.H2_ADDR || 'https://localhost:8444';
		const h3Addr = process.env.H3_ADDR || 'https://localhost:8443';
		const addr = h3Flag ? h3Addr : h2Addr;

		// Temporary file paths
		const tmpDir = os.tmpdir();
		const csvPath = path.join(tmpDir, `bench-${Date.now()}.csv`);

		// Get scenario binary name
		const binaryName = SCENARIO_MAP[body.scenario];

		const args = [
			'--addr', addr,
			'--h3=' + h3Flag,
			'--csv', csvPath,
			'--quiet'
		];

		// Add mode flag for cold_vs_resumed scenario
		if (body.scenario === 'cold_vs_resumed') {
			// Default to cold mode, could be parameterized later
			args.push('--mode', 'cold');
		}

		// Execute benchmark
		const result = await runBenchmark(binaryName, args);

		// Parse CSV results
		let summary: BenchmarkSummary = {
			Samples: 0,
			OKRatePct: 0,
			RPS: 0,
			DurationS: 0,
			P50ms: 0,
			P90ms: 0,
			P95ms: 0,
			P99ms: 0,
			Meanms: 0,
			Minms: 0,
			Maxms: 0
		};
		try {
			const csvContent = await fs.readFile(csvPath, 'utf-8');
			const records = parse(csvContent, {
				columns: true,
				skip_empty_lines: true
			}) as CSVRecord[];

			// Calculate summary
			summary = calculateSummary(records);

			// Cleanup
			await fs.unlink(csvPath).catch(() => {});
		} catch (err) {
			console.error('CSV parse error:', err);
		}

		return json({
			success: true,
			summary,
			stdout: result.stdout,
			stderr: result.stderr
		});
	} catch (error: any) {
		console.error('Benchmark error:', error);
		return json({ error: error.message || 'Benchmark failed' }, { status: 500 });
	}
};

function runBenchmark(binaryName: string, args: string[]): Promise<{ stdout: string; stderr: string }> {
	return new Promise((resolve, reject) => {
		const proc = spawn(binaryName, args, {
			env: { ...process.env, GOMAXPROCS: '8' }
		});

		let stdout = '';
		let stderr = '';

		proc.stdout?.on('data', (data) => {
			stdout += data.toString();
		});

		proc.stderr?.on('data', (data) => {
			stderr += data.toString();
		});

		proc.on('close', (code) => {
			if (code !== 0) {
				reject(new Error(`Process exited with code ${code}: ${stderr}`));
			} else {
				resolve({ stdout, stderr });
			}
		});

		proc.on('error', (err) => {
			reject(err);
		});
	});
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

function calculateSummary(records: CSVRecord[]): BenchmarkSummary {
	if (records.length === 0) {
		return {
			Samples: 0,
			OKRatePct: 0,
			RPS: 0,
			DurationS: 0,
			P50ms: 0,
			P90ms: 0,
			P95ms: 0,
			P99ms: 0,
			Meanms: 0,
			Minms: 0,
			Maxms: 0
		};
	}

	const latencies: number[] = records.map(r => parseFloat(r.latency_ns) / 1e6); // Convert to ms
	const timestamps: number[] = records.map(r => parseInt(r.ts_unix_ns));
	const okCount = records.filter(r => r.ok === 'true').length;

	latencies.sort((a, b) => a - b);

	const percentile = (p: number): number => {
		const pos = p * (latencies.length - 1);
		const i = Math.floor(pos);
		const f = pos - i;
		if (i + 1 < latencies.length) {
			return latencies[i] + f * (latencies[i + 1] - latencies[i]);
		}
		return latencies[i];
	};

	// Use iterative approach instead of spread operator to avoid stack overflow
	let minTS = timestamps[0];
	let maxTS = timestamps[0];
	for (let i = 1; i < timestamps.length; i++) {
		if (timestamps[i] < minTS) minTS = timestamps[i];
		if (timestamps[i] > maxTS) maxTS = timestamps[i];
	}
	const durationS = (maxTS - minTS) / 1e9;

	// Calculate mean iteratively
	let sum = 0;
	for (let i = 0; i < latencies.length; i++) {
		sum += latencies[i];
	}
	const meanMs = sum / latencies.length;

	// Min and max are already available from sorted array
	const minLatency = latencies[0];
	const maxLatency = latencies[latencies.length - 1];

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
		Minms: round6(minLatency),
		Maxms: round6(maxLatency)
	};
}

function round6(x: number): number {
	return Math.round(x * 1e6) / 1e6;
}

function testing() {
	console.log("testing")
}
