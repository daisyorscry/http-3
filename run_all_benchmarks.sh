#!/bin/bash
# run_all_benchmarks.sh
# Script untuk menjalankan semua 10 benchmark scenarios untuk HTTP/2 dan HTTP/3

set -e

# Color output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}HTTP/2 vs HTTP/3 Benchmark Suite${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "Error: docker-compose not found. Please install docker-compose first."
    exit 1
fi

# Create results directory
RESULTS_DIR="results/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$RESULTS_DIR"
echo -e "${GREEN}Results will be saved to: $RESULTS_DIR${NC}"
echo ""

# Ensure services are running
echo -e "${YELLOW}Starting services...${NC}"
docker-compose up -d server-h2 server-h3
sleep 5

echo -e "${YELLOW}Building bench-clients image...${NC}"
docker-compose build bench-clients
echo ""

# Define scenarios
SCENARIOS=(
  "bench-client:low-traffic"
  "bench-header-bloat:header-bloat"
  "bench-parallel:parallel"
  "bench-burst:burst"
  "bench-coldstart:coldstart-warm"
  "bench-churn:churn"
  "bench-mixed:mixed"
  "bench-uplink:uplink"
  "bench-migration:migration"
  "bench-stress:stress"
)

TOTAL=${#SCENARIOS[@]}
CURRENT=0

for scenario in "${SCENARIOS[@]}"; do
  CURRENT=$((CURRENT + 1))
  binary="${scenario%%:*}"
  name="${scenario##*:}"

  echo -e "${BLUE}[$CURRENT/$TOTAL] Running: $name${NC}"
  echo "─────────────────────────────────────"

  # HTTP/3
  echo -e "${YELLOW}  → HTTP/3...${NC}"
  docker-compose run --rm bench-clients $binary \
    -h3 -addr https://server-h3:8443 \
    -csv "/app/results/${name}-h3.csv" \
    -html "/app/results/${name}-h3.html" \
    -insecure -quiet

  # Copy from container volume to host
  docker cp grpc-bench-clients:/app/results/${name}-h3.csv "$RESULTS_DIR/" 2>/dev/null || true
  docker cp grpc-bench-clients:/app/results/${name}-h3.html "$RESULTS_DIR/" 2>/dev/null || true

  # HTTP/2
  echo -e "${YELLOW}  → HTTP/2...${NC}"
  docker-compose run --rm bench-clients $binary \
    -h3=false -addr https://server-h2:8444 \
    -csv "/app/results/${name}-h2.csv" \
    -html "/app/results/${name}-h2.html" \
    -insecure -quiet

  # Copy from container volume to host
  docker cp grpc-bench-clients:/app/results/${name}-h2.csv "$RESULTS_DIR/" 2>/dev/null || true
  docker cp grpc-bench-clients:/app/results/${name}-h2.html "$RESULTS_DIR/" 2>/dev/null || true

  echo -e "${GREEN}  ✓ Completed${NC}"
  echo ""

  # Cooldown between tests
  if [ $CURRENT -lt $TOTAL ]; then
    echo "Cooldown (5s)..."
    sleep 5
  fi
done

echo ""
echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}All benchmarks completed!${NC}"
echo -e "${GREEN}======================================${NC}"
echo -e "Results saved to: ${BLUE}$RESULTS_DIR${NC}"
echo ""
echo "You can now:"
echo "  1. View individual HTML reports in $RESULTS_DIR/"
echo "  2. Analyze CSV data with your preferred tools"
echo "  3. Compare HTTP/2 vs HTTP/3 results"
echo ""
