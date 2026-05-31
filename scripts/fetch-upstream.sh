#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
URL="https://github.com/FFmpeg/FFmpeg.git"
TAG="n8.0.1"
COMMIT="894da5ca7d742e4429ffb2af534fcda0103ef593"
DST="$ROOT/.upstream/ffmpeg-$TAG"

if [[ -d "$DST/.git" ]]; then
    have="$(git -C "$DST" rev-parse HEAD)"
    if [[ "$have" == "$COMMIT" ]]; then
        printf 'upstream cache already pinned at %s\n' "$COMMIT"
        exit 0
    fi
    printf 'upstream cache exists at %s but is %s, expected %s\n' "$DST" "$have" "$COMMIT" >&2
    exit 1
fi

mkdir -p "$(dirname "$DST")"
git -c advice.detachedHead=false clone \
    --depth 1 \
    --branch "$TAG" \
    --filter=blob:none \
    --sparse \
    "$URL" \
    "$DST"

git -C "$DST" sparse-checkout set --skip-checks \
    libavcodec \
    libavutil \
    tests/ref/fate \
    tests/fate \
    Makefile \
    configure \
    COPYING.LGPLv2.1 \
    LICENSE.md \
    CREDITS

have="$(git -C "$DST" rev-parse HEAD)"
if [[ "$have" != "$COMMIT" ]]; then
    printf 'fetched %s, expected %s\n' "$have" "$COMMIT" >&2
    exit 1
fi

printf 'upstream cache pinned at %s\n' "$COMMIT"

