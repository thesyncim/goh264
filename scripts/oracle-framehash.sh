#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
    printf 'usage: %s input.h264\n' "$0" >&2
    exit 2
fi

ffmpeg -v error -f h264 -i "$1" -an -sn -dn -f framemd5 -

