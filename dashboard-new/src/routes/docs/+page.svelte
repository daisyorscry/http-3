<script lang="ts">
  import {
    Server,
    ServerCog,
    Cable,
    Gauge,
    GitCompare,
    BarChart3,
    FileText,
  } from "lucide-svelte";
</script>

<div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
  <!-- Header -->
  <header class="mb-6">
    <div class="card px-4 sm:px-6 lg:px-8 py-6">
      <div class="flex items-center gap-3">
        <FileText class="h-6 w-6 text-primary-600" aria-hidden="true" />
        <h1
          class="text-2xl sm:text-3xl font-bold text-gray-900 dark:text-gray-100"
        >
          Documentation: gRPC over HTTP/2 vs HTTP/3
        </h1>
      </div>
      <p class="mt-2 text-sm muted">
        Arsitektur server, client, pipeline metrik, dan logika komparasi yang
        digunakan di dashboard ini.
      </p>
    </div>
  </header>

  <!-- Contents -->
  <div class="space-y-8">
    <!-- TOC -->
    <nav class="card p-4">
      <div class="text-sm font-semibold text-gray-800 dark:text-gray-100 mb-2">
        On this page
      </div>
      <ul class="text-sm muted grid sm:grid-cols-2 gap-2">
        <li>
          <a
            href="#overview"
            class="hover:text-primary-700 dark:hover:text-primary-400 underline-offset-2 hover:underline focus:outline-none focus:ring-2 focus:ring-primary-600 rounded"
            >Overview</a
          >
        </li>
        <li>
          <a
            href="#servers"
            class="hover:text-primary-700 dark:hover:text-primary-400 underline-offset-2 hover:underline focus:outline-none focus:ring-2 focus:ring-primary-600 rounded"
            >Servers (H2 &amp; H3)</a
          >
        </li>
        <li>
          <a
            href="#client"
            class="hover:text-primary-700 dark:hover:text-primary-400 underline-offset-2 hover:underline focus:outline-none focus:ring-2 focus:ring-primary-600 rounded"
            >Client &amp; Scenarios</a
          >
        </li>
        <li>
          <a
            href="#metrics"
            class="hover:text-primary-700 dark:hover:text-primary-400 underline-offset-2 hover:underline focus:outline-none focus:ring-2 focus:ring-primary-600 rounded"
            >Metrics &amp; Data</a
          >
        </li>
        <li>
          <a
            href="#compare"
            class="hover:text-primary-700 dark:hover:text-primary-400 underline-offset-2 hover:underline focus:outline-none focus:ring-2 focus:ring-primary-600 rounded"
            >Comparison Logic</a
          >
        </li>
        <li>
          <a
            href="#api"
            class="hover:text-primary-700 dark:hover:text-primary-400 underline-offset-2 hover:underline focus:outline-none focus:ring-2 focus:ring-primary-600 rounded"
            >Dashboard API</a
          >
        </li>
        <li>
          <a
            href="#run"
            class="hover:text-primary-700 dark:hover:text-primary-400 underline-offset-2 hover:underline focus:outline-none focus:ring-2 focus:ring-primary-600 rounded"
            >Run Locally</a
          >
        </li>
      </ul>
    </nav>

    <!-- Overview -->
    <section id="overview" class="card p-6 scroll-mt-20">
      <div class="flex items-center gap-2 mb-3">
        <BarChart3 class="h-5 w-5 text-primary-600" aria-hidden="true" />
        <h2 class="text-xl font-semibold text-gray-900 dark:text-gray-100">
          Overview
        </h2>
      </div>
      <p class="text-sm leading-6 text-gray-700 dark:text-gray-300">
        Tujuannya sederhana: membandingkan kinerja gRPC di atas HTTP/2 (TCP/TLS)
        dan HTTP/3 (QUIC/TLS 1.3). Dashboard menjalankan benchmark lewat Go
        client, menulis hasil ke CSV, lalu merangkum metrik seperti latency
        percentiles dan RPS. Di bawah ini dibahas cara server/klien disusun,
        bagaimana data diambil, dan bagaimana hasilnya dibandingkan.
      </p>
    </section>

    <!-- Servers -->
    <section id="servers" class="card p-6 scroll-mt-20">
      <div class="flex items-center gap-2 mb-3">
        <Server class="h-5 w-5 text-primary-600" aria-hidden="true" />
        <h2 class="text-xl font-semibold text-gray-900 dark:text-gray-100">
          Servers (gRPC over H2 &amp; H3)
        </h2>
      </div>

      <div class="space-y-5 text-sm leading-6 text-gray-700 dark:text-gray-300">
        <div>
          <div class="font-semibold text-gray-900 dark:text-gray-100">
            HTTP/2 Server
          </div>
          <p>
            Server H2 menggunakan <code>net/http</code> +
            <code>http2.ConfigureServer</code>
            dengan TLS 1.3 dan <code>NextProtos: ["h2"]</code>.
          </p>
          <p class="mt-1 muted">File: <code>cmd/server-h2/main.go:1</code></p>
        </div>

        <div>
          <div class="font-semibold text-gray-900 dark:text-gray-100">
            HTTP/3 Server
          </div>
          <p>
            Server H3 menggunakan <code
              >github.com/quic-go/quic-go/http3.Server</code
            >, TLS 1.3 dan <code>NextProtos: ["h3"]</code>, dengan konfigurasi
            QUIC.
          </p>
          <p class="mt-1 muted">File: <code>cmd/server-h3/main.go:1</code></p>
        </div>

        <div>
          <div class="font-semibold text-gray-900 dark:text-gray-100">
            gRPC Service (Connect)
          </div>
          <p>
            Service disediakan via <code>connectrpc</code> untuk Echo RPC. Handler
            di-mount pada mux HTTP.
          </p>
          <p class="muted">File: <code>internal/echo/handler.go:1</code></p>
        </div>

        <div>
          <div class="font-semibold text-gray-900 dark:text-gray-100">
            Cara serve (ringkas &amp; langkah)
          </div>
          <ol class="list-decimal pl-5 mt-1 space-y-1">
            <li>
              Siapkan sertifikat TLS (dev: <code>cert/dev.crt</code>,
              <code>cert/dev.key</code>).
            </li>
            <li>Buat mux gRPC/Connect: <code>echo.NewMux()</code>.</li>
            <li>
              H2: <code>*http.Server</code> + <code>http2.ConfigureServer</code>
              + <code>NextProtos: ["h2"]</code>.
            </li>
            <li>
              H3: <code>http3.Server</code> (quic-go) +
              <code>NextProtos: ["h3"]</code>.
            </li>
            <li>
              Jalankan di port berbeda: H2 <code>8444</code> (TCP), H3
              <code>8443</code> (UDP/QUIC).
            </li>
          </ol>

          <div class="mt-3 grid sm:grid-cols-2 gap-3">
            <div class="surface p-3 text-[12px]">
              <div class="font-semibold">Contoh H2</div>
              <pre
                class="mt-2 bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-800 rounded p-3 overflow-x-auto"><code
                  >s := &amp;http.Server&#123;Addr: ":8444", Handler: echo.NewMux(), TLSConfig: &amp;tls.Config&#123;MinVersion: tls.VersionTLS13, NextProtos: []string&#123;"h2"&#125;&#125;&#125;
http2.ConfigureServer(s, &amp;http2.Server&#123;&#125;)
log.Fatal(s.ListenAndServeTLS("cert/dev.crt", "cert/dev.key"))</code
                ></pre>
            </div>
            <div class="surface p-3 text-[12px]">
              <div class="font-semibold">Contoh H3</div>
              <pre
                class="mt-2 bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-800 rounded p-3 overflow-x-auto"><code
                  >s := &amp;http3.Server&#123;Addr: ":8443", Handler: echo.NewMux(), TLSConfig: &amp;tls.Config&#123;MinVersion: tls.VersionTLS13, NextProtos: []string&#123;"h3"&#125;&#125;&#125;
log.Fatal(s.ListenAndServeTLS("cert/dev.crt", "cert/dev.key"))</code
                ></pre>
            </div>
          </div>
        </div>

        <div>
          <div class="font-semibold text-gray-900 dark:text-gray-100">
            Perekaman request (client-side)
          </div>
          <p>Setiap request direkam langsung di klien dengan urutan berikut:</p>
          <ol class="list-decimal pl-5 mt-1 space-y-1">
            <li>
              Siapkan pesan dan header (opsional: <code>x-meta-bloat</code>).
            </li>
            <li>Ambil timestamp awal: <code>t0 := time.Now()</code>.</li>
            <li>Panggil RPC: <code>_, err := cl.Unary(ctx, req)</code>.</li>
            <li>
              Hitung latency: <code>lat := time.Since(t0)</code> (jam monotonik).
            </li>
            <li>
              Bentuk sampel: <code
                >rec&#123;TsUnixNS: t0.UnixNano(), LatencyNS: lat.Nanoseconds(),
                OK: err==nil&#125;</code
              >.
            </li>
            <li>
              Kirim ke channel besar non-blocking; jika buffer penuh, sampel
              boleh drop supaya jalur RPC tidak tersendat.
            </li>
          </ol>
          <pre
            class="mt-2 bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-800 rounded p-3 overflow-x-auto text-[12px]"><code
              >func doOne(...) &#123;
  req := connect.NewRequest(&amp;echov1.EchoRequest&#123;Message: "ping", Payload: make([]byte, payload)&#125;)
  if bloat &gt; 0 &#123; req.Header().Set("x-meta-bloat", string(make([]byte, bloat))) &#125;
  t0 := time.Now()
  _, err := cl.Unary(ctx, req)
  lat := time.Since(t0)
  ok := err == nil
  if ok &#123; totalOK.Add(1) &#125; else &#123; totalErr.Add(1) &#125;
  select &#123; case latCh &lt;- rec&#123;TsUnixNS: t0.UnixNano(), LatencyNS: lat.Nanoseconds(), OK: ok&#125;: default: &#125;
&#125;</code
            ></pre>
        </div>
      </div>
    </section>

    <!-- Client -->
    <section id="client" class="card p-6 scroll-mt-20">
      <div class="flex items-center gap-2 mb-3">
        <ServerCog class="h-5 w-5 text-primary-600" aria-hidden="true" />
        <h2 class="text-xl font-semibold text-gray-900 dark:text-gray-100">
          Client &amp; Scenario Generator
        </h2>
      </div>

      <div class="space-y-5 text-sm leading-6 text-gray-700 dark:text-gray-300">
        <div>
          <div class="font-semibold text-gray-900 dark:text-gray-100">
            Transport
          </div>
          <p>Client membangun <code>http.Client</code> seperti berikut:</p>
          <ul class="list-disc pl-5 mt-1">
            <li>HTTP/3: <code>http3.Transport</code> (quic-go) + TLS 1.3</li>
            <li>
              HTTP/2: <code>http.Transport</code> +
              <code>http2.ConfigureTransport</code> + TLS 1.3
            </li>
          </ul>
          <p class="mt-1 muted">
            File: <code>cmd/client/low-traffic/client.go:1</code>
          </p>
          <pre
            class="mt-2 bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-800 rounded p-3 overflow-x-auto text-[12px]"><code
              >func newHTTPClient(h3 bool, insecure bool) *http.Client &#123;
  tlsCfg := &amp;tls.Config&#123;MinVersion: tls.VersionTLS13, InsecureSkipVerify: insecure&#125;
  if h3 &#123; return &amp;http.Client&#123;Transport: &amp;http3.Transport&#123;TLSClientConfig: tlsCfg&#125;&#125; &#125;
  h2 := &amp;http.Transport&#123;TLSClientConfig: tlsCfg, ForceAttemptHTTP2: true&#125;
  _ = http2.ConfigureTransport(h2)
  return &amp;http.Client&#123;Transport: h2&#125;
&#125;</code
            ></pre>
        </div>

        <div>
          <div class="font-semibold text-gray-900 dark:text-gray-100">
            Scenarios
          </div>
          <p>
            Generator mendukung <code>low</code> (periodic), <code>medium</code>
            (constant RPS), dan <code>high</code> (ramp RPS). Dashboard memetakan
            10 skenario UI ke tiga mode ini.
          </p>
          <p class="muted">File: <code>cmd/client/low-traffic/*.go</code></p>
          <ul class="list-disc pl-5 mt-1">
            <li>
              <b>low</b>: tiap worker kirim request periodik dengan
              <code>period</code>
              + <code>jitter</code>.
            </li>
            <li>
              <b>medium</b>: dispatcher menjaga laju global <code>rps</code> konstan.
            </li>
            <li>
              <b>high</b>: dispatcher menjalankan serangkaian stage
              <code>dur@rps</code> (ramp naik/turun).
            </li>
          </ul>
        </div>

        <div>
          <div class="font-semibold text-gray-900 dark:text-gray-100">
            Request Path
          </div>
          <p>
            RPC target: <code>EchoService/Unary</code> dengan payload adjustable
            dan opsi header bloat (<code>x-meta-bloat</code>).
          </p>
          <p class="muted">File: <code>internal/echo/handler.go:1</code></p>
          <pre
            class="mt-2 bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-800 rounded p-3 overflow-x-auto text-[12px]"><code
              >req := connect.NewRequest(&amp;echov1.EchoRequest&#123;Message: "ping", Payload: make([]byte, payload)&#125;)
if bloat &gt; 0 &#123; req.Header().Set("x-meta-bloat", string(make([]byte, bloat))) &#125;
t0 := time.Now(); _, err := cl.Unary(ctx, req); lat := time.Since(t0)</code
            ></pre>
        </div>
      </div>
    </section>

    <!-- Metrics -->
    <section id="metrics" class="card p-6 scroll-mt-20">
      <div class="flex items-center gap-2 mb-3">
        <Gauge class="h-5 w-5 text-primary-600" aria-hidden="true" />
        <h2 class="text-xl font-semibold text-gray-900 dark:text-gray-100">
          Metrics &amp; Data Pipeline
        </h2>
      </div>

      <div class="space-y-5 text-sm leading-6 text-gray-700 dark:text-gray-300">
        <div>
          <div class="font-semibold text-gray-900 dark:text-gray-100">
            Pengumpulan Sampel
          </div>
          <p>
            Client merekam setiap request sebagai baris <code>rec</code>:
            timestamp (ns), latency (ns), dan status OK. Data dikirim ke
            channel-buffer besar agar perekaman non-blocking.
          </p>
          <p class="muted">
            File: <code>cmd/client/low-traffic/worker.go:1</code>
          </p>
          <pre
            class="mt-2 bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-800 rounded p-3 overflow-x-auto text-[12px]"><code
              >select &#123; case latCh &lt;- rec&#123;TsUnixNS: t0.UnixNano(), LatencyNS: lat.Nanoseconds(), OK: err==nil&#125;: default: &#125;</code
            ></pre>
        </div>

        <div>
          <div class="font-semibold text-gray-900 dark:text-gray-100">
            Ringkasan (Summary)
          </div>
          <p>
            Menghitung: Samples, OK rate, RPS, Duration, P50/P90/P99, Mean, Min,
            Max, CDF, throughput/s.
          </p>
          <p class="muted">
            File: <code>cmd/client/low-traffic/summary.go:1</code>
          </p>
          <pre
            class="mt-2 bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-800 rounded p-3 overflow-x-auto text-[12px]"><code
              >durationS := (maxTS - minTS) / 1e9
rps := float64(len(all)) / math.Max(durationS, 1e-9)
// Percentile p: pos = p*(n-1); i=floor(pos); f=pos-i; P = a[i] + f*(a[i+1]-a[i])</code
            ></pre>

          <div class="grid sm:grid-cols-2 gap-3 mt-2">
            <div class="surface p-3">
              <div class="font-semibold">Alasan metode</div>
              <ul class="list-disc pl-5 mt-1">
                <li>Interpolasi percentile stabil untuk n besar.</li>
                <li>
                  Durasi dari data (minTS..maxTS) merefleksikan waktu efektif.
                </li>
                <li>Histogram per detik mudah dibaca dan tahan jitter.</li>
              </ul>
            </div>
            <div class="surface p-3">
              <div class="font-semibold">Rounding &amp; I/O</div>
              <ul class="list-disc pl-5 mt-1">
                <li>
                  Rounding untuk konsistensi tampilan; perhitungan tetap presisi
                  penuh.
                </li>
                <li>
                  I/O dieksekusi setelah tes agar tidak mengganggu jalur panas.
                </li>
              </ul>
            </div>
          </div>
        </div>

        <div>
          <div class="font-semibold text-gray-900 dark:text-gray-100">
            Ekspor CSV &amp; HTML
          </div>
          <p>
            CSV format: <code>ts_unix_ns,latency_ns,ok</code>. HTML berisi chart
            CDF &amp; throughput (Chart.js).
          </p>
          <p class="muted">
            File: <code>cmd/client/low-traffic/output.go:1</code>
          </p>
        </div>
      </div>
    </section>

    <!-- Compare -->
    <section id="compare" class="card p-6 scroll-mt-20">
      <div class="flex items-center gap-2 mb-3">
        <GitCompare class="h-5 w-5 text-primary-600" aria-hidden="true" />
        <h2 class="text-xl font-semibold text-gray-900 dark:text-gray-100">
          Comparison Logic (H2 vs H3)
        </h2>
      </div>

      <div class="space-y-5 text-sm leading-6 text-gray-700 dark:text-gray-300">
        <div>
          <div class="font-semibold text-gray-900 dark:text-gray-100">
            Langkah
          </div>
          <ol class="list-decimal pl-5 mt-1 space-y-1">
            <li>Jalankan benchmark H2 dan H3 dengan konfigurasi yang sama.</li>
            <li>
              Parse CSV masing-masing menjadi <code>BenchmarkSummary</code>.
            </li>
            <li>Hitung delta P50, P99, dan RPS (persentase).</li>
            <li>
              Tentukan pemenang: latency winner = P50 lebih kecil; throughput
              winner = RPS lebih besar.
            </li>
          </ol>
          <p class="mt-1 muted">
            Code: <code
              >dashboard-new/src/routes/api/benchmark/compare/+server.ts:1</code
            >
          </p>
        </div>

        <div>
          <div class="font-semibold text-gray-900 dark:text-gray-100">
            Metode Perhitungan
          </div>
          <ul class="list-disc pl-5 mt-1">
            <li>
              <code>p50Diff = ((h2.P50 - h3.P50) / h2.P50) * 100</code> (positif
              = H3 lebih cepat)
            </li>
            <li><code>p99Diff = ((h2.P99 - h3.P99) / h2.P99) * 100</code></li>
            <li><code>rpsDiff = ((h3.RPS - h2.RPS) / h2.RPS) * 100</code></li>
          </ul>
          <p class="mt-1">
            Penentuan pemenang mengikuti aturan di atas; nilai <b
              >latencyImprovement</b
            >
            dapat diambil rata-rata dari <code>p50Diff</code> dan
            <code>p99Diff</code>.
          </p>
        </div>

        <div>
          <div class="font-semibold text-gray-900 dark:text-gray-100">
            Fairness
          </div>
          <ul class="list-disc pl-5 mt-1">
            <li>
              H2 dan H3 dijalankan berurutan (tidak paralel), menghindari
              kontensi CPU/bandwidth.
            </li>
            <li>Parameter identik: clients, payload, durasi, pola beban.</li>
            <li>
              Selama tes, logging dimatikan; CSV/HTML ditulis setelah selesai.
            </li>
            <li>
              Sampling non-blocking; drop kecil diperbolehkan untuk menjaga
              jalur panas.
            </li>
          </ul>
        </div>
      </div>
    </section>

    <!-- API -->
    <section id="api" class="card p-6 scroll-mt-20">
      <div class="flex items-center gap-2 mb-3">
        <Cable class="h-5 w-5 text-primary-600" aria-hidden="true" />
        <h2 class="text-xl font-semibold text-gray-900 dark:text-gray-100">
          Dashboard API
        </h2>
      </div>

      <div class="space-y-5 text-sm leading-6 text-gray-700 dark:text-gray-300">
        <div>
          <div class="font-semibold text-gray-900 dark:text-gray-100">
            Single Benchmark
          </div>
          <p>
            Endpoint memanggil Go client via <code
              >go run ./cmd/client/low-traffic</code
            >, kemudian mem-parse CSV untuk membentuk <code>summary</code>.
          </p>
          <p class="muted">
            File: <code
              >dashboard-new/src/routes/api/benchmark/+server.ts:1</code
            >
          </p>
          <pre
            class="mt-2 bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-800 rounded p-3 overflow-x-auto text-[12px]"><code
              >POST /api/benchmark → spawn Go client → tulis CSV ke /tmp → parse → kembalikan JSON</code
            ></pre>
        </div>

        <div>
          <div class="font-semibold text-gray-900 dark:text-gray-100">
            Comparison
          </div>
          <p>
            Endpoint menjalankan H2 lalu H3, menyatukan hasil sebagai <code
              >ComparisonResult</code
            > (H2, H3, plus metrik perbandingan).
          </p>
          <p class="muted">
            File: <code
              >dashboard-new/src/routes/api/benchmark/compare/+server.ts:1</code
            >
          </p>
          <div class="surface p-3 text-[12px]">
            <div class="font-semibold">Alasan parse CSV di server</div>
            <ul class="list-disc pl-5 mt-1">
              <li>Sumber data tunggal (CSV) → pipeline seragam.</li>
              <li>Parsing setelah tes → tidak menambah beban runtime.</li>
              <li>
                Algoritma sederhana (sort + percentile) efisien untuk ratusan
                ribu sampel.
              </li>
            </ul>
          </div>
        </div>
      </div>
    </section>

    <!-- Run -->
    <section id="run" class="card p-6 scroll-mt-20">
      <div class="flex items-center gap-2 mb-3">
        <Server class="h-5 w-5 text-primary-600" aria-hidden="true" />
        <h2 class="text-xl font-semibold text-gray-900 dark:text-gray-100">
          Run Locally
        </h2>
      </div>

      <ol
        class="list-decimal pl-5 text-sm leading-6 text-gray-700 dark:text-gray-300"
      >
        <li>
          Jalankan server H2 dan H3 (port 8444 dan 8443): <code
            >go run ./cmd/server-h2</code
          >, <code>go run ./cmd/server-h3</code>
        </li>
        <li>
          Jalankan dashboard (SvelteKit) dan buka halaman utama untuk
          menjalankan benchmark.
        </li>
        <li>Gunakan halaman ini sebagai referensi arsitektur dan metrik.</li>
      </ol>
      <p class="text-xs muted mt-3">
        Cert dev ada di <code>cert/dev.crt</code> dan <code>cert/dev.key</code>.
        Untuk lingkungan produksi, gunakan sertifikat yang valid.
      </p>
    </section>
  </div>
</div>
