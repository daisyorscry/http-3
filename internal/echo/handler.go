package echo

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"

	echov1 "h3-vs-h2-k6/echo/v1"
	"h3-vs-h2-k6/echo/v1/echov1connect"
)

const HeaderBloatKey = "x-meta-bloat"

// LogLevel controls server logging verbosity
type LogLevel int

const (
	LogLevelQuiet   LogLevel = iota // No request logging
	LogLevelMinimal                 // Only errors
	LogLevelNormal                  // Summary every N requests
	LogLevelVerbose                 // Every request
)

type svc struct {
	logLevel    LogLevel
	reqCount    atomic.Int64
	protocol    string
	lastLogTime atomic.Int64
}

// NewMux creates a new HTTP mux with echo service handler
func NewMux() *http.ServeMux {
	return NewMuxWithLogging(LogLevelNormal, "unknown")
}

// NewMuxWithLogging creates a new HTTP mux with configurable logging
func NewMuxWithLogging(level LogLevel, protocol string) *http.ServeMux {
	mux := http.NewServeMux()
	s := &svc{logLevel: level, protocol: protocol}
	path, h := echov1connect.NewEchoServiceHandler(s)
	mux.Handle(path, h)
	log.Printf("[%s] Echo service handler registered at %s", protocol, path)
	return mux
}

func (s *svc) Unary(ctx context.Context, req *connect.Request[echov1.EchoRequest]) (*connect.Response[echov1.EchoResponse], error) {
	reqID := s.reqCount.Add(1)
	t0 := time.Now()

	// Count header bloat
	totalBloatSize := 0
	bloatHeaderCount := 0
	for key, values := range req.Header() {
		if len(key) > 7 && key[:7] == "X-Bloat" {
			bloatHeaderCount++
			for _, v := range values {
				totalBloatSize += len(v)
			}
		}
	}

	// Also check old style bloat header
	if bloat := req.Header().Get(HeaderBloatKey); bloat != "" {
		totalBloatSize += len(bloat)
	}

	// Log request if verbose
	if s.logLevel >= LogLevelVerbose {
		log.Printf("[%s] REQ #%d: msg=%q payload=%d bytes, bloat_headers=%d total_bloat=%d bytes",
			s.protocol, reqID, req.Msg.GetMessage(), len(req.Msg.GetPayload()), bloatHeaderCount, totalBloatSize)
	}

	// Build response
	resp := connect.NewResponse(&echov1.EchoResponse{
		Message: req.Msg.GetMessage(),
		Payload: req.Msg.GetPayload(),
	})
	resp.Header().Set("server-recv-bloat-len", strconv.Itoa(totalBloatSize))
	resp.Header().Set("x-request-id", strconv.FormatInt(reqID, 10))

	// Simulate light work
	time.Sleep(1 * time.Millisecond)

	latency := time.Since(t0)

	// Log response if verbose
	if s.logLevel >= LogLevelVerbose {
		log.Printf("[%s] RES #%d: latency=%v", s.protocol, reqID, latency)
	}

	// Log summary every second if normal level
	if s.logLevel == LogLevelNormal {
		now := time.Now().Unix()
		last := s.lastLogTime.Load()
		if now > last && s.lastLogTime.CompareAndSwap(last, now) {
			log.Printf("[%s] STATS: total_requests=%d", s.protocol, reqID)
		}
	}

	return resp, nil
}
