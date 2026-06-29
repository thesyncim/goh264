# H.264 Decoder Assembly Map

This file tracks the decoder SIMD/assembly port from the FFmpeg n8.0.1
upstream snapshot pinned in `docs/source-truth.md` into the Go decoder. A row
is not complete until the Go entrypoint has an oracle test, architecture
dispatch, `purego` scalar fallback, and real-vector benchmark coverage.

Current profile evidence on Apple arm64 is dominated by CABAC slice decode plus
qpel/chroma motion compensation on `canl4` (`cabac`, `spatial-direct`,
`no-deblock`). Deblock is still mapped because it is a large upstream assembly
surface and expected to matter on deblocked streams, but it was not exercised by
that sample.

## Status Key

- `mapped`: upstream files and Go scalar owners are identified.
- `seam`: Go dispatch/ABI seam exists, scalar behavior unchanged.
- `partial`: at least one architecture-specific assembly leaf is enabled, but
  the family is not complete.
- `ported`: at least one architecture has an assembly implementation.
- `verified`: oracle tests and real-vector benchmarks pass for the ported path.

## Build Lanes

- Default builds on `amd64` and `arm64` use assembly-capable dispatch files.
  Until a kernel is ported, those dispatch files call the scalar reference.
- `-tags=purego` forces scalar kernels and must keep passing the same oracle,
  real-vector, allocation, and benchmark gates.
- Unsupported architectures use the scalar path by default.
- Benchmark evidence should include both default and `purego` lanes once an
  assembly kernel is enabled.
- Assembly-facing types must mirror the pinned FFmpeg prototypes:
  `uint8_t` buffers map directly to `uint8`, high-bit-depth scalar pixels map to
  `uint16_t`/`uint16` only inside Go scalar helpers, `ptrdiff_t` strides and
  offsets stay native signed pointer width, and C `int` selector arguments use
  32-bit Go values at dispatch boundaries. Public/internal Go wrappers reject
  out-of-range C `int` values before narrowing to assembly kernels.

## Upstream Families

| Family | Upstream amd64/x86 | Upstream arm64 | Go scalar owner | Status | Notes |
| --- | --- | --- | --- | --- | --- |
| Chroma motion compensation | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_chromamc.asm`, `h264_chromamc_10bit.asm`, `h264chroma_init.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/h264cmc_neon.S`, `h264chroma_init_aarch64.c` | `internal/h264/chroma.go`, `internal/h264/chroma_dispatch_asm.go`, `internal/h264/chroma_dispatch_purego.go`, `internal/h264/chroma_mc_amd64.s`, `internal/h264/chroma_mc_arm64.s`, call sites in `internal/h264/motion_comp.go` and `internal/h264/motion_comp_high.go` | partial | Widths 1/2/4/8, put/avg, 8-bit and high-bit-depth. `h264ChromaMCStridesKernel` keeps `uint8_t`/`ptrdiff_t`/`int` width parity. Enabled leaves: 8-bit widths 8/4/2/1, `x=0`, `y=0`, put/avg on amd64 and arm64, with separate source/destination stride support in the Go dispatch wrapper. High-bit-depth remains scalar until a byte-stride `uint8_t*` ABI is added for the FFmpeg 10-bit kernels. |
| Luma qpel motion compensation | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_qpel_8bit.asm`, `h264_qpel_10bit.asm`, `h264_qpel.c`, plus `x86/fpel.asm` and `x86/qpel.asm` support | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/h264qpel_neon.S`, `h264qpel_init_aarch64.c`, plus mc00 helpers in `aarch64/hpeldsp_neon.S` | `internal/h264/qpel.go`, `internal/h264/qpel_dispatch_asm.go`, `internal/h264/qpel_dispatch_purego.go`, call sites in `internal/h264/motion_comp.go` and `internal/h264/motion_comp_high.go` | seam | Largest motion-comp family and a direct profile hit. `h264QpelMCStridesKernel` is the current 8-bit assembly-capable seam and keeps `uint8_t`/`ptrdiff_t` selector width parity. Final assembly leaves should split by size/position like FFmpeg's qpel tables. High-bit-depth remains scalar until a byte-stride `uint8_t*` ABI is added for the 10-bit kernels. |
| Weighted prediction | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_weight.asm`, `h264_weight_10bit.asm` | covered by DSP/init paths in the arm64 snapshot when present upstream | `internal/h264/dsp.go`, weighted call sites in `internal/h264/motion_comp.go` and `internal/h264/motion_comp_high.go` | mapped | Explicit and implicit weighted prediction are hot in B/P weighted streams. Must preserve signed offset and rounding behavior. |
| IDCT and add-pixels | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_idct.asm`, `h264_idct_10bit.asm`, `h264dsp_init.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/h264idct_neon.S`, `h264dsp_init_aarch64.c` | `internal/h264/idct.go`, `internal/h264/dsp.go`, reconstruction call sites in `internal/h264/reconstruct.go` and `internal/h264/reconstruct_high.go` | mapped | Includes 4x4, 8x8, DC-only, chroma DC, and add-pixels clear variants. Needs exact block clearing semantics. |
| Loop/deblock filter | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_deblock.asm`, `h264_deblock_10bit.asm`, `h264dsp_init.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/h264dsp_neon.S`, `h264dsp_init_aarch64.c` | wrappers in `internal/h264/dsp.go`, integration in `internal/h264/loop_filter.go` | mapped | Likely high ROI on larger content. Must preserve MBAFF, chroma 4:2:2, intra/non-intra, and high-bit-depth variants. |
| Intra prediction | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_intrapred.asm`, `h264_intrapred_10bit.asm`, `h264_intrapred_init.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/h264pred_neon.S`, `h264pred_init.c` | `internal/h264/pred.go`, `internal/h264/pred_high.go`, dispatch by mode in `internal/h264/intra_prediction.go` and reconstruction files | mapped | Many small mode-specific kernels. Port after motion-comp and deblock unless profiles show intra-heavy workloads dominating. |
| CABAC helper optimizations | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/cabac.h`, `h264_cabac.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/cabac.h` | `internal/h264/cabac.go`, `internal/h264/cabac_mb.go`, `internal/h264/cabac_residual.go`, `internal/h264/cabac_frame.go` | mapped | Treat as a later micro-optimization lane. CABAC is profile-visible, but FFmpeg's asm/C helpers are tightly coupled to context layout and refill behavior. |
| VideoDSP support | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/videodsp.asm`, `videodsp_init.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/videodsp.S`, `videodsp_init.c` | edge-emulation and prefetch call sites under `internal/h264` | mapped | Track as support code for MC. Port only when a decoder profile shows edge emulation or prefetch wrappers matter. |

## Implementation Order

1. Luma qpel 8-bit size 16 and 8, all positions, on arm64 first, then amd64.
2. Chroma MC 8-bit width 8/4/2 put and avg on arm64 first, then amd64.
3. Loop/deblock luma 8-bit vertical/horizontal non-intra; swap with IDCT if
   fresh deblock profiles stay cold.
4. Luma qpel high-bit-depth.
5. Chroma MC high-bit-depth width 8/4 put and avg.
6. Loop/deblock high-bit-depth and chroma/MBAFF variants.
7. IDCT/add-pixels 4x4, 8x8, and DC-only.
8. Weighted prediction.
9. Intra prediction modes.
10. CABAC helper experiments only after the above have benchmark evidence.

## Porting Rules

- Keep scalar functions as the reference implementation and `purego` fallback.
- Add architecture dispatch only after a scalar equivalence test can force both
  scalar and assembly paths over the same generated cases.
- Assembly entrypoints must accept already-validated, offset-adjusted slices or
  pointers. Bounds checks and public error behavior stay in Go wrappers. Match
  FFmpeg type widths exactly at that boundary: `uint8_t` -> `uint8`,
  `uint16_t` -> `uint16`, `int16_t` -> `int16`, `ptrdiff_t` -> native signed
  pointer width, and C `int` -> 32-bit signed values.
- Every kernel must be covered by the existing oracle test family before it is
  enabled in decode paths:
  `chroma_oracle_test.go`, `qpel_oracle_test.go`, `dsp_oracle_test.go`,
  `idct_oracle_test.go`, `pred_oracle_test.go`, and real-vector decode tests.
- A port is not complete until it has benchmark evidence from
  `scripts/h264-real-vector-bench.sh` and the public benchmark canary in both
  default and `-tags=purego` lanes where the scripts support it.

## Active Work Queue

| Priority | Kernel group | amd64 | arm64 | Tracking notes |
| --- | --- | --- | --- | --- |
| 1 | Qpel 8-bit size 16/8 all positions | seam | seam | Kernel seam in `internal/h264/qpel.go` uses `[]uint8`, native signed pointer-width strides/offsets, and 32-bit C-int selectors. Next step is arm64 NEON and amd64 SIMD implementations behind the checked wrapper. Verify against `TestH264Qpel*`, `qpel_oracle_test.go`, and real-vector `caba3`/`canl4`. |
| 2 | Chroma MC 8-bit width 8/4/2 put and avg | partial | partial | Widths 8/4/2/1 `x=0,y=0` put/avg leaves are implemented in `internal/h264/chroma_mc_amd64.s` and `internal/h264/chroma_mc_arm64.s`. Remaining leaves: fractional x/y put/avg for all widths. Kernel seam uses `[]uint8`, native signed pointer-width strides, and 32-bit C-int selectors. Verify against `TestH264Chroma*`, `chroma_oracle_test.go`, and real-vector `caba3`/`canl4`. |
| 3 | Loop filter 8-bit luma/chroma | todo | todo | Needs MBAFF and chroma 4:2:2 rows in the forced-path test. Profile with deblock-heavy rows before enabling. |
| 4 | Qpel high-bit-depth size 16/8 | mapped | mapped | Current Go scalar helper uses `[]uint16`; the assembly port must add the FFmpeg-shaped byte-pointer/byte-stride ABI before enabling 10-bit leaves. |
| 5 | Chroma MC high-bit-depth width 8/4 | mapped | mapped | Current Go scalar helper uses `[]uint16`; the assembly port must add the FFmpeg-shaped byte-pointer/byte-stride ABI before enabling 10-bit leaves. |
| 6 | IDCT/add-pixels 4x4/8x8/DC | todo | todo | Must verify block clearing and transform-bypass exclusions. |
| 7 | Weighted prediction 8/high | todo | todo | Must preserve signed offsets, implicit B weighting, and high-bit-depth clipping. |
| 8 | Intra prediction 4x4/8x8/16x16/chroma | todo | todo | Many small kernels; enable in batches by mode. |
| 9 | CABAC helpers | todo | todo | Evidence-gated; do not start before allocation-free decoder path is stable. |
