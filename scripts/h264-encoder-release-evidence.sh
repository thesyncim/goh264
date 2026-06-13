#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

timestamp="${GOH264_ENCODER_RELEASE_TIMESTAMP:-$(date -u +%Y%m%dT%H%M%SZ)}"
out_dir="${GOH264_ENCODER_RELEASE_DIR:-$ROOT/.artifacts/h264-encoder-release-evidence/$timestamp}"
mkdir -p "$out_dir"

summary="$out_dir/summary.txt"
bench_pattern="${GOH264_ENCODER_BENCH_PATTERN:-BenchmarkEncode.*I420}"
bench_time="${GOH264_ENCODER_BENCHTIME:-20x}"

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
    printf 'bench_pattern=%s\n' "$bench_pattern"
    printf 'bench_time=%s\n' "$bench_time"
} >"$summary"

printf 'writing encoder release evidence to %s\n' "$out_dir" >&2

if [[ "${GOH264_ENCODER_RELEASE_ALLOW_DIRTY:-0}" != "1" ]]; then
    status="$(git status --short)"
    if [[ -n "$status" ]]; then
        {
            printf '\nworktree-clean: failed\n'
            printf '%s\n' "$status"
            printf 'set GOH264_ENCODER_RELEASE_ALLOW_DIRTY=1 only for local diagnostics\n'
        } | tee -a "$summary" >&2
        exit 1
    fi
fi
printf '\nworktree-clean: pass\n' | tee -a "$summary"

run_gate git-diff-check git diff --check
run_gate git-diff-cached-check git diff --cached --check
run_gate go-vet go vet ./...
run_gate go-test-all go test ./...
run_go_test_gate encoder-contract ./tests '^TestEncoder' -count=1 -v
run_go_test_gate encoder-api-surfaces ./tests '^(TestEncoderEncodeIntoRTPPacketsDoNotAliasAccessUnitData|TestEncoderReconfigureZeroScalarFieldsAreNoOps|TestEncoderZeroValueExplicitSettersRejectWithoutMutation|TestEncoderNonRTPConfigsRejectInvalidRTPControls|TestEncoderInvalidRTPControlsRejectForNonRTPOutputsWithoutMutation|TestEncoderReconfigureOutputFormatQueuesIDRBoundary|TestEncoderInvalidRTPSettersPreservePacketState|TestEncodedFrameNALDataRejectsInvalidIndexesAndMetadata|TestEncodedFrameRTPDataRejectsInvalidIndexesAndMetadata|TestEncodedFrameAppendNALAndAccessUnitDataReturnCallerOwnedBytes|TestEncodedFrameAppendRTPDataReturnsCallerOwnedBytes|TestEncoderAppendHelpersIsolateOverlappingSource|TestEncoderParameterSetsAVCCReturnsClippedView|TestEncoderRTPPacketDataHelpersReturnClippedCallerOwnedBytes|TestEncodedFrameCloneRejectsInvalidMetadata|TestEncoderCheckedCloneHelpersRejectOverflowedPublicStorage|TestEncoderCheckedAppendHelpersRejectOverflowedPublicStorage|TestEncodedFrameOutputHelpersRejectOverflowedPublicStorage)$' -count=1 -v
run_go_test_gate encoder-residual-boundary ./tests '^TestEncoderResidualShapedPDeltaRemainsPIntraPCMAcrossPublicOutputs$' -count=1 -v
run_go_test_gate encoder-allocation-canary ./tests '^TestEncoderEncodeIntoAllocationCanary$' -count=1 -v
run_go_test_gate encoder-writers ./internal/h264 '^(TestBitWriter|TestAppendNAL|TestAppendAVC|TestBuildEncoder|TestAppendSEI|TestCAVLCWriteResidual|TestWriteCAVLCInterPBoundedMacroblock|TestEncodeI420P16x16ResidualSliceRBSP)' -count=1 -v
run_gate encoder-benchmem go test . -run '^$' -bench "$bench_pattern" -benchmem -benchtime "$bench_time"

printf '\nall encoder release-evidence gates passed\n' | tee -a "$summary"
