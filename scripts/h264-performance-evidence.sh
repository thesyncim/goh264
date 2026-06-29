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
bench_json_purego="$out_dir/goh264bench.purego.json"
bench_log_purego="$out_dir/goh264bench.purego.stderr.txt"
metadata="$out_dir/metadata.txt"
cpu_profile="$out_dir/cpu.pprof"
mem_profile="$out_dir/heap.pprof"
cpu_profile_purego="$out_dir/cpu.purego.pprof"
mem_profile_purego="$out_dir/heap.purego.pprof"

export GOH264_BENCH_ITERS="${GOH264_BENCH_ITERS:-10}"
export GOH264_BENCH_REPEATS="${GOH264_BENCH_REPEATS:-5}"
export GOH264_BENCH_WARMUP="${GOH264_BENCH_WARMUP:-2}"
export GOH264_BENCH_MAX_ENTRIES="${GOH264_BENCH_MAX_ENTRIES:-1}"
export GOH264_BENCH_FFMPEG="${GOH264_BENCH_FFMPEG:-1}"
export GOH264_BENCH_FAIR_CPU_LANES="${GOH264_BENCH_FAIR_CPU_LANES:-1}"
export GOH264_BENCH_FORBID_GO_ALLOCATIONS="${GOH264_BENCH_FORBID_GO_ALLOCATIONS:-1}"
export GOH264_BENCHSTAT_PATTERN="${GOH264_BENCHSTAT_PATTERN:-Benchmark(Decode.*AnnexB.*High10IDRP|FrameAppendRawYUVBytesLEHigh10IDRP)}"
export GOH264_BENCHSTAT_TIME="${GOH264_BENCHSTAT_TIME:-${GOH264_BENCHSTAT_BENCHTIME:-100ms}}"
run_purego="${GOH264_PERF_RUN_PUREGO:-1}"

{
    printf 'commit=%s\n' "$(git -C "$ROOT" rev-parse HEAD)"
    printf 'branch=%s\n' "$(git -C "$ROOT" branch --show-current)"
    printf 'date_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    printf 'go=%s\n' "$(go version)"
    printf 'filter=%s\n' "$filter"
    printf 'benchstat_pattern=%s\n' "$GOH264_BENCHSTAT_PATTERN"
    printf 'benchstat_count=%s\n' "${GOH264_BENCHSTAT_COUNT:-5}"
    printf 'benchstat_time=%s\n' "$GOH264_BENCHSTAT_TIME"
    printf 'bench_iters=%s\n' "$GOH264_BENCH_ITERS"
    printf 'bench_repeats=%s\n' "$GOH264_BENCH_REPEATS"
    printf 'bench_warmup=%s\n' "$GOH264_BENCH_WARMUP"
    printf 'bench_max_entries=%s\n' "$GOH264_BENCH_MAX_ENTRIES"
    printf 'bench_ffmpeg=%s\n' "$GOH264_BENCH_FFMPEG"
    printf 'bench_fair_cpu_lanes=%s\n' "$GOH264_BENCH_FAIR_CPU_LANES"
    printf 'bench_forbid_go_allocations=%s\n' "$GOH264_BENCH_FORBID_GO_ALLOCATIONS"
    printf 'bench_run_purego=%s\n' "$run_purego"
    printf 'max_go_alloc_bytes_per_iter=%s\n' "${GOH264_BENCH_MAX_GO_ALLOC_BYTES_PER_ITER:-unset}"
    printf 'max_go_allocs_per_iter=%s\n' "${GOH264_BENCH_MAX_GO_ALLOCS_PER_ITER:-unset}"
} >"$metadata"

printf 'writing performance evidence to %s\n' "$out_dir" >&2
"$ROOT/scripts/h264-benchstat-canary.sh" | tee "$benchstat_out"
export GOH264_BENCH_CPU_PROFILE="$cpu_profile"
export GOH264_BENCH_MEM_PROFILE="$mem_profile"
"$ROOT/scripts/h264-real-vector-bench.sh" "$filter" "$@" >"$bench_json" 2>"$bench_log"
if [[ "$run_purego" == "1" ]]; then
    export GOH264_BENCH_CPU_PROFILE="$cpu_profile_purego"
    export GOH264_BENCH_MEM_PROFILE="$mem_profile_purego"
    purego_go_flags="${GOFLAGS:-}"
    purego_go_flags="${purego_go_flags:+$purego_go_flags }-tags=purego"
    GOFLAGS="$purego_go_flags" "$ROOT/scripts/h264-real-vector-bench.sh" "$filter" "$@" >"$bench_json_purego" 2>"$bench_log_purego"
fi

printf 'benchstat=%s\n' "$benchstat_out" >&2
printf 'benchmark_json=%s\n' "$bench_json" >&2
printf 'benchmark_log=%s\n' "$bench_log" >&2
if [[ "$run_purego" == "1" ]]; then
    printf 'benchmark_purego_json=%s\n' "$bench_json_purego" >&2
    printf 'benchmark_purego_log=%s\n' "$bench_log_purego" >&2
fi
printf 'cpu_profile=%s\n' "$cpu_profile" >&2
printf 'mem_profile=%s\n' "$mem_profile" >&2
if [[ "$run_purego" == "1" ]]; then
    printf 'purego_cpu_profile=%s\n' "$cpu_profile_purego" >&2
    printf 'purego_mem_profile=%s\n' "$mem_profile_purego" >&2
fi
printf 'metadata=%s\n' "$metadata" >&2
