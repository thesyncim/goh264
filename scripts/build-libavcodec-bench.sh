#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output="${1:-$ROOT/.artifacts/bin/goh264-libavcodec-bench}"
cc="${CC:-cc}"

if ! command -v pkg-config >/dev/null 2>&1; then
    printf 'pkg-config is required to locate libavcodec and libavutil\n' >&2
    exit 1
fi

mkdir -p "$(dirname "$output")"
read -r -a ffmpeg_flags <<<"$(pkg-config --cflags --libs libavcodec libavutil)"
"$cc" -std=c11 -O3 -Wall -Wextra -Werror -pthread \
    "$ROOT/tools/libavcodec-bench/main.c" \
    "${ffmpeg_flags[@]}" \
    -o "$output"
printf '%s\n' "$output"
