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

The H.264 prediction oracle compiles the pinned `libavcodec/h264pred_template.c`
and compares 8-bit plus 9/10/12/14-bit high-bit-depth luma/chroma prediction
outputs, including FFmpeg's 4:2:0 and 4:2:2 mad-cow unavailable-neighbor
chroma DC predictors and H.264 lossless prediction-add wrappers.

The H.264 DSP oracle compiles the pinned `libavcodec/h264dsp_template.c` and
`libavcodec/h264addpx_template.c`, comparing 8-bit reference kernels plus the
high-bit-depth add-pixels fallback, 9/10/12/14-bit weighted prediction cases,
and selected high-bit-depth deblocking variants.

The H.264 IDCT oracle compiles the pinned `libavcodec/h264idct_template.c` and
compares 8-bit transform/dequant fixtures plus 9/10/12/14-bit 4x4/8x8 inverse
transform add, DC-only add, luma DC dequant IDCT, and 4:2:0/4:2:2 chroma DC
dequant IDCT fixtures.

The H.264 reconstruction oracle compiles pinned prediction and IDCT templates
and compares source-shaped macroblock reconstruction fixtures, including
10-bit 4:2:0, 12-bit 4:2:2, and 14-bit 4:4:4 high-bit-depth IntraPCM payload
unpacking into uint16 planes plus high-bit-depth intra16x16, intra4x4, and
intra8x8 prediction/IDCT/dequant call-site fixtures.

The H.264 chroma MC and qpel oracles compile the pinned
`libavcodec/h264chroma_template.c` and `libavcodec/h264qpel_template.c`,
comparing 8-bit fixtures plus 9/10/12/14-bit high-bit-depth put/avg variants
across supported widths and fractional-pel positions.

The `ffprobe` header oracle now compares public `StreamInfo` SPS VUI sample
aspect ratio and timing rate for the black16 stream in addition to profile,
level, dimensions, and pixel format.

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
- deblock-enabled High 4:4:4 Predictive 32x32 CAVLC `testsrc2` IDR/P yuv444p rawvideo frame MD5s: `e6522cb7daa4278fa238f995daea8594`, `274c8ec306ee4705f93c3cc6bdedc948`, `d42015040093bf782173b1d8d00a5b74`, `9d93f36ffaeb8caa764f2b06240ba5d7`
- deblock-enabled High 4:4:4 Predictive 32x32 CABAC `testsrc2` IDR/P yuv444p rawvideo frame MD5s: `df7f5b803f967fcd46070b2b182c3805`, `5bc16fb5ebe5c3021e77c7c82c34127c`, `5e0f2020cfefc09d993a68c2963ad8ed`, `f14846abbb44addf3e1ce0e66394b683`
- qp=0 High 4:4:4 Predictive CAVLC/CABAC `testsrc2` IDR/P yuv420p lossless rawvideo frame MD5s: `69fcf25f35e829e5a3d96cbaaf22bbb6`, `8563271dc08ef4ed388ebc1f7016834c`, `1a054a3901101da0f6b6c58d8e71bbdb`, `a0addb72f5ea0957ef8a05b782f0e9ff`
- 16x16 no-skip non-direct B-frame CAVLC `testsrc2` yuv420p rawvideo frame MD5s: `4296e3dc95829cc27071a8685a428494`, `36f5a9b9064709ee891652e8f4e06992`, `aa778b981f96d21489196f6a0faa0959`
- 16x16 no-skip non-direct B-frame CABAC `testsrc2` yuv420p rawvideo frame MD5s: `f5c89cbdd198348f67b10b9e7cc511a7`, `fef9831ddd54882d715ceb50c382efde`, `4b6a7f1c59198ae9b8e31ef4de333e42`
- 16x16 temporal-direct B-frame CAVLC `testsrc2` yuv420p rawvideo frame MD5s: `dca1bb7607ebcd45d700a7b7f9feb2f6`, `6248c3284f9d89ac6346701f8f226ba8`, `0e1be965e4fb7e790038cda9d21845cf`
- 16x16 temporal-direct B-frame CABAC `testsrc2` yuv420p rawvideo frame MD5s: `dca1bb7607ebcd45d700a7b7f9feb2f6`, `6248c3284f9d89ac6346701f8f226ba8`, `0e1be965e4fb7e790038cda9d21845cf`
- 16x16 spatial-direct B-frame CAVLC `testsrc2` yuv420p rawvideo frame MD5s: `dca1bb7607ebcd45d700a7b7f9feb2f6`, `6248c3284f9d89ac6346701f8f226ba8`, `0e1be965e4fb7e790038cda9d21845cf`
- 16x16 spatial-direct B-frame CABAC `testsrc2` yuv420p rawvideo frame MD5s: `dca1bb7607ebcd45d700a7b7f9feb2f6`, `6248c3284f9d89ac6346701f8f226ba8`, `0e1be965e4fb7e790038cda9d21845cf`
- 16x16 non-neutral implicit B-weight CAVLC `testsrc2` yuv420p rawvideo frame MD5s: `4296e3dc95829cc27071a8685a428494`, `8747883f49707799806cf66a630e600e`, `0706cc9ae846c8aefe9597f9e83be042`, `143d2f0e79e82b9d5b6de6f48968c447`
- 16x16 non-neutral implicit B-weight CABAC `testsrc2` yuv420p rawvideo frame MD5s: `f5c89cbdd198348f67b10b9e7cc511a7`, `4a3834dbc6c0ea54fa46d9ec8fd4044e`, `eac9140384dc323ba6e4ef4e7a20c7f6`, `db30cd22f3204ef73b6b8e9ed3fd4e07`
- 32x32 no-skip non-direct B-frame CAVLC `testsrc2` yuv420p rawvideo frame MD5s: `2a9d9acd3e52356ad072de93fdbaca3d`, `96107676801850afd8aed8546397e3bf`, `3967b8bfe3a3a8cde4bc22334008eb1f`
- 32x32 no-skip non-direct B-frame CABAC `testsrc2` yuv420p rawvideo frame MD5s: `88a962a713f37e05f375eee6ee9f385b`, `a165d65aadbe1410829a22df4459539b`, `8d39f667da04571db61fc68919a64ade`
- same `testsrc2` encode with loop filter disabled: `b729e0367dccdfd707a7ea0c6e68c06e`
- dimensions: `16x16` and `32x32`
- frame payload size: `256` bytes (`gray`/`chroma_format_idc == 0`), `384` or `1536` bytes (`yuv420p`), `512` bytes (`yuv422p`), and `768` or `3072` bytes (`yuv444p`)

The AVC/NALFF packet-input tests mechanically convert those Annex B fixtures to
big-endian length-prefixed NAL units while preserving each raw NAL payload. The
default Go tests compare the same rawvideo MD5s through explicit `nal_length_size`
values 2, 3, and 4. The configured AVC tests additionally build FFmpeg-style
`avcC` extradata from SPS/PPS NAL units, remove those parameter sets from the
packet payload, and prove the separated-config CAVLC ref-list, CABAC IDR/P,
High 4:2:0 32x32 8x8-DCT CAVLC/CABAC, High 4:2:2 CAVLC/CABAC,
High 4:4:4 Predictive CAVLC/CABAC, monochrome CAVLC/CABAC, and qp=0
lossless CAVLC/CABAC packets against the same frame MD5s
both as bundled packets and as successive single-frame sample packets that
require DPB reference state to survive across public decoder calls. Native
FFmpeg framemd5 oracle checks cover the 32x32 High 4:2:0 8x8-DCT fixtures in
addition to the 16x16/32x32 families listed below. The
configured B-frame sample tests additionally decode one access unit per call and
then use the public delayed-frame flush to drain retained future P pictures,
covering FFmpeg's `last_pocs`/`has_b_frames` reorder inference and signaled VUI
reorder-depth handling. The generic public `DecodeFrames` tests exercise Annex B
and AVC4 auto-detection, packet-level `avcC` configuration storage, FFmpeg's
configured 4-byte AVC/Annex B sniffing heuristic, and empty-packet delayed flush
over the B-frame configured-sample fixtures. Packet side-data tests mirror
FFmpeg's `AV_PKT_DATA_NEW_EXTRADATA` ordering by applying non-empty side data
before packet NAL splitting, covering both `avcC` and Annex B parameter-set
extradata and repeated side data across sample-by-sample P-frame decode without
resetting DPB reference state. The same public packet surface maps
`AV_PKT_DATA_A53_CC`, `AV_PKT_DATA_AFD`, and
`AV_PKT_DATA_S12M_TIMECODE` onto decoded frame side data as frames are allocated,
so packet side data follows delayed B-frame output. The same path maps
FFmpeg's global video metadata packet side data for display matrix, Stereo3D,
spherical mapping, ICC profile, Dynamic HDR10+, LCEVC, mastering display, content light, ambient
viewing environment, and 3D reference displays, using the native struct layouts
and exact AVRational scaling into the public H.264/H.274 metadata units where
applicable; Dynamic HDR10+ and LCEVC are preserved as opaque byte side data.
The tests cover FFmpeg's first-matching packet entry, H.264's
packet-first A53/AFD/display/stereo ordering, S12M coded-timecode replacement
when picture-timing exports a timecode, coded-SEI precedence over global packet
HDR/ambient/LCEVC metadata, native variable-entry 3D reference display parsing, and
delayed B-frame carriage, while preserving the rawvideo MD5. Public frame
side-data tests prepend synthetic
leading SEI to the black16 fixture and prove the decoded frame retains x264
user-data, A53 closed captions, active-format description, recovery point,
green metadata, display orientation, frame packing, alternative transfer,
ambient viewing environment, H.274 film-grain characteristics, mastering
display, content-light metadata, and VNOVA LCEVC bytes while preserving the rawvideo MD5. The
same test proves FFmpeg/libavutil frame side-data projection for H.264 frame
packing into stereo3D metadata, display orientation into the native display
matrix, and mastering-display RGB ordering plus `has_primaries`/`has_luminance`
validation. Public picture-timing tests also cover FFmpeg's
`AV_FRAME_DATA_S12M_TIMECODE` projection from processed picture-timing SEI. The
two-frame side-data test additionally proves FFmpeg's one-shot handoff behavior
for unregistered SEI payloads, A53 captions, active-format descriptions,
VNOVA LCEVC payloads, picture-timing timecodes, and H.264 film grain with
`repetition_period == 0`.
Public picture-timing tests use a pic-struct-present SPS and synthetic leading
SEI to prove decoded `Frame` exposes FFmpeg-shaped `repeat_pict`, interlaced,
top-field-first metadata, and SMPTE 12M timecode words while preserving the
rawvideo MD5. A native opt-in C oracle compiles the pinned
`av_timecode_get_smpte` packing branch and compares the Go helper for 29.97,
50, and 60 fps cases. Recovery-point tests prove public key-frame flags for
IDR frames, non-IDR frames carrying `recovery_frame_cnt == 0`, and non-zero
recovery points, with an opt-in ffprobe frame-key oracle for the zero-count
promotion. Public rich-VUI tests synthesize a valid SPS and prove `StreamInfo`
exposes FFmpeg-normalized
SAR, video full-range signaling, color primaries/transfer/matrix, chroma
location, and timing fields.
Monochrome
native FFmpeg oracle checks
request `-pix_fmt gray` so the frame-MD5 surface compares only the luma plane
represented by `chroma_format_idc == 0`. The 16x16 High 4:4:4 Predictive
fixtures carry `disable_deblocking_filter_idc == 1`; the 32x32 High 4:4:4
Predictive fixtures keep deblocking enabled and prove the simple 8-bit
frame-picture loop filter over luma-shaped Cb/Cr planes and inter-macroblock
edges. The lossless fixtures carry `qpprime_y_zero_transform_bypass_flag` and
`8x8dct=0`, which keeps the oracle focused on qscale-0 scan selection,
add-pixels reconstruction, and lossless vertical/horizontal pred-add paths over
the simple progressive IDR/P subset.

The non-direct B fixtures use `testsrc2=size=16x16` and `testsrc2=size=32x32`
at `rate=1:duration=3` with
`bframes=1:b-adapt=0:b-pyramid=0:direct=none:no-skip=1:weightp=0:no-deblock=1`
and either `cabac=0` or `cabac=1`. They intentionally prove POC-backed B list0
and list1 construction, explicit L0/L1 motion, display-order output, Annex B,
AVC/NALFF, one-shot configured AVC paths, and qpel/chroma edge emulation when
the reference stride is smaller than FFmpeg's edge block width, while avoiding
B-direct/skip. The temporal-direct and spatial-direct B fixtures use the same
16x16 source with `partitions=none`, `subme=1`, skip enabled, and
`direct=temporal` or `direct=spatial`, proving the progressive frame-picture
`h264_direct.c` colocated-ref mapping, spatial neighbor-ref median path, and
B-skip write-back for both CAVLC and CABAC. The implicit-weight B fixtures use
the same 16x16 source and constraints with `duration=4:bframes=2`, which forces
non-symmetric B POC distances and proves FFmpeg's
`implicit_weight_table(field=-1)` path while still avoiding B-direct/skip.
The CAVLC and CABAC B 8x8 direct-sub fixtures are committed as 64x64 Annex B
bitstreams under `testdata/h264/`; they cover both spatial and temporal direct
prediction for sub-macroblocks across Annex B, AVC/NALFF, configured AVC,
sample-by-sample flush behavior, and native FFmpeg frame-MD5 oracle checks. The
paired B_SUB_4x4 fixtures are derived from the same streams by flipping only
the SPS `direct_8x8_inference_flag` bit to 0, which keeps the payload stable
while exercising FFmpeg's `h264_direct.c` direct-subdivision branch. The CAVLC
candidates also prove the FFmpeg `ff_h264_slice_context_init` internal
right-edge `ref_cache` sentinels needed by `fetch_diagonal_mv` for B 8x8
subpartition prediction.

Reference-picture unit coverage now includes FFmpeg's progressive frame-picture
long-term P-list behavior: default long refs after short refs, ref-list
modification op `2`, IDR/non-IDR long-term marking, short-to-long moves,
long-to-unused removal, max-long pruning, reset, mixed short/long sliding
window accounting, POC type 0 frame ordering, B-list sorting around current POC,
identical B-list swapping, B list1 reordering, FFmpeg `last_pocs` POC-gap
reorder-delay inference, and delayed display-output draining. A native long-ref
bitstream oracle is still pending.

SPS unit coverage includes source-shaped VUI/HRD bitstreams with Extended_SAR,
video signal/color description, chroma sample location, timing info, NAL HRD
state, pic-struct signaling, bitstream restriction, invalid HRD CPB counts,
invalid `num_reorder_frames`, and FFmpeg's derived reorder fallback when no
bitstream restriction is present.

SEI unit coverage includes FFmpeg-shaped SEI payload type/size accumulation,
extended type/size headers, strict truncated-payload rejection, non-fatal
buffering-period missing-SPS master errors, picture-timing HRD/pic-struct
timecode processing, buffering-period CPB delay extraction, registered ITU-T
T.35 ATSC AFD/A53 closed-caption parsing including multi-message A53 merge and
truncated-caption rejection, recovery point, green metadata, x264 unregistered
user data, display orientation, frame packing, alternative transfer, ambient
viewing environment with FFmpeg's invalid-value checks, H.274 film-grain
characteristics including the six-value component-model limit, mastering
display, and content light messages. Public side-data projection includes
SMPTE 12M picture-timing timecodes using the same VUI-derived frame-rate and
x264-build reset rules that feed FFmpeg's `h264_export_frame_props`. The simple
decoder now parses leading SEI NALs into decoder state while keeping SEI parser
failures non-fatal, matching FFmpeg's default behavior without `AV_EF_EXPLODE`,
consumes one-shot frame side data after export, and applies the simple
frame-picture portion of FFmpeg `h264_export_frame_props` for picture-timing
frame flags. The simple DPB now keeps FFmpeg-shaped IDR and SEI recovery marks
separate from the public output `KeyFrame` flag, including modulo
`recovery_frame` tracking and the `output_frame` rule that promotes
`recovery_frame_cnt == 0` frames to key frames.

## Decoder Boundary

Included:

- H.264 Annex B byte-stream parsing
- H.264 AVC/NALFF length-prefixed packet parsing when the caller supplies `nal_length_size`
- H.264 packet side-data handling for `AV_PKT_DATA_NEW_EXTRADATA`-style
  parameter-set updates plus A53 captions, AFD, display matrix, Stereo3D,
  spherical mapping, ICC profile, Dynamic HDR10+, LCEVC, SMPTE 12M timecode, mastering display,
  content light, ambient viewing environment, and 3D reference display
  packet-to-frame mapping
- H.264 NAL headers and RBSP handling
- SPS VUI public metadata for SAR, video range/format, colorimetry, chroma location, and timing
- Picture-timing-derived `repeat_pict`, interlaced, top-field-first, SMPTE 12M timecode, and key-frame public frame metadata for the simple frame-picture path
- Decoded frame SEI side data for the translated subset, including registered ITU-T T.35 ATSC AFD/A53 captions, registered VNOVA LCEVC bytes, stereo3D, display matrix, mastering-display validity, content light, ambient viewing environment, and H.274 film grain characteristics
- SPS/PPS, slice headers, entropy decode, macroblock decode, prediction, inverse transforms, loop filtering, reference picture management, and frame output as the port advances

Excluded unless directly required by decoder parity:

- H.264 encoder files
- Bitstream filters
- FFmpeg muxer/demuxer/filter frontends
- Hardware acceleration backends
- Non-H.264 codecs
