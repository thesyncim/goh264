#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

timestamp="${GOH264_RELEASE_EVIDENCE_TIMESTAMP:-$(date -u +%Y%m%dT%H%M%SZ)}"
out_dir="${GOH264_RELEASE_EVIDENCE_DIR:-$ROOT/.artifacts/h264-release-evidence/$timestamp}"
mkdir -p "$out_dir"

summary="$out_dir/summary.txt"
filter="${GOH264_RELEASE_PERF_FILTER:-canl4}"

run_gate() {
    local name="$1"
    shift
    local log="$out_dir/$name.log"
    {
        printf '\n== %s ==\n' "$name"
        printf 'command:'
        printf ' %q' "$@"
        printf '\n'
    } | tee -a "$summary"
    "$@" 2>&1 | tee "$log"
    printf 'status: pass\n' | tee -a "$summary"
}

run_env_gate() {
    local name="$1"
    shift
    run_gate "$name" env "$@"
}

{
    printf 'commit=%s\n' "$(git rev-parse HEAD)"
    printf 'branch=%s\n' "$(git branch --show-current)"
    printf 'date_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    printf 'go=%s\n' "$(go version)"
    printf 'corpus_cache=%s\n' "${GOH264_CORPUS_CACHE:-/tmp/goh264-fate-probe-cache}"
    printf 'release_perf_filter=%s\n' "$filter"
    printf 'release_alloc_filter=%s\n' "${GOH264_RELEASE_ALLOC_FILTER:-canl4}"
    printf 'benchstat_pattern=%s\n' "${GOH264_BENCHSTAT_PATTERN:-BenchmarkDecode.*AnnexBHigh10IDRP}"
} >"$summary"

printf 'writing decoder release evidence to %s\n' "$out_dir" >&2

if [[ "${GOH264_RELEASE_ALLOW_DIRTY:-0}" != "1" ]]; then
    status="$(git status --short)"
    if [[ -n "$status" ]]; then
        {
            printf '\nworktree-clean: failed\n'
            printf '%s\n' "$status"
            printf 'set GOH264_RELEASE_ALLOW_DIRTY=1 only for non-release diagnostics\n'
        } | tee -a "$summary" >&2
        exit 1
    fi
fi
printf '\nworktree-clean: pass\n' | tee -a "$summary"

run_gate git-diff-check git diff --check
run_gate git-diff-cached-check git diff --cached --check
run_gate go-test-all go test ./...

if [[ -s testdata/h264/realvectors/failures.jsonl && "${GOH264_RELEASE_ALLOW_KNOWN_RED:-0}" != "1" ]]; then
    printf '\nknown-red-failures: testdata/h264/realvectors/failures.jsonl is not empty; set GOH264_RELEASE_ALLOW_KNOWN_RED=1 only for non-release diagnostics\n' | tee -a "$summary" >&2
    exit 1
fi
printf '\nknown-red-failures: none\n' | tee -a "$summary"

run_env_gate real-vector-failure-ledger \
    GOH264_REAL_VECTOR_FAILURES=1 \
    GOH264_CORPUS_FETCH="${GOH264_CORPUS_FETCH:-1}" \
    go test ./tests -run '^TestH264RealVectorFailureLedgerFreshness$' -count=1 -v

run_env_gate real-vector-matrix \
    GOH264_REAL_VECTOR_MATRIX=1 \
    GOH264_CORPUS_FETCH="${GOH264_CORPUS_FETCH:-1}" \
    go test ./tests -run '^TestH264RealVectorFailureMatrix$' -count=1 -v

run_gate real-vector-strict scripts/h264-real-vector-strict.sh
run_gate real-vector-upstream-audit scripts/h264-real-vector-upstream-audit.sh
run_gate decoder-fuzz-smoke scripts/h264-decoder-fuzz-smoke.sh
run_gate release-allocation-canary scripts/h264-real-vector-release-alloc.sh
run_gate benchstat-canary scripts/h264-benchstat-canary.sh

run_env_gate performance-evidence \
    GOH264_PERF_DIR="$out_dir/performance-bundle" \
    scripts/h264-performance-evidence.sh "$filter"

printf '\nall decoder release-evidence gates passed\n' | tee -a "$summary"
