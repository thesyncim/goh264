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
            printf 'set GOH264_ENCODER_RELEASE_ALLOW_DIRTY=1 only for non-release diagnostics\n'
        } | tee -a "$summary" >&2
        exit 1
    fi
fi
printf '\nworktree-clean: pass\n' | tee -a "$summary"

run_gate git-diff-check git diff --check
run_gate git-diff-cached-check git diff --cached --check
run_gate go-vet go vet ./...
run_gate go-test-all go test ./...
run_gate encoder-contract go test ./tests -run '^TestEncoder' -count=1 -v
run_gate encoder-api-surfaces go test ./tests -run '^(TestEncoderEncodeIntoRTPPacketsDoNotAliasAccessUnitData|TestEncoderReconfigureZeroScalarFieldsAreNoOps|TestEncoderZeroValueExplicitSettersRejectWithoutMutation|TestEncoderNonRTPConfigsRejectInvalidRTPControls|TestEncoderInvalidRTPControlsRejectForNonRTPOutputsWithoutMutation|TestEncoderReconfigureOutputFormatQueuesIDRBoundary|TestEncodedFrameNALDataRejectsInvalidIndexesAndMetadata|TestEncodedFrameRTPDataRejectsInvalidIndexesAndMetadata)$' -count=1 -v
run_gate encoder-residual-boundary go test ./tests -run '^TestEncoderResidualShapedPDeltaRemainsPIntraPCMAcrossPublicOutputs$' -count=1 -v
run_gate encoder-allocation-canary go test ./tests -run '^TestEncoderEncodeIntoAllocationCanary$' -count=1 -v
run_gate encoder-writers go test ./internal/h264 -run '^(TestBitWriter|TestAppendNAL|TestAppendAVC|TestBuildEncoder|TestAppendSEI|TestCAVLCWriteResidual|TestWriteCAVLCInterPBoundedMacroblock|TestEncodeI420P16x16ResidualSliceRBSP)' -count=1 -v
run_gate encoder-benchmem go test . -run '^$' -bench "$bench_pattern" -benchmem -benchtime "$bench_time"

printf '\nall encoder release-evidence gates passed\n' | tee -a "$summary"
