package main

// rec merekam satu sample hasil request
type rec struct {
	TsUnixNS  int64 // Timestamp dalam Unix nanoseconds
	LatencyNS int64 // Latency dalam nanoseconds
	OK        bool  // Status sukses/gagal request
}

// job adalah token kosong untuk job queue (dispatcher → worker)
type job struct{} // 1 token = 1 RPC

// stage mendefinisikan satu stage untuk ramping scenario
type stage struct {
	dur int // Durasi stage dalam detik
	rps int // Target RPS untuk stage ini
}

// Summary adalah ringkasan hasil benchmark untuk output
type Summary struct {
	Samples   int     // Total sample
	OKRatePct float64 // Persentase request sukses
	RPS       float64 // Requests per second
	DurationS float64 // Durasi total test (seconds)
	P50ms     float64 // Latency percentile 50
	P90ms     float64 // Latency percentile 90
	P95ms     float64 // Latency percentile 95
	P99ms     float64 // Latency percentile 99
	Meanms    float64 // Mean latency
	Minms     float64 // Min latency
	Maxms     float64 // Max latency
	CDF_X_ms  []float64 // Data CDF (X-axis: latency)
	CDF_Y     []float64 // Data CDF (Y-axis: cumulative prob)
	THR_Ts    []int64   // Throughput timestamps
	THR_Val   []int     // Throughput values per second
}
