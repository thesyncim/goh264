#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FUZZTIME="${GOH264_DECODER_FUZZTIME:-1s}"
PATTERN="${GOH264_DECODER_FUZZ_PATTERN:-^FuzzDecodePublicSurfacesNoPanic$}"

cd "$ROOT"
exec go test ./tests -run '^$' -fuzz "$PATTERN" -fuzztime "$FUZZTIME"
