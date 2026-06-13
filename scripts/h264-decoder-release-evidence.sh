#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

timestamp="${GOH264_RELEASE_EVIDENCE_TIMESTAMP:-$(date -u +%Y%m%dT%H%M%SZ)}"
out_dir="${GOH264_RELEASE_EVIDENCE_DIR:-$ROOT/.artifacts/h264-release-evidence/$timestamp}"
mkdir -p "$out_dir"

summary="$out_dir/summary.txt"
filter="${GOH264_RELEASE_PERF_FILTER:-canl4}"
benchstat_time="${GOH264_BENCHSTAT_TIME:-${GOH264_BENCHSTAT_BENCHTIME:-100ms}}"
export GOH264_BENCHSTAT_TIME="$benchstat_time"

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

run_go_test_gate() {
    local name="$1"
    local pkg="$2"
    local pattern="$3"
    shift 3
    local list_log="$out_dir/$name-list.log"
    {
        printf '\n== %s-list ==\n' "$name"
        printf 'command: go test %q -list %q\n' "$pkg" "$pattern"
    } | tee -a "$summary"
    go test "$pkg" -list "$pattern" 2>&1 | tee "$list_log"
    if ! grep -Eq '^(Test|Benchmark|Fuzz|Example)' "$list_log"; then
        printf 'status: fail (no matching tests)\n' | tee -a "$summary" >&2
        exit 1
    fi
    printf 'status: pass\n' | tee -a "$summary"
    run_gate "$name" go test "$pkg" -run "$pattern" "$@"
}

{
    printf 'commit=%s\n' "$(git rev-parse HEAD)"
    printf 'branch=%s\n' "$(git branch --show-current)"
    printf 'date_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    printf 'go=%s\n' "$(go version)"
    printf 'corpus_cache=%s\n' "${GOH264_CORPUS_CACHE:-/tmp/goh264-fate-probe-cache}"
    printf 'release_perf_filter=%s\n' "$filter"
    printf 'release_alloc_filter=%s\n' "${GOH264_RELEASE_ALLOC_FILTER:-canl4}"
    printf 'benchstat_pattern=%s\n' "${GOH264_BENCHSTAT_PATTERN:-Benchmark(Decode.*AnnexBHigh10IDRP|FrameAppendRawYUVBytesLEHigh10IDRP|Encode.*I420)}"
    printf 'benchstat_time=%s\n' "$GOH264_BENCHSTAT_TIME"
} >"$summary"

printf 'writing decoder release evidence to %s\n' "$out_dir" >&2

if [[ "${GOH264_RELEASE_ALLOW_DIRTY:-0}" != "1" ]]; then
    status="$(git status --short)"
    if [[ -n "$status" ]]; then
        {
            printf '\nworktree-clean: failed\n'
            printf '%s\n' "$status"
            printf 'set GOH264_RELEASE_ALLOW_DIRTY=1 only for local diagnostics\n'
        } | tee -a "$summary" >&2
        exit 1
    fi
fi
printf '\nworktree-clean: pass\n' | tee -a "$summary"

run_gate git-diff-check git diff --check
run_gate git-diff-cached-check git diff --cached --check
run_gate go-vet go vet ./...
run_gate go-test-all go test ./...
run_go_test_gate decoder-api-surfaces ./tests '^(TestParseHeadersAnnexBBlack16|TestParseHeadersAVCBlack16|TestPackageAVCCParsersDoNotMutateDecoderState|TestDecodeAVCCFramesIncompatibleConfigurationDoesNotUseStalePFrameReference|TestDecodePacketFramesNewExtradataIncompatibleConfigurationDoesNotUseStalePFrameReference|TestDecodePacketFramesAnnexBNewExtradataIncompatibleConfigurationDoesNotUseStalePFrameReference|TestParseHeadersAnnexBIncompatibleHeadersDoNotUseStalePFrameReference|TestParseHeadersAVCIncompatibleHeadersDoNotUseStalePFrameReference|TestDecodeAVCCFramesSwitchesValidConfigurationWithoutReset|TestDecodePacketFramesNewExtradataSwitchesValidAVCConfiguration|TestDecodeAVCCFramesMultiSPSConfigurationUsesPacketActiveSPSForDPBReset|TestDecodeFramesStandaloneMultiSPSConfigurationResetsForNonFirstActiveSPS|TestDecodePacketFramesMultiSPSNewExtradataUsesPacketActiveSPSForDPBReset|TestDecodePacketFramesAnnexBMultiSPSNewExtradataUsesPacketActiveSPSForDPBReset|TestDecoderAVCConfigUsesAVCCFirstSPSForMultiSPSConfiguration|TestDecoderAVCConfigUsesPacketActiveSPSForMultiSPSConfiguration)$' -count=1 -v
run_go_test_gate decoder-ref-modifications ./internal/h264 '^(TestSimpleFrameDPBRejectsMissingShortRefModificationTarget|TestSimpleFrameDPBRejectsMissingLongRefModificationTarget|TestSimpleFrameDPBReordersShortRefs|TestSimpleFrameDPBReordersLongRefs)$' -count=1 -v

if [[ -s testdata/h264/realvectors/failures.jsonl && "${GOH264_RELEASE_ALLOW_KNOWN_RED:-0}" != "1" ]]; then
    printf '\nknown-red-failures: testdata/h264/realvectors/failures.jsonl is not empty; set GOH264_RELEASE_ALLOW_KNOWN_RED=1 only for local diagnostics\n' | tee -a "$summary" >&2
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
