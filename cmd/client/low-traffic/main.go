package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"h3-vs-h2-k6/echo/v1/echov1connect"
)

// =====================================
// FIXED CONFIGURATION - LOW TRAFFIC BASELINE
// =====================================
// Baseline scenario dengan beban ringan untuk membandingkan
// performa dasar HTTP/2 vs HTTP/3
//
// Config: 1000 clients, periodic requests (200ms ± 100ms jitter), 120s
// =====================================

const (
	fixedClients = 1000
	fixedPayload = 512
	fixedDur     = 120 * time.Second
	fixedPeriod  = 200 * time.Millisecond
	fixedJitter  = 100 * time.Millisecond
)

// ---- main ----
func main() {
	// -------- Flags --------
	var (
		addr     = flag.String("addr", "https://localhost:8443", "server URL")
		useH3    = flag.Bool("h3", true, "use HTTP/3 (true) or HTTP/2 (false)")
		insecure = flag.Bool("insecure", true, "skip TLS verify (dev)")

		// output only
		csvPath  = flag.String("csv", "", "write CSV after test")
		htmlPath = flag.String("html", "", "write HTML dashboard after test")
		label    = flag.String("label", "Low Traffic Baseline", "dashboard title label")
		quiet    = flag.Bool("quiet", false, "suppress progress logs during test")
	)
	flag.Parse()

	// ---- Startup logging ----
	cwd, _ := os.Getwd()
	log.Printf("[startup] pid=%d cwd=%s", os.Getpid(), cwd)
	log.Printf("[startup] argv=%q", os.Args)
	log.Printf("[config] clients=%d payload=%d duration=%v", fixedClients, fixedPayload, fixedDur)
	log.Printf("[config] mode=periodic period=%v jitter=%v", fixedPeriod, fixedJitter)

	// Absolutkan output path (kalau diisi user)
	csvAbs := absOrEmpty(*csvPath, cwd)
	htmlAbs := absOrEmpty(*htmlPath, cwd)

	// Build HTTP client
	httpClient, closer := newHTTPClient(*useH3, *insecure)
	defer closer()
	client := echov1connect.NewEchoServiceClient(httpClient, *addr)

	// Context & cancel
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Timer durasi
	go func() {
		select {
		case <-time.After(fixedDur):
			log.Printf("[timer] duration elapsed: %v → cancel()", fixedDur)
			cancel()
		case <-ctx.Done():
		}
	}()

	// Channels & counters
	latCh := make(chan rec, 1<<20)
	var totalOK, totalErr atomic.Uint64
	var errLogCount atomic.Int64

	if !*quiet {
		go progressPrinter(ctx, &totalOK, &totalErr)
	}

	// Worker pool
	var wg sync.WaitGroup
	wg.Add(fixedClients)

	// Worker launcher - periodic mode only
	for i := 0; i < fixedClients; i++ {
		go func(id int) {
			defer wg.Done()
			runPeriodicWorker(ctx, client, latCh, &totalOK, &totalErr, &errLogCount, fixedPeriod, fixedJitter, fixedPayload, 0)
		}(i)
	}

	// Collector
	var all []rec
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

	// Tunggu selesai
	start := time.Now()
	<-ctx.Done() // tunggu timer durasi atau signal
	wg.Wait()
	close(latCh)
	<-doneCol

	// Ringkas & tulis output
	mu.Lock()
	sum := summarize(all)
	mu.Unlock()

	log.Printf("done | samples=%d ok_rate=%.2f%% rps=%.2f p50=%.6fms p90=%.6fms p95=%.6fms p99=%.6fms",
		sum.Samples, sum.OKRatePct, sum.RPS, sum.P50ms, sum.P90ms, sum.P95ms, sum.P99ms)

	// Tulis CSV/HTML kalau diminta
	if csvAbs != "" {
		if err := writeCSV(csvAbs, all); err != nil {
			log.Printf("ERROR write csv to %s: %v", csvAbs, err)
		} else {
			log.Printf("csv written: %s", csvAbs)
		}
	}
	if htmlAbs != "" {
		if err := writeHTML(htmlAbs, *label, sum); err != nil {
			log.Printf("ERROR write html to %s: %v", htmlAbs, err)
		} else {
			log.Printf("html written: %s", htmlAbs)
		}
	}

	log.Printf("[runtime] %v", time.Since(start))
}
