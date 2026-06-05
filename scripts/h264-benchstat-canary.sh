#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COUNT="${GOH264_BENCHSTAT_COUNT:-5}"
BENCHTIME="${GOH264_BENCHSTAT_TIME:-100ms}"
PATTERN="${GOH264_BENCHSTAT_PATTERN:-BenchmarkDecodeAnnexBHigh10IDRP}"

cd "$ROOT_DIR"
exec go test \
  -run '^$' \
  -bench "$PATTERN" \
  -benchmem \
  -count "$COUNT" \
  -benchtime "$BENCHTIME" \
  .
