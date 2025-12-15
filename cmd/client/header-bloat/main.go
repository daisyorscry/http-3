package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"h3-vs-h2-k6/cmd/client/core"
	"h3-vs-h2-k6/echo/v1/echov1connect"
)

// =====================================
// FIXED CONFIGURATION - HEADER BLOAT
// =====================================
// Skenario untuk menguji efisiensi kompresi header HTTP/2 (HPACK) vs HTTP/3 (QPACK)
// dengan large metadata overhead
//
// Config: 1000 clients, 2000 RPS, 8KB headers (32 pairs), 120s
// =====================================

const (
	fixedClients     = 1000
	fixedPayload     = 512
	fixedDur         = 120 * time.Second
	fixedRPS         = 2000
	fixedHeaderSize  = 8 * 1024 // 8KB
	fixedHeaderPairs = 32
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
		label    = flag.String("label", "Header Bloat Benchmark", "dashboard title label")
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
	logger.Startup("header-bloat", map[string]interface{}{
		"pid":          os.Getpid(),
		"cwd":          cwd,
		"addr":         *addr,
		"protocol":     core.ProtocolName(*useH3),
		"insecure":     *insecure,
		"clients":      fixedClients,
		"payload":      fixedPayload,
		"duration":     fixedDur,
		"rps":          fixedRPS,
		"header-size":  fixedHeaderSize,
		"header-pairs": fixedHeaderPairs,
	})

	// Absolutkan output path
	csvAbs := core.AbsOrEmpty(*csvPath, cwd)
	htmlAbs := core.AbsOrEmpty(*htmlPath, cwd)

	// Build HTTP client
	httpClient, closer := core.NewHTTPClient(*useH3, *insecure, logger)
	defer closer()
	client := echov1connect.NewEchoServiceClient(httpClient, *addr)

	// Context & cancel
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Timer untuk durasi test
	go func() {
		select {
		case <-time.After(fixedDur):
			logger.Info("Duration elapsed: %v -> stopping", fixedDur)
			cancel()
		case <-ctx.Done():
		}
	}()

	// Channels & counters
	latCh := make(chan core.Record, 1<<20)
	jobs := make(chan struct{}, 1<<16)
	counters := core.NewCounters()
	var reqCounter atomic.Int64

	if !*quiet {
		go core.ProgressPrinter(ctx, counters, logger)
	}

	// Create request function with header bloat
	requestFn := core.HeaderBloatRequest(fixedPayload, fixedHeaderSize, fixedHeaderPairs)

	// Start workers
	var wg sync.WaitGroup
	wg.Add(fixedClients)
	for i := 0; i < fixedClients; i++ {
		go func(workerID int) {
			defer wg.Done()
			logger.Debug("Worker %d started", workerID)
			core.JobWorker(ctx, client, latCh, counters, logger, jobs, requestFn, &reqCounter)
			logger.Debug("Worker %d stopped", workerID)
		}(i)
	}

	// Start dispatcher (constant RPS)
	var dispWg sync.WaitGroup
	dispWg.Add(1)
	go func() {
		defer dispWg.Done()
		core.Dispatcher(ctx, jobs, fixedRPS, logger)
	}()

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

	// Wait for completion
	start := time.Now()
	dispWg.Wait()
	cancel()
	wg.Wait()
	close(latCh)
	<-doneCol

	// Calculate summary
	mu.Lock()
	sum := core.Summarize(all)
	mu.Unlock()

	// Print results
	fmt.Printf("\n")
	logger.Summary(map[string]interface{}{
		"scenario":     "header_bloat",
		"protocol":     core.ProtocolName(*useH3),
		"header_size":  fixedHeaderSize,
		"header_pairs": fixedHeaderPairs,
		"samples":      sum.Samples,
		"ok_rate_%":    fmt.Sprintf("%.2f", sum.OKRatePct),
		"rps":          fmt.Sprintf("%.2f", sum.RPS),
		"p50_ms":       fmt.Sprintf("%.6f", sum.P50ms),
		"p90_ms":       fmt.Sprintf("%.6f", sum.P90ms),
		"p95_ms":       fmt.Sprintf("%.6f", sum.P95ms),
		"p99_ms":       fmt.Sprintf("%.6f", sum.P99ms),
		"mean_ms":      fmt.Sprintf("%.6f", sum.Meanms),
		"min_ms":       fmt.Sprintf("%.6f", sum.Minms),
		"max_ms":       fmt.Sprintf("%.6f", sum.Maxms),
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
