export function pct(n: number, digits = 1) {
  if (!isFinite(n)) return "0%";
  return `${(n * 100).toFixed(digits)}%`;
}

export function ms(n: number, digits = 2) {
  if (n == null || !isFinite(n)) return "-";
  // sederhanakan: <1 ms tetap 2 desimal, >1 ms 2 desimal saja
  return `${n.toFixed(digits)} ms`;
}

export function verdictFromWinrate(win: number) {
  if (win >= 0.65) return { label: "Jelas unggul", tone: "good" };
  if (win >= 0.55) return { label: "Sedikit unggul", tone: "warn" };
  if (win > 0.45) return { label: "Imbang", tone: "neutral" };
  if (win > 0.35) return { label: "Sedikit kalah", tone: "warn" };
  return { label: "Jelas kalah", tone: "bad" };
}

export function explainOverall(overall: any) {
  // overall.winRates.latencyH3 & rpsH3; overall.latency.avgLatencyImprovementPct_vsH2; overall.throughput.avgRpsGainPct_vsH2
  const lWin = overall?.winRates?.latencyH3 ?? 0;
  const tWin = overall?.winRates?.rpsH3 ?? 0;
  const lImp = overall?.latency?.avgLatencyImprovementPct_vsH2 ?? 0; // + berarti H3 lebih cepat
  const tImp = overall?.throughput?.avgRpsGainPct_vsH2 ?? 0; // + berarti H3 lebih kencang

  const lat = verdictFromWinrate(lWin);
  const thr = verdictFromWinrate(tWin);

  const latPhrase =
    lImp > 0
      ? `HTTP/3 lebih cepat rata-rata ${pct(lImp, 2)}`
      : lImp < 0
      ? `HTTP/2 lebih cepat rata-rata ${pct(-lImp, 2)}`
      : "Kecepatan setara";

  const thrPhrase =
    tImp > 0
      ? `HTTP/3 throughput lebih tinggi ${pct(tImp, 2)}`
      : tImp < 0
      ? `HTTP/2 throughput lebih tinggi ${pct(-tImp, 2)}`
      : "Throughput setara";

  return {
    latency: {
      winrateText: `H3 win-rate: ${pct(lWin)}`,
      verdict: lat.label,
      detail: latPhrase,
      tone: lat.tone,
    },
    throughput: {
      winrateText: `H3 win-rate: ${pct(tWin)}`,
      verdict: thr.label,
      detail: thrPhrase,
      tone: thr.tone,
    },
  };
}
