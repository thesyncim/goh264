#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

timestamp="${GOH264_ENCODER_QUALITY_TIMESTAMP:-$(date -u +%Y%m%dT%H%M%SZ)}"
out_dir="${GOH264_ENCODER_QUALITY_DIR:-$ROOT/.artifacts/h264-encoder-quality-evidence/$timestamp}"
mkdir -p "$out_dir"

summary="$out_dir/summary.txt"
bench_pattern="${GOH264_ENCODER_BENCH_PATTERN:-BenchmarkEncode.*I420}"
bench_time="${GOH264_ENCODER_BENCHTIME:-20x}"
ffmpeg_bin="${GOH264_FFMPEG_BIN:-ffmpeg}"
encoder_api_surface_tests='^(TestEncoderEncodeIntoRTPPacketsDoNotAliasAccessUnitData|TestEncoderConfigExplicitQPZeroConstructsAndEncodes|TestEncoderSetQPZeroSurvivesNoopReconfigureAndEncodes|TestEncoderReconfigureLimitPointersDisableBudgets|TestEncoderReconfigureZeroScalarFieldsAreNoOps|TestEncoderReconfigureLimitsGroupUpdatesBudgetsAtomically|TestEncoderZeroValueExplicitSettersRejectWithoutMutation|TestEncoderNonRTPConfigsRejectInvalidRTPControls|TestEncoderInvalidRTPControlsRejectForNonRTPOutputsWithoutMutation|TestEncoderReconfigureOutputFormatQueuesIDRBoundary|TestEncoderInvalidRTPSettersPreservePacketState|TestEncoderFrameColorDoesNotOverrideConfigHeaders|TestEncoderValidSetterPreservesPendingIDR|TestEncoderInvalidSetterPreservesPendingIDR|TestEncoderValidReconfigurePreservesPendingIDR|TestEncoderValidOutputReconfigurePreservesPendingIDR|TestEncoderInvalidReconfigurePreservesPendingIDR|TestEncoderInvalidReconfigureWithForceIDRDoesNotQueueIDR|TestEncoderFrameRateInvalidUpdatesPreserveLiveState|TestEncoderSetIntraRefreshEnableIsUnsupportedAndPreservesState|TestEncoderSetIntraRefreshDisablePreservesLiveReference|TestEncoderSetLimitsUpdatesBudgetsAtomically|TestEncodedFrameNALDataRejectsInvalidIndexesAndMetadata|TestEncodedFrameRTPDataRejectsInvalidIndexesAndMetadata|TestEncodedFrameAppendNALAndAccessUnitDataReturnCallerOwnedBytes|TestEncodedFrameAppendRTPDataReturnsCallerOwnedBytes|TestEncoderAppendHelpersIsolateOverlappingSource|TestEncoderParameterSetsAVCCReturnsCallerOwnedBytes|TestEncoderRTPPacketDataHelpersReturnClippedCallerOwnedBytes|TestEncodedFrameCloneRejectsInvalidMetadata|TestEncoderHelperClonesRejectOverflowedPublicStorage|TestEncoderAppendHelpersRejectOverflowedPublicStorage|TestEncodedFrameOutputHelpersRejectOverflowedPublicStorage)$'
encoder_bitstream_oracle_tests='^(TestEncoderEncodeAnnexBIDRIntraPCMDecodesThroughLocalAndFFmpeg|TestEncoderEncodeCroppedAnnexBIDRIntraPCMDecodesVisibleFrame|TestEncoderEncodeAVCIDRIntraPCMDecodesThroughConfiguredSurface|TestEncoderEncodeIdenticalSecondFrameUsesPSkipReference|TestEncoderEncodeExactP16x16NoResidualMotion|TestEncoderEncodeExactP16x16NoResidualMotionForAVCAndRTP|TestEncoderEncodeExactP16x16NoResidualMotionWithDeblockControls|TestEncoderEncodeExactP16x16NoResidualMotionWithDeblockControlsForAVCAndRTP|TestEncoderEncodeMacroblockAlignedExactP16x16NoResidualMotion|TestEncoderEncodePerMacroblockExactP16x16NoResidualMotionForAnnexBAVCRTP|TestEncoderEncodePerMacroblockExactP16x16FallsBackWithDeblockControls|TestEncoderEncodePerMacroblockExactP16x16FallsBackWithDeblockControlsForAVCAndRTP|TestEncoderEncodeOddPixelExactP16x16NoResidualMotionWithConstantChroma|TestEncoderEncodeOddPixelExactP16x16FallsBackWithDeblockControls|TestEncoderEncodeOddPixelExactP16x16FallsBackWithDeblockControlsForAVCAndRTP|TestEncoderEncodeOddPixelExactP16x16NoResidualMotionForAVCAndRTP|TestEncoderEncodeOddPixelExactP16x16RequiresConstantChroma|TestEncoderEncodeChangedSecondFrameUsesPIntraPCM|TestEncoderEncodeChangedSecondFrameUsesPIntraPCMWithDefaultDeblock|TestEncoderEncodeChangedSecondFrameUsesPIntraPCMWithSliceBoundaryDeblock|TestEncoderEncodeChangedPIntraPCMRecoveryPointSEIForAVCAndRTP|TestEncoderResidualShapedPDeltaUsesResidualPAcrossPublicOutputs|TestEncoderChromaOnlyResidualPUsesResidualAcrossPublicOutputs|TestEncoderCombinedResidualPUsesResidualAcrossPublicOutputs|TestEncoderMultiMacroblockLumaDCResidualPUsesResidualAcrossPublicOutputs|TestEncoderSliceCountSplitsIDRPSkipAndPIntraPCMAccessUnits|TestEncoderSliceCountFeedsRTPMode1SingleNALPackets|TestEncoderEncodeForceIDRBypassesPSkipReference|TestEncoderEncodeRTPMode1FragmentsIDRAccessUnit|TestEncoderEncodeRTPMode1STAPAAggregatesParameterSets|TestEncoderEncodeRTPMode1STAPADoesNotAggregateChangedPRecoverySEI|TestEncoderEncodeRTPMode0EmitsSingleNALPackets|TestEncoderEncodeRTPMode0EmitsPFrameSingleNALPackets|TestEncoderRTPMode1STAPAFallbackAtSmallPayloadPreservesLiveState)$'
encoder_output_ownership_tests='^(TestEncoderFrameCloneDeepCopiesInputPlanes|TestEncoderParameterSetsReturnCallerOwnedSurfaces|TestEncoderParameterSetsCloneDeepCopiesSurfaces|TestEncoderParameterSetsAppendHelpersReturnCallerOwnedBytes|TestEncoderSEICloneDeepCopiesSurfaces|TestEncoderSEIAppendHelpersReturnCallerOwnedBytes|TestEncoderEncodeIntoRTPMode0RejectPreservesCallerBuffer|TestEncoderEncodeIntoLateDropPreservesCallerBuffer|TestEncoderEncodeIntoBitrateDropPreservesCallerBuffer|TestEncoderEncodeIntoRTPPacketsDoNotAliasAccessUnitData|TestEncodedFrameAppendNALAndAccessUnitDataReturnCallerOwnedBytes|TestEncodedFrameAppendRTPDataReturnsCallerOwnedBytes|TestEncoderRTPPacketDataHelpersReturnClippedCallerOwnedBytes|TestEncoderDoesNotRetainInputFramePlanes|TestEncoderEncodeResultsSurviveLaterEncode|TestEncodedFrameCloneDeepCopiesResultStorage|TestEncoderEncodeNALUnitsAppendDoesNotAliasLaterResult|TestEncoderEncodeRTPPacketSlicesAppendDoesNotAliasNextPacket|TestEncoderEncodeRTPPacketsAppendDoesNotAliasLaterResult|TestEncoderRTPPacketsDoNotAliasEncodedFrameData|TestEncoderEncodeIntoRTPPacketsDoNotAliasCallerBuffer|TestEncoderRTPPacketCallbackPacketsSurviveLaterEncode)$'

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

run_exact_go_test_gate() {
    local name="$1"
    local pkg="$2"
    local pattern="$3"
    shift 3
    if [[ "${pattern:0:2}" != "^(" || "${pattern: -2}" != ")$" ]]; then
        printf 'run_exact_go_test_gate %s requires an anchored exact-name pattern\n' "$name" | tee -a "$summary" >&2
        exit 1
    fi
    local expected="${pattern:2:${#pattern}-4}"
    local -a expected_names=()
    IFS='|' read -r -a expected_names <<<"$expected"
    local list_log="$out_dir/$name-list.log"
    {
        printf '\n== %s-list ==\n' "$name"
        printf 'command: go test %q -run %q -list %q\n' "$pkg" '^$' "$pattern"
    } | tee -a "$summary"
    go test "$pkg" -run '^$' -list "$pattern" 2>&1 | tee "$list_log"
    for test_name in "${expected_names[@]}"; do
        if ! grep -Fxq "$test_name" "$list_log"; then
            printf 'status: fail (missing focused test %s)\n' "$test_name" | tee -a "$summary" >&2
            exit 1
        fi
    done
    printf 'status: pass\n' | tee -a "$summary"
    run_gate "$name" go test "$pkg" -run "$pattern" "$@"
}

{
    printf 'commit=%s\n' "$(git rev-parse HEAD)"
    printf 'branch=%s\n' "$(git branch --show-current)"
    printf 'date_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    printf 'go=%s\n' "$(go version)"
    printf 'ffmpeg_bin=%s\n' "$ffmpeg_bin"
    printf 'bench_pattern=%s\n' "$bench_pattern"
    printf 'bench_time=%s\n' "$bench_time"
} >"$summary"

printf 'writing encoder quality evidence to %s\n' "$out_dir" >&2

if [[ "${GOH264_ENCODER_QUALITY_ALLOW_DIRTY:-0}" != "1" ]]; then
    status="$(git status --short)"
    if [[ -n "$status" ]]; then
        {
            printf '\nworktree-clean: failed\n'
            printf '%s\n' "$status"
            printf 'set GOH264_ENCODER_QUALITY_ALLOW_DIRTY=1 only for local diagnostics\n'
        } | tee -a "$summary" >&2
        exit 1
    fi
fi
printf '\nworktree-clean: pass\n' | tee -a "$summary"

if ! ffmpeg_path="$(command -v "$ffmpeg_bin")"; then
    {
        printf '\nffmpeg-oracle: fail\n'
        printf 'missing ffmpeg binary %q; set GOH264_FFMPEG_BIN to the oracle binary\n' "$ffmpeg_bin"
    } | tee -a "$summary" >&2
    exit 1
fi
{
    printf '\nffmpeg-oracle: pass\n'
    printf 'path=%s\n' "$ffmpeg_path"
    "$ffmpeg_path" -version | sed -n '1p'
} | tee -a "$summary"
export GOH264_ENCODER_REQUIRE_FFMPEG=1

run_gate git-diff-check git diff --check
run_gate git-diff-cached-check git diff --cached --check
run_gate go-vet go vet ./...
run_gate go-test-all go test ./...
run_go_test_gate encoder-contract ./tests '^TestEncoder' -count=1 -v
run_exact_go_test_gate encoder-api-surfaces ./tests "$encoder_api_surface_tests" -count=1 -v
run_exact_go_test_gate encoder-output-ownership ./tests "$encoder_output_ownership_tests" -count=1 -v
run_exact_go_test_gate encoder-bitstream-oracles ./tests "$encoder_bitstream_oracle_tests" -count=1 -v
run_go_test_gate encoder-residual-boundary ./tests '^(TestEncoderResidualShapedPDeltaUsesResidualPAcrossPublicOutputs|TestEncoderChromaOnlyResidualPUsesResidualAcrossPublicOutputs|TestEncoderCombinedResidualPUsesResidualAcrossPublicOutputs|TestEncoderMultiMacroblockLumaDCResidualPUsesResidualAcrossPublicOutputs)$' -count=1 -v
run_go_test_gate encoder-allocation-canary ./tests '^TestEncoderEncodeIntoAllocationCanary$' -count=1 -v
run_go_test_gate encoder-writers ./internal/h264 '^(TestBitWriter|TestAppendNAL|TestAppendAVC|TestBuildEncoder|TestAppendSEI|TestCAVLCWriteResidual|TestWriteCAVLCInterPBoundedMacroblock|TestEncodeI420P16x16ResidualSliceRBSP)' -count=1 -v
run_gate encoder-benchmem go test . -run '^$' -bench "$bench_pattern" -benchmem -benchtime "$bench_time"

printf '\nall encoder quality-evidence gates passed\n' | tee -a "$summary"
