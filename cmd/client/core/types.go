package core

import (
	"sync/atomic"
	"time"
)

// Record stores a single request sample
type Record struct {
	TsUnixNS  int64 // Timestamp in Unix nanoseconds
	LatencyNS int64 // Latency in nanoseconds
	OK        bool  // Request success status
}

// Summary contains aggregated benchmark statistics
type Summary struct {
	Samples   int       // Total samples
	OKRatePct float64   // Success rate percentage
	RPS       float64   // Requests per second
	DurationS float64   // Total duration in seconds
	P50ms     float64   // 50th percentile latency
	P90ms     float64   // 90th percentile latency
	P95ms     float64   // 95th percentile latency
	P99ms     float64   // 99th percentile latency
	Meanms    float64   // Mean latency
	Minms     float64   // Minimum latency
	Maxms     float64   // Maximum latency
	CDF_X_ms  []float64 // CDF X-axis (latency values)
	CDF_Y     []float64 // CDF Y-axis (cumulative probability)
	THR_Ts    []int64   // Throughput timestamps
	THR_Val   []int     // Throughput values per second
}

// Counters holds atomic counters for tracking request stats
type Counters struct {
	TotalOK     atomic.Uint64
	TotalErr    atomic.Uint64
	ErrLogCount atomic.Int64
}

// NewCounters creates a new Counters instance
func NewCounters() *Counters {
	return &Counters{}
}

// BaseConfig contains common configuration for all benchmark clients
type BaseConfig struct {
	Addr     string        // Server address
	UseH3    bool          // Use HTTP/3 (true) or HTTP/2 (false)
	Insecure bool          // Skip TLS verification
	Clients  int           // Number of concurrent clients
	Payload  int           // Request payload size in bytes
	Duration time.Duration // Test duration
	CSVPath  string        // Output CSV path
	HTMLPath string        // Output HTML path
	Label    string        // Dashboard label
	Quiet    bool          // Suppress progress logs
	Verbose  bool          // Enable verbose logging
}

// RequestInfo contains information about a single request for logging
type RequestInfo struct {
	ID        int64
	Method    string
	Addr      string
	Headers   map[string]string
	StartTime time.Time
}

// ResponseInfo contains information about a response for logging
type ResponseInfo struct {
	RequestID int64
	OK        bool
	Latency   time.Duration
	Size      int
	Error     error
}
