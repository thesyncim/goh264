#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COUNT="${GOH264_BENCHSTAT_COUNT:-5}"
BENCHTIME="${GOH264_BENCHSTAT_TIME:-${GOH264_BENCHSTAT_BENCHTIME:-100ms}}"
PATTERN="${GOH264_BENCHSTAT_PATTERN:-Benchmark(Decode.*AnnexBHigh10IDRP|FrameAppendRawYUVBytesLEHigh10IDRP|Encode.*I420)}"

cd "$ROOT_DIR"
exec go test \
  -run '^$' \
  -bench "$PATTERN" \
  -benchmem \
  -count "$COUNT" \
  -benchtime "$BENCHTIME" \
  .
