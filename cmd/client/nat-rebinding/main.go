package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
	"h3-vs-h2-k6/cmd/client/core"
	"h3-vs-h2-k6/echo/v1/echov1connect"
)

// =====================================
// NAT REBINDING / IP ADDRESS MIGRATION
// =====================================
// Skenario untuk menguji migrasi koneksi saat terjadi perubahan
// IP address atau NAT rebinding (simulasi mobile handoff)
//
// Pattern:
// - Phase 1: Establish connection, send N requests
// - Phase 2: Close old connection, create new connection (simulate IP change)
// - Phase 3: Continue sending M requests on new connection
// - Repeat cycles
//
// Catatan:
// HTTP/3 memiliki connection migration yang seharusnya lebih baik
// menangani perubahan IP dibanding HTTP/2 yang harus rebuild connection
//
// Untuk simulasi real IP migration, gunakan:
// - Mobile device switching between WiFi/4G
// - VM migration tools
// - Network namespace switching (Linux)
//
// FIXED CONFIGURATIONS PER LEVEL:
// FIXED CONFIGURATION:
// Config: 1000 workers, 50 cycles, 1 req/phase, 500ms interval, 512B payload
// Total: 1000 * 50 * 1 * 2 phases = 100,000 requests
// =====================================

const (
	fixedWorkers          = 1000
	fixedCycles           = 50
	fixedRequestsPerPhase = 1
	fixedMigrationInterval = 1 * time.Second
	fixedPayload          = 512
)


func main() {
	// -------- Flags --------
	var (
		addr     = flag.String("addr", "https://localhost:8443", "server URL")
		useH3    = flag.Bool("h3", true, "use HTTP/3 (true) or HTTP/2 (false)")
		insecure = flag.Bool("insecure", true, "skip TLS verify (dev)")

		// Output only
		csvPath  = flag.String("csv", "", "write CSV after test")
		htmlPath = flag.String("html", "", "write HTML dashboard after test")
		label    = flag.String("label", "NAT Rebinding Benchmark", "dashboard title label")
		quiet    = flag.Bool("quiet", false, "suppress progress logs during test")
		verbose  = flag.Bool("verbose", false, "enable verbose request/response logging")
	)
	flag.Parse()


	// ---- Setup Logger ----
	logLevel := core.LogLevelNormal
	if *quiet {
		logLevel = core.LogLevelMinimal
	}
	if *verbose {
		logLevel = core.LogLevelVerbose
	}
	logger := core.NewLogger(logLevel)

	// ---- Startup logging ----
	cwd, _ := os.Getwd()
	logger.Startup("nat-rebinding", map[string]interface{}{
		"pid":                os.Getpid(),
		"cwd":                cwd,
		"addr":               *addr,
		"protocol":           core.ProtocolName(*useH3),
		
		"insecure":           *insecure,
		"workers":            fixedWorkers,
		"cycles":             fixedCycles,
		"requests_per_phase": fixedRequestsPerPhase,
		"migration_interval": fixedMigrationInterval,
		"payload":            fixedPayload,
		"total_requests":     fixedWorkers * fixedCycles * fixedRequestsPerPhase * 2, // 2 phases per cycle
	})

	// Migration simulation note
	logger.Info("NOTE: This simulates connection migration by forcing reconnection")
	logger.Info("HTTP/3 connection migration (real IP change) requires:")
	logger.Info("  - Mobile device switching WiFi/4G")
	logger.Info("  - Network namespace switching (Linux)")
	logger.Info("  - VM migration tools")
	logger.Info("This benchmark measures reconnection overhead as proxy for migration cost")

	// Absolutkan output path
	csvAbs := core.AbsOrEmpty(*csvPath, cwd)
	htmlAbs := core.AbsOrEmpty(*htmlPath, cwd)

	// Context & cancel
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Channels & counters
	latCh := make(chan core.Record, 1<<20)
	counters := core.NewCounters()
	var reqCounter atomic.Int64
	var migrationCount atomic.Int64

	if !*quiet {
		go core.ProgressPrinter(ctx, counters, logger)
	}

	// Collector goroutine
	var all []core.Record
	var mu sync.Mutex
	doneCol := make(chan struct{})
	go func() {
		defer close(doneCol)
		for r := range latCh {
			mu.Lock()
			all = append(all, r)
			mu.Unlock()
		}
	}()

	// Start benchmark
	start := time.Now()
	logger.Info("Starting NAT rebinding/migration benchmark...")

	// Start workers - each simulates migration cycles
	var wg sync.WaitGroup
	wg.Add(fixedWorkers)

	for i := 0; i < fixedWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()
			logger.Debug("Worker %d started", workerID)

			for cycle := 0; cycle < fixedCycles; cycle++ {
				select {
				case <-ctx.Done():
					logger.Debug("Worker %d cancelled at cycle %d", workerID, cycle)
					return
				default:
				}

				// PHASE 1: Create connection and send requests
				httpClient1, closer1 := newHTTPClient(*useH3, *insecure)
				client1 := echov1connect.NewEchoServiceClient(httpClient1, *addr)
				requestFn := core.SimpleRequest(fixedPayload)

				for req := 0; req < fixedRequestsPerPhase; req++ {
					select {
					case <-ctx.Done():
						closer1()
						return
					default:
					}

					reqID := reqCounter.Add(1)
					core.DoRequest(ctx, client1, latCh, counters, logger, reqID, requestFn)
				}

				// Simulate migration interval (network switch delay)
				time.Sleep(fixedMigrationInterval)

				// SIMULATE MIGRATION: Close old connection
				closer1()
				migrationCount.Add(1)

				// PHASE 2: Create NEW connection (simulate post-migration)
				httpClient2, closer2 := newHTTPClient(*useH3, *insecure)
				client2 := echov1connect.NewEchoServiceClient(httpClient2, *addr)

				for req := 0; req < fixedRequestsPerPhase; req++ {
					select {
					case <-ctx.Done():
						closer2()
						return
					default:
					}

					reqID := reqCounter.Add(1)
					core.DoRequest(ctx, client2, latCh, counters, logger, reqID, requestFn)
				}

				// Close connection before next cycle
				closer2()
			}

			logger.Debug("Worker %d completed all %d cycles", workerID, fixedCycles)
		}(i)
	}

	// Wait for all workers to finish
	wg.Wait()
	cancel()
	close(latCh)
	<-doneCol

	// Calculate summary
	mu.Lock()
	sum := core.Summarize(all)
	mu.Unlock()

	migrations := migrationCount.Load()

	// Print results
	fmt.Printf("\n")
	logger.Summary(map[string]interface{}{
		"scenario":           "nat_rebinding",
		
		"protocol":           core.ProtocolName(*useH3),
		"workers":            fixedWorkers,
		"cycles":             fixedCycles,
		"migrations":         migrations,
		"requests_per_phase": fixedRequestsPerPhase,
		"total_requests":     fixedWorkers * fixedCycles * fixedRequestsPerPhase * 2,
		"samples":            sum.Samples,
		"ok_rate_%":          fmt.Sprintf("%.2f", sum.OKRatePct),
		"rps":                fmt.Sprintf("%.2f", sum.RPS),
		"p50_ms":             fmt.Sprintf("%.6f", sum.P50ms),
		"p90_ms":             fmt.Sprintf("%.6f", sum.P90ms),
		"p95_ms":             fmt.Sprintf("%.6f", sum.P95ms),
		"p99_ms":             fmt.Sprintf("%.6f", sum.P99ms),
		"mean_ms":            fmt.Sprintf("%.6f", sum.Meanms),
		"min_ms":             fmt.Sprintf("%.6f", sum.Minms),
		"max_ms":             fmt.Sprintf("%.6f", sum.Maxms),
	})

	// Also log in standard format for backward compatibility
	log.Printf("done | samples=%d ok_rate=%.2f%% rps=%.2f p50=%.6fms p90=%.6fms p95=%.6fms p99=%.6fms",
		sum.Samples, sum.OKRatePct, sum.RPS, sum.P50ms, sum.P90ms, sum.P95ms, sum.P99ms)

	// Write CSV/HTML if requested
	if csvAbs != "" {
		if err := core.WriteCSV(csvAbs, all, logger); err != nil {
			log.Printf("ERROR write csv: %v", err)
		}
	}
	if htmlAbs != "" {
		if err := core.WriteHTML(htmlAbs, *label, sum, logger); err != nil {
			log.Printf("ERROR write html: %v", err)
		}
	}

	logger.Info("Total runtime: %v", time.Since(start))
}

// newHTTPClient creates an HTTP client (H2 or H3) with TLS config
func newHTTPClient(useH3 bool, insecure bool) (*http.Client, func()) {
	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: insecure,
	}

	if useH3 {
		tr := &http3.Transport{TLSClientConfig: tlsCfg}
		return &http.Client{Transport: tr, Timeout: 0}, func() { tr.CloseIdleConnections() }
	}

	h2 := &http.Transport{
		TLSClientConfig:   tlsCfg,
		ForceAttemptHTTP2: true,
	}
	_ = http2.ConfigureTransport(h2)
	return &http.Client{Transport: h2, Timeout: 0}, func() {}
}
