package core

import (
	"crypto/tls"
	"net/http"

	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
)

// NewHTTPClient creates an HTTP client (H2 or H3) with TLS config
// Returns the HTTP client and a cleanup function
func NewHTTPClient(useH3 bool, insecure bool, logger *Logger) (*http.Client, func()) {
	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: insecure,
	}

	if useH3 {
		tr := &http3.Transport{TLSClientConfig: tlsCfg}
		logger.Info("HTTP client initialized: HTTP/3 (QUIC) insecure=%v", insecure)
		return &http.Client{Transport: tr, Timeout: 0}, func() { tr.CloseIdleConnections() }
	}

	h2 := &http.Transport{
		TLSClientConfig:   tlsCfg,
		ForceAttemptHTTP2: true,
	}
	_ = http2.ConfigureTransport(h2)
	logger.Info("HTTP client initialized: HTTP/2 (TCP) insecure=%v", insecure)
	return &http.Client{Transport: h2, Timeout: 0}, func() {}
}

// ProtocolName returns human-readable protocol name
func ProtocolName(useH3 bool) string {
	if useH3 {
		return "HTTP/3"
	}
	return "HTTP/2"
}
