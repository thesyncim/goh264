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
  path still rejects non-8-bit frames before reconstruction.
- `internal/h264/simple_decode.go` keeps `DecodedFrame.Y/Cb/Cr` as `[]uint8`.
  `newSimpleDecodedFrame` rejects `BitDepthLuma != 8` or
  `BitDepthChroma != 8`, and `validateSimpleFrameSliceDecodeInputs` rejects
  high-bit-depth slice reconstruction.
- `decoder.go` exposes public `Frame.Y/Cb/Cr` as `[]byte`. `Frame.AppendRawYUV`
  is deliberately 8-bit-only and returns `ErrUnsupported` for high-bit-depth
  frames.
- Entropy/state layers are farther along than output: CAVLC and CABAC frame-MB
  paths already size and hand off high-bit-depth IntraPCM payloads, carry high
  QP values, and persist residual/motion/direct state in bit-depth-neutral table
  shapes.
- Kernel-level high-bit-depth DSP is present and oracle-covered: IDCT/dequant,
  add-pixels, intra prediction, weighted/biweighted prediction, qpel, chroma MC,
  and luma/chroma deblocking have 9/10/12/14-bit reference implementations.
- `internal/h264/reconstruct_high.go` has a separate `h264PicturePlanesHigh`
  surface and internal high-bit-depth IntraPCM/intra reconstruction helpers for
  4:2:0, 4:2:2, and 4:4:4. That high path is not called from the slice loop,
  still rejects non-intra macroblocks, and rejects deblocking/border-exchange
  modes at its API boundary.
- `internal/h264/motion_comp_high.go` now mirrors the 8-bit `hl_motion`
  call-site layer over uint16 planes. It covers standard and weighted
  macroblock partitions, 4:2:0/4:2:2 chroma MC, 4:4:4 qpel-shaped Cb/Cr, and
  uint16 edge-emulation scratch in sample units.
- `internal/h264/reconstruct_high.go` now consumes high motion internally for
  inter macroblocks. The public simple slice path still rejects non-8-bit frame
  decode, so this is MB-level parity rather than public high-bit-depth output.
- `internal/h264/loop_filter.go` ports the generic frame-picture loop-filter
  strength and call-site wiring for 8-bit planes. High-bit-depth DSP kernels
  exist in `dsp.go`, but the frame filter still validates `BitDepthLuma == 8`
  and uses 52-entry local threshold tables instead of FFmpeg's `52*3` high-depth
  index model with `qp_bd_offset`.
- The simple DPB/ref-list path stores `*DecodedFrame` references and builds
  `[]*h264PicturePlanes` byte-plane references for motion compensation. High
  decoded frames must therefore either extend `DecodedFrame` with high planes or
  introduce a parallel high frame/ref context before inter prediction can work.

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
- Add public output only after internal high frame storage and at least one
  internal high-bit-depth bitstream fixture prove decode correctness.
- Every implementation step should land as a safe point with focused tests plus
  at least one oracle/framemd5 fixture once bitstream decode is involved.

## Gap Ledger

| Area | Current state | Remaining high-bit-depth work |
| --- | --- | --- |
| SPS/PPS/slice metadata | High bit depths parse; PPS/dequant tables cover 9/10/12/14; slice QP uses bit-depth max. | Preserve this behavior while removing simple-path high-bit-depth rejects only when the matching high decode path exists. |
| Entropy-to-state | CAVLC/CABAC frame-MB handoff, residuals, motion caches, direct motion, and high IntraPCM payload sizing exist. | Prove that high QP and high residual buffers survive a full high slice loop; add high-specific regression cases where QP exceeds 51. |
| Internal frame storage | `DecodedFrame` and simple DPB are byte-backed; `h264PicturePlanesHigh` exists only for isolated helpers. | Add uint16 frame planes and DPB/ref-list views, or a parallel high decoded-frame type that still carries the existing metadata, side data, POC, ref entries, and macroblock tables. |
| Intra reconstruction | Internal high IntraPCM/intra16x16/intra4x4/intra8x8 call sites exist and are oracle-covered. | Wire the high path into frame slice decode for deblock-disabled intra frames first, then for mixed I/P/B streams after high motion exists. |
| Inter/motion reconstruction | 8-bit `hl_motion` is integrated for P/B, weighted P, implicit B, direct B, and 4:4:4 planes. High `h264HLMotionFrame*` is now ported for internal MB-level 4:2:0/4:2:2/4:4:4 motion, explicit/implicit weighting, and edge emulation. | Wire high refs and motion through the simple slice/frame path, then prove public high P/B bitstreams and direct-motion consumption with framemd5. |
| Loop filter integration | 8-bit frame-picture strength/call-site integration works post-frame for the simple path; high deblock kernels exist. | Add high frame-picture filter wiring over uint16 planes, source-shaped high threshold indexing, chroma 4:2:0/4:2:2 and 4:4:4 edge dispatch, and high bitstream fixtures with deblocking enabled. |
| Public output | Public `Frame` exposes byte planes and `AppendRawYUV` only supports 8-bit. | Add a value-preserving high-bit-depth output surface and raw little-endian export helper for oracle comparison without breaking the 8-bit API. |
| Oracle fixtures | Kernel oracles cover high primitives; public frame-MD5 fixtures cover 8-bit High-profile streams. | Add true high-bit-depth framemd5 fixtures and default tests in the same style as current Annex B/AVC/configured packet tests. |

## Internal Frame And Plane Work

The first code safe point should make high-bit-depth frames representable without
yet changing public output. A conservative shape is to extend `DecodedFrame` with
high planes:

- Keep existing `Y`, `Cb`, `Cr []uint8` for 8-bit frames.
- Add `Y16`, `Cb16`, `Cr16 []uint16` for high-bit-depth frames.
- Keep `LumaStride` and `ChromaStride` as sample strides for both 8-bit and
  high-bit-depth paths.
- Add `picturePlanesHigh() h264PicturePlanesHigh` next to `picturePlanes()`.
- Add validation helpers that require exactly one backing representation for a
  decoded frame: byte planes for 8-bit, uint16 planes for high.

This should be done before any inter work because the DPB and ref-list builders
must be able to hand high reference planes to motion compensation. The simple DPB
can keep owning `*DecodedFrame`; the ref context can grow a high-plane sibling:

- `simpleFrameRefContext` currently carries `[2][]*h264PicturePlanes`.
- Add `[2][]*h264PicturePlanesHigh` or a separate high context.
- Build high refs from `entry.frame.picturePlanesHigh()` only when the current
  frame is high-bit-depth.
- Keep direct-motion context and `simpleRefEntry` frame identity unchanged, since
  POC, long/short marking, and direct-motion provenance are not sample-depth
  dependent.

Do not expose high planes publicly in this first safe point. Internal tests can
assert allocation, plane sizes, strides, chroma sizing, metadata copy, and DPB
ref-list construction without changing public behavior.

## High Intra Slice Integration

After high frames can be allocated, wire a narrow decode path for high
deblock-disabled intra frames:

- Split the simple slice decode dispatch by sample depth after headers are
  parsed and frame storage is allocated.
- Add `h264FrameSliceDecodeInputHigh` carrying high refs later, but initially it
  can carry only `SliceNum`.
- Add high CAVLC/CABAC slice loops or shared entropy loops that call
  `h264HLDecodeFrameMacroblockHigh`.
- Extend `h264FrameMBReconstructInputHigh` from the current isolated helper shape
  only as needed for parity; for the intra safe point it needs the existing
  intra fields, residual, PPS, transform-bypass flag, bit depth, and IntraPCM
  payload.
- Keep `DeblockingFilter` false at the reconstruction helper boundary. The
  current simple 8-bit path filters post-frame rather than doing row-time border
  exchange; high should follow the same simple-path sequencing until the
  row-threaded lane exists.

Focused fixtures for this step:

- 10-bit 4:2:0 CAVLC IDR/I, deblocking disabled.
- 10-bit 4:2:0 CABAC IDR/I, deblocking disabled.
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

- Add high decoded-frame storage and high ref-list construction.
- Dispatch high CAVLC/CABAC frame slices to `h264HLDecodeFrameMacroblockHigh`.
- Preserve the current public `BitDepthLuma == 8` guard until a high output
  surface and framemd5 oracle are present.

Suggested safe-point order:

1. High P-skip/P 16x16 with no residual and deblocking disabled. This proves
   high refs, zero/pskip motion, DPB state, and public decode sequencing with
   minimal transform interaction.
2. High P inter with residual add and 8x8-DCT/non-8x8-DCT cases. This proves
   inter residual IDCT over predicted uint16 planes.
3. High explicit weighted P. This proves high luma/chroma weight dispatch and
   sample clipping.
4. High non-direct B with explicit L0/L1 refs. This proves bidirectional avg and
   display-order output with high planes.
5. High temporal/spatial direct B and B 8x8/B_SUB_4x4 direct-sub. This should
   reuse the existing direct-motion tables and focus on high motion consumption.
6. High implicit weighted B. This proves DPB-fed implicit weights over uint16
   planes.

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

Public output should preserve sample values and keep the existing 8-bit API
stable. A low-risk public shape is:

- Keep `Frame.Y`, `Frame.Cb`, `Frame.Cr []byte` populated only for 8-bit frames.
- Add `Frame.Y16`, `Frame.Cb16`, `Frame.Cr16 []uint16` for high-bit-depth
  frames.
- Keep `Frame.YStride` and `Frame.CStride` in samples, not bytes.
- Keep `Frame.BitDepthLuma` and `Frame.BitDepthChroma` as the authoritative
  selector for which plane set is populated.
- Keep `AppendRawYUV` 8-bit-only for compatibility.
- Add a high-depth helper with an explicit name, for example
  `AppendRawYUV16(dst []uint16)` for sample-order output and/or
  `AppendRawYUVBytesLE(dst []byte)` for FFmpeg rawvideo framemd5 parity.

Cropping must be implemented in sample units:

- Luma crop uses `CropLeft` and `CropTop` directly.
- Chroma crop must reuse the existing chroma crop geometry for 4:2:0 and 4:2:2.
- Monochrome high output should append only the luma plane.
- 4:4:4 high output should use full-resolution Cb/Cr planes.

Do not expose high output until at least one high-bit-depth IDR/I fixture passes
against FFmpeg rawvideo bytes. Once exposed, every new high inter/deblock safe
point should compare the public output helper against FFmpeg frame MD5s.

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
3. High-depth IDR/P with P-skip and residual inter.
4. High-depth explicit weighted P.
5. High-depth explicit non-direct B.
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
   - Add high planes to internal decoded frames or a parallel high frame type.
   - Add high plane validation and DPB/ref-list high views.
   - Tests: allocation, chroma sizing, crop geometry, DPB ref-list high view,
     and no public API behavior change.

2. **Decode High Intra Frames Internally**
   - Allow high-bit-depth simple slice decode only for deblock-disabled intra
     frames.
   - Call `h264HLDecodeFrameMacroblockHigh` from CAVLC/CABAC slice loops.
   - Tests: internal high IDR/I CAVLC/CABAC fixtures and IntraPCM path.

3. **Expose Public High Output**
   - Add value-preserving high planes and raw little-endian export.
   - Tests: public high IDR/I frame MD5 against FFmpeg, plus crop/chroma layout
     unit tests.

4. **Wire High P Inter Motion**
   - Add high motion scratch, edge emulation, qpel/chroma dispatch, and P inter
     reconstruction.
   - Tests: P-skip, residual P, edge-emulation P, Annex B and public packet
     surfaces.

5. **Wire High Weighted P**
   - Add high explicit weight dispatch through public decode.
   - Tests: luma/chroma weight cases with clipping-sensitive samples.

6. **Wire High B Motion**
   - Add high standard bidirectional avg, direct B, direct-sub, and implicit
     weight coverage.
   - Tests: non-direct B, temporal/spatial direct B, B 8x8/B_SUB_4x4 direct-sub,
     implicit weighted B, delayed output and flush.

7. **Wire High Loop Filter**
   - Add high threshold tables/indexing and high edge application.
   - Tests: deblock-enabled 4:2:0, 4:2:2, 4:4:4, plus disabled-filter controls.

8. **Broaden Packet Surfaces**
   - Mirror the current 8-bit fixture matrix for high-bit-depth streams:
     Annex B, AVC/NALFF, configured AVC, sample-by-sample configured decode,
     generic packet intake, side-data-new-extradata retention, and delayed flush.

9. **Performance And Cleanup**
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
