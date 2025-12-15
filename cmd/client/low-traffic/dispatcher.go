package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// constantRPSDispatcher adalah dispatcher untuk MEDIUM scenario (constant RPS)
// Mengirim token ke channel jobs dengan rate konstan sesuai RPS yang ditargetkan
func constantRPSDispatcher(ctx context.Context, out chan<- job, rps int) {
	if rps <= 0 {
		close(out)
		return
	}
	defer close(out)
	interval := time.Second / time.Duration(rps)
	if interval <= 0 {
		interval = time.Nanosecond
	}
	tk := time.NewTicker(interval)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			select {
			case out <- job{}:
			default:
				// drop token jika penuh
			}
		}
	}
}

// parseRamp mem-parse string ramp definition jadi array of stages
// Input format: "30@1000,30@2000,30@4000" (durationSec@rps)
// Output: slice of stage structs, total duration, dan error jika format invalid
func parseRamp(s string) ([]stage, time.Duration, error) {
	var stages []stage
	var total time.Duration
	for _, part := range strings.Split(strings.TrimSpace(s), ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.Split(part, "@")
		if len(kv) != 2 {
			return nil, 0, fmt.Errorf("bad stage: %q", part)
		}
		sec, err := strconv.Atoi(strings.TrimSpace(kv[0]))
		if err != nil || sec <= 0 {
			return nil, 0, fmt.Errorf("bad seconds: %q", kv[0])
		}
		rps, err := strconv.Atoi(strings.TrimSpace(kv[1]))
		if err != nil || rps < 0 {
			return nil, 0, fmt.Errorf("bad rps: %q", kv[1])
		}
		d := time.Duration(sec) * time.Second
		total += d
		stages = append(stages, stage{dur: sec, rps: rps})
	}
	if len(stages) == 0 {
		return nil, 0, fmt.Errorf("no stages")
	}
	return stages, total, nil
}

// rampDispatcher adalah dispatcher untuk HIGH scenario (ramping RPS)
// Mengirim token dengan rate yang berubah sesuai stages yang diberikan
// Setiap stage memiliki durasi dan target RPS sendiri
func rampDispatcher(ctx context.Context, out chan<- job, stages []stage) {
	defer close(out)
	for _, st := range stages {
		if st.rps <= 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(st.dur) * time.Second):
			}
			continue
		}
		interval := time.Second / time.Duration(st.rps)
		if interval <= 0 {
			interval = time.Nanosecond
		}
		tk := time.NewTicker(interval)
		stop := time.NewTimer(time.Duration(st.dur) * time.Second)
	loop:
		for {
			select {
			case <-ctx.Done():
				tk.Stop()
				stop.Stop()
				return
			case <-stop.C:
				tk.Stop()
				break loop
			case <-tk.C:
				select {
				case out <- job{}:
				default:
				}
			}
		}
	}
}
