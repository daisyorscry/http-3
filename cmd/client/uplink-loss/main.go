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
// SMALL UPLOADS UNDER UPLINK LOSS
// =====================================
// Skenario untuk menilai ketahanan protokol pada uplink yang tidak stabil
//
// Catatan: Skenario ini memerlukan network impairment tools untuk
// mensimulasikan packet loss pada uplink (misalnya tc/netem di Linux,
// Network Link Conditioner di macOS, atau clumsy di Windows)
//
// Tanpa network impairment, skenario ini akan berfungsi sebagai
// baseline upload benchmark untuk kemudian dibandingkan dengan
// kondisi loss yang disimulasikan.
//
// Recommended network impairment setup:
// - macOS: Network Link Conditioner (Xcode Additional Tools)
//   - Profile: Custom (1% - 5% packet loss uplink)
// - Linux: tc netem
//   - sudo tc qdisc add dev eth0 root netem loss 1%
// - Windows: clumsy
//   - Outbound packet drop 1% - 5%
//
// FIXED CONFIGURATION:
// Config: 1000 workers, 100 uploads per worker, 8KB payload, 2000 RPS, 120s
// Total: 1000 * 100 = 100,000 uploads
// =====================================

const (
	fixedWorkers      = 1000
	fixedTotalUploads = 100
	fixedUploadSize   = 8 * 1024  // 8KB
	fixedTargetRPS    = 2000
	fixedDuration     = 120 * time.Second
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
		label    = flag.String("label", "Uplink Loss Benchmark", "dashboard title label")
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
	logger.Startup("uplink-loss", map[string]interface{}{
		"pid":           os.Getpid(),
		"cwd":           cwd,
		"addr":          *addr,
		"protocol":      core.ProtocolName(*useH3),
		
		"insecure":      *insecure,
		"workers":       fixedWorkers,
		"total_uploads": fixedTotalUploads,
		"upload_size":   fixedUploadSize,
		"target_rps":    fixedTargetRPS,
		"est_duration":  fixedDuration,
	})

	// Network impairment reminder
	logger.Info("NOTE: For uplink loss testing, configure network impairment:")
	logger.Info("  - macOS: Network Link Conditioner (1-5%% uplink loss)")
	logger.Info("  - Linux: tc qdisc add dev <iface> root netem loss 1%%")
	logger.Info("  - Windows: clumsy (outbound packet drop 1-5%%)")
	logger.Info("Without impairment, this runs as baseline upload benchmark")

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

	// Timer untuk durasi maksimum (safety timeout)
	go func() {
		select {
		case <-time.After(fixedDuration + 60*time.Second):
			logger.Info("Safety timeout reached -> stopping")
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

	// Create upload request function
	requestFn := core.SimpleRequest(fixedUploadSize)

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
			core.JobWorker(ctx, client, latCh, counters, logger, jobs, requestFn, &reqCounter)
			logger.Debug("Worker %d stopped", workerID)
		}(i)
	}

	// Start dispatcher
	var dispWg sync.WaitGroup
	dispWg.Add(1)
	go func() {
		defer dispWg.Done()
		uplinkLossDispatcher(ctx, jobs, logger)
	}()

	// Start benchmark
	start := time.Now()
	logger.Info("Starting uplink loss benchmark...")

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

	// Print results
	fmt.Printf("\n")
	logger.Summary(map[string]interface{}{
		"scenario":      "uplink_loss",
		
		"protocol":      core.ProtocolName(*useH3),
		"workers":       fixedWorkers,
		"upload_size":   fixedUploadSize,
		"target_rps":    fixedTargetRPS,
		"total_uploads": fixedWorkers * fixedTotalUploads,
		"samples":       sum.Samples,
		"ok_rate_%":     fmt.Sprintf("%.2f", sum.OKRatePct),
		"rps":           fmt.Sprintf("%.2f", sum.RPS),
		"p50_ms":        fmt.Sprintf("%.6f", sum.P50ms),
		"p90_ms":        fmt.Sprintf("%.6f", sum.P90ms),
		"p95_ms":        fmt.Sprintf("%.6f", sum.P95ms),
		"p99_ms":        fmt.Sprintf("%.6f", sum.P99ms),
		"mean_ms":       fmt.Sprintf("%.6f", sum.Meanms),
		"min_ms":        fmt.Sprintf("%.6f", sum.Minms),
		"max_ms":        fmt.Sprintf("%.6f", sum.Maxms),
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

// uplinkLossDispatcher dispatches upload jobs at constant RPS
func uplinkLossDispatcher(ctx context.Context, jobs chan<- struct{}, logger *core.Logger) {
	logger.Info("Uplink loss dispatcher started: target RPS=%d", fixedTargetRPS)

	totalJobs := fixedWorkers * fixedTotalUploads
	interval := time.Second / time.Duration(fixedTargetRPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	sent := 0
	for sent < totalJobs {
		select {
		case <-ctx.Done():
			logger.Info("Uplink loss dispatcher stopped (context cancelled)")
			return
		case <-ticker.C:
			select {
			case jobs <- struct{}{}:
				sent++
			default:
				// Drop if workers can't keep up
			}
		}
	}

	logger.Info("Uplink loss dispatcher completed: sent %d jobs", sent)
}
