package core

import (
	"context"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	echov1 "h3-vs-h2-k6/echo/v1"
	"h3-vs-h2-k6/echo/v1/echov1connect"
)

// RequestFunc is a function that builds and executes a request
// Returns the response size and any error
type RequestFunc func(ctx context.Context, cl echov1connect.EchoServiceClient, reqID int64) (int, error)

// DoRequest executes a single request with logging and metrics collection
func DoRequest(
	ctx context.Context,
	cl echov1connect.EchoServiceClient,
	latCh chan<- Record,
	counters *Counters,
	logger *Logger,
	reqID int64,
	requestFn RequestFunc,
) {
	t0 := time.Now()
	respSize, err := requestFn(ctx, cl, reqID)
	lat := time.Since(t0)

	ok := err == nil
	if ok {
		counters.TotalOK.Add(1)
	} else {
		counters.TotalErr.Add(1)
		errCount := counters.ErrLogCount.Add(1)
		logger.ErrorThrottled(errCount, err, 10, 1000)
	}

	// Log request completion
	logger.RequestEnd(reqID, ok, lat, respSize, err)

	// Send record to collector
	select {
	case latCh <- Record{TsUnixNS: t0.UnixNano(), LatencyNS: lat.Nanoseconds(), OK: ok}:
	default:
		// Drop if channel is full
	}
}

// SimpleRequest creates a basic echo request
func SimpleRequest(payload int) RequestFunc {
	return func(ctx context.Context, cl echov1connect.EchoServiceClient, reqID int64) (int, error) {
		req := connect.NewRequest(&echov1.EchoRequest{
			Message: "ping",
			Payload: make([]byte, payload),
		})
		resp, err := cl.Unary(ctx, req)
		if err != nil {
			return 0, err
		}
		return len(resp.Msg.GetPayload()), nil
	}
}

// HeaderBloatRequest creates a request with multiple bloated headers
func HeaderBloatRequest(payload int, headerSize int, headerPairs int) RequestFunc {
	// Pre-generate header values
	headerValues := make([]string, headerPairs)
	sizePerHeader := headerSize / headerPairs
	if sizePerHeader < 1 {
		sizePerHeader = 1
	}
	for i := 0; i < headerPairs; i++ {
		headerValues[i] = generateHeaderValue(sizePerHeader)
	}

	return func(ctx context.Context, cl echov1connect.EchoServiceClient, reqID int64) (int, error) {
		req := connect.NewRequest(&echov1.EchoRequest{
			Message: "header-bloat-test",
			Payload: make([]byte, payload),
		})

		// Add bloated headers
		for i, val := range headerValues {
			req.Header().Set(headerKey(i), val)
		}

		resp, err := cl.Unary(ctx, req)
		if err != nil {
			return 0, err
		}
		return len(resp.Msg.GetPayload()), nil
	}
}

// RequestWithHeaders creates a request with custom headers
func RequestWithHeaders(payload int, headers map[string]string) RequestFunc {
	return func(ctx context.Context, cl echov1connect.EchoServiceClient, reqID int64) (int, error) {
		req := connect.NewRequest(&echov1.EchoRequest{
			Message: "ping",
			Payload: make([]byte, payload),
		})

		for k, v := range headers {
			req.Header().Set(k, v)
		}

		resp, err := cl.Unary(ctx, req)
		if err != nil {
			return 0, err
		}
		return len(resp.Msg.GetPayload()), nil
	}
}

// ProgressPrinter prints progress every second
func ProgressPrinter(ctx context.Context, counters *Counters, logger *Logger) {
	tk := time.NewTicker(1 * time.Second)
	defer tk.Stop()
	var lastOK, lastErr uint64
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			o := counters.TotalOK.Load()
			e := counters.TotalErr.Load()
			dOK := o - lastOK
			dErr := e - lastErr
			lastOK, lastErr = o, e
			logger.Progress(o, e, dOK, dErr)
		}
	}
}

// Dispatcher sends job tokens at constant RPS
func Dispatcher(ctx context.Context, jobs chan<- struct{}, rps int, logger *Logger) {
	if rps <= 0 {
		rps = 1000
	}
	interval := time.Second / time.Duration(rps)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Info("Dispatcher started: target RPS=%d, interval=%v", rps, interval)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Dispatcher stopped")
			return
		case <-ticker.C:
			select {
			case jobs <- struct{}{}:
			default:
				// Drop if workers can't keep up
			}
		}
	}
}

// JobWorker runs requests based on job tokens from dispatcher
func JobWorker(
	ctx context.Context,
	cl echov1connect.EchoServiceClient,
	latCh chan<- Record,
	counters *Counters,
	logger *Logger,
	jobs <-chan struct{},
	requestFn RequestFunc,
	reqCounter *atomic.Int64,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-jobs:
			if !ok {
				return
			}
			reqID := reqCounter.Add(1)
			DoRequest(ctx, cl, latCh, counters, logger, reqID, requestFn)
		}
	}
}

// PeriodicWorker runs requests at periodic intervals with jitter
func PeriodicWorker(
	ctx context.Context,
	cl echov1connect.EchoServiceClient,
	latCh chan<- Record,
	counters *Counters,
	logger *Logger,
	period, jitter time.Duration,
	requestFn RequestFunc,
	reqCounter *atomic.Int64,
) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		reqID := reqCounter.Add(1)
		DoRequest(ctx, cl, latCh, counters, logger, reqID, requestFn)

		sleep := period
		if jitter > 0 {
			sleep += time.Duration(randInt63n(int64(jitter)))
		}
		time.Sleep(sleep)
	}
}

// Helper functions

func generateHeaderValue(size int) string {
	if size <= 0 {
		return ""
	}
	b := make([]byte, size)
	for i := 0; i < size; i++ {
		b[i] = 'a' + byte(i%26)
	}
	return string(b)
}

func headerKey(i int) string {
	return "x-bloat-" + itoa(i)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	idx := len(b)
	for i > 0 {
		idx--
		b[idx] = byte('0' + i%10)
		i /= 10
	}
	return string(b[idx:])
}

// Simple random without importing math/rand
var randState uint64 = uint64(time.Now().UnixNano())

func randInt63n(n int64) int64 {
	randState = randState*6364136223846793005 + 1442695040888963407
	return int64(randState>>1) % n
}
