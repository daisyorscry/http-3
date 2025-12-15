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
// FIXED CONFIGURATION - CONNECTION CHURN
// =====================================
// Skenario yang mencerminkan koneksi singkat ala IoT devices
// dimana setiap device connect, send data, disconnect secara cepat
//
// Pattern:
// - Setiap "device" (worker) melakukan N cycles
// - Setiap cycle: buat koneksi baru → send M requests → close koneksi
// - Short-lived connections dengan rapid turnover
//
// Config: 1000 devices, 50 cycles, 2 requests per cycle @ 500ms interval, 512B payload
// Total connections = 1000 * 50 = 50,000 short-lived connections
// Total requests = 1000 * 50 * 2 = 100,000 requests
// =====================================

const (
	fixedDevices          = 1000
	fixedCycles           = 50
	fixedRequestsPerCycle = 2
	fixedCycleInterval    = 500 * time.Millisecond
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
		label    = flag.String("label", "Connection Churn Benchmark", "dashboard title label")
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
	logger.Startup("connection-churn", map[string]interface{}{
		"pid":                os.Getpid(),
		"cwd":                cwd,
		"addr":               *addr,
		"protocol":           core.ProtocolName(*useH3),
		"insecure":           *insecure,
		"devices":            fixedDevices,
		"cycles":             fixedCycles,
		"requests_per_cycle": fixedRequestsPerCycle,
		"cycle_interval":     fixedCycleInterval,
		"payload":            fixedPayload,
		"total_requests":     fixedDevices * fixedCycles * fixedRequestsPerCycle,
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
	logger.Info("Starting connection churn benchmark...")

	// Start device workers
	var wg sync.WaitGroup
	wg.Add(fixedDevices)

	for i := 0; i < fixedDevices; i++ {
		go func(deviceID int) {
			defer wg.Done()
			logger.Debug("Device %d started", deviceID)

			for cycle := 0; cycle < fixedCycles; cycle++ {
				select {
				case <-ctx.Done():
					logger.Debug("Device %d cancelled at cycle %d", deviceID, cycle)
					return
				default:
				}

				// Create NEW connection for this cycle
				httpClient, closer := newHTTPClient(*useH3, *insecure)
				client := echov1connect.NewEchoServiceClient(httpClient, *addr)
				requestFn := core.SimpleRequest(fixedPayload)

				// Send multiple requests on this connection
				for req := 0; req < fixedRequestsPerCycle; req++ {
					select {
					case <-ctx.Done():
						closer()
						logger.Debug("Device %d cancelled during cycle %d", deviceID, cycle)
						return
					default:
					}

					reqID := reqCounter.Add(1)
					core.DoRequest(ctx, client, latCh, counters, logger, reqID, requestFn)
				}

				// Close connection immediately after requests
				closer()

				// Sleep before next cycle (if not last cycle)
				if cycle < fixedCycles-1 {
					time.Sleep(fixedCycleInterval)
				}
			}

			logger.Debug("Device %d completed all %d cycles", deviceID, fixedCycles)
		}(i)
	}

	// Wait for all devices to finish
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
		"scenario":           "connection_churn",
		"protocol":           core.ProtocolName(*useH3),
		"devices":            fixedDevices,
		"cycles":             fixedCycles,
		"requests_per_cycle": fixedRequestsPerCycle,
		"total_connections":  fixedDevices * fixedCycles,
		"total_requests":     fixedDevices * fixedCycles * fixedRequestsPerCycle,
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
