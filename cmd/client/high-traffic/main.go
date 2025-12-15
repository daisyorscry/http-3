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
// HIGH TRAFFIC STRESS TEST
// =====================================
// Skenario untuk mengetahui batas kapasitas dan degradasi performa
// sistem dengan beban sangat tinggi
//
// Pattern:
// - Ramp-up phase: gradually increase RPS
// - Sustained high load: maintain peak RPS
// - Ramp-down phase: gradually decrease RPS
//
// FIXED CONFIGURATION:
// Config: 1000 workers, 60s ramp-up to 15K RPS, 120s sustained, 60s ramp-down, 512B payload
// Total duration: 240s
// =====================================

const (
	fixedWorkers       = 1000
	fixedRampUpTime    = 60 * time.Second
	fixedSustainedTime = 120 * time.Second
	fixedRampDownTime  = 60 * time.Second
	fixedPeakRPS       = 15000
	fixedPayload       = 512
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
		label    = flag.String("label", "High Traffic Stress Test", "dashboard title label")
		quiet    = flag.Bool("quiet", false, "suppress progress logs during test")
		verbose  = flag.Bool("verbose", false, "enable verbose request/response logging")
	)
	flag.Parse()


	totalDuration := fixedRampUpTime + fixedSustainedTime + fixedRampDownTime

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
	logger.Startup("high-traffic", map[string]interface{}{
		"pid":            os.Getpid(),
		"cwd":            cwd,
		"addr":           *addr,
		"protocol":       core.ProtocolName(*useH3),
		
		"insecure":       *insecure,
		"workers":        fixedWorkers,
		"peak_rps":       fixedPeakRPS,
		"ramp_up_time":   fixedRampUpTime,
		"sustained_time": fixedSustainedTime,
		"ramp_down_time": fixedRampDownTime,
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

	// Safety timeout (total duration + buffer)
	go func() {
		select {
		case <-time.After(totalDuration + 60*time.Second):
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
	wg.Add(fixedWorkers)
	for i := 0; i < fixedWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()
			logger.Debug("Worker %d started", workerID)
			core.JobWorker(ctx, client, latCh, counters, logger, jobs, requestFn, &reqCounter)
			logger.Debug("Worker %d stopped", workerID)
		}(i)
	}

	// Start stress test dispatcher
	var dispWg sync.WaitGroup
	dispWg.Add(1)
	go func() {
		defer dispWg.Done()
		stressTestDispatcher(ctx, jobs, logger)
	}()

	// Start benchmark
	start := time.Now()
	logger.Info("Starting high traffic stress test...")

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
		"scenario":       "high_traffic_stress",
		
		"protocol":       core.ProtocolName(*useH3),
		"workers":        fixedWorkers,
		"peak_rps":       fixedPeakRPS,
		"total_duration": totalDuration,
		"samples":        sum.Samples,
		"ok_rate_%":      fmt.Sprintf("%.2f", sum.OKRatePct),
		"rps":            fmt.Sprintf("%.2f", sum.RPS),
		"p50_ms":         fmt.Sprintf("%.6f", sum.P50ms),
		"p90_ms":         fmt.Sprintf("%.6f", sum.P90ms),
		"p95_ms":         fmt.Sprintf("%.6f", sum.P95ms),
		"p99_ms":         fmt.Sprintf("%.6f", sum.P99ms),
		"mean_ms":        fmt.Sprintf("%.6f", sum.Meanms),
		"min_ms":         fmt.Sprintf("%.6f", sum.Minms),
		"max_ms":         fmt.Sprintf("%.6f", sum.Maxms),
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

// stressTestDispatcher implements ramp-up -> sustained -> ramp-down pattern
func stressTestDispatcher(ctx context.Context, jobs chan<- struct{}, logger *core.Logger) {
	logger.Info("Stress test dispatcher started")

	start := time.Now()

	// PHASE 1: RAMP-UP
	logger.Info("PHASE 1: RAMP-UP (0 -> %d RPS over %v)", fixedPeakRPS, fixedRampUpTime)
	rampUpEnd := start.Add(fixedRampUpTime)

	for time.Now().Before(rampUpEnd) {
		select {
		case <-ctx.Done():
			logger.Info("Stress test dispatcher stopped during ramp-up")
			return
		default:
		}

		elapsed := time.Since(start)
		progress := float64(elapsed) / float64(fixedRampUpTime)
		if progress > 1.0 {
			progress = 1.0
		}
		currentRPS := int(float64(fixedPeakRPS) * progress)
		if currentRPS < 100 {
			currentRPS = 100
		}

		interval := time.Second / time.Duration(currentRPS)
		select {
		case jobs <- struct{}{}:
		default:
		}
		time.Sleep(interval)
	}

	// PHASE 2: SUSTAINED HIGH LOAD
	logger.Info("PHASE 2: SUSTAINED (maintain %d RPS for %v)", fixedPeakRPS, fixedSustainedTime)
	sustainedEnd := time.Now().Add(fixedSustainedTime)
	interval := time.Second / time.Duration(fixedPeakRPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for time.Now().Before(sustainedEnd) {
		select {
		case <-ctx.Done():
			logger.Info("Stress test dispatcher stopped during sustained phase")
			return
		case <-ticker.C:
			select {
			case jobs <- struct{}{}:
			default:
			}
		}
	}

	// PHASE 3: RAMP-DOWN
	logger.Info("PHASE 3: RAMP-DOWN (%d RPS -> 0 over %v)", fixedPeakRPS, fixedRampDownTime)
	rampDownStart := time.Now()
	rampDownEnd := rampDownStart.Add(fixedRampDownTime)

	for time.Now().Before(rampDownEnd) {
		select {
		case <-ctx.Done():
			logger.Info("Stress test dispatcher stopped during ramp-down")
			return
		default:
		}

		elapsed := time.Since(rampDownStart)
		progress := float64(elapsed) / float64(fixedRampDownTime)
		if progress > 1.0 {
			progress = 1.0
		}
		currentRPS := int(float64(fixedPeakRPS) * (1.0 - progress))
		if currentRPS < 100 {
			currentRPS = 100
		}

		interval := time.Second / time.Duration(currentRPS)
		select {
		case jobs <- struct{}{}:
		default:
		}
		time.Sleep(interval)
	}

	logger.Info("Stress test dispatcher completed all phases")
}
