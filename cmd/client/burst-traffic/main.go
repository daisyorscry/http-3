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
// FIXED CONFIGURATION - BURST TRAFFIC
// =====================================
// Skenario untuk menguji kemampuan protokol menghadapi
// lonjakan beban mendadak seperti autocomplete/search-as-you-type
//
// Pola: idle period -> burst -> idle -> burst -> ...
//
// Config: 1000 clients, 3s idle + 3s burst @3000 RPS, 20 cycles, 512B payload
// Total duration = 20 * (3+3) = 120s
// =====================================

const (
	fixedClients     = 1000
	fixedIdlePeriod  = 3 * time.Second
	fixedBurstPeriod = 3 * time.Second
	fixedBurstRPS    = 3000
	fixedCycles      = 20
	fixedPayload     = 512
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
		label    = flag.String("label", "Burst Traffic Benchmark", "dashboard title label")
		quiet    = flag.Bool("quiet", false, "suppress progress logs during test")
		verbose  = flag.Bool("verbose", false, "enable verbose request/response logging")
	)
	flag.Parse()

	totalDuration := time.Duration(fixedCycles) * (fixedIdlePeriod + fixedBurstPeriod)

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
	logger.Startup("burst-traffic", map[string]interface{}{
		"pid":            os.Getpid(),
		"cwd":            cwd,
		"addr":           *addr,
		"protocol":       core.ProtocolName(*useH3),
		"insecure":       *insecure,
		"clients":        fixedClients,
		"idle_period":    fixedIdlePeriod,
		"burst_period":   fixedBurstPeriod,
		"burst_rps":      fixedBurstRPS,
		"cycles":         fixedCycles,
		"total_duration": totalDuration,
		"payload":        fixedPayload,
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
	jobs := make(chan struct{}, 1<<16)
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

	// Start burst dispatcher
	var dispWg sync.WaitGroup
	dispWg.Add(1)
	go func() {
		defer dispWg.Done()
		burstDispatcher(ctx, jobs, logger)
	}()

	// Start benchmark
	start := time.Now()
	logger.Info("Starting burst traffic benchmark...")

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
		"scenario":     "burst_traffic",
		"protocol":     core.ProtocolName(*useH3),
		"clients":      fixedClients,
		"burst_rps":    fixedBurstRPS,
		"cycles":       fixedCycles,
		"idle_period":  fixedIdlePeriod,
		"burst_period": fixedBurstPeriod,
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

// burstDispatcher implements idle-burst-idle-burst pattern
func burstDispatcher(ctx context.Context, jobs chan<- struct{}, logger *core.Logger) {
	logger.Info("Burst dispatcher started: cycles=%d, idle=%v, burst=%v @%d RPS",
		fixedCycles, fixedIdlePeriod, fixedBurstPeriod, fixedBurstRPS)

	for cycle := 0; cycle < fixedCycles; cycle++ {
		select {
		case <-ctx.Done():
			logger.Info("Burst dispatcher stopped (context cancelled)")
			return
		default:
		}

		// IDLE PERIOD - no requests sent
		logger.Info("Cycle %d/%d: IDLE for %v", cycle+1, fixedCycles, fixedIdlePeriod)
		idleTimer := time.NewTimer(fixedIdlePeriod)
		select {
		case <-ctx.Done():
			idleTimer.Stop()
			logger.Info("Burst dispatcher stopped during idle")
			return
		case <-idleTimer.C:
		}

		// BURST PERIOD - send requests at high RPS
		logger.Info("Cycle %d/%d: BURST for %v @%d RPS", cycle+1, fixedCycles, fixedBurstPeriod, fixedBurstRPS)

		interval := time.Second / time.Duration(fixedBurstRPS)
		ticker := time.NewTicker(interval)
		burstTimer := time.NewTimer(fixedBurstPeriod)

		burstLoop:
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				burstTimer.Stop()
				logger.Info("Burst dispatcher stopped during burst")
				return
			case <-burstTimer.C:
				ticker.Stop()
				break burstLoop
			case <-ticker.C:
				select {
				case jobs <- struct{}{}:
				default:
					// Drop if workers can't keep up
				}
			}
		}
	}

	logger.Info("Burst dispatcher completed all %d cycles", fixedCycles)
}
