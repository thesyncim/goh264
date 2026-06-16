#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

timestamp="${GOH264_FULL_QUALITY_TIMESTAMP:-$(date -u +%Y%m%dT%H%M%SZ)}"
out_dir="${GOH264_FULL_QUALITY_DIR:-$ROOT/.artifacts/h264-full-quality-evidence/$timestamp}"
mkdir -p "$out_dir"

summary="$out_dir/summary.txt"

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

{
    printf 'commit=%s\n' "$(git rev-parse HEAD)"
    printf 'branch=%s\n' "$(git branch --show-current)"
    printf 'date_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    printf 'go=%s\n' "$(go version)"
    printf 'decoder_dir=%s\n' "$out_dir/decoder"
} >"$summary"

printf 'writing full quality evidence to %s\n' "$out_dir" >&2

if [[ "${GOH264_FULL_QUALITY_ALLOW_DIRTY:-0}" != "1" ]]; then
    status="$(git status --short)"
    if [[ -n "$status" ]]; then
        {
            printf '\nworktree-clean: failed\n'
            printf '%s\n' "$status"
            printf 'set GOH264_FULL_QUALITY_ALLOW_DIRTY=1 only for local diagnostics\n'
        } | tee -a "$summary" >&2
        exit 1
    fi
    printf '\nworktree-clean: pass\n' | tee -a "$summary"
else
    status="$(git status --short)"
    {
        printf '\nworktree-clean: allowed-dirty\n'
        if [[ -n "$status" ]]; then
            printf '%s\n' "$status"
        else
            printf 'git status --short: empty\n'
        fi
    } | tee -a "$summary"
    export GOH264_QUALITY_ALLOW_DIRTY="${GOH264_QUALITY_ALLOW_DIRTY:-1}"
fi

run_gate go-test-race go test -race ./...

run_env_gate decoder-quality-evidence \
    GOH264_QUALITY_EVIDENCE_DIR="$out_dir/decoder" \
    GOH264_QUALITY_EVIDENCE_TIMESTAMP="$timestamp" \
    scripts/h264-decoder-quality-evidence.sh

printf '\nall full quality-evidence gates passed\n' | tee -a "$summary"
