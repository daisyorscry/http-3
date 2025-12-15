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
// FIXED CONFIGURATION - PARALLEL REQUESTS
// =====================================
// Skenario untuk mengevaluasi efek multiplexing
// setiap worker mengirim N request paralel secara bersamaan
//
// Config: 1000 clients, 20 parallel streams, 50 batches @ 30ms interval, 512B payload
// Total: 1000 * 20 * 50 = 1,000,000 requests
// Total duration ~= 50 * 0.03 = ~1.5s per worker, dengan 1000 workers parallel
// =====================================

const (
	fixedClients        = 1000
	fixedParallelStreams = 20
	fixedBatches        = 50
	fixedBatchInterval  = 30 * time.Millisecond
	fixedPayload        = 512
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
		label    = flag.String("label", "Parallel Requests Benchmark", "dashboard title label")
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
	logger.Startup("parallel-requests", map[string]interface{}{
		"pid":              os.Getpid(),
		"cwd":              cwd,
		"addr":             *addr,
		"protocol":         core.ProtocolName(*useH3),
		"insecure":         *insecure,
		"clients":          fixedClients,
		"parallel_streams": fixedParallelStreams,
		"batches":          fixedBatches,
		"batch_interval":   fixedBatchInterval,
		"payload":          fixedPayload,
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

	// Channels & counters
	latCh := make(chan core.Record, 1<<20)
	counters := core.NewCounters()
	var reqCounter atomic.Int64

	if !*quiet {
		go core.ProgressPrinter(ctx, counters, logger)
	}

	// Create simple request function
	requestFn := core.SimpleRequest(fixedPayload)

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
	logger.Info("Starting parallel requests benchmark...")

	// Start workers
	var wg sync.WaitGroup
	wg.Add(fixedClients)

	for i := 0; i < fixedClients; i++ {
		go func(workerID int) {
			defer wg.Done()
			logger.Debug("Worker %d started", workerID)

			// Each worker runs N batches
			for batch := 0; batch < fixedBatches; batch++ {
				select {
				case <-ctx.Done():
					logger.Debug("Worker %d cancelled at batch %d", workerID, batch)
					return
				default:
				}

				// Send parallelStreams requests concurrently
				var batchWg sync.WaitGroup
				batchWg.Add(fixedParallelStreams)

				for stream := 0; stream < fixedParallelStreams; stream++ {
					go func() {
						defer batchWg.Done()
						reqID := reqCounter.Add(1)
						core.DoRequest(ctx, client, latCh, counters, logger, reqID, requestFn)
					}()
				}

				// Wait for all parallel streams in this batch to complete
				batchWg.Wait()

				// Sleep before next batch (if not the last batch)
				if batch < fixedBatches-1 {
					time.Sleep(fixedBatchInterval)
				}
			}

			logger.Debug("Worker %d completed all batches", workerID)
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

	// Print results
	fmt.Printf("\n")
	logger.Summary(map[string]interface{}{
		"scenario":         "parallel_requests",
		"protocol":         core.ProtocolName(*useH3),
		"clients":          fixedClients,
		"parallel_streams": fixedParallelStreams,
		"batches":          fixedBatches,
		"total_requests":   fixedClients * fixedBatches * fixedParallelStreams,
		"samples":          sum.Samples,
		"ok_rate_%":        fmt.Sprintf("%.2f", sum.OKRatePct),
		"rps":              fmt.Sprintf("%.2f", sum.RPS),
		"p50_ms":           fmt.Sprintf("%.6f", sum.P50ms),
		"p90_ms":           fmt.Sprintf("%.6f", sum.P90ms),
		"p95_ms":           fmt.Sprintf("%.6f", sum.P95ms),
		"p99_ms":           fmt.Sprintf("%.6f", sum.P99ms),
		"mean_ms":          fmt.Sprintf("%.6f", sum.Meanms),
		"min_ms":           fmt.Sprintf("%.6f", sum.Minms),
		"max_ms":           fmt.Sprintf("%.6f", sum.Maxms),
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
