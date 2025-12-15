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

	"connectrpc.com/connect"
	"h3-vs-h2-k6/cmd/client/core"
	echov1 "h3-vs-h2-k6/echo/v1"
	"h3-vs-h2-k6/echo/v1/echov1connect"
)

// =====================================
// MIXED LOAD WITH QUEUEING EFFECTS
// =====================================
// Skenario untuk mengamati dampak antrian pada lalu lintas heterogen
// dengan mix dari small/fast requests dan large/slow requests
//
// Request Types:
// - SMALL: 512B payload, fast processing
// - MEDIUM: 8KB payload, moderate processing
// - LARGE: 64KB payload, slow processing
//
// Config: 1000 workers, 120s, 3000 RPS mixed (50% small/30% medium/20% large)
// =====================================

const (
	fixedWorkers       = 1000
	fixedDuration      = 120 * time.Second
	fixedTargetRPS     = 3000
	fixedSmallPct      = 50
	fixedMediumPct     = 30
	fixedLargePct      = 20
	fixedSmallPayload  = 512
	fixedMediumPayload = 8 * 1024  // 8KB
	fixedLargePayload  = 64 * 1024 // 64KB
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
		label    = flag.String("label", "Mixed Load Benchmark", "dashboard title label")
		quiet    = flag.Bool("quiet", false, "suppress progress logs during test")
		verbose  = flag.Bool("verbose", false, "enable verbose request/response logging")
	)
	flag.Parse()

	// Get fixed config for level

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
	logger.Startup("mixed-load", map[string]interface{}{
		"pid":            os.Getpid(),
		"cwd":            cwd,
		"addr":           *addr,
		"protocol":       core.ProtocolName(*useH3),
		
		"insecure":       *insecure,
		"workers":        fixedWorkers,
		"duration":       fixedDuration,
		"target_rps":     fixedTargetRPS,
		"small_pct":      fixedSmallPct,
		"medium_pct":     fixedMediumPct,
		"large_pct":      fixedLargePct,
		"small_payload":  fixedSmallPayload,
		"medium_payload": fixedMediumPayload,
		"large_payload":  fixedLargePayload,
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
		case <-time.After(fixedDuration):
			logger.Info("Duration elapsed: %v -> stopping", fixedDuration)
			cancel()
		case <-ctx.Done():
		}
	}()

	// Channels & counters
	latCh := make(chan core.Record, 1<<20)
	jobs := make(chan requestJob, 1<<16)
	counters := core.NewCounters()
	var reqCounter atomic.Int64

	// Track request type distribution
	var smallCount, mediumCount, largeCount atomic.Int64

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

	// Start workers
	var wg sync.WaitGroup
	wg.Add(fixedWorkers)
	for i := 0; i < fixedWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()
			logger.Debug("Worker %d started", workerID)
			mixedLoadWorker(ctx, client, latCh, counters, logger, jobs, &reqCounter,
				&smallCount, &mediumCount, &largeCount)
			logger.Debug("Worker %d stopped", workerID)
		}(i)
	}

	// Start dispatcher
	var dispWg sync.WaitGroup
	dispWg.Add(1)
	go func() {
		defer dispWg.Done()
		mixedLoadDispatcher(ctx, jobs, logger)
	}()

	// Start benchmark
	start := time.Now()
	logger.Info("Starting mixed load benchmark...")

	// Wait for completion
	dispWg.Wait()
	cancel()
	wg.Wait()
	close(latCh)
	<-doneCol

	// Calculate summary
	mu.Lock()
	sum := core.Summarize(all)
	mu.Unlock()

	// Get request type counts
	small := smallCount.Load()
	medium := mediumCount.Load()
	large := largeCount.Load()
	total := small + medium + large

	// Print results
	fmt.Printf("\n")
	logger.Summary(map[string]interface{}{
		"scenario":        "mixed_load",
		
		"protocol":        core.ProtocolName(*useH3),
		"workers":         fixedWorkers,
		"target_rps":      fixedTargetRPS,
		"small_requests":  small,
		"medium_requests": medium,
		"large_requests":  large,
		"total_requests":  total,
		"small_pct_actual":  fmt.Sprintf("%.1f", float64(small)/float64(total)*100),
		"medium_pct_actual": fmt.Sprintf("%.1f", float64(medium)/float64(total)*100),
		"large_pct_actual":  fmt.Sprintf("%.1f", float64(large)/float64(total)*100),
		"samples":         sum.Samples,
		"ok_rate_%":       fmt.Sprintf("%.2f", sum.OKRatePct),
		"rps":             fmt.Sprintf("%.2f", sum.RPS),
		"p50_ms":          fmt.Sprintf("%.6f", sum.P50ms),
		"p90_ms":          fmt.Sprintf("%.6f", sum.P90ms),
		"p95_ms":          fmt.Sprintf("%.6f", sum.P95ms),
		"p99_ms":          fmt.Sprintf("%.6f", sum.P99ms),
		"mean_ms":         fmt.Sprintf("%.6f", sum.Meanms),
		"min_ms":          fmt.Sprintf("%.6f", sum.Minms),
		"max_ms":          fmt.Sprintf("%.6f", sum.Maxms),
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

// requestJob represents a request job with specific payload size
type requestJob struct {
	payloadSize int
	reqType     string // "small", "medium", "large"
}

// mixedLoadDispatcher dispatches mixed request types at target RPS
func mixedLoadDispatcher(ctx context.Context, jobs chan<- requestJob, logger *core.Logger) {
	logger.Info("Mixed load dispatcher started: target RPS=%d", fixedTargetRPS)

	interval := time.Second / time.Duration(fixedTargetRPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	counter := 0
	for {
		select {
		case <-ctx.Done():
			logger.Info("Mixed load dispatcher stopped")
			return
		case <-ticker.C:
			// Determine request type based on distribution
			job := requestJob{}
			counter++
			mod := counter % 100

			if mod < fixedSmallPct {
				job.payloadSize = fixedSmallPayload
				job.reqType = "small"
			} else if mod < fixedSmallPct+fixedMediumPct {
				job.payloadSize = fixedMediumPayload
				job.reqType = "medium"
			} else {
				job.payloadSize = fixedLargePayload
				job.reqType = "large"
			}

			select {
			case jobs <- job:
			default:
				// Drop if workers can't keep up (queueing effect)
			}
		}
	}
}

// mixedLoadWorker handles mixed request types
func mixedLoadWorker(
	ctx context.Context,
	cl echov1connect.EchoServiceClient,
	latCh chan<- core.Record,
	counters *core.Counters,
	logger *core.Logger,
	jobs <-chan requestJob,
	reqCounter *atomic.Int64,
	smallCount, mediumCount, largeCount *atomic.Int64,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			// Track request type
			switch job.reqType {
			case "small":
				smallCount.Add(1)
			case "medium":
				mediumCount.Add(1)
			case "large":
				largeCount.Add(1)
			}

			// Create request function with specific payload size
			requestFn := func(ctx context.Context, cl echov1connect.EchoServiceClient, reqID int64) (int, error) {
				req := connect.NewRequest(&echov1.EchoRequest{
					Message: job.reqType,
					Payload: make([]byte, job.payloadSize),
				})
				resp, err := cl.Unary(ctx, req)
				if err != nil {
					return 0, err
				}
				return len(resp.Msg.GetPayload()), nil
			}

			reqID := reqCounter.Add(1)
			core.DoRequest(ctx, cl, latCh, counters, logger, reqID, requestFn)
		}
	}
}
