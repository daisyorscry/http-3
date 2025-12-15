package main

import (
	"math"
	"sort"
)

// summarize mengaggregate raw records jadi summary statistics
// Menghitung metrics seperti latency percentiles, RPS, CDF, dan throughput per second
func summarize(all []rec) Summary {
	if len(all) == 0 {
		return Summary{}
	}
	var minTS, maxTS int64 = math.MaxInt64, math.MinInt64
	var okCount int
	latms := make([]float64, 0, len(all))
	var sum float64
	min := math.MaxFloat64
	max := -1.0

	for _, r := range all {
		if r.TsUnixNS < minTS {
			minTS = r.TsUnixNS
		}
		if r.TsUnixNS > maxTS {
			maxTS = r.TsUnixNS
		}
		if r.OK {
			okCount++
		}
		ms := float64(r.LatencyNS) / 1e6
		latms = append(latms, ms)
		sum += ms
		if ms < min {
			min = ms
		}
		if ms > max {
			max = ms
		}
	}
	durationS := float64(maxTS-minTS) / 1e9
	if durationS <= 0 {
		durationS = 1e-9
	}
	rps := float64(len(all)) / durationS

	sort.Float64s(latms)
	percentile := func(p float64) float64 {
		pos := p * float64(len(latms)-1)
		i := int(pos)
		f := pos - float64(i)
		if i+1 < len(latms) {
			return latms[i] + f*(latms[i+1]-latms[i])
		}
		return latms[i]
	}

	// CDF
	y := make([]float64, len(latms))
	for i := range latms {
		y[i] = float64(i+1) / float64(len(latms))
	}

	// throughput per second
	m := make(map[int64]int)
	var minSec, maxSec int64 = math.MaxInt64, math.MinInt64
	for _, r := range all {
		sec := r.TsUnixNS / 1e9
		m[sec]++
		if sec < minSec {
			minSec = sec
		}
		if sec > maxSec {
			maxSec = sec
		}
	}
	var ts []int64
	var val []int
	if minSec <= maxSec && minSec != math.MaxInt64 {
		n := int(maxSec - minSec + 1)
		ts = make([]int64, n)
		val = make([]int, n)
		for i := 0; i < n; i++ {
			s := minSec + int64(i)
			ts[i] = s
			val[i] = m[s]
		}
	}

	return Summary{
		Samples:   len(all),
		OKRatePct: 100 * float64(okCount) / float64(len(all)),
		RPS:       rps,
		DurationS: durationS,
		P50ms:     round6(percentile(0.50)),
		P90ms:     round6(percentile(0.90)),
		P95ms:     round6(percentile(0.95)),
		P99ms:     round6(percentile(0.99)),
		Meanms:    round6(sum / float64(len(all))),
		Minms:     round6(min),
		Maxms:     round6(max),
		CDF_X_ms:  latms,
		CDF_Y:     y,
		THR_Ts:    ts,
		THR_Val:   val,
	}
}

// round6 membulatkan float ke 6 decimal places untuk konsistensi formatting
func round6(x float64) float64 {
	const p = 1e6
	return math.Round(x*p) / p
}
