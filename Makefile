# Makefile untuk HTTP/3 vs HTTP/2 Benchmark Project
# Author: Daisy
# Description: Automation untuk build, run, dan test benchmark tools

.PHONY: help install build clean run-servers run-dashboard proto dev all \
	test-baseline test-burst test-coldstart test-parallel test-header-bloat \
	test-uplink test-churn test-migration test-mixed test-stress \
	test-h3-baseline test-h3-burst test-h3-coldstart test-h3-parallel test-h3-header-bloat \
	test-h3-uplink test-h3-churn test-h3-migration test-h3-mixed test-h3-stress \
	test-all-h2 test-all-h3 \
	compare-baseline compare-burst compare-coldstart compare-parallel compare-header-bloat \
	compare-uplink compare-churn compare-migration compare-mixed compare-stress compare-all \
	docker-build docker-up docker-down docker-restart docker-logs docker-clean \
	docker-test docker-run-all docker-status

# Default target
.DEFAULT_GOAL := help

##@ General

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Installation & Setup

install: install-go install-dashboard ## Install all dependencies (Go + Dashboard)

install-go: ## Install Go module dependencies
	@echo "📦 Installing Go dependencies..."
	go mod download
	go mod tidy

install-dashboard: ## Install dashboard dependencies (npm)
	@echo "📦 Installing dashboard dependencies..."
	cd dashboard-new && npm install

.PHONY: install-db-optional
install-db-optional: ## (Optional) Install DB modules (@mikro-orm/better-sqlite)
	@echo "📦 Installing optional DB modules (@mikro-orm/core and @mikro-orm/better-sqlite)..."
	@echo "   Note: requires Node build toolchain (Python, C/C++ compiler)."
	cd dashboard-new && npm install @mikro-orm/core @mikro-orm/better-sqlite || echo "(optional) install skipped/failed"

proto: ## Generate protobuf code
	@echo "🔨 Generating protobuf code..."
	buf generate

##@ Build

build: build-servers build-client build-dashboard ## Build all components

build-servers: ## Build HTTP/2 and HTTP/3 servers
	@echo "🔨 Building servers..."
	go build -o bin/bench-server-h2 ./cmd/server-h2
	go build -o bin/bench-server-h3 ./cmd/server-h3
	@echo "✅ Servers built: bin/bench-server-h2, bin/bench-server-h3"

build-client: ## Build all 10 benchmark clients
	@echo "🔨 Building all 10 clients..."
	go build -o bin/bench-client ./cmd/client/low-traffic
	go build -o bin/bench-header-bloat ./cmd/client/header-bloat
	go build -o bin/bench-parallel ./cmd/client/parallel-requests
	go build -o bin/bench-burst ./cmd/client/burst-traffic
	go build -o bin/bench-coldstart ./cmd/client/cold-start
	go build -o bin/bench-churn ./cmd/client/connection-churn
	go build -o bin/bench-mixed ./cmd/client/mixed-load
	go build -o bin/bench-uplink ./cmd/client/uplink-loss
	go build -o bin/bench-migration ./cmd/client/nat-rebinding
	go build -o bin/bench-stress ./cmd/client/high-traffic
	@echo "✅ All 10 clients built in bin/"

build-dashboard: ## Build dashboard for production
	@echo "🔨 Building dashboard..."
	cd dashboard-new && npm run build
	@echo "✅ Dashboard built: dashboard-new/build/"

install-bins: build-servers build-client ## Install all binaries to /usr/local/bin
	@echo "📦 Installing binaries to /usr/local/bin..."
	sudo cp bin/bench-server-h2 /usr/local/bin/
	sudo cp bin/bench-server-h3 /usr/local/bin/
	sudo cp bin/bench-client /usr/local/bin/
	sudo cp bin/bench-header-bloat /usr/local/bin/
	sudo cp bin/bench-parallel /usr/local/bin/
	sudo cp bin/bench-burst /usr/local/bin/
	sudo cp bin/bench-coldstart /usr/local/bin/
	sudo cp bin/bench-churn /usr/local/bin/
	sudo cp bin/bench-mixed /usr/local/bin/
	sudo cp bin/bench-uplink /usr/local/bin/
	sudo cp bin/bench-migration /usr/local/bin/
	sudo cp bin/bench-stress /usr/local/bin/
	@echo "✅ All binaries installed to /usr/local/bin"

##@ Run

run-servers: ## Run both HTTP/2 and HTTP/3 servers
	@echo "🚀 Starting servers..."
	@echo "  - HTTP/2 server on :8444"
	@echo "  - HTTP/3 server on :8443"
	@trap 'kill 0' INT; \
	go run ./cmd/server-h2 & \
	go run ./cmd/server-h3 & \
	wait

run-h2: ## Run HTTP/2 server only
	@echo "🚀 Starting HTTP/2 server on :8444..."
	go run ./cmd/server-h2

run-h3: ## Run HTTP/3 server only
	@echo "🚀 Starting HTTP/3 server on :8443..."
	go run ./cmd/server-h3

run-dashboard: ## Run dashboard in development mode
	@echo "🌐 Starting dashboard on http://localhost:5000..."
	cd dashboard-new && npm run dev

run-dashboard-prod: build-dashboard ## Run dashboard in production mode
	@echo "🌐 Starting dashboard (production) on http://localhost:5000..."
	cd dashboard-new && npm start

##@ Development

dev: ## Run servers + dashboard concurrently for development
	@echo "🚀 Starting development environment..."
	@echo "  - HTTP/2 server: :8444"
	@echo "  - HTTP/3 server: :8443"
	@echo "  - Dashboard: http://localhost:5000"
	@trap 'kill 0' INT; \
	go run ./cmd/server-h2 --verbose & \
	go run ./cmd/server-h3 --verbose & \
	cd dashboard-new && npm run dev & \
	wait

dev-servers: run-servers ## Alias for run-servers

##@ Testing - All 10 Scenarios (Fixed Config)

test-baseline: ## Run baseline (low-traffic) scenario on HTTP/2
	@echo "📊 Running BASELINE scenario (HTTP/2)..."
	go run ./cmd/client/low-traffic --addr https://localhost:8444 --h3=false

test-burst: ## Run burst traffic scenario on HTTP/2
	@echo "📊 Running BURST TRAFFIC scenario (HTTP/2)..."
	go run ./cmd/client/burst-traffic --addr https://localhost:8444 --h3=false

test-coldstart: ## Run cold-start scenario on HTTP/2
	@echo "📊 Running COLD-START scenario (HTTP/2)..."
	go run ./cmd/client/cold-start --addr https://localhost:8444 --h3=false --mode cold

test-parallel: ## Run parallel streams scenario on HTTP/2
	@echo "📊 Running PARALLEL STREAMS scenario (HTTP/2)..."
	go run ./cmd/client/parallel-requests --addr https://localhost:8444 --h3=false

test-header-bloat: ## Run header bloat scenario on HTTP/2
	@echo "📊 Running HEADER BLOAT scenario (HTTP/2)..."
	go run ./cmd/client/header-bloat --addr https://localhost:8444 --h3=false

test-uplink: ## Run uplink loss scenario on HTTP/2
	@echo "📊 Running UPLINK LOSS scenario (HTTP/2)..."
	go run ./cmd/client/uplink-loss --addr https://localhost:8444 --h3=false

test-churn: ## Run connection churn scenario on HTTP/2
	@echo "📊 Running CONNECTION CHURN scenario (HTTP/2)..."
	go run ./cmd/client/connection-churn --addr https://localhost:8444 --h3=false

test-migration: ## Run NAT rebinding/migration scenario on HTTP/2
	@echo "📊 Running NAT REBINDING scenario (HTTP/2)..."
	go run ./cmd/client/nat-rebinding --addr https://localhost:8444 --h3=false

test-mixed: ## Run mixed load scenario on HTTP/2
	@echo "📊 Running MIXED LOAD scenario (HTTP/2)..."
	go run ./cmd/client/mixed-load --addr https://localhost:8444 --h3=false

test-stress: ## Run high traffic stress test on HTTP/2
	@echo "📊 Running STRESS TEST scenario (HTTP/2)..."
	go run ./cmd/client/high-traffic --addr https://localhost:8444 --h3=false

# HTTP/3 versions
test-h3-baseline: ## Run baseline scenario on HTTP/3
	@echo "📊 Running BASELINE scenario (HTTP/3)..."
	go run ./cmd/client/low-traffic --addr https://localhost:8443 --h3=true

test-h3-burst: ## Run burst traffic scenario on HTTP/3
	@echo "📊 Running BURST TRAFFIC scenario (HTTP/3)..."
	go run ./cmd/client/burst-traffic --addr https://localhost:8443 --h3=true

test-h3-coldstart: ## Run cold-start scenario on HTTP/3
	@echo "📊 Running COLD-START scenario (HTTP/3)..."
	go run ./cmd/client/cold-start --addr https://localhost:8443 --h3=true --mode cold

test-h3-parallel: ## Run parallel streams scenario on HTTP/3
	@echo "📊 Running PARALLEL STREAMS scenario (HTTP/3)..."
	go run ./cmd/client/parallel-requests --addr https://localhost:8443 --h3=true

test-h3-header-bloat: ## Run header bloat scenario on HTTP/3
	@echo "📊 Running HEADER BLOAT scenario (HTTP/3)..."
	go run ./cmd/client/header-bloat --addr https://localhost:8443 --h3=true

test-h3-uplink: ## Run uplink loss scenario on HTTP/3
	@echo "📊 Running UPLINK LOSS scenario (HTTP/3)..."
	go run ./cmd/client/uplink-loss --addr https://localhost:8443 --h3=true

test-h3-churn: ## Run connection churn scenario on HTTP/3
	@echo "📊 Running CONNECTION CHURN scenario (HTTP/3)..."
	go run ./cmd/client/connection-churn --addr https://localhost:8443 --h3=true

test-h3-migration: ## Run NAT rebinding/migration scenario on HTTP/3
	@echo "📊 Running NAT REBINDING scenario (HTTP/3)..."
	go run ./cmd/client/nat-rebinding --addr https://localhost:8443 --h3=true

test-h3-mixed: ## Run mixed load scenario on HTTP/3
	@echo "📊 Running MIXED LOAD scenario (HTTP/3)..."
	go run ./cmd/client/mixed-load --addr https://localhost:8443 --h3=true

test-h3-stress: ## Run high traffic stress test on HTTP/3
	@echo "📊 Running STRESS TEST scenario (HTTP/3)..."
	go run ./cmd/client/high-traffic --addr https://localhost:8443 --h3=true

test-all-h2: ## Run all 10 scenarios on HTTP/2
	@echo "📊 Running ALL scenarios on HTTP/2..."
	@make test-baseline
	@make test-burst
	@make test-coldstart
	@make test-parallel
	@make test-header-bloat
	@make test-uplink
	@make test-churn
	@make test-migration
	@make test-mixed
	@make test-stress

test-all-h3: ## Run all 10 scenarios on HTTP/3
	@echo "📊 Running ALL scenarios on HTTP/3..."
	@make test-h3-baseline
	@make test-h3-burst
	@make test-h3-coldstart
	@make test-h3-parallel
	@make test-h3-header-bloat
	@make test-h3-uplink
	@make test-h3-churn
	@make test-h3-migration
	@make test-h3-mixed
	@make test-h3-stress

##@ Benchmark Comparison

compare-baseline: ## Compare H2 vs H3 for baseline scenario
	@echo "📊 Comparing BASELINE: HTTP/2 vs HTTP/3..."
	@mkdir -p results
	@echo "\n=== HTTP/2 Baseline ==="
	go run ./cmd/client/low-traffic --addr https://localhost:8444 --h3=false \
		--csv results/baseline-h2.csv --html results/baseline-h2.html \
		--label "HTTP/2 Baseline"
	@echo "\n=== HTTP/3 Baseline ==="
	go run ./cmd/client/low-traffic --addr https://localhost:8443 --h3=true \
		--csv results/baseline-h3.csv --html results/baseline-h3.html \
		--label "HTTP/3 Baseline"
	@echo "✅ Results: results/baseline-h2.html & results/baseline-h3.html"

compare-burst: ## Compare H2 vs H3 for burst scenario
	@echo "📊 Comparing BURST: HTTP/2 vs HTTP/3..."
	@mkdir -p results
	go run ./cmd/client/burst-traffic --addr https://localhost:8444 --h3=false \
		--csv results/burst-h2.csv --html results/burst-h2.html --label "HTTP/2 Burst"
	go run ./cmd/client/burst-traffic --addr https://localhost:8443 --h3=true \
		--csv results/burst-h3.csv --html results/burst-h3.html --label "HTTP/3 Burst"
	@echo "✅ Results: results/burst-h2.html & results/burst-h3.html"

compare-coldstart: ## Compare H2 vs H3 for cold-start scenario
	@echo "📊 Comparing COLD-START: HTTP/2 vs HTTP/3..."
	@mkdir -p results
	go run ./cmd/client/cold-start --addr https://localhost:8444 --h3=false --mode cold \
		--csv results/coldstart-h2.csv --html results/coldstart-h2.html --label "HTTP/2 Cold-Start"
	go run ./cmd/client/cold-start --addr https://localhost:8443 --h3=true --mode cold \
		--csv results/coldstart-h3.csv --html results/coldstart-h3.html --label "HTTP/3 Cold-Start"
	@echo "✅ Results: results/coldstart-h2.html & results/coldstart-h3.html"

compare-parallel: ## Compare H2 vs H3 for parallel streams scenario
	@echo "📊 Comparing PARALLEL STREAMS: HTTP/2 vs HTTP/3..."
	@mkdir -p results
	go run ./cmd/client/parallel-requests --addr https://localhost:8444 --h3=false \
		--csv results/parallel-h2.csv --html results/parallel-h2.html --label "HTTP/2 Parallel"
	go run ./cmd/client/parallel-requests --addr https://localhost:8443 --h3=true \
		--csv results/parallel-h3.csv --html results/parallel-h3.html --label "HTTP/3 Parallel"
	@echo "✅ Results: results/parallel-h2.html & results/parallel-h3.html"

compare-header-bloat: ## Compare H2 vs H3 for header bloat scenario
	@echo "📊 Comparing HEADER BLOAT: HTTP/2 vs HTTP/3..."
	@mkdir -p results
	go run ./cmd/client/header-bloat --addr https://localhost:8444 --h3=false \
		--csv results/header-h2.csv --html results/header-h2.html --label "HTTP/2 Header Bloat"
	go run ./cmd/client/header-bloat --addr https://localhost:8443 --h3=true \
		--csv results/header-h3.csv --html results/header-h3.html --label "HTTP/3 Header Bloat"
	@echo "✅ Results: results/header-h2.html & results/header-h3.html"

compare-uplink: ## Compare H2 vs H3 for uplink loss scenario
	@echo "📊 Comparing UPLINK LOSS: HTTP/2 vs HTTP/3..."
	@mkdir -p results
	go run ./cmd/client/uplink-loss --addr https://localhost:8444 --h3=false \
		--csv results/uplink-h2.csv --html results/uplink-h2.html --label "HTTP/2 Uplink Loss"
	go run ./cmd/client/uplink-loss --addr https://localhost:8443 --h3=true \
		--csv results/uplink-h3.csv --html results/uplink-h3.html --label "HTTP/3 Uplink Loss"
	@echo "✅ Results: results/uplink-h2.html & results/uplink-h3.html"

compare-churn: ## Compare H2 vs H3 for connection churn scenario
	@echo "📊 Comparing CONNECTION CHURN: HTTP/2 vs HTTP/3..."
	@mkdir -p results
	go run ./cmd/client/connection-churn --addr https://localhost:8444 --h3=false \
		--csv results/churn-h2.csv --html results/churn-h2.html --label "HTTP/2 Churn"
	go run ./cmd/client/connection-churn --addr https://localhost:8443 --h3=true \
		--csv results/churn-h3.csv --html results/churn-h3.html --label "HTTP/3 Churn"
	@echo "✅ Results: results/churn-h2.html & results/churn-h3.html"

compare-migration: ## Compare H2 vs H3 for NAT rebinding scenario
	@echo "📊 Comparing NAT REBINDING: HTTP/2 vs HTTP/3..."
	@mkdir -p results
	go run ./cmd/client/nat-rebinding --addr https://localhost:8444 --h3=false \
		--csv results/migration-h2.csv --html results/migration-h2.html --label "HTTP/2 Migration"
	go run ./cmd/client/nat-rebinding --addr https://localhost:8443 --h3=true \
		--csv results/migration-h3.csv --html results/migration-h3.html --label "HTTP/3 Migration"
	@echo "✅ Results: results/migration-h2.html & results/migration-h3.html"

compare-mixed: ## Compare H2 vs H3 for mixed load scenario
	@echo "📊 Comparing MIXED LOAD: HTTP/2 vs HTTP/3..."
	@mkdir -p results
	go run ./cmd/client/mixed-load --addr https://localhost:8444 --h3=false \
		--csv results/mixed-h2.csv --html results/mixed-h2.html --label "HTTP/2 Mixed Load"
	go run ./cmd/client/mixed-load --addr https://localhost:8443 --h3=true \
		--csv results/mixed-h3.csv --html results/mixed-h3.html --label "HTTP/3 Mixed Load"
	@echo "✅ Results: results/mixed-h2.html & results/mixed-h3.html"

compare-stress: ## Compare H2 vs H3 for stress test scenario
	@echo "📊 Comparing STRESS TEST: HTTP/2 vs HTTP/3..."
	@mkdir -p results
	go run ./cmd/client/high-traffic --addr https://localhost:8444 --h3=false \
		--csv results/stress-h2.csv --html results/stress-h2.html --label "HTTP/2 Stress"
	go run ./cmd/client/high-traffic --addr https://localhost:8443 --h3=true \
		--csv results/stress-h3.csv --html results/stress-h3.html --label "HTTP/3 Stress"
	@echo "✅ Results: results/stress-h2.html & results/stress-h3.html"

compare-all: ## Run all 10 scenario comparisons (H2 vs H3)
	@echo "📊 Running ALL 10 scenario comparisons..."
	@make compare-baseline
	@make compare-burst
	@make compare-coldstart
	@make compare-parallel
	@make compare-header-bloat
	@make compare-uplink
	@make compare-churn
	@make compare-migration
	@make compare-mixed
	@make compare-stress
	@echo "\n✅ All comparisons complete! Check results/ directory"

##@ Utilities

clean: ## Clean build artifacts and cache
	@echo "🧹 Cleaning..."
	rm -rf bin/
	rm -rf dashboard-new/build/
	rm -rf dashboard-new/.svelte-kit/
	rm -rf dashboard-new/node_modules/.vite/
	rm -rf results/*.csv
	@echo "✅ Clean complete"

clean-all: clean ## Clean everything including node_modules
	@echo "🧹 Deep cleaning..."
	rm -rf dashboard-new/node_modules/
	@echo "✅ Deep clean complete"

fmt: ## Format Go code
	@echo "🎨 Formatting Go code..."
	go fmt ./...
	@echo "✅ Format complete"

lint: ## Run Go linter
	@echo "🔍 Running linter..."
	golangci-lint run ./...

check: fmt lint ## Run format and lint

##@ Info

info: ## Display project information
	@echo "\n📋 Project Information"
	@echo "======================"
	@echo "Go version:       $$(go version)"
	@echo "Node version:     $$(node --version)"
	@echo "npm version:      $$(npm --version)"
	@echo ""
	@echo "Project structure:"
	@echo "  cmd/server-h2/              - HTTP/2 gRPC server"
	@echo "  cmd/server-h3/              - HTTP/3 gRPC server"
	@echo "  cmd/client/low-traffic/     - Baseline scenario"
	@echo "  cmd/client/burst-traffic/   - Burst traffic scenario"
	@echo "  cmd/client/cold-start/      - Cold-start vs resumed"
	@echo "  cmd/client/parallel-requests/ - Parallel streams"
	@echo "  cmd/client/header-bloat/    - Header bloat scenario"
	@echo "  cmd/client/uplink-loss/     - Uplink loss scenario"
	@echo "  cmd/client/connection-churn/ - Connection churn"
	@echo "  cmd/client/nat-rebinding/   - NAT rebinding/migration"
	@echo "  cmd/client/mixed-load/      - Mixed load scenario"
	@echo "  cmd/client/high-traffic/    - Stress test scenario"
	@echo "  dashboard-new/              - SvelteKit dashboard"
	@echo "  proto/                      - Protobuf definitions"
	@echo ""
	@echo "Ports:"
	@echo "  HTTP/2 Server:  8444"
	@echo "  HTTP/3 Server:  8443"
	@echo "  Dashboard:      5000"
	@echo ""
	@echo "All 10 benchmark scenarios use FIXED configurations for fair comparison"
	@echo ""

ports: ## Show which processes are using benchmark ports
	@echo "🔍 Checking ports..."
	@echo "\nPort 8444 (HTTP/2):"
	@lsof -i :8444 || echo "  Not in use"
	@echo "\nPort 8443 (HTTP/3):"
	@lsof -i :8443 || echo "  Not in use"
	@echo "\nPort 5000 (Dashboard):"
	@lsof -i :5000 || echo "  Not in use"

kill-servers: ## Kill all running servers
	@echo "🛑 Killing servers..."
	@-pkill -f "server-h2" || true
	@-pkill -f "server-h3" || true
	@-lsof -ti:8444 | xargs kill -9 2>/dev/null || true
	@-lsof -ti:8443 | xargs kill -9 2>/dev/null || true
	@echo "✅ Servers stopped"

kill-dashboard: ## Kill dashboard process
	@echo "🛑 Killing dashboard..."
	@-lsof -ti:5000 | xargs kill -9 2>/dev/null || true
	@echo "✅ Dashboard stopped"

kill-all: kill-servers kill-dashboard ## Kill all processes

##@ Quick Start

all: install build ## Install dependencies and build everything

quickstart: ## Quick start guide
	@echo "\n🚀 Quick Start Guide"
	@echo "==================="
	@echo ""
	@echo "1. Install dependencies:"
	@echo "   make install"
	@echo ""
	@echo "2. Start development environment (servers + dashboard):"
	@echo "   make dev"
	@echo ""
	@echo "3. Open dashboard:"
	@echo "   http://localhost:5000"
	@echo ""
	@echo "4. Test individual scenarios:"
	@echo "   make test-baseline      # HTTP/2"
	@echo "   make test-h3-baseline   # HTTP/3"
	@echo ""
	@echo "5. Compare HTTP/2 vs HTTP/3 for any scenario:"
	@echo "   make compare-baseline"
	@echo "   make compare-burst"
	@echo "   make compare-stress"
	@echo "   make compare-all        # Run all 10 comparisons"
	@echo ""
	@echo "6. Build all binaries:"
	@echo "   make build"
	@echo ""
	@echo "Available scenarios:"
	@echo "  - baseline, burst, coldstart, parallel, header-bloat"
	@echo "  - uplink, churn, migration, mixed, stress"
	@echo ""
	@echo "For complete command list, run: make help"
	@echo ""

##@ Docker

docker-build: ## Build all Docker images
	@echo "🐳 Building Docker images..."
	docker-compose build

docker-up: ## Start all services with Docker Compose
	@echo "🚀 Starting Docker services..."
	docker-compose up -d
	@echo "✅ Services started"
	@echo "   - Dashboard: http://localhost:5000"
	@echo "   - HTTP/2 Server: https://localhost:8444"
	@echo "   - HTTP/3 Server: https://localhost:8443"

docker-down: ## Stop and remove all Docker containers
	@echo "🛑 Stopping Docker services..."
	docker-compose down

docker-restart: ## Restart all Docker services
	@echo "🔄 Restarting Docker services..."
	docker-compose restart

docker-logs: ## Show logs from all services
	docker-compose logs -f

docker-status: ## Show status of all Docker services
	@echo "📊 Docker services status:"
	@docker-compose ps

docker-clean: ## Clean up Docker containers, volumes, and images
	@echo "🧹 Cleaning Docker resources..."
	docker-compose down -v
	docker system prune -f
	@echo "✅ Docker cleanup complete"

docker-test: ## Run a quick test with Docker clients
	@echo "🧪 Running quick test (low-traffic baseline)..."
	@echo "  HTTP/3:"
	@docker-compose run --rm bench-clients bench-client -h3 -addr https://server-h3:8443 -insecure
	@echo ""
	@echo "  HTTP/2:"
	@docker-compose run --rm bench-clients bench-client -h3=false -addr https://server-h2:8444 -insecure

docker-run-all: ## Run all benchmark scenarios using Docker
	@echo "🏃 Running all benchmarks with Docker..."
	@./run_all_benchmarks.sh

docker-dev: docker-build docker-up ## Build and start development environment with Docker
	@echo "✅ Development environment ready!"
	@echo "   Open http://localhost:5000 in your browser"

