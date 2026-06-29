#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COUNT="${GOH264_BENCHSTAT_COUNT:-5}"
BENCHTIME="${GOH264_BENCHSTAT_TIME:-${GOH264_BENCHSTAT_BENCHTIME:-100ms}}"
PATTERN="${GOH264_BENCHSTAT_PATTERN:-Benchmark(Decode.*AnnexB.*High10IDRP|FrameAppendRawYUVBytesLEHigh10IDRP)}"

cd "$ROOT_DIR"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/goh264-benchstat-canary.XXXXXX")"
trap 'rm -rf "$tmp_dir"' EXIT
list_log="$tmp_dir/list.log"

go test -run '^$' -list "$PATTERN" . 2>&1 | tee "$list_log"
if ! grep -Eq '^Benchmark' "$list_log"; then
  printf 'status: fail (no matching benchmarks)\n' >&2
  exit 1
fi

log="$tmp_dir/bench.log"
go test \
  -run '^$' \
  -bench "$PATTERN" \
  -benchmem \
  -count "$COUNT" \
  -benchtime "$BENCHTIME" \
  . 2>&1 | tee "$log"
if ! grep -Eq '^Benchmark' "$log"; then
  printf 'status: fail (benchmark did not run)\n' >&2
  exit 1
fi
