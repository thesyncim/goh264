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

The embedded smoke bitstreams currently have these decoded-frame oracles:

- single-IDR rawvideo frame MD5: `8aaefe0adcea094cfb5161a060bab4e2`
- two-frame IDR/P rawvideo frame MD5s: `8aaefe0adcea094cfb5161a060bab4e2`, `8aaefe0adcea094cfb5161a060bab4e2`
- deblock-enabled `testsrc2` IDR rawvideo frame MD5: `54b049d05d99dc31d270402e798d4af4`
- deblock-enabled `testsrc2` IDR/P rawvideo frame MD5s: `54b049d05d99dc31d270402e798d4af4`, `681e6d4ef3058d3880346e8039e95b94`
- deblock-enabled `testsrc2` IDR/P rawvideo frame MD5s with PPS default `num_ref_idx_l0_active_minus1 = 1`: `54b049d05d99dc31d270402e798d4af4`, `681e6d4ef3058d3880346e8039e95b94`, `ef38cc80fb47f60e38abc2502af7e5f9`, `0cee44ff1f8279a97bc3e56e4f58f802`
- Main-profile CAVLC weighted-P `testsrc2` fade rawvideo frame MD5s: `8aaefe0adcea094cfb5161a060bab4e2`, `50de7a9591980d98580e8cc5bdf907cb`, `c6df9314a9f54e22d49db2316f12eb99`, `9244803e5a615a34427608350be0fbda`
- same `testsrc2` encode with loop filter disabled: `b729e0367dccdfd707a7ea0c6e68c06e`
- dimensions: `16x16`
- frame payload size: `384` bytes (`yuv420p`)

The AVC/NALFF packet-input tests mechanically convert those Annex B fixtures to
big-endian length-prefixed NAL units while preserving each raw NAL payload. The
default Go tests compare the same rawvideo MD5s through explicit `nal_length_size`
values 2, 3, and 4; avcC extradata parsing remains outside this safe point.

Reference-picture unit coverage now includes FFmpeg's progressive frame-picture
long-term P-list behavior: default long refs after short refs, ref-list
modification op `2`, IDR/non-IDR long-term marking, short-to-long moves,
long-to-unused removal, max-long pruning, reset, and mixed short/long sliding
window accounting. A native long-ref bitstream oracle is still pending.

## Decoder Boundary

Included:

- H.264 Annex B byte-stream parsing
- H.264 AVC/NALFF length-prefixed packet parsing when the caller supplies `nal_length_size`
- H.264 NAL headers and RBSP handling
- SPS/PPS, slice headers, entropy decode, macroblock decode, prediction, inverse transforms, loop filtering, reference picture management, and frame output as the port advances

Excluded unless directly required by decoder parity:

- H.264 encoder files
- Bitstream filters
- FFmpeg muxer/demuxer/filter frontends
- Hardware acceleration backends
- Non-H.264 codecs
