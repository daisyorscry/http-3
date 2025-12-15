# ===== build client (Go 1.25) =====
FROM golang:1.25 AS bench-client
WORKDIR /src

# Cache modul
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build ALL benchmark client binaries
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -trimpath -buildvcs=false -ldflags="-s -w" \
      -o /out/bench-client ./cmd/client/low-traffic && \
    go build -trimpath -buildvcs=false -ldflags="-s -w" \
      -o /out/bench-header-bloat ./cmd/client/header-bloat && \
    go build -trimpath -buildvcs=false -ldflags="-s -w" \
      -o /out/bench-parallel ./cmd/client/parallel-requests && \
    go build -trimpath -buildvcs=false -ldflags="-s -w" \
      -o /out/bench-burst ./cmd/client/burst-traffic && \
    go build -trimpath -buildvcs=false -ldflags="-s -w" \
      -o /out/bench-coldstart ./cmd/client/cold-start && \
    go build -trimpath -buildvcs=false -ldflags="-s -w" \
      -o /out/bench-churn ./cmd/client/connection-churn && \
    go build -trimpath -buildvcs=false -ldflags="-s -w" \
      -o /out/bench-mixed ./cmd/client/mixed-load && \
    go build -trimpath -buildvcs=false -ldflags="-s -w" \
      -o /out/bench-uplink ./cmd/client/uplink-loss && \
    go build -trimpath -buildvcs=false -ldflags="-s -w" \
      -o /out/bench-migration ./cmd/client/nat-rebinding && \
    go build -trimpath -buildvcs=false -ldflags="-s -w" \
      -o /out/bench-stress ./cmd/client/high-traffic

# ===== build dashboard (Node) =====
FROM node:20-alpine AS build-dashboard
WORKDIR /app

# Pasang deps
COPY dashboard-new/package*.json ./
RUN npm ci

# Build SvelteKit
COPY dashboard-new ./
RUN npm run build

# ===== runtime (dashboard + client binary) =====
FROM node:20-alpine AS runner
WORKDIR /app

# Install build dependencies for better-sqlite3
RUN apk add --no-cache python3 make g++

# App build hasil adapter-node akan di ./build
COPY --from=build-dashboard /app/build ./build
COPY --from=build-dashboard /app/package*.json ./
RUN npm ci --omit=dev

# Copy ALL benchmark binaries to PATH
COPY --from=bench-client /out/bench-client /usr/local/bin/bench-client
COPY --from=bench-client /out/bench-header-bloat /usr/local/bin/bench-header-bloat
COPY --from=bench-client /out/bench-parallel /usr/local/bin/bench-parallel
COPY --from=bench-client /out/bench-burst /usr/local/bin/bench-burst
COPY --from=bench-client /out/bench-coldstart /usr/local/bin/bench-coldstart
COPY --from=bench-client /out/bench-churn /usr/local/bin/bench-churn
COPY --from=bench-client /out/bench-mixed /usr/local/bin/bench-mixed
COPY --from=bench-client /out/bench-uplink /usr/local/bin/bench-uplink
COPY --from=bench-client /out/bench-migration /usr/local/bin/bench-migration
COPY --from=bench-client /out/bench-stress /usr/local/bin/bench-stress

# (opsional) kalau perlu non-root:
# USER node

EXPOSE 5000
CMD ["node","build/index.js"]
