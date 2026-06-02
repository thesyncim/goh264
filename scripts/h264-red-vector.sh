#!/usr/bin/env bash
set -euo pipefail

filter="${1:-${GOH264_CORPUS_FILTER:-}}"
export GOH264_CORPUS_CACHE="${GOH264_CORPUS_CACHE:-/tmp/goh264-fate-probe-cache}"
export GOH264_CORPUS_FETCH="${GOH264_CORPUS_FETCH:-1}"
if [[ -n "$filter" ]]; then
    export GOH264_CORPUS_FILTER="$filter"
fi

printf 'known-red filter preflight'
if [[ -n "${GOH264_CORPUS_FILTER:-}" ]]; then
    printf ' filter=%s' "$GOH264_CORPUS_FILTER"
fi
printf '\n'
GOH264_REAL_VECTOR_RED=1 go test . -run '^TestH264RealVectorKnownRedFilterSelected$' -count=1 -v

printf '\n'
printf 'real-vector matrix (safe-point gate)'
if [[ -n "${GOH264_CORPUS_FILTER:-}" ]]; then
    printf ' filter=%s' "$GOH264_CORPUS_FILTER"
fi
printf '\n'
GOH264_REAL_VECTOR_MATRIX=1 go test . -run '^TestH264RealVectorFailureMatrix$' -count=1 -v

printf '\nknown-red strict oracle (expected to fail until the lane is fixed)'
if [[ -n "${GOH264_CORPUS_FILTER:-}" ]]; then
    printf ' filter=%s' "$GOH264_CORPUS_FILTER"
fi
printf '\n'
set +e
GOH264_REAL_VECTOR_RED=1 go test . -run '^TestH264RealVectorKnownRedStrict$' -count=1 -v
status=$?
set -e

if [[ "$status" -eq 0 ]]; then
    printf 'known-red strict oracle unexpectedly passed; remove fixed row(s) from testdata/h264/realvectors/failures.jsonl and rerun the matrix\n' >&2
    exit 1
fi

printf 'known-red strict oracle failed as expected; use the first strict corpus failure above as the next parity target\n'
