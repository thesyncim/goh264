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
GOH264_REAL_VECTOR_RED=1 go test ./tests -run '^TestH264RealVectorKnownRedFilterSelected$' -count=1 -v

printf '\n'
printf 'known-red freshness (stale-ledger gate)'
if [[ -n "${GOH264_CORPUS_FILTER:-}" ]]; then
    printf ' filter=%s' "$GOH264_CORPUS_FILTER"
fi
printf '\n'
GOH264_REAL_VECTOR_FAILURES=1 go test ./tests -run '^TestH264RealVectorFailureLedgerFreshness$' -count=1 -v

printf '\n'
printf 'real-vector matrix (safe-point gate)'
if [[ -n "${GOH264_CORPUS_FILTER:-}" ]]; then
    printf ' filter=%s' "$GOH264_CORPUS_FILTER"
fi
printf '\n'
GOH264_REAL_VECTOR_MATRIX=1 go test ./tests -run '^TestH264RealVectorFailureMatrix$' -count=1 -v

printf '\nraw-diff diagnostics (raw-MD5 lanes exit here with the first divergent raw byte)'
if [[ -n "${GOH264_CORPUS_FILTER:-}" ]]; then
    printf ' filter=%s' "$GOH264_CORPUS_FILTER"
fi
printf '\n'
set +e
GOH264_REAL_VECTOR_RAWDIFF=1 go test ./tests -run '^TestH264RealVectorRawDiffDiagnostics$' -count=1 -v
rawdiff_status=$?
set -e
if [[ "$rawdiff_status" -ne 0 ]]; then
    exit "$rawdiff_status"
fi

printf '\nframe-MD5 diagnostics (fallback: raw-MD5 lanes exit here with the first divergent frame)'
if [[ -n "${GOH264_CORPUS_FILTER:-}" ]]; then
    printf ' filter=%s' "$GOH264_CORPUS_FILTER"
fi
printf '\n'
set +e
GOH264_REAL_VECTOR_FRAMEMD5=1 go test ./tests -run '^TestH264RealVectorFrameMD5Diagnostics$' -count=1 -v
framemd5_status=$?
set -e
if [[ "$framemd5_status" -ne 0 ]]; then
    exit "$framemd5_status"
fi

printf '\nknown-red red queue (expected to fail while ledger rows are current)'
if [[ -n "${GOH264_CORPUS_FILTER:-}" ]]; then
    printf ' filter=%s' "$GOH264_CORPUS_FILTER"
fi
printf '\n'
GOH264_REAL_VECTOR_RED_QUEUE=1 go test ./tests -run '^TestH264RealVectorRedQueue$' -count=1 -v
