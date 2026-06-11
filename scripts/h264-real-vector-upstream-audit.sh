#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

scripts/fetch-upstream.sh

export GOH264_REAL_VECTOR_UPSTREAM_AUDIT=1
exec go test ./tests -run '^TestH264RealVector(ImportedUpstreamInventory|PinnedFATEInventory|DocumentationCounts|UpstreamFATECoverage)$' -count=1 -v "$@"
