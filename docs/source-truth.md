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
intra8x8 prediction/IDCT/dequant call-site fixtures and a 10-bit 4:2:0
inter P16x16 motion-then-residual reconstruction fixture.

The H.264 chroma MC and qpel oracles compile the pinned
`libavcodec/h264chroma_template.c` and `libavcodec/h264qpel_template.c`,
comparing 8-bit fixtures plus 9/10/12/14-bit high-bit-depth put/avg variants
across supported widths and fractional-pel positions.

The H.264 motion-compensation call-site oracle compiles the pinned qpel,
chroma, weighting, and edge-emulation templates and compares 8-bit plus
10/12-bit high-bit-depth `hl_motion`/weighted fixtures over 4:2:0, 4:2:2, and
4:4:4 macroblock partitions.

The high-bit-depth frame-storage boundary follows pinned FFmpeg
`libavcodec/h264_slice.c` source: `h264_slice_header_init` accepts bit depths
8, 9, 10, 12, and 14, sets `bits_per_raw_sample`, and sets `pixel_shift` for
depths greater than 8; `get_pixel_format` selects the software YUV/GBR
`AVPixelFormat`; `alloc_picture` and `h264_frame_start` allocate the `AVFrame`
and derive byte offsets from `pixel_shift` and frame linesizes. Locally,
`internal/h264/simple_decode.go` keeps one `DecodedFrame` type with byte planes
for 8-bit frames and uint16 `Y16/Cb16/Cr16` planes for high frames. This is a
storage and narrow public-decode safe point: `internal/h264/simple_dpb.go` can
expose high ref-list views over those uint16 planes, and
`decodeSimpleNALUnitsWithState` dispatches high CAVLC/CABAC slices only for the
proved High 10 4:2:0 deblock-disabled I subset, the proved High 10 4:2:0
deblock-disabled P-skip/P16x16 no-residual subset, the proved High 10 4:2:0
frame-only deblock-disabled exact P16x16 L0 residual subset, explicit weighted
P over those P-skip/P16x16 L0 lanes, the High 10 4:2:0 deblock-enabled
32x32 IDR/P subset, the CAVLC-only High 10 4:2:0 slice-boundary
`disable_deblocking_filter_idc == 2` IDR/P subset, and the narrow High 10 4:2:2/4:4:4 deblock-enabled
32x32 IDR/P subset. The exact High 10 4:2:0
frame-only deblock-disabled non-direct B16x16 bidirectional lane, top-level
temporal/spatial B_Direct lane resolving to B16x16, temporal/spatial B-skip
lane, CAVLC/CABAC B 8x8/B_SUB_4x4 direct-sub lane with CBP zero, implicit
weighted B16x16 lane, explicit partitioned B16x8/B8x16/B8x8 lane,
partitioned implicit weighted B16x8/B8x16/B8x8 lane, the narrow
CAVLC/CABAC non-direct and top-level direct B16x16 deblock-enabled lanes, mixed-P
Intra4x4/Intra16x16 lane, CAVLC/CABAC partitioned P16x8/P8x16/P8x8 lane,
and High 4:4:4 Predictive-compatible yuv420p12le CAVLC IDR/I IntraPCM lane
are opened for the proved surfaces below. P IntraPCM, P 8x8-DCT intra,
weighted partitioned P, mixed direct/explicit B8x8, residual-bearing direct-sub
B, broader partitioned implicit weighted B outside the proved B16x8/B8x16/B8x8
shapes, partitioned/direct-sub/skip/implicit high B deblocking, CABAC/chroma/B-slice public high
slice-boundary mode, broader 12-bit and all 14-bit public high bitstreams, and
MBAFF remain outside the supported boundary.

The public high-depth raw output helper surface follows FFmpeg rawvideo byte
layout. `decoder.go` `RawPixelFormat`, `RawYUVSize`, `BytesPerSample`,
`AppendRawYUV16`, and `AppendRawYUVBytesLE` preserve sample values;
`AppendRawYUV` remains 8-bit-only.
`AppendRawYUVBytesLE` is shaped for FFmpeg `libavcodec/rawenc.c` `raw_encode`,
which sizes/copies frames through `av_image_get_buffer_size` and
`av_image_copy_to_buffer`, and for `libavutil/pixfmt.h`/`pixdesc.c` planar
little-endian high-depth pixel formats. Samples are written unshifted into
little-endian uint16 slots and rejected if they exceed the declared bit depth.

| Local chroma/depth | FFmpeg source selection | Rawvideo oracle `pix_fmt` |
| --- | --- | --- |
| `chroma_format_idc == 0`, 9/10/12/14-bit | Local luma-only oracle surface; FFmpeg H.264 software selection falls through to the non-4:2:2/non-4:4:4 branch in `get_pixel_format`. | `gray9le`, `gray10le`, `gray12le`, `gray14le` |
| `chroma_format_idc == 1`, 9/10/12/14-bit | `AV_PIX_FMT_YUV420P9/10/12/14` | `yuv420p9le`, `yuv420p10le`, `yuv420p12le`, `yuv420p14le` |
| `chroma_format_idc == 2`, 9/10/12/14-bit | `AV_PIX_FMT_YUV422P9/10/12/14` | `yuv422p9le`, `yuv422p10le`, `yuv422p12le`, `yuv422p14le` |
| `chroma_format_idc == 3`, YCbCr, 9/10/12/14-bit | `AV_PIX_FMT_YUV444P9/10/12/14` | `yuv444p9le`, `yuv444p10le`, `yuv444p12le`, `yuv444p14le` |
| `chroma_format_idc == 3`, RGB colorspace | `AV_PIX_FMT_GBRP9/10/12/14` | Unsupported by the current Y/Cb/Cr helper surface. |

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
- true High 10 4:2:0 deblock-disabled CAVLC/CABAC IDR/I rawvideo frame MD5s:
  `fd302f00e365b8502c44005ea308c468`,
  `38ed4870a1ba82aeb0c45b09d67e3e2a`
- true High 10 4:2:0 deblock-disabled CAVLC/CABAC IDR/P P-skip rawvideo frame
  MD5s: `87e217773d3e8b548fdf2002955cfcb9`,
  `87e217773d3e8b548fdf2002955cfcb9`
- true High 10 4:2:0 deblock-disabled CAVLC/CABAC 32x16 IDR/P P16x16
  no-residual rawvideo frame MD5s: `e0f04baf1c5940cf72857345ca05bbee`,
  `c356cd5790ea90f599ad5c2230869f06`
- true High 10 4:2:0 deblock-disabled CAVLC 16x16 IDR/P exact P16x16 L0
  residual rawvideo frame MD5s: `95893f95fdce0f45e7593f4eca8bd834`,
  `22ace8bfddbddf2958ef31f3d56ab09d`; concatenated rawvideo MD5
  `42e8d152117304a86b492cd0d529e90e`
- true High 10 4:2:0 deblock-disabled CABAC 16x16 IDR/P exact P16x16 L0
  residual rawvideo frame MD5s: `b47c39a842e4395e1ed527f2339c10ee`,
  `94edd171434db39321da0bc98328f421`; concatenated rawvideo MD5
  `f2c1ffc6f537acf9afcb10beecbedb1e`
- true High 10 4:2:0 deblock-disabled CAVLC/CABAC 32x16 IDR/P/P/P explicit
  weighted P16x16 rawvideo frame MD5s:
  `4b1f34db2851def469994d3f52eee679`,
  `914bd8170a17a4ff2800d632af8b4e0b`,
  `968ca595fffbfded0f4fbc1c0840cdde`,
  `36e2a95ad8461d4f280bab116f6087e6`; concatenated rawvideo MD5
  `c9f7de8ec190db53525801f41b473de9`
- true High 10 4:2:0 deblock-disabled CAVLC/CABAC 64x16 mixed IDR/P with
  P-slice Intra16x16 macroblocks rawvideo frame MD5s:
  `d8763101b7caf84ef313361b2c509966`,
  `bc57ab466b36d17a8f97bce6d2778fd2`,
  `120c1cc35907941cf8adacb9289389a3`,
  `270c2deffbbf00fbab9f02a4646f6a70`,
  `288f0ebfec67eb2cd02622d6e84e4b78`,
  `1dfb9457bc1c92737c21a9eeb0d3c5c0`,
  `8588fa7d5b458765c9c23943d089d955`,
  `1814743e97f04868d87b1060194606c7`,
  `d8105649af4d4c96dc72ec39dd84186d`,
  `ba305727a671c37d878ca128112dc50a`,
  `8ce69b5e4ceda53f964d9187dd08754f`,
  `bc6c9b3bdfb4007e95957c0d1de7bc04`; concatenated rawvideo MD5
  `79ab32c577ba4992c3c259bd3a0948ec`; stripped Annex B MD5s
  `2f7cf7da83f2bb10eda8092c9cc3bdc3` and
  `f98517c8acf532fb1005b88e4235bc08`
- true High 10 4:2:0 deblock-disabled CAVLC/CABAC 64x64 IDR/P/P/P/P
  partitioned P16x8/P8x16/P8x8 rawvideo frame MD5s:
  `1e10f859d4a3be85a0b4057dd7bff92c`,
  `a87c5d14c468e549ae461bd63d21e7d6`,
  `1e079388524aab8937783f56d36383c6`,
  `360bf39f49dbbd060fdbb52e68f1c5ce`,
  `d85d56ee1073b087635fcedb5d229025`; concatenated rawvideo MD5
  `447dd2695f723fc336ddb1a6c0b710cc` for CAVLC and
  `d37c9f22040bed0d61923dd6af57147a` for CABAC; stripped Annex B MD5s
  `1855c563913e2b4372d655417d333cdd` and
  `4300b297e11dc082735c4f784c46ed62`.
- true High 10 4:2:0 deblock-disabled CAVLC 16x16 exact non-direct B16x16
  bidirectional rawvideo frame MD5s:
  `95893f95fdce0f45e7593f4eca8bd834`,
  `9e8ad599e09f708487e0614412596665`,
  `b7edf8a2678e03b0495ba6a6efebc063`; concatenated rawvideo MD5
  `1ccf5f80b965f0e5788e592b2496e432`; stripped Annex B MD5
  `5a18eb8a8156a259ae2c3c915116fd7f`
- true High 10 4:2:0 deblock-disabled CABAC 16x16 exact non-direct B16x16
  bidirectional rawvideo frame MD5s:
  `b43174bc46328c029e698e5b27960dcd`,
  `8b7a30d943aeacb4c000a53bb1dbc212`,
  `6c997570b55af8ecd2ad29fbf56386a3`; concatenated rawvideo MD5
  `70c7595de7146ac9b0aec7a2cf2d116b`; stripped Annex B MD5
  `0067912e1f4bb582a1a6accf6930ab8d`
- true High 10 4:2:0 deblock-disabled CAVLC 32x16 temporal direct B16x16
  rawvideo frame MD5s:
  `dde20d70a08020b7171c068825ceab33`,
  `6e6d6501898f05aa0f8efd391a783b25`,
  `a4524920d19b25b23be978e8479039d0`; concatenated rawvideo MD5
  `865b30bbd64725fd8bb720c0576e19d0`; stripped Annex B MD5
  `1d30ac7b5a3aebfa9b360e43dd1747c1`
- true High 10 4:2:0 deblock-disabled CABAC 32x16 temporal direct B16x16
  rawvideo frame MD5s:
  `4737f86fe82079c689aec065ca6bb09f`,
  `dc494068394c583d86e4650b4635d8c4`,
  `4d9ce06c29c67bf8454164832e1ca92f`; concatenated rawvideo MD5
  `779cd7a6b9f8555bf0930465ded641e2`; stripped Annex B MD5
  `9ed2b7d4183f1fbdee66af5a3124eac3`
- true High 10 4:2:0 deblock-disabled CAVLC 32x16 spatial direct B16x16
  rawvideo frame MD5s:
  `dde20d70a08020b7171c068825ceab33`,
  `6e6d6501898f05aa0f8efd391a783b25`,
  `a4524920d19b25b23be978e8479039d0`; concatenated rawvideo MD5
  `865b30bbd64725fd8bb720c0576e19d0`; stripped Annex B MD5
  `d266bc4b06acc6835899d9e18fa6fa47`
- true High 10 4:2:0 deblock-disabled CABAC 32x16 spatial direct B16x16
  rawvideo frame MD5s:
  `4737f86fe82079c689aec065ca6bb09f`,
  `dc494068394c583d86e4650b4635d8c4`,
  `4d9ce06c29c67bf8454164832e1ca92f`; concatenated rawvideo MD5
  `779cd7a6b9f8555bf0930465ded641e2`; stripped Annex B MD5
  `8c12df946dc2a5620753b3e81c000c4c`
- true High 10 4:2:0 deblock-disabled 16x16 B-skip rawvideo frame MD5s
  for temporal/spatial CAVLC/CABAC:
  `d73be6c1b3e4082e402d67d810323786`,
  `d73be6c1b3e4082e402d67d810323786`,
  `d73be6c1b3e4082e402d67d810323786`; concatenated rawvideo MD5
  `bed8c5ab899fe974cae09585e60b151f`; stripped Annex B MD5s
  `a3d29c7a7a11a5c9da642487de5a4c37` (temporal CAVLC),
  `4ae312697d364153195deec6da9a1973` (spatial CAVLC),
  `74a9b632842600c57c0e20c03800c772` (temporal CABAC), and
  `961a79bdc2278420951d4662a1a2c2f3` (spatial CABAC)
- true High 10 4:2:0 deblock-disabled 16x16 CAVLC/CABAC
  B 8x8/B_SUB_4x4 direct-sub
  rawvideo frame MD5s for temporal/spatial:
  `d73be6c1b3e4082e402d67d810323786`,
  `d73be6c1b3e4082e402d67d810323786`,
  `d73be6c1b3e4082e402d67d810323786`; concatenated rawvideo MD5
  `bed8c5ab899fe974cae09585e60b151f`; Annex B MD5s
  `737b17dbc09f1d038fabccad1308afd4` (B8x8 temporal CAVLC),
  `87dc52d6a6ca8d0309c3bf064ab36eeb` (B8x8 spatial CAVLC),
  `c1abf23eeb9ccb84465e8b701886c9e8` (B_SUB_4x4 temporal CAVLC),
  `33bafb77f946ce2d9fe1168e8f9de609` (B_SUB_4x4 spatial CAVLC),
  `ac402e6f18e176ba51da9899b3285e66` (B8x8 temporal CABAC),
  `70b723a521824437321dca37b6b4f335` (B8x8 spatial CABAC),
  `56fbd77d91f0ce2e22d485e77c98a491` (B_SUB_4x4 temporal CABAC), and
  `3f567a5a22de5ae171658d71264a83f5` (B_SUB_4x4 spatial CABAC)
- true High 10 4:2:0 deblock-disabled 16x16 implicit weighted B rawvideo
  frame MD5s for CAVLC/CABAC:
  `857cc91515b2182f4444a4d746b9d721`,
  `734370de9ff1562a091bd9da2e7388f4`,
  `0278043f7918f89fb326a88e60c9c01b`,
  `1b742676a4555b46109892813b9feaa6`,
  `aed2dfa63ba343c3f2ef494bff5e3f74`; concatenated rawvideo MD5
  `fb94a7906e135740b49588c257f4bc15`; stripped Annex B MD5s
  `41bfa783c0361d76fbc8e0df36a6edca` (CAVLC) and
  `5865569a46cdb4f1692f1a1a589cd16b` (CABAC)
- true High 10 4:2:0 deblock-disabled 16x16 explicit partitioned B16x8
  rawvideo frame MD5s for CAVLC/CABAC:
  `da42dbbc6702ac820c7162dd19030ea3`,
  `6dc0b7afff881b7f69b9176db6c5155e`,
  `ae723753e3ae671a34e4f57f325d2cb8`; concatenated rawvideo MD5
  `8057ca8e0ee9e2f51fc59b824333e0da`; stripped Annex B MD5s
  `2798d9490dcb9f4b1495faee8e23c998` (CAVLC) and
  `74d9dd3315d2a1b45406508786722c25` (CABAC)
- true High 10 4:2:0 deblock-disabled 16x16 explicit partitioned B8x16
  rawvideo frame MD5s for CAVLC/CABAC:
  `3de0d9ec87d2b43d34b08554de5509e0`,
  `6dc0b7afff881b7f69b9176db6c5155e`,
  `360499a4bb17c8730018ce06b58180b7`; concatenated rawvideo MD5
  `d927d8a41788f89e93b8d66d54347ec7`; stripped Annex B MD5s
  `8f041ebd2075c5ee3195c6e4ea197d69` (CAVLC) and
  `0b7b7c3094532f5fff464f7a3819635a` (CABAC)
- true High 10 4:2:0 deblock-disabled 16x16 explicit partitioned B8x8
  rawvideo frame MD5s for CAVLC:
  `41ea931c1df0c87907ca7627beeb1dfc`,
  `ca7db1692b52de6fd7be03eae5d6b121`,
  `e355a7851b20224a769b798c9a63c8b3`; concatenated rawvideo MD5
  `017a85619aefcae9c7c98f11f6b829ee`; stripped Annex B MD5
  `9bd955daf127957bc6684c012a91df6a`
- true High 10 4:2:0 deblock-disabled 16x16 explicit partitioned B8x8
  rawvideo frame MD5s for CABAC:
  `541565314ead228ebda2b21fc3ee25d6`,
  `ccd7e4a2a29432b1db826acd229b78cd`,
  `730d70dba915767dc72964eb71a28ae4`; concatenated rawvideo MD5
  `63bbee01f26a0382dd58777ccb6c05e3`; stripped Annex B MD5
  `880484c1f22f9ac1846f5f9cd7652917`
- true High 10 4:2:0 deblock-disabled partitioned implicit weighted
  B16x8/B8x16/B8x8 rawvideo MD5s:
  `b85b69946077d6e700034f18e03afa02`/`5954cb46ad68184de947dbb604748924`
  for B16x8 CAVLC/CABAC,
  `0b5de5fe0388cb1f75b2a462f8b9252a`/`8d8aca4b4693bee11d56c99cf139007f`
  for B8x16 CAVLC/CABAC, and
  `d9feb695639d1c22e395c150e8f7f99f`/`2306e0d4cd6e403f86776208ccd87c3f`
  for B8x8 CAVLC/CABAC; stripped Annex B MD5s
  `f7a8b5d2e8e06a91f9e2b3a011fb2c9f`,
  `aa7076b8e6ffe06af2af84cdf381cb52`,
  `34cdb3fd5c7a9e3346acd2187d918c03`,
  `161bcc46653e699e834eff53c0e4df9d`,
  `cf2cc71caf7d42bfac77844b6e3c80cf`, and
  `558e36221572460fdd1d77b44aaa691a`
- true High 10 4:2:0 deblock-enabled 32x32 IDR/P rawvideo frame MD5s
  for CAVLC/CABAC:
  `ba8f5dc7f864b5cd854ee7d30e89fde1`,
  `108cc5e767fced5c958a56f4e65a2278`; concatenated rawvideo MD5
  `b635135b4e7db55894f75c390cf194c2`; stripped Annex B MD5s
  `9b221dc5e9937f2a4e9e95c06e00eb3f` (CAVLC) and
  `3d38082a580bf4945f6c7e6edfea81b3` (CABAC)
- true High 10 4:2:0 slice-boundary `disable_deblocking_filter_idc == 2`
  32x32 IDR/P rawvideo frame MD5s for CAVLC:
  `07f4ecbe2f86634c4de5b715ce1183c5`,
  `2395db9c9fd32c34e3705708c566177e`; concatenated rawvideo MD5
  `fc65b48f2855bd3a33b1f3cc1a6e9e16`; stripped Annex B MD5
  `c929a27027d7d3e77041ac3ed79e13a1`
- true High 10 4:2:0 CAVLC/CABAC non-direct B16x16 deblock-enabled rawvideo
  frame MD5s for CAVLC:
  `95893f95fdce0f45e7593f4eca8bd834`,
  `6be70b93adcb7bb8f78d667776b774dc`,
  `b7edf8a2678e03b0495ba6a6efebc063`; concatenated rawvideo MD5
  `35a2a24c460551f2c43e759dde953583`; stripped Annex B MD5
  `b8c45671afd9b919b7f391e09f9eced0`. CABAC frame MD5s:
  `b43174bc46328c029e698e5b27960dcd`,
  `1246d5f5c2fe36f2e658491be7309b5d`,
  `53eebcc181d70b4c0a0d0bf5dd4a5778`; concatenated rawvideo MD5
  `6200d3c83441e33c2cb1aac56d6882b3`; stripped Annex B MD5
  `0681332c3a5e40b6b6f2ad387e534432`
- true High 10 4:2:0 CAVLC/CABAC top-level direct B16x16 deblock-enabled
  rawvideo frame MD5s for temporal direct CAVLC:
  `86945e69a42629edd0fa46f7b8032c1d`,
  `46eebce937687169972bc95b770f2953`,
  `6185d7575b0476622e2317ad84de9ca8`; concatenated rawvideo MD5
  `663118a3e79cd6b41bb20a14867f7015`; stripped Annex B MD5
  `94c4f9b73c8a8b59f756320f20cf7def`. Temporal direct CABAC frame MD5s:
  `7e34fc5b9647628681a446de7c88c108`,
  `b24f513ee6c045f5c1add2a1e89e1af5`,
  `50ecc1b26b4ddd9582d37e1703e3a31e`; concatenated rawvideo MD5
  `411680af6618b27159866c456c28f6ff`; stripped Annex B MD5
  `59b29d60becffa83b095cd1eafc72757`. Spatial direct CAVLC shares
  rawvideo MD5 `663118a3e79cd6b41bb20a14867f7015` with stripped Annex B MD5
  `6d64382e77d76c28a17f31208d50a751`; spatial direct CABAC shares rawvideo
  MD5 `411680af6618b27159866c456c28f6ff` with stripped Annex B MD5
  `a5c947ab318d1ef5a4eac96fb19cbacf`
- true High 10 4:2:2/4:4:4 deblock-enabled 32x32 IDR/P rawvideo
  frame MD5s for CAVLC/CABAC:
  `754ac4c117c705808e87230f2d39a521`,
  `accfc50bf3e08afaf0e073d0849992dc`; concatenated rawvideo MD5
  `710f36ec1dd547e5b584144bb299ee7a`; stripped Annex B MD5
  `095b3897df89b12b6fba734931771d8b` (4:2:2 CAVLC);
  `77bd0e8f2c734a359d2238bbeffab77b`,
  `b5fd410a1bb665f5c10f8268fbfd2d53`; concatenated rawvideo MD5
  `1a011c767ac1131c7eb4b07c32f8a1ab`; stripped Annex B MD5
  `a697f204f63ac7d5d5eab7df23c16755` (4:2:2 CABAC);
  `b456b84535b2b0241a9ad973edaccd25`,
  `b0b7fc22ee4cb292a902d4949365c040`; concatenated rawvideo MD5
  `6cd1945a6daefd4ab1bc257f6be1d906`; stripped Annex B MD5
  `91ac19688e8e9fa26ad3941954b7948f` (4:4:4 CAVLC);
  `e0e3b6a956484218ee7c5979780ed9d6`,
  `b169bd10fc31bb91aa50a040b1358838`; concatenated rawvideo MD5
  `1f70a47728f816c0406fd7aed90bcbb2`; stripped Annex B MD5
  `f3ed8d65e4a600c331770ec9acb4d8f6` (4:4:4 CABAC)
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
- dimensions: `16x16`, `32x16`, and `32x32`
- frame payload size: `256` bytes (`gray`/`chroma_format_idc == 0`), `384`
  bytes (8-bit 16x16 `yuv420p`), `768` bytes (10-bit 16x16 `yuv420p10le`),
  `1536` bytes (8-bit 32x32 `yuv420p` or 10-bit 32x16 `yuv420p10le`),
  `512` bytes (`yuv422p`), and `768` or `3072` bytes (`yuv444p`)

The AVC/NALFF packet-input tests mechanically convert those Annex B fixtures to
big-endian length-prefixed NAL units while preserving each raw NAL payload. The
default Go tests compare the same rawvideo MD5s through explicit `nal_length_size`
values 2, 3, and 4. The configured AVC tests additionally build FFmpeg-style
`avcC` extradata from SPS/PPS NAL units, remove those parameter sets from the
packet payload, and prove the separated-config CAVLC ref-list, CABAC IDR/P,
High 4:2:0 32x32 8x8-DCT CAVLC/CABAC, High 4:2:2 CAVLC/CABAC,
High 4:4:4 Predictive CAVLC/CABAC, true High 10 4:2:0 deblock-disabled
CAVLC/CABAC IDR/I, P-skip/P16x16 no-residual, exact P16x16 L0 residual, and
explicit weighted P16x16, non-direct B16x16, temporal/spatial direct B16x16,
monochrome CAVLC/CABAC, and qp=0 lossless CAVLC/CABAC packets against the same frame MD5s
both as bundled packets and as successive single-frame sample packets that
require DPB reference state to survive across public decoder calls. Native
FFmpeg framemd5 oracle checks cover the 32x32 High 4:2:0 8x8-DCT fixtures in
addition to the true High 10 4:2:0 deblock-disabled CAVLC/CABAC IDR/I and
P-skip/P16x16 no-residual fixtures, the exact P16x16 L0 residual fixtures,
explicit weighted P16x16 fixtures, and the 16x16/32x32 families listed below. The
High 10 non-direct B16x16 CAVLC/CABAC fixtures are accepted packet and frame-MD5
proof for the exact B16x16 bidirectional subset only; the High 10
temporal/spatial direct B16x16 fixtures add top-level B_Direct proof for both
entropy modes, and the High 10 temporal/spatial B-skip fixtures add skip-run/
skip-flag direct-motion proof for both entropy modes. The High 10 CAVLC/CABAC
B 8x8/B_SUB_4x4 direct-sub fixtures open the direct-sub no-residual lane, the
High 10 implicit weighted B16x16 fixtures open `weighted_bipred_idc == 2` over
P16x16 anchors, the High 10 explicit partitioned B16x8/B8x16/B8x8 fixtures
open non-direct partitioned B, the High 10 partitioned implicit weighted
B16x8/B8x16/B8x8 fixtures open the same explicit partition shapes with
DPB-built non-neutral implicit bipred weights, the High10 mixed-P fixtures open
P Intra4x4/Intra16x16, and the CAVLC/CABAC partitioned-P fixtures open
P16x8/P8x16/P8x8 without opening P IntraPCM, P 8x8-DCT intra, or weighted
partitioned P. The narrow High 10 CAVLC/CABAC B16x16 deblock fixtures now open
only non-direct and top-level temporal/spatial direct B16x16 high loop filtering
with neutral weighting and keep partitioned, direct-sub, skip, and implicit
high B deblocking guarded.
Configured B-frame sample
tests additionally decode one access unit per call and
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

The High 10 non-direct B fixtures mirror the 16x16 no-skip non-direct B shape,
but request true High 10 4:2:0 output and compare `yuv420p10le` rawvideo bytes.
They prove only B16x16 standard bidirectional averaging over high refs, B-list
POC ordering, delayed display-order output, configured sample-by-sample decode,
and explicit flush. The bitstreams keep deblocking disabled, avoid B
skip/direct prediction, avoid 16x8/8x16/8x8 partitioned B, and keep
`weighted_bipred_idc == 0`.

The High 10 implicit weighted B fixtures use a 16x16 linear-luma
`nullsrc,geq` source with `bframes=2`, `b-pyramid=none`, `ref=1`,
`weightb=1`, `weightp=0`, `partitions=none`, `direct=none`,
`no-8x8dct=1`, and deblocking disabled. They keep the two P anchors inside the
proved P16x16 high subset and code the B macroblocks as bidirectional 16x16,
so the lane proves FFmpeg's progressive frame-picture
`implicit_weight_table(field=-1)` path over uint16 refs without admitting P
IntraPCM, P 8x8-DCT intra, partitioned P, B direct/skip, direct-sub,
partitioned B, or high deblocking.

The High 10 temporal/spatial direct B fixtures use a 32x16 `testsrc2` source
with a luma bump on the right macroblock of the B display frame. The forced
IBP encode keeps the reference P picture at P16 100%, skip 0%, intra 0%, and
keeps the B picture at direct 100%, skip 0%. They prove resolved top-level
B_Direct 16x16 motion over uint16 refs for both CAVLC and CABAC, including
Annex B, AVC/NALFF, configured AVC, sample-by-sample decode, delayed output,
public flush, and FFmpeg `yuv420p10le` rawvideo MD5 parity. The fixtures keep
deblocking disabled, `partitions=none`, direct mode fixed to temporal or
spatial, `weighted_bipred_idc == 0`, and avoid P intra, B-skip,
B 8x8/direct-sub, 16x8/8x16/8x8 partitioned B, and implicit weighted B.

The High 10 temporal/spatial B-skip fixtures use a static 16x16 source encoded
as forced IBP with `bframes=1:b-adapt=0:b-pyramid=0:ref=2`, `direct=temporal`
or `direct=spatial`, `weightp=0:weightb=0`, `partitions=none`, and deblocking
disabled. x264 reports P skip 100% and B skip 100%. They prove CAVLC
`mb_skip_run` and CABAC `mb_skip_flag` handoff through FFmpeg-shaped
temporal/spatial direct motion, high ref-list consumption, delayed output,
Annex B, AVC/NALFF, configured AVC, sample-by-sample decode, public flush, and
FFmpeg `yuv420p10le` rawvideo MD5 parity. The fixtures keep direct-sub,
explicit partitioned B, broader high deblocking, and broader chroma/depth
outside that fixture family. The public B-skip tests also
compare the embedded Annex B hex byte-for-byte with the file-backed corpus
fixtures so the manifest rows and public API oracle checks cannot drift apart.

The High 10 CAVLC/CABAC B 8x8/B_SUB_4x4 direct-sub fixtures are one-macroblock
static IBP streams derived from the matching High 10 B-skip shape by replacing the B slice
macroblock payload with `mb_skip_run=0`, B_8x8 type, four direct sub-MB types,
and CBP zero. The CABAC pair uses a synthesized CABAC body
`be27feed80` for `mb_skip_flag=0`, B_8x8, four B_Direct_8x8 sub-MBs, CBP
zero, and end-of-slice termination. The B_SUB_4x4 pair flips only the SPS
`direct_8x8_inference_flag` bit to 0. They prove the high CAVLC direct-sub
and CABAC direct-sub entropy-to-state paths, FFmpeg-shaped temporal/spatial direct sub-motion,
uint16 motion compensation, delayed output, Annex B, AVC/NALFF, configured AVC,
sample-by-sample decode, public flush, and FFmpeg `yuv420p10le` rawvideo MD5
parity.

The High 10 CAVLC/CABAC explicit partitioned B fixtures are small 16x16
frame-only IBP streams with deblocking disabled, neutral B weighting, no direct
mode in the B payload, and x264-selected B16x8, B8x16, or B8x8 explicit
partitions. They prove CAVLC and CABAC entropy-to-state, high ref-list
consumption, uint16 motion compensation, delayed output, Annex B, AVC/NALFF,
configured AVC, sample-by-sample decode, public flush, corpus manifest rows,
and FFmpeg `yuv420p10le` rawvideo MD5 parity for explicit non-direct
partitioned B without opening mixed direct/explicit B8x8, residual-bearing
direct-sub, implicit weighting, or partitioned/direct-sub/skip/implicit high B
deblocking.

The High 10 CAVLC/CABAC partitioned implicit weighted B fixtures combine the
explicit B16x8, B8x16, and B8x8 partition shapes with `weighted_bipred_idc == 2`,
one L0/L1 ref per B slice, temporal direct flag disabled, and deblocking
disabled. They prove DPB-fed implicit bipred weighting through uint16 motion
compensation for partitioned B while still excluding mixed direct/explicit
B8x8, residual-bearing direct-sub, and partitioned/direct-sub/skip/implicit
high B deblocking.

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
- Internal uint16 frame storage for high-bit-depth frames and public raw helper
  methods for value-preserving sample output when a high frame is available
- Public High 10 4:2:0 deblock-disabled CAVLC/CABAC IDR/I decode through the
  high raw helper surface, covered by Annex B, AVC/NALFF, configured AVC, and
  FFmpeg rawvideo MD5 oracle tests
- Public High 10 4:2:0 deblock-disabled CAVLC/CABAC IDR/P P-skip and P16x16
  no-residual decode through the high raw helper surface, covered by Annex B,
  AVC/NALFF, configured AVC, configured sample-by-sample decode, and FFmpeg
  rawvideo MD5 oracle tests
- Public High 10 4:2:0 deblock-disabled CAVLC/CABAC IDR/P/P/P explicit
  weighted P16x16 decode through the high raw helper surface, covered by Annex
  B, AVC/NALFF, configured AVC, configured sample-by-sample decode, and FFmpeg
  rawvideo/framemd5 oracle tests
- Public High 10 4:2:0 deblock-disabled CAVLC/CABAC exact non-direct B16x16
  bidirectional decode through the high raw helper surface, with Annex B, AVC,
  configured AVC, sample-by-sample flush, and FFmpeg rawvideo oracle proof.
- Public High 10 4:2:0 deblock-disabled CAVLC/CABAC temporal/spatial direct
  B16x16 decode through the high raw helper surface, with Annex B, AVC,
  configured AVC, sample-by-sample flush, and FFmpeg rawvideo oracle proof.
- Public High 10 4:2:0 deblock-disabled CAVLC/CABAC temporal/spatial B-skip
  decode through the high raw helper surface, with Annex B, AVC, configured
  AVC, sample-by-sample flush, and FFmpeg rawvideo oracle proof.
- Public High 10 4:2:0 deblock-disabled CAVLC/CABAC B 8x8/B_SUB_4x4 direct-sub
  decode through the high raw helper surface, with Annex B, AVC, configured
  AVC, sample-by-sample flush, and FFmpeg rawvideo oracle proof.
- Public High 10 4:2:0 deblock-disabled CAVLC/CABAC partitioned P16x8/P8x16/P8x8
  decode through the high raw helper surface, with Annex B, AVC, configured
  AVC, sample-by-sample flush, auto nil-flush, corpus, and FFmpeg rawvideo
  oracle proof.
- Public High 10 4:2:0 deblock-enabled CAVLC/CABAC 32x32 IDR/P decode
  through the high raw helper surface, covered by Annex B, AVC/NALFF,
  configured AVC, and FFmpeg rawvideo oracle proof.
- Public High 10 4:2:0 slice-boundary `disable_deblocking_filter_idc == 2`
  CAVLC-only 32x32 IDR/P decode through the high raw helper surface, covered by
  Annex B, AVC/NALFF, configured AVC, access-unit sample flush, corpus manifest,
  and FFmpeg rawvideo oracle proof.
- Public High 10 4:2:0 deblock-enabled CAVLC/CABAC B16x16 decode through the high
  raw helper surface, covered by Annex B, AVC/NALFF, configured AVC,
  sample-by-sample flush, corpus manifest, and FFmpeg rawvideo oracle proof.
- Public High 10 4:2:2/4:4:4 deblock-enabled CAVLC/CABAC 32x32 IDR/P decode
  through the high raw helper surface, covered by Annex B, AVC/NALFF,
  configured AVC, configured sample-by-sample decode, corpus manifest rows, and
  FFmpeg rawvideo/framemd5 oracle proof.
- Manifest-driven H.264 corpus runner `decoder_corpus_test.go`, with default
  JSONL entries in `testdata/h264/corpus/manifest.jsonl` and external corpus
  override through `GOH264_CORPUS_MANIFEST` or a path-list through
  `GOH264_CORPUS_MANIFESTS`. Decode-ok rows require bitstream, per-frame raw, and
  concatenated rawvideo MD5s; unsupported rows must name guard tags and assert
  `ErrUnsupported`. The committed manifest now file-backs the
  local 8-bit B direct-sub vectors plus the proved High 10 4:2:0 IDR/P,
  residual P16x16, explicit weighted P16x16, CAVLC/CABAC partitioned P16x8/P8x16/P8x8,
  non-direct B16x16,
  temporal/spatial direct B16x16, temporal/spatial B-skip, CAVLC/CABAC
  B 8x8/B_SUB_4x4 direct-sub, implicit weighted B16x16, partitioned implicit
  weighted B16x8/B8x16/B8x8, the narrow CAVLC/CABAC non-direct/direct B16x16 deblock rows, and
  deblock-enabled 32x32 IDR/P vectors including the
  narrow High 10 4:2:2/4:4:4 rows, plus the CAVLC-only High10 slice-boundary
  row and the High 4:4:4 Predictive-compatible yuv420p12le CAVLC IDR/I
  IntraPCM row.
- Decoder benchmark harness `cmd/goh264bench`, including Go decode/raw-output
  timing, raw MD5 reporting, allocation counters, repeated samples/statistics,
  machine-readable input/host/VCS/FFmpeg metadata, Go raw pixel-format reporting,
  FFmpeg `-pix_fmt` auto-selection for raw-MD5 parity, an optional FFmpeg CLI
  rawvideo baseline over the same input with explicit timed-scope caveats, and a
  manifest benchmark mode that runs decode-ok corpus rows only after bitstream
  MD5, raw shape, and rawvideo MD5 oracle checks pass.
- Production readiness corpus/benchmark plan in `docs/production-readiness.md`,
  naming external manifest tiers, command-profile recipes, benchstat-friendly
  output shape, allocation-gate fields, and FFmpeg CLI comparator caveats without
  widening the decoder's supported surface.
- SPS/PPS, slice headers, entropy decode, macroblock decode, prediction, inverse transforms, loop filtering, reference picture management, and frame output as the port advances

Excluded unless directly required by decoder parity:

- H.264 encoder files
- Bitstream filters
- FFmpeg muxer/demuxer/filter frontends
- Hardware acceleration backends
- Non-H.264 codecs
- Public high-bit-depth behavior outside the proved High 10 deblock-disabled I,
  P-skip/P16x16 no-residual, exact P16x16 L0 residual, explicit weighted P,
  exact non-direct plus temporal/spatial direct B16x16, temporal/spatial
  B-skip, CAVLC/CABAC B 8x8/B_SUB_4x4 direct-sub, explicit partitioned
  B16x8/B8x16/B8x8, implicit weighted B16x16, partitioned implicit weighted
  B16x8/B8x16/B8x8, mixed-P Intra4x4/Intra16x16, CAVLC/CABAC partitioned
  P16x8/P8x16/P8x8, CAVLC/CABAC non-direct/direct B16x16 high deblocking,
  deblock-enabled 4:2:0 32x32 IDR/P, CAVLC-only High10 4:2:0
  slice-boundary deblocking IDR/P, High 4:4:4 Predictive-compatible
  yuv420p12le CAVLC IDR/I IntraPCM, and deblock-enabled 4:2:2/4:4:4 32x32 IDR/P subsets
  remains explicitly unsupported. In particular, P IntraPCM,
  P 8x8-DCT intra, weighted partitioned P, mixed direct/explicit B8x8,
  residual-bearing direct-sub B, broader partitioned implicit weighted B beyond
  the proved B16x8/B8x16/B8x8 shapes, partitioned/direct-sub/skip/implicit high B deblocking,
  CABAC/chroma/B-slice public high slice-boundary mode, broader 12-bit and all
  14-bit public high bitstreams, and MBAFF remain later lanes.
- Full conformance/testvector corpus passing and production benchmark claims
  remain pending until curated external corpora are added and manifest benchmark
  reports cover stable larger clips with profile presets, benchstat-friendly
  output, allocation gates, and a fair in-process/native baseline.
