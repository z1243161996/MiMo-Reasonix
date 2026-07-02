#!/usr/bin/env bash
#
# run-benchmarks.sh — Run all MiMo-Reasonix benchmarks with optional profiling.
#
# Usage:
#   ./scripts/run-benchmarks.sh              # run all benchmarks
#   ./scripts/run-benchmarks.sh -short       # skip slow benchmarks
#   ./scripts/run-benchmarks.sh -cpu         # CPU profile only
#   ./scripts/run-benchmarks.sh -mem         # memory profile only
#   ./scripts/run-benchmarks.sh -all         # all profiles + comparison
#   ./scripts/run-benchmarks.sh -count N     # repeat each benchmark N times

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RESULTS_DIR="$REPO_ROOT/benchmarks/results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RUN_DIR="$RESULTS_DIR/$TIMESTAMP"

# Defaults
CPU_PROFILE=false
MEM_PROFILE=false
COUNT=1
BENCH_FLAGS=""
EXTRA_FLAGS=""

usage() {
    echo "Usage: $0 [-short] [-cpu] [-mem] [-all] [-count N] [extra go test flags]"
    echo ""
    echo "Options:"
    echo "  -short    Skip slow benchmarks (short mode)"
    echo "  -cpu      Generate CPU profile"
    echo "  -mem      Generate memory profile"
    echo "  -all      Generate CPU, memory, and memalloc profiles + comparison"
    echo "  -count N  Repeat each benchmark N times for statistical analysis"
    echo ""
    echo "Output is saved to: benchmarks/results/<timestamp>/"
    exit 0
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help) usage ;;
        -short)    BENCH_FLAGS="$BENCH_FLAGS -short"; shift ;;
        -cpu)      CPU_PROFILE=true; shift ;;
        -mem)      MEM_PROFILE=true; shift ;;
        -all)      CPU_PROFILE=true; MEM_PROFILE=true; shift ;;
        -count)    COUNT="$2"; shift 2 ;;
        *)         EXTRA_FLAGS="$EXTRA_FLAGS $1"; shift ;;
    esac
done

if [[ "$CPU_PROFILE" == true && "$MEM_PROFILE" == true ]]; then
    # -all mode
    :
fi

mkdir -p "$RUN_DIR"

echo "=== MiMo-Reasonix Benchmark Suite ==="
echo "Results directory: $RUN_DIR"
echo "Timestamp: $TIMESTAMP"
echo ""

# --- Phase 1: Run all benchmarks ---
BENCH_FILE="$RUN_DIR/benchmarks.txt"
echo "--- Phase 1: Running benchmarks (count=$COUNT) ---"
cd "$REPO_ROOT"
go test -bench=. -benchmem -count="$COUNT" -timeout=300s \
    $BENCH_FLAGS $EXTRA_FLAGS \
    ./benchmarks/... 2>&1 | tee "$BENCH_FILE"

echo ""

# --- Phase 2: CPU profile (optional) ---
if [[ "$CPU_PROFILE" == true ]]; then
    CPU_FILE="$RUN_DIR/cpu.prof"
    echo "--- Phase 2: Generating CPU profile ---"
    go test -bench=. -benchmem -cpuprofile="$CPU_FILE" -timeout=300s \
        $BENCH_FLAGS $EXTRA_FLAGS \
        ./benchmarks/...
    echo "CPU profile saved to: $CPU_FILE"
    echo ""
fi

# --- Phase 3: Memory profile (optional) ---
if [[ "$MEM_PROFILE" == true ]]; then
    MEM_FILE="$RUN_DIR/mem.prof"
    echo "--- Phase 3: Generating memory profile ---"
    go test -bench=. -benchmem -memprofile="$MEM_FILE" -timeout=300s \
        $BENCH_FLAGS $EXTRA_FLAGS \
        ./benchmarks/...
    echo "Memory profile saved to: $MEM_FILE"

    # Generate memory allocation profile
    MEM_ALLOC="$RUN_DIR/memalloc.prof"
    go test -bench=. -benchmem -memprofile="$MEM_ALLOC" -timeout=300s \
        $BENCH_FLAGS $EXTRA_FLAGS \
        ./benchmarks/...
    echo "Memory allocation profile saved to: $MEM_ALLOC"
    echo ""
fi

# --- Phase 4: Profile analysis (optional) ---
if [[ "$CPU_PROFILE" == true ]]; then
    echo "--- Phase 4: CPU profile analysis (top 20) ---"
    go tool pprof -top -nodecount=20 "$RUN_DIR/cpu.prof" 2>/dev/null || true
    echo ""
fi

if [[ "$MEM_PROFILE" == true ]]; then
    echo "--- Phase 5: Memory profile analysis (top 20) ---"
    go tool pprof -top -nodecount=20 "$RUN_DIR/mem.prof" 2>/dev/null || true
    echo ""
fi

# --- Summary ---
echo "=== Benchmark Run Complete ==="
echo "Results: $RUN_DIR/"
ls -la "$RUN_DIR/"
echo ""
echo "To analyze profiles:"
if [[ "$CPU_PROFILE" == true ]]; then
    echo "  CPU:    go tool pprof $RUN_DIR/cpu.prof"
fi
if [[ "$MEM_PROFILE" == true ]]; then
    echo "  Memory: go tool pprof $RUN_DIR/mem.prof"
fi
echo ""
echo "To compare with previous runs:"
echo "  go tool pprof -diff_base=$RESULTS_DIR/<previous>/benchmarks.txt $RUN_DIR/benchmarks.txt"
