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
- Main-profile CABAC `testsrc2` IDR/P rawvideo frame MD5s: `57948a884e4468c79f3291b2693263de`, `4fb1e27b7087e9f1aa485402993ca525`, `a7e3e74bb19403d111dd2ffdb4455102`, `1202e58b9b15f56a341fea8787bcc769`
- High 4:2:0 32x32 CAVLC 8x8-DCT `testsrc2` IDR/P rawvideo frame MD5s: `4d912de8c22019c29a46f3966607408c`, `11d6e207060405262de9a91bbdd298a9`, `6bf6d4689852ae04c3c5f7da495e5e48`, `559d2dfec6c93d5b03fd9f179f8216c4`
- High 4:2:0 32x32 CABAC 8x8-DCT `testsrc2` IDR/P rawvideo frame MD5s: `2f01a945ea8e10134c1c80077e62ca3f`, `2dcdacc98ced800818b6fe09c2e7fa2b`, `20e5d5b88002dcf514d3772316464476`, `8ac7c3f6f20b7e002fdf895532a3fd9b`
- High 4:2:2 CAVLC `testsrc2` IDR/P rawvideo frame MD5s: `b37a1f7943ce6c7d9646786f348f4ce9`, `e705648238ec1a68ce2fc83f8d1b7293`, `13cfed6389834373ccb5b6bb61f6cf9d`, `f0b4d1caf4e666cc4767cfe273de480e`
- High 4:2:2 CABAC `testsrc2` IDR/P rawvideo frame MD5s: `e06b0f34fe689940304653e5c3840a53`, `424fb373278235a5d2b0808968cb0e58`, `b6e4d159f8c0b0bb452de55824214ac6`, `892dfdee5dbf37558f99a6fe0c278abb`
- High monochrome CAVLC `testsrc2` IDR/P gray rawvideo frame MD5s: `7d7c6b5414619f78c6303e94f6c69dba`, `6ae5ffb09f3156812deccefdf58a6c74`, `f1dd36e9dbc0f928b6e57afc2022a8f2`, `504e78844c238b097aa59235df29ec07`
- High monochrome CABAC `testsrc2` IDR/P gray rawvideo frame MD5s: `cf88b0a4244f7df1c3c54613f6290345`, `d003fa3ed4b3edd4622c36e4c2b5249c`, `677639d3d5857b18931e727d46e6a4cc`, `fb50b49ba64db3576559b442d3c4a6ad`
- High 4:4:4 Predictive CAVLC `testsrc2` IDR/P yuv444p rawvideo frame MD5s: `0ff3893d32b4b1875412d88a6fa4a5b1`, `008c471027c25eab150c1cc4a30fb9ac`, `ef107480f4c8b836d91e422e1f3c0b75`, `6acd1f8bc304066008a32acf64228305`
- High 4:4:4 Predictive CABAC `testsrc2` IDR/P yuv444p rawvideo frame MD5s: `8539237f1ecaf659fa36c0f76cde8815`, `6f594f9f9f10d12a399d54882ce6c8e5`, `5e4250996d28cff7f2e85b95d78995ff`, `452f232c9a94da5220babd530117a395`
- same `testsrc2` encode with loop filter disabled: `b729e0367dccdfd707a7ea0c6e68c06e`
- dimensions: `16x16` and `32x32`
- frame payload size: `256` bytes (`gray`/`chroma_format_idc == 0`), `384` or `1536` bytes (`yuv420p`), `512` bytes (`yuv422p`), and `768` bytes (`yuv444p`)

The AVC/NALFF packet-input tests mechanically convert those Annex B fixtures to
big-endian length-prefixed NAL units while preserving each raw NAL payload. The
default Go tests compare the same rawvideo MD5s through explicit `nal_length_size`
values 2, 3, and 4. The configured AVC tests additionally build FFmpeg-style
`avcC` extradata from SPS/PPS NAL units, remove those parameter sets from the
packet payload, and prove the separated-config CAVLC ref-list, CABAC IDR/P,
High 4:2:2 CAVLC/CABAC, High 4:4:4 Predictive CAVLC/CABAC, and monochrome
CAVLC/CABAC packets against the same frame MD5s both as bundled packets and as
successive single-frame sample packets that require DPB reference state to
survive across public decoder calls. Monochrome native FFmpeg oracle checks
request `-pix_fmt gray` so the frame-MD5 surface compares only the luma plane
represented by `chroma_format_idc == 0`. The High 4:4:4 Predictive fixtures
carry `disable_deblocking_filter_idc == 1`; they prove 4:4:4 entropy,
prediction, residual reconstruction, motion, and public packet integration, but
not 4:4:4 loop-filter parity.

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
