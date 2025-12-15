package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/http2"

	"h3-vs-h2-k6/internal/echo"
)

func main() {
	var (
		addr    = flag.String("addr", ":8444", "listen addr (TLS/TCP)")
		cert    = flag.String("cert", "cert/dev.crt", "TLS cert")
		key     = flag.String("key", "cert/dev.key", "TLS key")
		verbose = flag.Bool("verbose", false, "enable verbose request logging")
	)
	flag.Parse()

	log.Printf("[HTTP/2] ====== SERVER STARTUP ======")
	log.Printf("[HTTP/2] pid=%d", os.Getpid())
	log.Printf("[HTTP/2] addr=%s", *addr)
	log.Printf("[HTTP/2] cert=%s key=%s", *cert, *key)
	log.Printf("[HTTP/2] verbose=%v", *verbose)
	log.Printf("[HTTP/2] =============================")

	logLevel := echo.LogLevelNormal
	if *verbose {
		logLevel = echo.LogLevelVerbose
	}

	s := &http.Server{
		Addr:    *addr,
		Handler: echo.NewMuxWithLogging(logLevel, "HTTP/2"),
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
			NextProtos: []string{"h2"},
		},
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	http2.ConfigureServer(s, &http2.Server{})

	log.Printf("[HTTP/2] gRPC server listening at https://localhost%s", *addr)
	log.Fatal(s.ListenAndServeTLS(*cert, *key))
}
