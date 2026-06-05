#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

filter="${GOH264_PERF_FILTER:-canl4}"
if [[ $# -gt 0 && "$1" != -* ]]; then
    filter="$1"
    shift
fi

timestamp="${GOH264_PERF_TIMESTAMP:-$(date -u +%Y%m%dT%H%M%SZ)}"
out_dir="${GOH264_PERF_DIR:-$ROOT/.artifacts/h264-performance-evidence/$timestamp}"
mkdir -p "$out_dir"

benchstat_out="$out_dir/benchstat.txt"
bench_json="$out_dir/goh264bench.json"
bench_log="$out_dir/goh264bench.stderr.txt"
metadata="$out_dir/metadata.txt"
cpu_profile="$out_dir/cpu.pprof"
mem_profile="$out_dir/heap.pprof"

export GOH264_BENCH_ITERS="${GOH264_BENCH_ITERS:-1}"
export GOH264_BENCH_REPEATS="${GOH264_BENCH_REPEATS:-2}"
export GOH264_BENCH_WARMUP="${GOH264_BENCH_WARMUP:-1}"
export GOH264_BENCH_MAX_ENTRIES="${GOH264_BENCH_MAX_ENTRIES:-1}"
export GOH264_BENCH_MAX_GO_ALLOC_BYTES_PER_ITER="${GOH264_BENCH_MAX_GO_ALLOC_BYTES_PER_ITER:-64000000}"
export GOH264_BENCH_MAX_GO_ALLOCS_PER_ITER="${GOH264_BENCH_MAX_GO_ALLOCS_PER_ITER:-10000}"
export GOH264_BENCH_CPU_PROFILE="$cpu_profile"
export GOH264_BENCH_MEM_PROFILE="$mem_profile"

{
    printf 'commit=%s\n' "$(git -C "$ROOT" rev-parse HEAD)"
    printf 'branch=%s\n' "$(git -C "$ROOT" branch --show-current)"
    printf 'date_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    printf 'go=%s\n' "$(go version)"
    printf 'filter=%s\n' "$filter"
    printf 'benchstat_pattern=%s\n' "${GOH264_BENCHSTAT_PATTERN:-BenchmarkDecodeAnnexBHigh10IDRP}"
    printf 'benchstat_count=%s\n' "${GOH264_BENCHSTAT_COUNT:-5}"
    printf 'benchstat_time=%s\n' "${GOH264_BENCHSTAT_TIME:-100ms}"
    printf 'bench_iters=%s\n' "$GOH264_BENCH_ITERS"
    printf 'bench_repeats=%s\n' "$GOH264_BENCH_REPEATS"
    printf 'bench_warmup=%s\n' "$GOH264_BENCH_WARMUP"
    printf 'bench_max_entries=%s\n' "$GOH264_BENCH_MAX_ENTRIES"
    printf 'max_go_alloc_bytes_per_iter=%s\n' "$GOH264_BENCH_MAX_GO_ALLOC_BYTES_PER_ITER"
    printf 'max_go_allocs_per_iter=%s\n' "$GOH264_BENCH_MAX_GO_ALLOCS_PER_ITER"
} >"$metadata"

printf 'writing performance evidence to %s\n' "$out_dir" >&2
"$ROOT/scripts/h264-benchstat-canary.sh" | tee "$benchstat_out"
"$ROOT/scripts/h264-real-vector-bench.sh" "$filter" "$@" >"$bench_json" 2>"$bench_log"

printf 'benchstat=%s\n' "$benchstat_out" >&2
printf 'benchmark_json=%s\n' "$bench_json" >&2
printf 'benchmark_log=%s\n' "$bench_log" >&2
printf 'cpu_profile=%s\n' "$cpu_profile" >&2
printf 'mem_profile=%s\n' "$mem_profile" >&2
printf 'metadata=%s\n' "$metadata" >&2
