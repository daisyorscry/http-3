FROM golang:1.25 AS builder
WORKDIR /app
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOMAXPROCS=2 GOMEMLIMIT=1GiB GOGC=80
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -p 1 -trimpath -buildvcs=false -ldflags="-s -w" \
      -o /out/server-h2 ./cmd/server-h2

FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=builder /out/server-h2 /app/server-h2
COPY --chown=65532:65532 cert /app/cert
ENV ADDR=:8444
EXPOSE 8444
USER 65532:65532
ENTRYPOINT ["/app/server-h2"]
