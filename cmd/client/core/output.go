package core

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
)

// WriteCSV writes raw records to CSV file
func WriteCSV(path string, rows []Record, logger *Logger) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	_ = w.Write([]string{"ts_unix_ns", "latency_ns", "ok"})
	for _, r := range rows {
		_ = w.Write([]string{
			strconv.FormatInt(r.TsUnixNS, 10),
			strconv.FormatInt(r.LatencyNS, 10),
			strconv.FormatBool(r.OK),
		})
	}

	logger.Info("CSV written: %s (%d records)", path, len(rows))
	return nil
}

// WriteHTML generates a self-contained HTML dashboard
func WriteHTML(path string, label string, s Summary, logger *Logger) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}

	toJS := func(v any) template.JS {
		b, _ := json.Marshal(v)
		return template.JS(b)
	}

	data := struct {
		Title    string
		S        Summary
		CDF_X_ms template.JS
		CDF_Y    template.JS
		THR_Ts   template.JS
		THR_Val  template.JS
	}{
		Title:    label,
		S:        s,
		CDF_X_ms: toJS(s.CDF_X_ms),
		CDF_Y:    toJS(s.CDF_Y),
		THR_Ts:   toJS(s.THR_Ts),
		THR_Val:  toJS(s.THR_Val),
	}

	t, err := template.New("page").Parse(htmlTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := t.Execute(f, data); err != nil {
		return err
	}

	logger.Info("HTML written: %s", path)
	return nil
}

const htmlTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{ .Title }} – Benchmark Summary</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; margin: 24px; }
h1 { margin-bottom: 0; }
.sub { color: #666; margin-top: 4px; }
table { border-collapse: collapse; margin-top: 16px; }
td, th { border: 1px solid #ddd; padding: 6px 10px; text-align: left; }
.grid { display: grid; grid-template-columns: 1fr; gap: 16px; margin-top: 18px; }
.chart { width: 100%; height: 360px; }
.code { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; background: #f6f8fa; padding: 2px 6px; border-radius: 4px; }
@media (min-width: 900px) { .grid { grid-template-columns: 1fr 1fr; } }
</style>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4"></script>
</head>
<body>
<h1>{{ .Title }}</h1>
<div class="sub">Auto-generated benchmark dashboard</div>

<h2>Summary</h2>
<table>
<tbody>
	<tr><td>samples</td><td>{{ .S.Samples }}</td></tr>
	<tr><td>ok_rate_%</td><td>{{ printf "%.3f" .S.OKRatePct }}</td></tr>
	<tr><td>rps</td><td>{{ printf "%.2f" .S.RPS }}</td></tr>
	<tr><td>duration_s</td><td>{{ printf "%.3f" .S.DurationS }}</td></tr>
	<tr><td>p50_ms</td><td>{{ printf "%.6f" .S.P50ms }}</td></tr>
	<tr><td>p90_ms</td><td>{{ printf "%.6f" .S.P90ms }}</td></tr>
	<tr><td>p95_ms</td><td>{{ printf "%.6f" .S.P95ms }}</td></tr>
	<tr><td>p99_ms</td><td>{{ printf "%.6f" .S.P99ms }}</td></tr>
	<tr><td>mean_ms</td><td>{{ printf "%.6f" .S.Meanms }}</td></tr>
	<tr><td>min_ms</td><td>{{ printf "%.6f" .S.Minms }}</td></tr>
	<tr><td>max_ms</td><td>{{ printf "%.6f" .S.Maxms }}</td></tr>
</tbody>
</table>

<div class="grid">
<div>
	<h3>Latency CDF</h3>
	<canvas id="cdf" class="chart"></canvas>
</div>
<div>
	<h3>Throughput per Second</h3>
	<canvas id="thr" class="chart"></canvas>
</div>
</div>

<p style="margin-top:22px;color:#666">
Source columns: <span class="code">ts_unix_ns, latency_ns, ok</span>. Latency in ns; converted to ms.
</p>

<script>
const CDF_X = {{ .CDF_X_ms }};
const CDF_Y = {{ .CDF_Y }};
const THR_T = {{ .THR_Ts }};
const THR_V = {{ .THR_Val }};

new Chart(document.getElementById('cdf'), {
type: 'line',
data: { labels: CDF_X, datasets: [{ label: 'CDF', data: CDF_Y, pointRadius: 0, borderWidth: 1 }] },
options: {
	animation: false, parsing: false,
	scales: {
	x: { type: 'linear', title: { text: 'Latency (ms)', display: true } },
	y: { min: 0, max: 1, title: { text: 'CDF', display: true } }
	},
	elements: { line: { tension: 0 } }
}
});

new Chart(document.getElementById('thr'), {
type: 'line',
data: { labels: THR_T, datasets: [{ label: 'RPS', data: THR_V, pointRadius: 0, borderWidth: 1 }] },
options: {
	animation: false, parsing: false,
	scales: {
	x: { title: { text: 'Unix second', display: true } },
	y: { title: { text: 'Requests/sec', display: true } }
	},
	elements: { line: { tension: 0 } }
}
});
</script>
</body>
</html>`
