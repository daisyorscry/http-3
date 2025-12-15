package main

import (
	"context"
	"log"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

// absOrEmpty mengkonversi relative path ke absolute path
// Return empty string jika input kosong
func absOrEmpty(p, cwd string) string {
	if strings.TrimSpace(p) == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(cwd, p)
}

// progressPrinter adalah goroutine yang print progress setiap 1 detik
// Menampilkan total OK/error requests dan delta sejak print terakhir
func progressPrinter(ctx context.Context, ok, er *atomic.Uint64) {
	tk := time.NewTicker(1 * time.Second)
	defer tk.Stop()
	var lastOK, lastErr uint64
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			o := ok.Load()
			e := er.Load()
			dOK := o - lastOK
			dErr := e - lastErr
			lastOK, lastErr = o, e
			log.Printf("progress ok=%d (+%d) err=%d (+%d)", o, dOK, e, dErr)
		}
	}
}
