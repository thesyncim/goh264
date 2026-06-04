#!/usr/bin/env bash
set -euo pipefail

filter="${1:-${GOH264_CORPUS_FILTER:-}}"
export GOH264_CORPUS_CACHE="${GOH264_CORPUS_CACHE:-/tmp/goh264-fate-probe-cache}"
export GOH264_CORPUS_FETCH="${GOH264_CORPUS_FETCH:-1}"
if [[ -n "$filter" ]]; then
    export GOH264_CORPUS_FILTER="$filter"
fi

printf 'real-vector failure-ledger freshness'
if [[ -n "${GOH264_CORPUS_FILTER:-}" ]]; then
    printf ' filter=%s' "$GOH264_CORPUS_FILTER"
fi
printf '\n'
GOH264_REAL_VECTOR_FAILURES=1 go test ./tests -run '^TestH264RealVectorFailureLedgerFreshness$' -count=1 -v

printf '\nreal-vector matrix (safe-point gate)'
if [[ -n "${GOH264_CORPUS_FILTER:-}" ]]; then
    printf ' filter=%s' "$GOH264_CORPUS_FILTER"
fi
printf '\n'
GOH264_REAL_VECTOR_MATRIX=1 go test ./tests -run '^TestH264RealVectorFailureMatrix$' -count=1 -v

printf '\nreal-vector red queue (expected to fail until selected rows are fixed; script exits non-zero while red)'
if [[ -n "${GOH264_CORPUS_FILTER:-}" ]]; then
    printf ' filter=%s' "$GOH264_CORPUS_FILTER"
fi
printf '\n'
GOH264_REAL_VECTOR_RED_QUEUE=1 go test ./tests -run '^TestH264RealVectorRedQueue$' -count=1 -v
