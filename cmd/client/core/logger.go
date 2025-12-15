package core

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// LogLevel defines logging verbosity
type LogLevel int

const (
	LogLevelQuiet   LogLevel = iota // No logs
	LogLevelMinimal                 // Only errors and summary
	LogLevelNormal                  // Progress + errors
	LogLevelVerbose                 // All requests/responses
	LogLevelDebug                   // Everything including headers
)

// Logger handles structured logging for benchmark clients
type Logger struct {
	level     LogLevel
	mu        sync.Mutex
	reqCount  int64
	startTime time.Time
}

// NewLogger creates a new logger with specified level
func NewLogger(level LogLevel) *Logger {
	return &Logger{
		level:     level,
		startTime: time.Now(),
	}
}

// NewLoggerFromQuiet creates logger based on quiet flag
func NewLoggerFromQuiet(quiet bool) *Logger {
	if quiet {
		return NewLogger(LogLevelQuiet)
	}
	return NewLogger(LogLevelNormal)
}

// SetLevel changes the log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// Startup logs startup information
func (l *Logger) Startup(component string, config map[string]interface{}) {
	if l.level < LogLevelMinimal {
		return
	}
	log.Printf("[%s] ====== STARTUP ======", component)
	for k, v := range config {
		log.Printf("[%s] %s = %v", component, k, v)
	}
	log.Printf("[%s] ======================", component)
}

// RequestStart logs the beginning of a request (verbose mode)
func (l *Logger) RequestStart(reqID int64, method, addr string, headers map[string]string) {
	if l.level < LogLevelVerbose {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	log.Printf("[REQ #%d] --> %s %s", reqID, method, addr)
	if l.level >= LogLevelDebug && len(headers) > 0 {
		for k, v := range headers {
			// Truncate long header values
			if len(v) > 100 {
				v = v[:100] + "..."
			}
			log.Printf("[REQ #%d]     Header: %s = %s", reqID, k, v)
		}
	}
}

// RequestEnd logs the completion of a request (verbose mode)
func (l *Logger) RequestEnd(reqID int64, statusOK bool, latency time.Duration, respSize int, err error) {
	if l.level < LogLevelVerbose {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	status := "OK"
	if !statusOK {
		status = "FAIL"
	}

	if err != nil {
		log.Printf("[REQ #%d] <-- %s latency=%v error=%v", reqID, status, latency, err)
	} else {
		log.Printf("[REQ #%d] <-- %s latency=%v size=%d bytes", reqID, status, latency, respSize)
	}
}

// Error logs an error (always shown unless quiet)
func (l *Logger) Error(reqID int64, err error) {
	if l.level < LogLevelMinimal {
		return
	}
	log.Printf("[ERROR #%d] %v", reqID, err)
}

// ErrorThrottled logs errors with throttling (first N + every Mth)
func (l *Logger) ErrorThrottled(errCount int64, err error, firstN int64, everyM int64) {
	if l.level < LogLevelMinimal {
		return
	}
	if errCount <= firstN || errCount%everyM == 0 {
		log.Printf("[ERROR #%d] %v", errCount, err)
	}
}

// Progress logs progress information
func (l *Logger) Progress(okCount, errCount uint64, deltaOK, deltaErr uint64) {
	if l.level < LogLevelNormal {
		return
	}
	log.Printf("[PROGRESS] ok=%d (+%d) err=%d (+%d)", okCount, deltaOK, errCount, deltaErr)
}

// Summary logs final summary
func (l *Logger) Summary(stats map[string]interface{}) {
	if l.level < LogLevelMinimal {
		return
	}
	log.Printf("[SUMMARY] ====== RESULTS ======")
	for k, v := range stats {
		log.Printf("[SUMMARY] %s = %v", k, v)
	}
	log.Printf("[SUMMARY] ======================")
}

// Info logs informational message
func (l *Logger) Info(format string, args ...interface{}) {
	if l.level < LogLevelNormal {
		return
	}
	log.Printf("[INFO] "+format, args...)
}

// Debug logs debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.level < LogLevelDebug {
		return
	}
	log.Printf("[DEBUG] "+format, args...)
}

// FormatDuration formats duration for display
func FormatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.3fµs", float64(d.Nanoseconds())/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%.3fms", float64(d.Nanoseconds())/1e6)
	}
	return fmt.Sprintf("%.3fs", d.Seconds())
}

// FormatBytes formats bytes for display
func FormatBytes(b int) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	}
	if b < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(b)/1024)
	}
	return fmt.Sprintf("%.2f MB", float64(b)/(1024*1024))
}
