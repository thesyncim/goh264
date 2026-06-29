#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

filter="${GOH264_CORPUS_FILTER:-}"
if [[ $# -gt 0 && "$1" != -* ]]; then
    filter="$1"
    shift
fi

export GOH264_CORPUS_CACHE="${GOH264_CORPUS_CACHE:-/tmp/goh264-fate-probe-cache}"
export GOH264_CORPUS_FETCH="${GOH264_CORPUS_FETCH:-1}"
if [[ -n "$filter" ]]; then
    export GOH264_CORPUS_FILTER="$filter"
fi

iters="${GOH264_BENCH_ITERS:-10}"
repeats="${GOH264_BENCH_REPEATS:-5}"
warmup="${GOH264_BENCH_WARMUP:-2}"
max_entries="${GOH264_BENCH_MAX_ENTRIES:-0}"

alloc_args=()
if [[ -n "${GOH264_BENCH_MAX_GO_ALLOC_BYTES_PER_ITER:-}" ]]; then
    alloc_args+=(-max-go-alloc-bytes-per-iter "${GOH264_BENCH_MAX_GO_ALLOC_BYTES_PER_ITER}")
fi
if [[ -n "${GOH264_BENCH_MAX_GO_ALLOCS_PER_ITER:-}" ]]; then
    alloc_args+=(-max-go-allocs-per-iter "${GOH264_BENCH_MAX_GO_ALLOCS_PER_ITER}")
fi

profile_args=()
if [[ -n "${GOH264_BENCH_CPU_PROFILE:-}" ]]; then
    profile_args+=(-cpuprofile "${GOH264_BENCH_CPU_PROFILE}")
fi
if [[ -n "${GOH264_BENCH_MEM_PROFILE:-}" ]]; then
    profile_args+=(-memprofile "${GOH264_BENCH_MEM_PROFILE}")
fi

ffmpeg_args=()
if [[ "${GOH264_BENCH_FFMPEG:-0}" == "1" ]]; then
    ffmpeg_args=(
        -ffmpeg
        -ffmpeg-bin "${GOH264_FFMPEG_BIN:-ffmpeg}"
        -ffmpeg-threads "${GOH264_FFMPEG_THREADS:-1}"
        -strict-pix-fmt
    )
    if [[ "${GOH264_BENCH_FAIR_CPU_LANES:-0}" == "1" ]]; then
        ffmpeg_args+=(-fair-cpu-lanes)
    elif [[ "${GOH264_BENCH_FFMPEG_PURE_C:-0}" == "1" ]]; then
        ffmpeg_args+=(-ffmpeg-pure-c)
    elif [[ -n "${GOH264_FFMPEG_CPUFLAGS:-}" ]]; then
        ffmpeg_args+=(-ffmpeg-cpuflags "${GOH264_FFMPEG_CPUFLAGS}")
    fi
    if [[ "${GOH264_BENCH_FFMPEG_PROCESS_PER_ITER:-0}" == "1" ]]; then
        ffmpeg_args+=(-ffmpeg-process-per-iter)
    fi
fi

printf 'real-vector benchmark cache=%s fetch=%s' "$GOH264_CORPUS_CACHE" "$GOH264_CORPUS_FETCH" >&2
if [[ -n "${GOH264_CORPUS_FILTER:-}" ]]; then
    printf ' filter=%s' "$GOH264_CORPUS_FILTER" >&2
fi
printf ' iters=%s repeats=%s warmup=%s max_entries=%s' "$iters" "$repeats" "$warmup" "$max_entries" >&2
if [[ "${#alloc_args[@]}" -ne 0 ]]; then
    if [[ -n "${GOH264_BENCH_MAX_GO_ALLOC_BYTES_PER_ITER:-}" ]]; then
        printf ' max_go_alloc_bytes_per_iter=%s' "$GOH264_BENCH_MAX_GO_ALLOC_BYTES_PER_ITER" >&2
    fi
    if [[ -n "${GOH264_BENCH_MAX_GO_ALLOCS_PER_ITER:-}" ]]; then
        printf ' max_go_allocs_per_iter=%s' "$GOH264_BENCH_MAX_GO_ALLOCS_PER_ITER" >&2
    fi
fi
if [[ "${#profile_args[@]}" -ne 0 ]]; then
    if [[ -n "${GOH264_BENCH_CPU_PROFILE:-}" ]]; then
        printf ' cpuprofile=%s' "$GOH264_BENCH_CPU_PROFILE" >&2
    fi
    if [[ -n "${GOH264_BENCH_MEM_PROFILE:-}" ]]; then
        printf ' memprofile=%s' "$GOH264_BENCH_MEM_PROFILE" >&2
    fi
fi
if [[ "${#ffmpeg_args[@]}" -ne 0 ]]; then
    printf ' ffmpeg=1' >&2
    if [[ "${GOH264_BENCH_FAIR_CPU_LANES:-0}" == "1" ]]; then
        printf ' fair_cpu_lanes=1' >&2
    elif [[ "${GOH264_BENCH_FFMPEG_PURE_C:-0}" == "1" ]]; then
        printf ' ffmpeg_cpuflags=0' >&2
    elif [[ -n "${GOH264_FFMPEG_CPUFLAGS:-}" ]]; then
        printf ' ffmpeg_cpuflags=%s' "$GOH264_FFMPEG_CPUFLAGS" >&2
    fi
    if [[ "${GOH264_BENCH_FFMPEG_PROCESS_PER_ITER:-0}" == "1" ]]; then
        printf ' ffmpeg_process_per_iter=1' >&2
    else
        printf ' ffmpeg_amortized=1' >&2
    fi
fi
printf '\n' >&2

cd "$ROOT"
cmd=(go run ./cmd/goh264bench \
    -manifest testdata/h264/realvectors/manifest.jsonl \
    -failure-ledger auto \
    -filter "${GOH264_CORPUS_FILTER:-}" \
    -iters "$iters" \
    -repeats "$repeats" \
    -warmup "$warmup" \
    -max-entries "$max_entries" \
    -json)
if [[ "${#alloc_args[@]}" -ne 0 ]]; then
    cmd+=("${alloc_args[@]}")
fi
if [[ "${#profile_args[@]}" -ne 0 ]]; then
    cmd+=("${profile_args[@]}")
fi
if [[ "${#ffmpeg_args[@]}" -ne 0 ]]; then
    cmd+=("${ffmpeg_args[@]}")
fi
cmd+=("$@")
"${cmd[@]}"
