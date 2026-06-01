# High-Bit-Depth Decoder Roadmap

This is a working roadmap for the remaining high-bit-depth H.264 decoder work in
this repository. It is intentionally decoder-only and source-shaped: FFmpeg
`n8.0.1` remains the source truth, with the current ledger and implementation as
the local state of record.

High-bit-depth here means decoded sample depth greater than 8 bits, using the
already-supported FFmpeg bit-depth family of 9, 10, 12, and 14 bits. Do not
confuse this with the existing public "High profile" fixtures, which mostly
exercise 8-bit High/High 4:2:2/High 4:4:4 syntax and reconstruction.

## Current Source Truth

- Upstream source truth is pinned in `docs/source-truth.md`: FFmpeg
  `libavcodec` tag `n8.0.1`, peeled commit
  `894da5ca7d742e4429ffb2af534fcda0103ef593`.
- SPS/PPS parsing already accepts high-profile chroma formats and equal luma/
  chroma bit depths through 14 bits. PPS explicitly rejects 11-bit and 13-bit
  decode because local DSP support is only present for 9, 10, 12, and 14.
- QP/dequant tables are sized to `qpMaxNum == 87`, so 14-bit QP storage exists.
  Slice parsing computes `maxQP` from `BitDepthLuma`; the public simple decode
  path now dispatches high-bit-depth CAVLC/CABAC slices for the proved High 10
  4:2:0 deblock-disabled I subset, P-skip/P16x16 no-residual subset, exact
  P16x16 L0 residual subset, and explicit weighted P16x16 subset.
- `internal/h264/simple_decode.go` now represents decoded frames with either
  byte planes (`DecodedFrame.Y/Cb/Cr`) or uint16 planes
  (`DecodedFrame.Y16/Cb16/Cr16`). `newSimpleDecodedFrame` allocates high planes
  for 9/10/12/14-bit SPS values and validates them through
  `picturePlanesHigh()`. `decodeSimpleNALUnitsWithState` routes high pictures
  through a separate uint16 slice loop when validation proves a High 10 4:2:0
  I slice, no-residual P-slice, exact P16x16 L0 residual P-slice, or explicit
  weighted P16x16 P-slice with deblocking disabled. The B route is proved only
  for exact non-direct B16x16 bidirectional pictures with neutral B weighting.
- `decoder.go` exposes public `Frame.Y16/Cb16/Cr16`, `BytesPerSample`,
  `RawPixelFormat`, `RawYUVSize`, `AppendRawYUV16`, and
  `AppendRawYUVBytesLE` alongside the existing 8-bit `Frame.Y/Cb/Cr` and
  `AppendRawYUV`. These helpers carry the proved public High 10
  deblock-disabled I/P output fixtures without downconverting samples, and are
  the oracle surface for the proved non-direct B16x16 lane.
- Entropy/state layers are farther along than output: CAVLC and CABAC frame-MB
  paths already size and hand off high-bit-depth IntraPCM payloads, carry high
  QP values, and persist residual/motion/direct state in bit-depth-neutral table
  shapes.
- Kernel-level high-bit-depth DSP is present and oracle-covered: IDCT/dequant,
  add-pixels, intra prediction, weighted/biweighted prediction, qpel, chroma MC,
  and luma/chroma deblocking have 9/10/12/14-bit reference implementations.
- The current B-picture safe point is deliberately narrow: exact High 10 4:2:0
  frame-only, deblock-disabled, non-direct B16x16 bidirectional motion with
  explicit L0/L1 references and neutral B weighting. It is the only high B lane
  opened in this step; direct, implicit weighted, and partitioned B remain
  separate oracle lanes.
- `internal/h264/reconstruct_high.go` has a separate `h264PicturePlanesHigh`
  surface and internal high-bit-depth IntraPCM/intra/inter reconstruction
  helpers for 4:2:0, 4:2:2, and 4:4:4. The public simple slice loop now calls
  the high path for deblock-disabled I pictures, High 10 P-skip/P16x16
  no-residual pictures, exact P16x16 L0 residual pictures, and explicit
  weighted P16x16 pictures. The High 10 non-direct B16x16 lane is now proved;
  partitioned P, direct B, implicit weighted B, partitioned B, and high
  deblocking/border-exchange modes remain at the unsupported boundary.
- `internal/h264/motion_comp_high.go` now mirrors the 8-bit `hl_motion`
  call-site layer over uint16 planes. It covers standard and weighted
  macroblock partitions, 4:2:0/4:2:2 chroma MC, 4:4:4 qpel-shaped Cb/Cr, and
  uint16 edge-emulation scratch in sample units.
- `internal/h264/reconstruct_high.go` now consumes high motion internally for
  inter macroblocks. The public simple slice path now admits the High 10
  P-skip/P16x16 no-residual lane plus exact High 10 4:2:0 frame-only,
  deblock-disabled P16x16 L0 residual slices, explicit weighted P16x16
  slices, and exact non-direct B16x16 bidirectional slices for CAVLC and CABAC.
  The High 10 B lane only uses standard bidirectional averaging with explicit
  L0/L1 references and neutral B weighting.
  Partitioned P, P intra macroblocks, direct B, implicit weighted B,
  partitioned B, high deblocking, other chroma/depth combinations, and MBAFF
  remain guarded.
- `internal/h264/loop_filter.go` ports the generic frame-picture loop-filter
  strength and call-site wiring for 8-bit planes. High-bit-depth DSP kernels
  exist in `dsp.go`, but the frame filter still validates `BitDepthLuma == 8`
  and uses 52-entry local threshold tables instead of FFmpeg's `52*3` high-depth
  index model with `qp_bd_offset`.
- The simple DPB stores `*DecodedFrame` references that can now carry high
  planes, and `simpleFrameRefContext` exposes either byte-plane refs or
  `[]*h264PicturePlanesHigh` refs from the same short/long ordering. The proved
  high P and exact non-direct B16x16 lanes consume those refs for explicit list
  entries without opening direct or implicit-weighted B.

## Non-Goals For This Roadmap

- Encoder support, muxing, filtering outside the H.264 decoder dependency cone,
  and non-H.264 video DSP surfaces remain out of scope.
- Field pictures, MBAFF, row threading, reference-progress waits, FMO, full
  error resilience, and full libavcodec delayed-output semantics remain later
  lanes unless a high-bit-depth safe point explicitly needs a narrow subset.
- SIMD and architecture dispatch should wait until scalar high-bit-depth byte
  parity is proved.
- Do not broaden public support by silently downconverting high-bit-depth samples
  to 8-bit output. Any public surface must preserve sample values.

## Design Ground Rules

- Keep the 8-bit public and internal paths behaviorally unchanged while adding
  high-bit-depth support. The current byte-backed path is well-covered by
  frame-MD5 fixtures and should remain the regression baseline.
- Prefer source-shaped high variants over clever generic abstraction until the
  high path is proven. The upstream split is effectively `pixel_shift == 0` vs
  `pixel_shift != 0`; local code can reflect that with byte and uint16 paths.
- Keep strides measured in samples inside Go plane structs. When mirroring
  FFmpeg byte-pointer math, translate `pixel_shift` offsets carefully at API
  boundaries instead of mixing byte and sample strides.
- Allow public high-bit-depth decode output only after internal high frame
  storage, a high slice path, and at least one high-bit-depth bitstream fixture
  prove decode correctness.
- Every implementation step should land as a safe point with focused tests plus
  at least one oracle/framemd5 fixture once bitstream decode is involved.

## Gap Ledger

| Area | Current state | Remaining high-bit-depth work |
| --- | --- | --- |
| SPS/PPS/slice metadata | High bit depths parse; PPS/dequant tables cover 9/10/12/14; slice QP uses bit-depth max. | Preserve this behavior while removing simple-path high-bit-depth rejects only when the matching high decode path exists. |
| Entropy-to-state | CAVLC/CABAC frame-MB handoff, residuals, motion caches, direct motion, and high IntraPCM payload sizing exist; high CAVLC/CABAC slice loops now carry deblock-disabled I pictures plus proved High 10 P-skip/P16x16 no-residual, exact P16x16 L0 residual, explicit weighted P16x16, and exact non-direct B16x16 bidirectional subsets through reconstruction. | Add high-specific regression cases where QP exceeds 51 only with matching public proof, then continue to partitioned P and B variants with separate oracles. Direct B, implicit weighted B, partitioned B, and high loop filtering stay guarded. |
| Internal frame storage | `DecodedFrame` now has uint16 high planes, `newSimpleDecodedFrame` allocates them for 9/10/12/14-bit SPS values, `picturePlanesHigh()` validates them, the simple DPB can expose `RefsHigh`, and public `Frame` can carry `Y16/Cb16/Cr16`. | Keep direct B, implicit weighted B, partitioned B, high deblocking, GBR, and unproved depth/chroma combinations guarded until matching bitstream oracles land. |
| Intra reconstruction | Internal high IntraPCM/intra16x16/intra4x4/intra8x8 call sites exist and are oracle-covered; the simple high slice path now decodes deblock-disabled High 10 4:2:0 CAVLC/CABAC IDR/I fixtures through public output. | Broaden intra coverage to 12/14-bit, 4:2:2/4:4:4, IntraPCM/lossless variants, and then mixed I/P/B streams after broader high motion is public. |
| Inter/motion reconstruction | 8-bit `hl_motion` is integrated for P/B, weighted P, implicit B, direct B, and 4:4:4 planes. High `h264HLMotionFrame*` is now ported for internal MB-level 4:2:0/4:2:2/4:4:4 motion, explicit/implicit weighting, and edge emulation; High 10 P-skip/P16x16 no-residual, exact P16x16 L0 residual, explicit weighted P16x16, and exact non-direct B16x16 standard bidirectional avg are now wired through public slice/frame output. | Partitioned P, P intra, direct B, implicit weighted B, partitioned B, high deblocking, other chroma/depth, and MBAFF stay guarded until each gets bitstream/oracle proof. |
| Loop filter integration | 8-bit frame-picture strength/call-site integration works post-frame for the simple path; high deblock kernels exist. | Add high frame-picture filter wiring over uint16 planes, source-shaped high threshold indexing, chroma 4:2:0/4:2:2 and 4:4:4 edge dispatch, and high bitstream fixtures with deblocking enabled. |
| Public output | Public `Frame` exposes `Y16/Cb16/Cr16`, `RawPixelFormat`, `RawYUVSize`, `BytesPerSample`, `AppendRawYUV16`, and `AppendRawYUVBytesLE`; `AppendRawYUV` remains 8-bit-only; High 10 deblock-disabled I output, no-residual P-skip/P16x16 output, exact P16x16 L0 residual output, explicit weighted P16x16 output, and exact non-direct B16x16 output are proved against FFmpeg rawvideo MD5s. | Keep partitioned P, direct B, implicit weighted B, partitioned B, high deblocking, GBR, MBAFF, and unproved chroma/depth combinations guarded. |
| Oracle fixtures | Kernel oracles cover high primitives; public frame-MD5 fixtures cover 8-bit High-profile streams, true High 10 CAVLC/CABAC deblock-disabled IDR/I fixtures, true High 10 IDR/P P-skip/P16x16 no-residual fixtures, true High 10 exact P16x16 L0 residual fixtures, true High 10 explicit weighted P16x16 fixtures, and true High 10 non-direct B16x16 CAVLC/CABAC fixtures across Annex B/AVC/configured surfaces. | Build later oracle targets for direct B, implicit weighted B, partitioned B, and high loop filtering without widening this guard. |

## Internal Frame And Plane Work

The original uint16 frame-storage safe point made high-bit-depth frames
representable; the current safe point exposes the proved High 10 4:2:0
deblock-disabled I, P-skip/P16x16 no-residual, exact P16x16 L0 residual, and
explicit weighted P subsets, plus exact non-direct B16x16 bidirectional output.
It mirrors FFmpeg's separation between selected pixel format, `pixel_shift`, and
`AVFrame` buffer ownership in `libavcodec/h264_slice.c` `get_pixel_format`,
`h264_slice_header_init`, `alloc_picture`, and `h264_frame_start`.

Completed storage rules:

- Keep existing `Y`, `Cb`, `Cr []uint8` for 8-bit frames.
- Use `Y16`, `Cb16`, `Cr16 []uint16` for 9/10/12/14-bit frames.
- Keep `LumaStride` and `ChromaStride` as sample strides for both 8-bit and
  high-bit-depth paths. FFmpeg byte-offset math that uses `pixel_shift` is
  translated only at high helper and DSP boundaries.
- Keep exactly one backing representation populated by construction: byte
  planes for 8-bit frames, uint16 planes for high frames.
- Use `picturePlanesHigh() h264PicturePlanesHigh` next to `picturePlanes()` so
  high prediction, motion, reconstruction, and future high deblocking consume
  the same sample-unit layout.
- Preserve frame metadata, side data, POC fields, direct-motion provenance, and
  macroblock tables on the same `DecodedFrame` type; those fields are not
  sample-depth dependent.

Completed ref-facing storage rules:

- `simpleFrameRefContext` carries `[2][]*h264PicturePlanes` and
  `[2][]*h264PicturePlanesHigh`.
- `buildRefListsHigh` and the shared ref context build high refs from
  `entry.frame.picturePlanesHigh()` only when the current frame is
  high-bit-depth.
- High ref tests prove P short/long ordering, B POC ordering, preserved byte refs
  for 8-bit frames, and uint16 backing-plane identity for high refs.
- Keep `simpleRefEntry` frame identity unchanged, since POC, long/short marking,
  delayed output, and direct-motion provenance are shared between byte and uint16
  frames.

The high-bit-depth public decode guard is now narrowed rather than blanket:
deblock-disabled High 10 4:2:0 I slices, P-skip/P16x16 no-residual slices,
exact P16x16 L0 residual slices, and explicit weighted P16x16 slices may reach
public output. The only B guard opened is the exact High 10 4:2:0
frame-only, deblock-disabled, non-direct B16x16 bidirectional lane, now proved
by CAVLC/CABAC rawvideo oracles. Partitioned P, P intra macroblocks, direct B,
implicit weighted B, partitioned B, high
deblocking, and unproved depth/chroma combinations remain guarded.
Storage tests should continue to assert high plane allocation, plane sizes,
strides, chroma sizing for `chroma_format_idc` 0/1/2/3, crop geometry, public
helper error behavior, and no change in 8-bit frame MD5s.

## High Intra Slice Integration

After high frames can be allocated, the first narrow decode path for high
deblock-disabled intra frames is now wired:

- Split the simple slice decode dispatch by sample depth after headers are
  parsed and frame storage is allocated.
- Add `h264FrameSliceDecodeInputHigh` carrying `SliceNum`, high refs, direct
  context, prediction weights, and high motion scratch so the next inter safe
  points can extend the same call surface.
- Add high CAVLC/CABAC slice loops that call
  `h264HLDecodeFrameMacroblockHigh`.
- Extend `h264FrameMBReconstructInputHigh` from the current isolated helper shape
  only as needed for parity; for the intra safe point it carries the existing
  intra fields, residual, PPS, transform-bypass flag, bit depth, refs/weights,
  high motion scratch, and IntraPCM payload.
- Keep `DeblockingFilter` false at the reconstruction helper boundary. The
  current simple 8-bit path filters post-frame rather than doing row-time border
  exchange; high should follow the same simple-path sequencing until the
  row-threaded lane exists.

Focused fixtures for this step:

- 10-bit 4:2:0 CAVLC IDR/I, deblocking disabled. Done with public rawvideo MD5
  parity.
- 10-bit 4:2:0 CABAC IDR/I, deblocking disabled. Done with public rawvideo MD5
  parity.
- 12-bit 4:2:2 intra fixture if the local FFmpeg/x264 stack can generate it.
- 14-bit 4:4:4 intra fixture, preferably small and deblock-disabled, to prove
  the luma-shaped plane path.
- IntraPCM and qscale-0/lossless variants should stay separate unless they are
  naturally tiny; otherwise they become their own safe point.

## Internal Inter And Motion Reconstruction

The first high inter safe point is now in place at MB level: high motion
compensation is source-shaped in `motion_comp_high.go`, and
`h264HLDecodeFrameMacroblockHigh` can run P16x16 motion before high residual
add. The local byte path in `motion_comp.go` remains the template; avoid
generalizing the byte and uint16 paths until public parity is boring.

Completed MB-level pieces:

- Add `h264MotionCompScratchHigh` with `Y`, `Cb`, `Cr`, and `Edge` as
  `[]uint16`.
- Add `h264EdgeScratchHigh`, `h264EdgeScratchSizeHigh` in tests, and
  `h264EmulatedEdgeMCHigh` that pads samples rather than bytes.
- Port `h264HLMotionFrameCore` to high planes, preserving the same 16x16,
  16x8, 8x16, and 8x8 subpartition sequencing.
- Port `h264MCPartFrameStd` and `h264MCPartFrameWeighted` to high planes.
- Dispatch luma through `h264QpelMCStridesHigh` with `bitDepth`.
- Dispatch 4:2:0/4:2:2 chroma through `h264ChromaMCStridesHigh`.
- Dispatch 4:4:4 Cb/Cr through high qpel, matching the byte path's
  qpel-shaped 4:4:4 handling.
- Dispatch explicit and implicit weights through `h264WeightPixelsHigh` and
  `h264BiweightPixelsHigh`.
- Extend `h264FrameMBReconstructInputHigh` with `ListCount`, `Motion`, high
  `Refs`, `PredWeight`, and high `MotionScratch`.
- Removed the `!isIntra` rejection in `h264HLDecodeFrameMacroblockHigh` after
  high motion dispatch was covered by tests.

Remaining slice/frame pieces:

- Keep high ref-list construction wired through the simple slice path for the
  proved High 10 P-skip/P16x16 no-residual, exact P16x16 residual, explicit
  weighted P16x16, and exact non-direct B16x16 bidirectional lanes.
- High CAVLC/CABAC frame slices now extend into residual P and explicit weighted
  P only for High 10 4:2:0 frame-only, deblock-disabled exact P16x16 L0
  macroblocks. This proves residual IDCT/add and luma/chroma weighted
  prediction after uint16 motion without admitting partitioned P.
- The B lane admits only non-direct B16x16 macroblocks that use
  explicit L0 and L1 references with `weighted_bipred_idc == 0`. It proves
  standard bidirectional avg over uint16 planes, B-list DPB ordering, delayed
  display-order output, and flush behavior without admitting B skip/direct
  prediction or B partitions.
- Keep the narrowed public high-bit-depth guards for residual P outside this
  exact lane, partitioned P, direct B, implicit weighted B, partitioned B, high
  deblock, and unproved chroma/depth modes until each path passes a
  framemd5/rawvideo oracle.

Suggested safe-point order:

1. High P-skip/P16x16 with no residual and deblocking disabled. Done for High
   10 4:2:0 CAVLC/CABAC with public Annex B, AVC, configured AVC, and FFmpeg
   frame-MD5 proof.
2. High 10 4:2:0 exact P16x16 L0 residual, frame-only, and deblock-disabled.
   Done for CAVLC/CABAC with public Annex B, AVC, configured AVC,
   sample-by-sample configured decode, and FFmpeg rawvideo/framemd5 proof.
3. High explicit weighted P. Done for High 10 4:2:0 frame-only,
   deblock-disabled P16x16 CAVLC/CABAC with luma/chroma weights, public Annex B,
   AVC, configured AVC, sample-by-sample configured decode, and FFmpeg
   rawvideo/framemd5 proof.
4. High non-direct B with explicit L0/L1 refs. Done for exact High 10 4:2:0
   frame-only, deblock-disabled B16x16 CAVLC/CABAC with standard bidirectional
   avg, display-order output, configured sample decode, and FFmpeg rawvideo
   proof.
5. High temporal/spatial direct B and B 8x8/B_SUB_4x4 direct-sub. This should
   reuse the existing direct-motion tables and focus on high motion consumption.
6. High implicit weighted B. This proves DPB-fed implicit weights over uint16
   planes.

The High non-direct B safe point is intentionally not a general B unlock.
Acceptance criteria:

- The fixture is true High 10 4:2:0 and exports `yuv420p10le` rawvideo bytes.
- The stream is progressive frame-only, deblock-disabled, and small enough for
  first-divergence debugging, preferably 16x16.
- Macroblocks stay B16x16 non-direct with explicit L0/L1 refs. No B skip,
  top-level B_Direct, B 8x8/direct-sub, 16x8, 8x16, or 8x8 partitioned B.
- PPS weighted bipred remains neutral (`weighted_bipred_idc == 0`); implicit
  weighted B is a later safe point.
- Public checks cover rawvideo MD5, configured sample-by-sample decode, delayed
  output, and explicit flush before this lane is marked done.

Each inter step should include at least one edge-emulation case where the
reference stride is smaller than the FFmpeg edge block width, because that was a
real parity trap in the 8-bit path.

## High Loop Filter Integration

High loop filtering should be a separate safe point after high motion and public
high output have a reliable raw compare surface. The current 8-bit post-frame
filter sequencing is acceptable for the simple non-threaded frame-picture path;
do not block on FFmpeg row-time border exchange unless a fixture proves the
post-frame model diverges.

Required pieces:

- Add high threshold tables or table generation matching FFmpeg's
  `alpha_table[52*3]`, `beta_table[52*3]`, and `tc0_table[52*3][4]`.
- Compute `qp_bd_offset = 6 * (bitDepthLuma - 8)` and pass
  `a = 52 + slice_alpha_c0_offset - qp_bd_offset`,
  `b = 52 + slice_beta_offset - qp_bd_offset` into high edge wrappers, matching
  FFmpeg's `qp + a` and `qp + b` indexing.
- Keep the current strength calculation unless source truth shows high-depth
  differences; bS rules are sample-depth independent in the current simple
  frame-picture subset.
- Add `h264ApplyLoopFilterEdgeHigh` over `h264PicturePlanesHigh`.
- Dispatch high luma edges through `h264*LoopFilterLuma*High`.
- Dispatch high 4:2:0/4:2:2 chroma edges through
  `h264*LoopFilterChroma*High` and `h264*LoopFilterChroma422*High`.
- Dispatch high 4:4:4 Cb/Cr through luma high filter kernels with chroma QPs,
  matching the existing 8-bit 4:4:4 branch.
- Validate both `disable_deblocking_filter_idc == 1` no-op behavior and
  `disable_deblocking_filter_idc == 0/2` edge behavior.

High loop-filter fixture order:

1. 10-bit 4:2:0 IDR/P with deblocking enabled.
2. 10-bit 4:2:0 same encode with deblocking disabled, to isolate filter impact.
3. 12-bit 4:2:2 deblocking enabled.
4. 14-bit 4:4:4 deblocking enabled, proving luma-shaped chroma plane filtering.
5. Slice-boundary mode `disable_deblocking_filter_idc == 2` once multi-slice
   high fixtures exist.

## Public High-Bit-Depth Output

Public output helpers should preserve sample values and keep the existing 8-bit
API stable:

- Keep `Frame.Y`, `Frame.Cb`, `Frame.Cr []byte` populated only for 8-bit frames.
- Keep `Frame.Y16`, `Frame.Cb16`, `Frame.Cr16 []uint16` populated only for
  high-bit-depth frames.
- Keep `Frame.YStride` and `Frame.CStride` in samples, not bytes.
- Keep `Frame.BitDepthLuma` and `Frame.BitDepthChroma` as the authoritative
  selector for which plane set is populated.
- Keep `AppendRawYUV` 8-bit-only for compatibility.
- Use `AppendRawYUV16(dst []uint16)` for caller-side sample-order output.
- Use `AppendRawYUVBytesLE(dst []byte)` for FFmpeg rawvideo/framemd5 parity. It
  writes each 9/10/12/14-bit sample as an unshifted little-endian uint16; do not
  left-shift to MSB alignment, normalize to 16-bit range, or downconvert. Samples
  above the declared bit depth are invalid.
- Use `RawPixelFormat`, `RawYUVSize`, and `BytesPerSample` to size oracle
  buffers and command-line `-pix_fmt` arguments.

Cropping must be implemented in sample units:

- Luma crop uses `CropLeft` and `CropTop` directly.
- Chroma crop must reuse the existing chroma crop geometry for 4:2:0 and 4:2:2.
- Monochrome high output should append only the luma plane.
- 4:4:4 high output should use full-resolution Cb/Cr planes.

The helper mapping is source-cited by FFmpeg `libavcodec/h264_slice.c`
`get_pixel_format`, `libavutil/pixfmt.h`, `libavutil/pixdesc.c`, and
`libavcodec/rawenc.c` `raw_encode` through `av_image_copy_to_buffer`:

| `ChromaFormatIDC` | FFmpeg software candidate | Raw helper `-pix_fmt` names |
| --- | --- | --- |
| 0 | Local luma-only oracle surface; FFmpeg H.264 software selection falls through the non-4:2:2/non-4:4:4 branch. | `gray9le`, `gray10le`, `gray12le`, `gray14le` |
| 1 | `AV_PIX_FMT_YUV420P9/10/12/14` | `yuv420p9le`, `yuv420p10le`, `yuv420p12le`, `yuv420p14le` |
| 2 | `AV_PIX_FMT_YUV422P9/10/12/14` | `yuv422p9le`, `yuv422p10le`, `yuv422p12le`, `yuv422p14le` |
| 3, YCbCr | `AV_PIX_FMT_YUV444P9/10/12/14` | `yuv444p9le`, `yuv444p10le`, `yuv444p12le`, `yuv444p14le` |
| 3, RGB colorspace | `AV_PIX_FMT_GBRP9/10/12/14` | Unsupported by the Y/Cb/Cr public helper until a GBR surface is designed. |

The `AV_PIX_FMT_*` names above are native-endian macros in FFmpeg; the oracle
surface must request explicit little-endian names so raw bytes are stable across
hosts. Monochrome output intentionally follows the existing local gray oracle
practice and appends only luma samples.

The high-bit-depth public decode guard is removed only for proved subsets:
High 10 4:2:0 deblock-disabled I pictures and High 10 4:2:0 deblock-disabled
P-skip/P16x16 no-residual pictures, High 10 4:2:0 frame-only deblock-disabled
exact P16x16 L0 residual pictures, and High 10 4:2:0 frame-only
deblock-disabled explicit weighted P16x16 pictures. Every later
partitioned/deblock/chroma-depth safe point should compare the public output
helper against FFmpeg frame MD5s before broadening the guard again.

## Oracle And Fixture Plan

The current kernel oracles are valuable but not sufficient. The remaining work
needs true high-bit-depth bitstream fixtures whose expected output is generated
by the pinned native FFmpeg tools.

Fixture principles:

- Use small dimensions first: 16x16 and 32x32 keep raw output and debugging
  tractable.
- Store compressed H.264 bitstreams under `testdata/h264/` only when default
  tests need them. Avoid committing raw frames.
- Capture expected rawvideo MD5s in tests/docs as the existing fixtures do.
- For public parity, compare little-endian rawvideo byte order for formats such
  as `yuv420p10le`, `yuv422p12le`, and `yuv444p14le`.
- Keep opt-in native oracle tests behind `GOH264_ORACLE=1`, but default tests
  should use committed bitstreams plus known MD5s once a fixture is accepted.
- Where local encoder support is uncertain, first add an oracle-probe note/test
  that skips cleanly if the local FFmpeg/x264 cannot generate the requested
  high-bit-depth stream.

Minimum fixture ladder:

1. High-depth IDR/I, CAVLC and CABAC, deblocking disabled.
2. High-depth IntraPCM and qscale-0/lossless cases.
3. High-depth IDR/P with P-skip/P16x16 no-residual first, then exact P16x16 L0
   residual, then explicit weighted P before any partitioned P.
4. High-depth partitioned P.
5. High-depth exact non-direct B16x16, then later non-direct B partitions only
   after a separate oracle.
6. High-depth temporal and spatial direct B.
7. High-depth implicit weighted B.
8. High-depth deblocking enabled for 4:2:0, 4:2:2, and 4:4:4.
9. Annex B, explicit AVC/NALFF, configured AVC, sample-by-sample configured
   decode, generic packet intake, and delayed flush coverage after each public
   high fixture family is stable.

The first fixture in each family should be narrow and boring. Add variations
only when they prove a new source-shaped branch: chroma format, entropy mode,
weighted mode, direct-motion mode, deblocking mode, or packet/public-output
surface.

## Safe-Point Sequence

1. **Represent High Frames Internally**
   - Done for frame storage and DPB views: internal and public frame structs
     carry uint16 high planes, the simple DPB exposes high ref lists, and raw
     helper surfaces use sample-unit strides.
   - Tests: allocation, chroma sizing, crop geometry, helper sizing/format
     behavior, DPB ref-list high view, and no 8-bit API behavior change.

2. **Decode High Intra Frames Internally**
   - Done for deblock-disabled intra frames: the simple slice dispatch calls
     `h264HLDecodeFrameMacroblockHigh` from high CAVLC/CABAC loops.
   - Tests: internal high CAVLC/CABAC slice-loop tests, including high
     IntraPCM path coverage.

3. **Expose Public High Output**
   - Partly done for High 10 4:2:0 deblock-disabled CAVLC/CABAC IDR/I pictures
     through Annex B, explicit AVC, and configured AVC public surfaces.
   - Tests: public high IDR/I rawvideo MD5 against FFmpeg, plus crop/chroma
     layout unit tests.

4. **Wire High 10 P-Slice No-Residual**
   - Done for High 10 4:2:0 deblock-disabled P-skip/P16x16:
     high refs, high motion scratch, CAVLC/CABAC frame-slice handoff, P-skip
     write-back, and high P16x16 motion reconstruction.
   - Guard after the weighted safe point: partitioned P, P intra macroblocks,
     direct B, implicit weighted B, partitioned B, high deblocking, and
     unproved depth/chroma combinations remain unsupported.
   - Tests/proof: Annex B, explicit AVC/NALFF, configured AVC,
     sample-by-sample configured decode, internal CAVLC/CABAC P-skip/P16x16
     tests, and opt-in FFmpeg framemd5 oracle checks.

5. **Wire High 10 P-Slice P16x16 Residual**
   - Done for High 10 4:2:0 frame-only, deblock-disabled exact P16x16 L0
     residual slices for CAVLC and CABAC.
   - The per-MB high guard now admits only this residual MB shape, residual high
     reconstruction runs through the public slice loop, and residual fixtures
     cover Annex B, explicit AVC/NALFF, configured AVC, sample-by-sample
     configured decode, and FFmpeg oracle checks.
   - Guard: 16x8/8x16/8x8 P partitions, P intra macroblocks, direct B,
     implicit weighted B, partitioned B, high deblocking, other chroma/depth
     combinations, and MBAFF remain unsupported.

6. **Wire High Weighted P**
   - Done for High 10 4:2:0 frame-only, deblock-disabled explicit weighted
     P16x16 CAVLC and CABAC streams.
   - Tests/proof: internal CAVLC/CABAC weighted P-skip/P16x16 slice-loop tests,
     syntax assertions for luma/chroma weights, public Annex B, AVC/NALFF,
     configured AVC, sample-by-sample configured decode, and FFmpeg
     rawvideo/framemd5 oracle checks.

7. **Wire High B Motion**
   - Done for exact non-direct B16x16 standard bidirectional avg, with
     direct B, direct-sub, implicit weighted B, and partitioned B left for later
     safe points.
   - Tests: non-direct B16x16 CAVLC/CABAC rawvideo MD5, configured
     sample-by-sample decode, delayed output and flush; later tests cover
     temporal/spatial direct B, B 8x8/B_SUB_4x4 direct-sub, implicit weighted B,
     and partitioned B.

8. **Wire High Loop Filter**
   - Add high threshold tables/indexing and high edge application.
   - Tests: deblock-enabled 4:2:0, 4:2:2, 4:4:4, plus disabled-filter controls.

9. **Broaden Packet Surfaces**
   - Mirror the current 8-bit fixture matrix for high-bit-depth streams:
     Annex B, AVC/NALFF, configured AVC, sample-by-sample configured decode,
     generic packet intake, side-data-new-extradata retention, and delayed flush.

10. **Performance And Cleanup**
   - Run allocation checks and benchmarks after scalar parity.
   - Only then consider shared abstractions or SIMD dispatch.

## Review Checklist For Future Workers

- Does the change preserve every existing 8-bit fixture and public API behavior?
- Is the high path using sample strides internally and explicit little-endian
  byte export only at the public/oracle boundary?
- Is each removal of an `ErrUnsupported` high-bit-depth guard paired with a
  high implementation and test?
- Does the fixture prove a new branch rather than just another profile label?
- Are FFmpeg `pixel_shift`, `qp_bd_offset`, chroma format, and transform-bypass
  semantics handled at the same source boundary as upstream?
- Are high reference frames retained in the DPB without copying or downconverting
  their samples?
- Are disabled tracing/oracle paths free from hot-loop allocations and public
  side effects?
- Is the safe point coherent enough to bisect: one semantic capability, focused
  tests, and no unrelated docs or ledger churn?
