#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

timestamp="${GOH264_QUALITY_EVIDENCE_TIMESTAMP:-$(date -u +%Y%m%dT%H%M%SZ)}"
out_dir="${GOH264_QUALITY_EVIDENCE_DIR:-$ROOT/.artifacts/h264-quality-evidence/$timestamp}"
mkdir -p "$out_dir"

summary="$out_dir/summary.txt"
filter="${GOH264_QUALITY_PERF_FILTER:-canl4}"
benchstat_time="${GOH264_BENCHSTAT_TIME:-${GOH264_BENCHSTAT_BENCHTIME:-100ms}}"
export GOH264_BENCHSTAT_TIME="$benchstat_time"
decoder_output_ownership_tests='^(TestDecodeConfiguredAVCFramesDoesNotAliasCallerBuffer|TestDecodeAVCCFramesDoesNotAliasCallerBuffer|TestDecodeAnnexBFramesDoesNotAliasCallerBuffer|TestDecodeAVCFramesDoesNotAliasCallerBuffer|TestDecodeFramesAutoAnnexBDoesNotAliasCallerBuffer|TestDecodeFramesAutoAVCDoesNotAliasCallerBuffer|TestDecodePacketFramesDoesNotAliasCallerBuffer|TestConfigureAVCCDoesNotAliasCallerBuffer|TestParseHeadersAnnexBDoesNotAliasCallerBuffer|TestParseHeadersAVCDoesNotAliasCallerBuffer|TestPacketSideDataCloneDeepCopiesPayload|TestPacketCloneDeepCopiesDataAndSideData|TestFrameSideDataCloneDeepCopiesNestedStorage|TestFrameCloneDeepCopiesPlanesAndSideData|TestDecodePacketFramesNewExtradataDoesNotAliasCallerBuffers|TestDecodePacketFramesAnnexBNewExtradataDoesNotAliasCallerBuffers|TestDecodeFrameStructuredSideDataIsCallerOwned|TestDecodeFrameSideDataByteSlicesAreCallerOwned|TestDecodeFrameSEIStructuredSideDataIsCallerOwned|TestDecodeFramePictureTimingSlicesAreCallerOwned|TestFrameRawYUVBytesLEReturnsOwnedExactBuffer|TestFrameAppendRawYUVIsolatesOverlappingSource|TestFrameAppendRawYUVErrorPreservesCallerBuffer|TestFrameAppendRawYUVHighErrorPreservesCallerBuffer)$'

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

require_oracle_command() {
    local name="$1"
    local path
    if ! path="$(command -v "$name")"; then
        printf '\n%s-oracle: fail\nmissing command %q\n' "$name" "$name" | tee -a "$summary" >&2
        exit 1
    fi
    printf '\n%s-oracle: pass\npath=%s\n' "$name" "$path" | tee -a "$summary"
}

require_oracle_file() {
    local path="$1"
    if [[ ! -f "$path" ]]; then
        printf '\nupstream-oracle-file: fail\nmissing %s\n' "$path" | tee -a "$summary" >&2
        exit 1
    fi
    printf '\nupstream-oracle-file: pass\npath=%s\n' "$path" | tee -a "$summary"
}

run_exact_oracle_go_test_gate() {
    local name="$1"
    local pkg="$2"
    local pattern="$3"
    shift 3
    if [[ "${pattern:0:2}" != "^(" || "${pattern: -2}" != ")$" ]]; then
        printf 'run_exact_oracle_go_test_gate %s requires an anchored exact-name pattern\n' "$name" | tee -a "$summary" >&2
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

    local log="$out_dir/$name.log"
    {
        printf '\n== %s ==\n' "$name"
        printf 'command: env GOH264_ORACLE=1 go test %q -run %q' "$pkg" "$pattern"
        printf ' %q' "$@"
        printf '\n'
    } | tee -a "$summary"
    env GOH264_ORACLE=1 go test "$pkg" -run "$pattern" "$@" 2>&1 | tee "$log"
    if grep -Eq '^--- SKIP: ' "$log"; then
        printf 'status: fail (oracle test skipped)\n' | tee -a "$summary" >&2
        exit 1
    fi
    printf 'status: pass\n' | tee -a "$summary"
}

{
    printf 'commit=%s\n' "$(git rev-parse HEAD)"
    printf 'branch=%s\n' "$(git branch --show-current)"
    printf 'date_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    printf 'go=%s\n' "$(go version)"
    printf 'corpus_cache=%s\n' "${GOH264_CORPUS_CACHE:-/tmp/goh264-fate-probe-cache}"
    printf 'quality_perf_filter=%s\n' "$filter"
    printf 'quality_alloc_filter=%s\n' "${GOH264_QUALITY_ALLOC_FILTER:-canl4}"
    printf 'benchstat_pattern=%s\n' "${GOH264_BENCHSTAT_PATTERN:-Benchmark(Decode.*AnnexBHigh10IDRP|FrameAppendRawYUVBytesLEHigh10IDRP|Encode.*I420)}"
    printf 'benchstat_time=%s\n' "$GOH264_BENCHSTAT_TIME"
} >"$summary"

printf 'writing decoder quality evidence to %s\n' "$out_dir" >&2

if [[ "${GOH264_QUALITY_ALLOW_DIRTY:-0}" != "1" ]]; then
    status="$(git status --short)"
    if [[ -n "$status" ]]; then
        {
            printf '\nworktree-clean: failed\n'
            printf '%s\n' "$status"
            printf 'set GOH264_QUALITY_ALLOW_DIRTY=1 only for local diagnostics\n'
        } | tee -a "$summary" >&2
        exit 1
    fi
fi
printf '\nworktree-clean: pass\n' | tee -a "$summary"

run_gate git-diff-check git diff --check
run_gate git-diff-cached-check git diff --cached --check
run_gate go-vet go vet ./...
run_gate go-test-all go test ./...
run_exact_go_test_gate decoder-api-surfaces ./tests '^(TestParseHeadersAnnexBBlack16|TestParseHeadersAVCBlack16|TestPackageAVCCParsersDoNotMutateDecoderState|TestDecodeAVCOneByteLengthSizePublicSurfaces|TestFrameCloneRejectsOverflowedPublicStorage|TestDecoderCheckedCloneHelpersRejectOverflowedPublicStorage|TestDecodePacketFramesRejectsOverflowedSideDataListWithoutDroppingPacket|TestDecodeAVCCFramesIncompatibleConfigurationDoesNotUseStalePFrameReference|TestDecodePacketFramesNewExtradataIncompatibleConfigurationDoesNotUseStalePFrameReference|TestDecodePacketFramesAnnexBNewExtradataIncompatibleConfigurationDoesNotUseStalePFrameReference|TestDecodePacketFramesDuplicateNewExtradataFirstEntryWins|TestDecodePacketFramesMalformedDuplicateNewExtradataSuppressesLaterEntries|TestDecodeFramesInBandIncompatibleParameterSetsDoNotUseStalePFrameReference|TestDecodePacketFramesInBandIncompatibleParameterSetsDoNotUseStalePFrameReference|TestDecodeConfiguredAVCFramesInBandIncompatibleParameterSetsDoNotUseStalePFrameReference|TestParseHeadersAnnexBIncompatibleHeadersDoNotUseStalePFrameReference|TestParseHeadersAVCIncompatibleHeadersDoNotUseStalePFrameReference|TestDecodeFramesValidInBandParameterSetsBeforeDamagedSliceUpdateConfigAndRecover|TestValidAVCCBeforeDamagedSliceUpdatesConfigAndRecover|TestDecodeAVCCFramesSwitchesValidConfigurationWithoutReset|TestDecodePacketFramesNewExtradataSwitchesValidAVCConfiguration|TestDecodeAVCCFramesMultiSPSConfigurationUsesPacketActiveSPSForDPBReset|TestDecodeFramesStandaloneMultiSPSConfigurationResetsForNonFirstActiveSPS|TestDecodePacketFramesMultiSPSNewExtradataUsesPacketActiveSPSForDPBReset|TestDecodePacketFramesAnnexBMultiSPSNewExtradataUsesPacketActiveSPSForDPBReset|TestDecoderAVCConfigUsesAVCCFirstSPSForMultiSPSConfiguration|TestDecoderAVCConfigUsesPacketActiveSPSForMultiSPSConfiguration)$' -count=1 -v
run_exact_go_test_gate decoder-output-ownership ./tests "$decoder_output_ownership_tests" -count=1 -v
run_exact_go_test_gate decoder-ref-modifications ./internal/h264 '^(TestSimpleFrameDPBRejectsMissingShortRefModificationTarget|TestSimpleFrameDPBRejectsMissingLongRefModificationTarget|TestSimpleFrameDPBReordersShortRefs|TestSimpleFrameDPBReordersLongRefs)$' -count=1 -v
require_oracle_command ffmpeg
require_oracle_command ffprobe
require_oracle_command cc
require_oracle_file .upstream/ffmpeg-n8.0.1/libavcodec/cabac.c
require_oracle_file .upstream/ffmpeg-n8.0.1/libavcodec/h264idct_template.c
run_exact_oracle_go_test_gate decoder-ffmpeg-oracle-smoke ./tests '^(TestS12MTimecodePackingMatchesNativeFFmpegOracle|TestFFprobeOracleBlack16|TestFFmpegFrameMD5OracleBlack16)$' -count=1 -v
run_exact_oracle_go_test_gate decoder-native-oracle-smoke ./internal/h264 '^(TestCABACPrimitiveSequenceUpstreamOracle|TestH264IDCTUpstreamOracle)$' -count=1 -v

if [[ -s testdata/h264/realvectors/failures.jsonl && "${GOH264_QUALITY_ALLOW_KNOWN_RED:-0}" != "1" ]]; then
    printf '\nknown-red-failures: testdata/h264/realvectors/failures.jsonl is not empty; set GOH264_QUALITY_ALLOW_KNOWN_RED=1 only for local diagnostics\n' | tee -a "$summary" >&2
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
run_gate quality-allocation-canary scripts/h264-real-vector-quality-alloc.sh
run_gate benchstat-canary scripts/h264-benchstat-canary.sh

run_env_gate performance-evidence \
    GOH264_PERF_DIR="$out_dir/performance-bundle" \
    scripts/h264-performance-evidence.sh "$filter"

printf '\nall decoder quality-evidence gates passed\n' | tee -a "$summary"
