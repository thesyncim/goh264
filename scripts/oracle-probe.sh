#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
    printf 'usage: %s input.h264\n' "$0" >&2
    exit 2
fi

ffprobe -v error \
    -select_streams v:0 \
    -show_entries stream=codec_name,profile,width,height,level,pix_fmt \
    -of json \
    "$1"

