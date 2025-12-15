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
// FIXED CONFIGURATION - COLD-START VS RESUMED
// =====================================
// Skenario untuk melihat dampak pembukaan koneksi baru (cold-start)
// vs koneksi berkelanjutan (resumed/warm connections)
//
// Mode:
// - cold: setiap worker buat client baru untuk setiap request (close connection)
// - warm: workers reuse persistent connection
//
// Config: 1000 workers, 100 requests per worker @ 30ms interval, 512B payload
// Total: 1000 * 100 = 100,000 requests
// =====================================

const (
	fixedWorkers          = 1000
	fixedRequestsPerWorker = 100
	fixedRequestInterval  = 30 * time.Millisecond
	fixedPayload          = 512
)

func main() {
	// -------- Flags --------
	var (
		addr     = flag.String("addr", "https://localhost:8443", "server URL")
		useH3    = flag.Bool("h3", true, "use HTTP/3 (true) or HTTP/2 (false)")
		insecure = flag.Bool("insecure", true, "skip TLS verify (dev)")
		mode     = flag.String("mode", "warm", "cold|warm (connection mode)")

		// Output only
		csvPath  = flag.String("csv", "", "write CSV after test")
		htmlPath = flag.String("html", "", "write HTML dashboard after test")
		label    = flag.String("label", "Cold-Start vs Resumed Benchmark", "dashboard title label")
		quiet    = flag.Bool("quiet", false, "suppress progress logs during test")
		verbose  = flag.Bool("verbose", false, "enable verbose request/response logging")
	)
	flag.Parse()

	// Validate mode
	if *mode != "cold" && *mode != "warm" {
		log.Fatalf("unknown --mode: %s (valid: cold, warm)", *mode)
	}

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
	logger.Startup("cold-start", map[string]interface{}{
		"pid":                 os.Getpid(),
		"cwd":                 cwd,
		"addr":                *addr,
		"protocol":            core.ProtocolName(*useH3),
		"mode":                *mode,
		"insecure":            *insecure,
		"workers":             fixedWorkers,
		"requests_per_worker": fixedRequestsPerWorker,
		"request_interval":    fixedRequestInterval,
		"payload":             fixedPayload,
	})

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
	logger.Info("Starting cold-start benchmark in %s mode...", *mode)

	// Start workers
	var wg sync.WaitGroup
	wg.Add(fixedWorkers)

	if *mode == "warm" {
		// WARM MODE: reuse persistent connection
		// Build shared HTTP client
		httpClient, closer := newHTTPClient(*useH3, *insecure)
		defer closer()
		client := echov1connect.NewEchoServiceClient(httpClient, *addr)
		requestFn := core.SimpleRequest(fixedPayload)

		for i := 0; i < fixedWorkers; i++ {
			go func(workerID int) {
				defer wg.Done()
				logger.Debug("Worker %d started (warm mode)", workerID)

				for req := 0; req < fixedRequestsPerWorker; req++ {
					select {
					case <-ctx.Done():
						logger.Debug("Worker %d cancelled at request %d", workerID, req)
						return
					default:
					}

					reqID := reqCounter.Add(1)
					core.DoRequest(ctx, client, latCh, counters, logger, reqID, requestFn)

					if req < fixedRequestsPerWorker-1 {
						time.Sleep(fixedRequestInterval)
					}
				}

				logger.Debug("Worker %d completed", workerID)
			}(i)
		}
	} else {
		// COLD MODE: create new connection for each request
		for i := 0; i < fixedWorkers; i++ {
			go func(workerID int) {
				defer wg.Done()
				logger.Debug("Worker %d started (cold mode)", workerID)

				for req := 0; req < fixedRequestsPerWorker; req++ {
					select {
					case <-ctx.Done():
						logger.Debug("Worker %d cancelled at request %d", workerID, req)
						return
					default:
					}

					// Create NEW client for each request
					httpClient, closer := newHTTPClient(*useH3, *insecure)
					client := echov1connect.NewEchoServiceClient(httpClient, *addr)
					requestFn := core.SimpleRequest(fixedPayload)

					reqID := reqCounter.Add(1)
					core.DoRequest(ctx, client, latCh, counters, logger, reqID, requestFn)

					// Close connection immediately
					closer()

					if req < fixedRequestsPerWorker-1 {
						time.Sleep(fixedRequestInterval)
					}
				}

				logger.Debug("Worker %d completed", workerID)
			}(i)
		}
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

	// Print results
	fmt.Printf("\n")
	logger.Summary(map[string]interface{}{
		"scenario":            "cold_start",
		"mode":                *mode,
		"protocol":            core.ProtocolName(*useH3),
		"workers":             fixedWorkers,
		"requests_per_worker": fixedRequestsPerWorker,
		"total_requests":      fixedWorkers * fixedRequestsPerWorker,
		"samples":             sum.Samples,
		"ok_rate_%":           fmt.Sprintf("%.2f", sum.OKRatePct),
		"rps":                 fmt.Sprintf("%.2f", sum.RPS),
		"p50_ms":              fmt.Sprintf("%.6f", sum.P50ms),
		"p90_ms":              fmt.Sprintf("%.6f", sum.P90ms),
		"p95_ms":              fmt.Sprintf("%.6f", sum.P95ms),
		"p99_ms":              fmt.Sprintf("%.6f", sum.P99ms),
		"mean_ms":             fmt.Sprintf("%.6f", sum.Meanms),
		"min_ms":              fmt.Sprintf("%.6f", sum.Minms),
		"max_ms":              fmt.Sprintf("%.6f", sum.Maxms),
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
