#!/usr/bin/env bash
set -euo pipefail

filter="${1:-${GOH264_CORPUS_FILTER:-}}"
export GOH264_CORPUS_CACHE="${GOH264_CORPUS_CACHE:-/tmp/goh264-fate-probe-cache}"
export GOH264_CORPUS_FETCH="${GOH264_CORPUS_FETCH:-1}"
if [[ -n "$filter" ]]; then
    export GOH264_CORPUS_FILTER="$filter"
fi

printf 'real-vector strict oracle'
if [[ -n "${GOH264_CORPUS_FILTER:-}" ]]; then
    printf ' filter=%s' "$GOH264_CORPUS_FILTER"
fi
printf '\n'

GOH264_REAL_VECTOR_STRICT=1 go test . -run '^TestH264RealVectorStrictOracle$' -count=1 -v
