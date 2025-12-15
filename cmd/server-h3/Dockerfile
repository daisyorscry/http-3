# ===== builder (Go 1.25) =====
FROM golang:1.25 AS builder
WORKDIR /app

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
ENV GOMAXPROCS=2 GOMEMLIMIT=1GiB GOGC=80

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -p 1 -trimpath -buildvcs=false -ldflags="-s -w" \
      -o /out/server-h3 ./cmd/server-h3

FROM gcr.io/distroless/static-debian12
WORKDIR /app

# Binary + cert dev (struktur kamu: /cert/dev.crt & /cert/dev.key)
COPY --from=builder /out/server-h3 /app/server-h3
COPY --chown=65532:65532 cert /app/cert
ENV TLS_CERT_FILE=/app/cert/dev.crt
ENV TLS_KEY_FILE=/app/cert/dev.key
ENV ADDR=:8443

# UDP port
EXPOSE 8443/udp
USER 65532:65532
ENTRYPOINT ["/app/server-h3"]