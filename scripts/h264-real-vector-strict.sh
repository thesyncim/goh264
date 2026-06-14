#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

filter="${1:-${GOH264_CORPUS_FILTER:-}}"
export GOH264_CORPUS_CACHE="${GOH264_CORPUS_CACHE:-/tmp/goh264-fate-probe-cache}"
export GOH264_CORPUS_FETCH="${GOH264_CORPUS_FETCH:-1}"
if [[ -n "$filter" ]]; then
    export GOH264_CORPUS_FILTER="$filter"
fi

printf 'real-vector strict oracle'
if [[ -n "${GOH264_CORPUS_FILTER:-}" ]]; then
    printf ' filter=%s' "$GOH264_CORPUS_FILTER"
fi
printf '\n'

pattern='^TestH264RealVectorStrictOracle$'
expected_tests=(TestH264RealVectorStrictOracle)
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/goh264-real-vector-strict.XXXXXX")"
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
GOH264_REAL_VECTOR_STRICT=1 go test ./tests -run "$pattern" -count=1 -v 2>&1 | tee "$log"
if grep -Eq '^--- SKIP: ' "$log"; then
    printf 'status: fail (focused test skipped)\n' >&2
    exit 1
fi
