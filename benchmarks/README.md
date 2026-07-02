# MiMo-Reasonix Performance Benchmarks

This directory contains Go benchmarks that measure the performance of MiMo-Reasonix's core subsystems. All benchmarks use Go's built-in `testing.B` framework and are designed to be deterministic, with no external dependencies or network calls.

## Running Benchmarks

### Quick start

```bash
# Run all benchmarks
make bench

# Or directly
go test -bench=. -benchmem ./benchmarks/...

# Run a specific benchmark
go test -bench=BenchmarkProviderInitialization -benchmem ./benchmarks/...

# Run with more iterations for stability
go test -bench=. -benchmem -count=5 ./benchmarks/...
```

### Benchmark runner script

The `scripts/run-benchmarks.sh` script provides a structured runner with profiling support:

```bash
# Run all benchmarks
./scripts/run-benchmarks.sh

# Generate CPU profile
./scripts/run-benchmarks.sh -cpu

# Generate memory profile
./scripts/run-benchmarks.sh -mem

# Generate all profiles
./scripts/run-benchmarks.sh -all

# Repeat for statistical stability
./scripts/run-benchmarks.sh -count 3

# Skip slow benchmarks
./scripts/run-benchmarks.sh -short
```

Results are saved to `benchmarks/results/<timestamp>/`.

### Makefile targets

| Target | Description |
|--------|-------------|
| `make bench` | Run all benchmarks |
| `make bench-cpu` | Run with CPU profiling |
| `make bench-memory` | Run with memory profiling |

## Benchmark Categories

### Provider Benchmarks (`provider_benchmark_test.go`)

| Benchmark | What it measures | Target |
|-----------|-----------------|--------|
| `BenchmarkProviderInitialization` | Factory lookup + struct assembly + HTTP client creation | < 50us |
| `BenchmarkProviderInitializationParallel` | Concurrent provider creation (registry contention) | < 100us |
| `BenchmarkModelResolution` | Model reference -> ProviderEntry resolution | < 2us |
| `BenchmarkPricingCalculation` | Token usage -> cost estimate | < 100ns |
| `BenchmarkNormalizeMessages/well_formed` | Tool-call pairing repair (fast path, zero alloc) | < 200ns |
| `BenchmarkNormalizeMessages/with_orphan_tools` | Repair with orphan tool messages | < 5us |
| `BenchmarkNormalizeMessages/with_truncated_args` | JSON truncation repair | < 5us |
| `BenchmarkCanonicalizeSchema/simple` | Schema normalization (5 properties) | < 2us |

### Config Benchmarks (`config_benchmark_test.go`)

| Benchmark | What it measures | Target |
|-----------|-----------------|--------|
| `BenchmarkConfigLoading/defaults_only` | Default config allocation | < 1us |
| `BenchmarkConfigLoading/from_toml_file` | Full config load (defaults + TOML merge) | < 5ms |
| `BenchmarkConfigLoading/from_large_toml` | Config load with 8 providers | < 10ms |
| `BenchmarkConfigParsing/minimal` | Raw TOML decode (small config) | < 200us |
| `BenchmarkConfigParsing/large` | Raw TOML decode (large config) | < 2ms |
| `BenchmarkConfigParsing/providers_heavy` | TOML decode with 20 providers | < 5ms |
| `BenchmarkCredentialResolution` | Credential key lookup (env + .env files) | < 50us |
| `BenchmarkProviderEntryModelList` | Model list retrieval | < 100ns |
| `BenchmarkPriceForModel` | Per-model price lookup | < 100ns |

### Agent Benchmarks (`agent_benchmark_test.go`)

| Benchmark | What it measures | Target |
|-----------|-----------------|--------|
| `BenchmarkAgentLoop/session_add` | Single message append | < 100ns |
| `BenchmarkAgentLoop/session_snapshot` | Copy 200 messages (thread-safe) | < 10us |
| `BenchmarkAgentLoop/session_replace` | Swap message log (compaction) | < 1us |
| `BenchmarkToolExecution/registry_add` | Add 10 tools to registry | < 2us |
| `BenchmarkToolExecution/registry_get` | Tool lookup by name | < 100ns |
| `BenchmarkToolExecution/registry_schemas` | Generate schema list for 25 tools | < 5us |
| `BenchmarkToolExecution/registry_remove_prefix` | Remove MCP tool namespace | < 5us |
| `BenchmarkContextManagement/normalize_messages_large` | Normalize 200-turn history | < 50us |
| `BenchmarkContextManagement/session_concurrent_add_snapshot` | Concurrent add + snapshot | < 10us |

## Interpreting Results

### Key metrics

- **ns/op** -- Nanoseconds per operation. Lower is better.
- **B/op** -- Bytes allocated per operation. Lower is better.
- **allocs/op** -- Number of heap allocations per operation. Zero is ideal for hot paths.

### Reading output

```
BenchmarkPricingCalculation/large_turn_with_cache-8   1234567   987 ns/op   0 B/op   0 allocs/op
                                 ^^^^^^^^^^^^^^^^       ^^^^      ^^^        ^^^        ^^^
                                 sub-benchmark          iter   ns/op      bytes     allocs
```

### Profiling

```bash
# CPU profile
go tool pprof benchmarks/results/<run>/cpu.prof
> top 20          # show top 20 functions by CPU time
> web             # visualize (requires graphviz)
> list FuncName   # disassemble a specific function

# Memory profile
go tool pprof benchmarks/results/<run>/mem.prof
> top 20          # top 20 by allocated bytes
> list FuncName   # per-line allocation details
```

### Comparing runs

```bash
# Save a baseline
./scripts/run-benchmarks.sh
# ... modify code ...
./scripts/run-benchmarks.sh

# Compare
benchstat benchmarks/results/<baseline>/benchmarks.txt benchmarks/results/<new>/benchmarks.txt
```

Install benchstat: `go install golang.org/x/perf/cmd/benchstat@latest`

## Performance Goals

These are target ceilings for the core hot paths that execute on every turn:

| Hot path | Budget | Rationale |
|----------|--------|-----------|
| Provider request construction | < 50us | Called once per turn; 20 turns/sec budget = 50us |
| Tool schema serialization | < 5us | Included in request construction |
| Message normalization | < 10us | Called before every provider request |
| Pricing calculation | < 100ns | Called on every streamed chunk |
| Session snapshot | < 10us | Called by frontends during active turns |
| Tool registry lookup | < 100ns | Called per tool call in the agent loop |

## Design Principles

1. **Deterministic**: No network calls, no file I/O in hot paths, no random data. Temp dirs for config benchmarks.
2. **Self-contained**: Each benchmark file is independent. Provider benchmarks import only the provider packages; config benchmarks use `config.Default()` as baseline.
3. **Zero external deps**: Only standard library and project-internal packages. No third-party benchmark libraries.
4. **Profileable**: All benchmarks work with `-cpuprofile`, `-memprofile`, and `-benchmem` flags.
5. **Scalable**: Sub-benchmarks isolate individual operations for targeted optimization.
