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

ffmpeg_args=()
if [[ "${GOH264_BENCH_FFMPEG:-0}" == "1" ]]; then
    ffmpeg_args=(
        -ffmpeg
        -ffmpeg-bin "${GOH264_FFMPEG_BIN:-ffmpeg}"
        -ffmpeg-threads "${GOH264_FFMPEG_THREADS:-1}"
        -strict-pix-fmt
    )
fi

printf 'real-vector benchmark cache=%s fetch=%s' "$GOH264_CORPUS_CACHE" "$GOH264_CORPUS_FETCH" >&2
if [[ -n "${GOH264_CORPUS_FILTER:-}" ]]; then
    printf ' filter=%s' "$GOH264_CORPUS_FILTER" >&2
fi
printf ' iters=%s repeats=%s warmup=%s max_entries=%s' "$iters" "$repeats" "$warmup" "$max_entries" >&2
if [[ "${#ffmpeg_args[@]}" -ne 0 ]]; then
    printf ' ffmpeg=1' >&2
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
if [[ "${#ffmpeg_args[@]}" -ne 0 ]]; then
    cmd+=("${ffmpeg_args[@]}")
fi
cmd+=("$@")
"${cmd[@]}"
