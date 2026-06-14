#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FUZZTIME="${GOH264_DECODER_FUZZTIME:-1s}"
PATTERN="${GOH264_DECODER_FUZZ_PATTERN:-^FuzzDecodePublicSurfacesNoPanic$}"

cd "$ROOT"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/goh264-decoder-fuzz-smoke.XXXXXX")"
trap 'rm -rf "$tmp_dir"' EXIT
list_log="$tmp_dir/list.log"

go test ./tests -run '^$' -list "$PATTERN" 2>&1 | tee "$list_log"
if ! grep -Eq '^Fuzz' "$list_log"; then
    printf 'status: fail (no matching fuzz targets)\n' >&2
    exit 1
fi

log="$tmp_dir/fuzz.log"
go test ./tests -run '^$' -fuzz "$PATTERN" -fuzztime "$FUZZTIME" 2>&1 | tee "$log"
if grep -Eq 'testing: warning: no fuzz tests to fuzz|^--- SKIP: ' "$log"; then
    printf 'status: fail (fuzz target did not run)\n' >&2
    exit 1
fi
