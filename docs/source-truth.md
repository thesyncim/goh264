# Source Truth

This port follows the `c-cpp-parity-cloner` workflow: source-shaped translation first, oracle parity before optimization.

## Upstream Pin

- Project: FFmpeg `libavcodec`
- Scope: H.264 decoder dependency cone only
- Upstream URL: `https://github.com/FFmpeg/FFmpeg.git`
- Release tag: `n8.0.1`
- Annotated tag object: `d22ecc4f6f3fca77b3e71b18641ceddb25973e97`
- Peeled commit: `894da5ca7d742e4429ffb2af534fcda0103ef593`
- Local cache path: `.upstream/ffmpeg-n8.0.1`

The cache is intentionally ignored by git. Recreate it with:

```sh
scripts/fetch-upstream.sh
```

## Native Oracle

The first oracle surface uses the installed FFmpeg command line tools:

- `ffmpeg version 8.0.1`
- `libavcodec 62.11.100`
- `libavutil 60.8.100`
- Platform observed at baseline: `darwin/arm64`

The oracle tests are opt-in because they depend on local binaries:

```sh
GOH264_ORACLE=1 go test ./...
```

The CABAC arithmetic oracle also requires a local C compiler. It compiles the
pinned FFmpeg `libavcodec/cabac.c` and `cabac_functions.h` from
`.upstream/ffmpeg-n8.0.1` in a temporary directory and compares primitive traces
against the Go port.

The embedded smoke bitstream currently has this decoded-frame oracle:

- rawvideo frame MD5: `8aaefe0adcea094cfb5161a060bab4e2`
- dimensions: `16x16`
- frame payload size: `384` bytes (`yuv420p`)

## Decoder Boundary

Included:

- H.264 Annex B byte-stream parsing
- H.264 NAL headers and RBSP handling
- SPS/PPS, slice headers, entropy decode, macroblock decode, prediction, inverse transforms, loop filtering, reference picture management, and frame output as the port advances

Excluded unless directly required by decoder parity:

- H.264 encoder files
- Bitstream filters
- FFmpeg muxer/demuxer/filter frontends
- Hardware acceleration backends
- Non-H.264 codecs
