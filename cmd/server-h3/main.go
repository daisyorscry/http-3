package main

import (
	"crypto/tls"
	"flag"
	"log"
	"os"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"

	"h3-vs-h2-k6/internal/echo"
)

func main() {
	var (
		addr    = flag.String("addr", ":8443", "listen addr (UDP/QUIC)")
		cert    = flag.String("cert", "cert/dev.crt", "TLS cert")
		key     = flag.String("key", "cert/dev.key", "TLS key")
		verbose = flag.Bool("verbose", false, "enable verbose request logging")
	)
	flag.Parse()

	log.Printf("[HTTP/3] ====== SERVER STARTUP ======")
	log.Printf("[HTTP/3] pid=%d", os.Getpid())
	log.Printf("[HTTP/3] addr=%s (UDP/QUIC)", *addr)
	log.Printf("[HTTP/3] cert=%s key=%s", *cert, *key)
	log.Printf("[HTTP/3] verbose=%v", *verbose)
	log.Printf("[HTTP/3] =============================")

	logLevel := echo.LogLevelNormal
	if *verbose {
		logLevel = echo.LogLevelVerbose
	}

	s := &http3.Server{
		Addr:    *addr,
		Handler: echo.NewMuxWithLogging(logLevel, "HTTP/3"),
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
			NextProtos: []string{"h3"},
		},
		QUICConfig: &quic.Config{
			HandshakeIdleTimeout: 10 * time.Second,
			MaxIdleTimeout:       15 * time.Second,
		},
	}

	log.Printf("[HTTP/3] gRPC server listening at https://localhost%s", *addr)
	log.Fatal(s.ListenAndServeTLS(*cert, *key))
}
