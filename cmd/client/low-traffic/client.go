package main

import (
	"crypto/tls"
	"log"
	"net/http"

	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
)

// newHTTPClient membangun HTTP client (H2 atau H3) dengan TLS config
// Return HTTP client yang sudah dikonfigurasi + cleanup function
func newHTTPClient(h3 bool, insecure bool) (*http.Client, func()) {
	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: insecure, // dev only
	}
	if h3 {
		tr := &http3.Transport{TLSClientConfig: tlsCfg}
		log.Printf("[http] using HTTP/3 transport (insecure=%v)", insecure)
		return &http.Client{Transport: tr, Timeout: 0}, func() { tr.CloseIdleConnections() }
	}
	h2 := &http.Transport{
		TLSClientConfig:   tlsCfg,
		ForceAttemptHTTP2: true,
	}
	_ = http2.ConfigureTransport(h2)
	log.Printf("[http] using HTTP/2 transport (insecure=%v)", insecure)
	return &http.Client{Transport: h2, Timeout: 0}, func() {}
}
