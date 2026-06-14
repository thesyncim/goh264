#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

scripts/fetch-upstream.sh

export GOH264_REAL_VECTOR_UPSTREAM_AUDIT=1
pattern='^TestH264RealVector(ImportedUpstreamInventory|PinnedFATEInventory|DocumentationCounts|UpstreamFATECoverage)$'
expected_tests=(
    TestH264RealVectorImportedUpstreamInventory
    TestH264RealVectorPinnedFATEInventory
    TestH264RealVectorDocumentationCounts
    TestH264RealVectorUpstreamFATECoverage
)
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/goh264-real-vector-upstream-audit.XXXXXX")"
trap 'rm -rf "$tmp_dir"' EXIT
list_log="$tmp_dir/list.log"

go test ./tests -run '^$' -list "$pattern" 2>&1 | tee "$list_log"
for test_name in "${expected_tests[@]}"; do
    if ! grep -Fxq "$test_name" "$list_log"; then
        printf 'status: fail (missing focused test %s)\n' "$test_name" >&2
        exit 1
    fi
done

log="$tmp_dir/test.log"
go test ./tests -run "$pattern" -count=1 -v "$@" 2>&1 | tee "$log"
if grep -Eq '^--- SKIP: ' "$log"; then
    printf 'status: fail (focused test skipped)\n' >&2
    exit 1
fi
