package main

import (
	"context"
	"log"
	"math/rand"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	echov1 "h3-vs-h2-k6/echo/v1"
	"h3-vs-h2-k6/echo/v1/echov1connect"
)

// runPeriodicWorker adalah worker untuk LOW traffic scenario (periodic requests)
// Setiap worker menjalankan request secara berkala dengan interval tetap + jitter
func runPeriodicWorker(
	ctx context.Context,
	cl echov1connect.EchoServiceClient,
	latCh chan<- rec,
	totalOK, totalErr *atomic.Uint64,
	errLogCount *atomic.Int64,
	period, jitter time.Duration,
	payload, bloat int,
) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		doOne(ctx, cl, latCh, totalOK, totalErr, errLogCount, payload, bloat)
		sleep := period
		if jitter > 0 {
			sleep += time.Duration(rng.Int63n(int64(jitter)))
		}
		time.Sleep(sleep)
	}
}

// runJobWorker adalah worker untuk MEDIUM & HIGH traffic scenario (job-based)
// Worker menunggu token dari channel jobs, setiap dapat token execute request
func runJobWorker(
	ctx context.Context,
	cl echov1connect.EchoServiceClient,
	latCh chan<- rec,
	totalOK, totalErr *atomic.Uint64,
	errLogCount *atomic.Int64,
	jobs <-chan job,
	payload, bloat int,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-jobs:
			if !ok {
				return
			}
			doOne(ctx, cl, latCh, totalOK, totalErr, errLogCount, payload, bloat)
		}
	}
}

// doOne melakukan satu gRPC request dan merekam hasilnya
// Fungsi ini mengukur latency, update counter, dan log error jika perlu
func doOne(
	ctx context.Context,
	cl echov1connect.EchoServiceClient,
	latCh chan<- rec,
	totalOK, totalErr *atomic.Uint64,
	errLogCount *atomic.Int64,
	payload, bloat int,
) {
	req := connect.NewRequest(&echov1.EchoRequest{
		Message: "ping",
		Payload: make([]byte, payload),
	})
	if bloat > 0 {
		req.Header().Set("x-meta-bloat", string(make([]byte, bloat)))
	}
	t0 := time.Now()
	_, err := cl.Unary(ctx, req)
	lat := time.Since(t0)

	ok := err == nil
	if ok {
		totalOK.Add(1)
	} else {
		totalErr.Add(1)
		n := errLogCount.Add(1)
		// log 10 error pertama + setiap 1000th supaya nggak banjir
		if n <= 10 || n%1000 == 0 {
			log.Printf("[req error #%d] %v", n, err)
		}
	}
	select {
	case latCh <- rec{TsUnixNS: t0.UnixNano(), LatencyNS: lat.Nanoseconds(), OK: ok}:
	default:
		// drop jika penuh
	}
}
