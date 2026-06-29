# H.264 Decoder Assembly Map

This file tracks the decoder SIMD/assembly port from the FFmpeg n8.0.1
upstream snapshot pinned in `docs/source-truth.md` into the Go decoder. A row
is not complete until the Go entrypoint has an oracle test, architecture
dispatch, `purego` scalar fallback, and real-vector benchmark coverage.

Current profile evidence on Apple arm64 is dominated by CABAC slice decode,
motion compensation, and loop filtering. On `caba3` (`cabac`,
`temporal-direct`, `deblock`), the fair table is still behind FFmpeg after
using amortized FFmpeg CLI timing over repeated Annex B input, raw-MD5
validation, truthful Go-backend comparison lanes, and a zero-allocation Go
gate. With 10 timed iterations, 5 repeats, and 2 warmups after the arm64 NEON
luma deblock port, default Go partial-asm is about 1.8x slower than FFmpeg
pure-C and 2.4x slower than FFmpeg native C+asm, while `-tags=purego` is about
2.4x slower than FFmpeg pure-C and 3.2x slower than FFmpeg native C+asm. The
same run shows default partial assembly about 1.31x faster than `purego`.
Recent profiles no longer put edge emulation at the top; the remaining frontier
is real SIMD deblock, more motion-comp work where profile-visible, weighted
prediction, IDCT/add-pixels, intra prediction, and only then CABAC experiments.

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
| Chroma motion compensation | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_chromamc.asm`, `h264_chromamc_10bit.asm`, `h264chroma_init.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/h264cmc_neon.S`, `h264chroma_init_aarch64.c` | `internal/h264/chroma.go`, `internal/h264/chroma_dispatch_asm.go`, `internal/h264/chroma_dispatch_purego.go`, `internal/h264/chroma_dispatch_high_asm.go`, `internal/h264/chroma_dispatch_high.go`, `internal/h264/chroma_mc_amd64.s`, `internal/h264/chroma_mc_arm64.s`, `internal/h264/chroma_fractional_amd64.s`, `internal/h264/chroma_fractional_arm64.s`, `internal/h264/chroma_high_amd64.s`, `internal/h264/chroma_high_arm64.s`, call sites in `internal/h264/motion_comp.go` and `internal/h264/motion_comp_high.go` | verified | Widths 1/2/4/8, put/avg, 8-bit and high-bit-depth. The 8-bit seam uses `uint8_t`/`ptrdiff_t`/`int` width parity directly. The high-bit-depth seam keeps the FFmpeg `uint8_t*` byte-pointer ABI at the assembly boundary, converts Go `uint16` sample strides to native byte strides in dispatch, and keeps C `int` selectors as `int32`. Enabled leaves: all 8-bit and high-bit-depth copy/fractional x/y cases for widths 8/4/2/1, put/avg on amd64 and arm64, with separate source/destination stride support in the Go dispatch wrapper. The pinned upstream arm64 init only wires 8-bit chroma MC, so the local arm64 high-bit leaf mirrors the upstream C template rather than an upstream NEON high-bit leaf. |
| Luma qpel motion compensation | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_qpel_8bit.asm`, `h264_qpel_10bit.asm`, `h264_qpel.c`, plus `x86/fpel.asm` and `x86/qpel.asm` support | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/h264qpel_neon.S`, `h264qpel_init_aarch64.c`, plus mc00 helpers in `aarch64/hpeldsp_neon.S` | `internal/h264/qpel.go`, `internal/h264/qpel_dispatch_asm.go`, `internal/h264/qpel_dispatch_purego.go`, `internal/h264/qpel_dispatch_high_asm.go`, `internal/h264/qpel_dispatch_high.go`, `internal/h264/qpel_mc_amd64.s`, `internal/h264/qpel_mc_arm64.s`, `internal/h264/qpel_hvblend_amd64.s`, `internal/h264/qpel_hvblend_arm64.s`, `internal/h264/qpel_high_amd64.s`, `internal/h264/qpel_high_arm64.s`, call sites in `internal/h264/motion_comp.go` and `internal/h264/motion_comp_high.go` | verified | Largest motion-comp family and a direct profile hit. `h264QpelMCStridesKernel` keeps `uint8_t`/`ptrdiff_t` selector width parity with separate source/destination strides in the Go dispatch wrapper. Enabled leaves: all 8-bit and high-bit-depth qpel positions for sizes 16/8/4/2, put/avg, on amd64 and arm64. High-bit-depth entrypoints use a `uint8_t*` byte-pointer ABI and native byte strides, with Go `[]uint16` wrappers only at the validated scalar boundary. The upstream C oracle covers FFmpeg's put2/4/8/16 and avg4/8/16 leaves; local scalar dispatch parity covers the local avg2 helper. |
| Weighted prediction | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_weight.asm`, `h264_weight_10bit.asm` | covered by DSP/init paths in the arm64 snapshot when present upstream | `internal/h264/dsp.go`, weighted call sites in `internal/h264/motion_comp.go` and `internal/h264/motion_comp_high.go` | mapped | Explicit and implicit weighted prediction are hot in B/P weighted streams. Must preserve signed offset and rounding behavior. |
| IDCT and add-pixels | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_idct.asm`, `h264_idct_10bit.asm`, `h264dsp_init.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/h264idct_neon.S`, `h264dsp_init_aarch64.c` | `internal/h264/idct.go`, `internal/h264/dsp.go`, reconstruction call sites in `internal/h264/reconstruct.go` and `internal/h264/reconstruct_high.go` | mapped | Includes 4x4, 8x8, DC-only, chroma DC, and add-pixels clear variants. Needs exact block clearing semantics. |
| Loop/deblock filter | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_deblock.asm`, `h264_deblock_10bit.asm`, `h264dsp_init.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/h264dsp_neon.S`, `h264dsp_init_aarch64.c` | wrappers in `internal/h264/dsp.go`, seam in `internal/h264/deblock_dispatch.go`, asm in `internal/h264/deblock_amd64.s` and `internal/h264/deblock_arm64.s`, integration in `internal/h264/loop_filter.go` | partial | Visible on deblocked `caba3`. The Go dispatch seam narrows FFmpeg C `int` thresholds to checked `int32` values and keeps strides as native signed `int`. Normal 8-bit luma vertical/horizontal now uses a real FFmpeg-shaped arm64 NEON leaf; amd64 remains scalar-equivalent assembly. MBAFF, intra, chroma/chroma 4:2:2, and high-bit-depth variants still fall back to scalar. |
| Intra prediction | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_intrapred.asm`, `h264_intrapred_10bit.asm`, `h264_intrapred_init.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/h264pred_neon.S`, `h264pred_init.c` | `internal/h264/pred.go`, `internal/h264/pred_high.go`, dispatch by mode in `internal/h264/intra_prediction.go` and reconstruction files | mapped | Many small mode-specific kernels. Port after motion-comp and deblock unless profiles show intra-heavy workloads dominating. |
| CABAC helper optimizations | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/cabac.h`, `h264_cabac.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/cabac.h` | `internal/h264/cabac.go`, `internal/h264/cabac_mb.go`, `internal/h264/cabac_residual.go`, `internal/h264/cabac_frame.go` | mapped | Treat as a later micro-optimization lane. CABAC is profile-visible, but FFmpeg's asm/C helpers are tightly coupled to context layout and refill behavior. |
| VideoDSP support | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/videodsp.asm`, `videodsp_init.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/videodsp.S`, `videodsp_init.c` | edge-emulation and prefetch call sites under `internal/h264/motion_comp.go` and `internal/h264/motion_comp_high.go` | mapped | Edge emulation is profile-visible on `caba3`; the scalar helper now mirrors FFmpeg's valid-rectangle copy plus left/right extension instead of clipping every pixel. Dedicated amd64/arm64 asm remains todo; the pinned upstream arm64 file only wires prefetch, while x86 wires 8-bit SSE2/AVX2 edge emulation. |

## Implementation Order

1. Replace the remaining scalar-equivalent amd64 8-bit luma deblock asm leaves
   with FFmpeg-shaped SSE2/AVX math for normal vertical/horizontal non-intra
   edges.
2. Extend deblock coverage to intra, MBAFF, chroma/chroma 4:2:2, and high-bit
   leaves once the normal luma SIMD path is verified.
3. Add VideoDSP edge-emulation asm only if fresh real-vector profiles keep it
   material after the checked-once motion validation split.
4. Add weighted prediction leaves for weighted P/B streams.
5. Add IDCT/add-pixels 4x4, 8x8, DC-only, and clear variants after choosing the
   local block ABI.
6. Add intra prediction mode batches for intra-heavy workloads.
7. Treat CABAC helper asm as an evidence-gated later lane because it is tightly
   coupled to decoder context layout and refill behavior.

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
| 1 | Loop filter 8-bit luma SIMD | partial | verified | arm64 now mirrors FFmpeg `ff_h264_v_loop_filter_luma_neon`/`ff_h264_h_loop_filter_luma_neon` with raw NEON instruction words behind the existing Go ABI. The focused luma deblock benchmark improved from roughly 55-57 ns/op purego to about 11 ns/op vertical and 18-21 ns/op horizontal on Apple arm64, with 0 allocs/op. amd64 still needs real FFmpeg `ff_deblock_v_luma_8_*`/`ff_deblock_h_luma_8_*` SSE2/AVX math instead of the current scalar-equivalent leaf. ABI remains `*uint8` pixels, native signed `int` stride, `int32` alpha/beta, and `*int8` tc0. |
| 2 | Loop filter intra/MBAFF/chroma/high | todo | todo | Add after normal luma SIMD is verified. Cover `ff_deblock_h_luma_mbaff_8_*`, intra luma, chroma, chroma 4:2:2, and high-bit leaves. Keep high-bit assembly boundaries byte-pointer/byte-stride shaped where mirroring FFmpeg. |
| 3 | VideoDSP edge emulation and prefetch | todo | todo | `h264EmulatedEdgeMC`/`h264EmulatedEdgeMCHigh` now use FFmpeg-shaped scalar rectangle copy plus border extension with 0-alloc focused benchmarks. Add dedicated asm only after default and `purego` real-vector benches show the remaining edge helper cost is still material. |
| 4 | Weighted prediction 8/high | todo | todo | Source-equivalent families are `ff_h264_weight_*` and `ff_h264_biweight_*` on x86 plus NEON weighted-pixels leaves on arm64. Preserve signed offsets, implicit B weighting, high-bit-depth clipping, and C-int width at the dispatch boundary. |
| 5 | IDCT/add-pixels 4x4/8x8/DC | todo | todo | Must choose the local ABI first because FFmpeg uses `int16_t *block` while the current scalar code uses `[]int32`. Verify block clearing and transform-bypass exclusions. |
| 6 | Intra prediction 4x4/8x8/16x16/chroma | todo | todo | Many small kernels; enable in batches by mode only when intra-heavy profiles justify it. |
| 7 | CABAC helpers | todo | todo | Evidence-gated; do not start before the larger DSP/motion/deblock families stop dominating. |
| done | Qpel 8-bit sizes 16/8/4/2 all positions | verified | verified | All qpel positions `mc00` through `mc33` are assembly-backed for sizes 16/8/4/2 in `internal/h264/qpel_mc_amd64.s`, `internal/h264/qpel_mc_arm64.s`, `internal/h264/qpel_hvblend_amd64.s`, and `internal/h264/qpel_hvblend_arm64.s`. Kernel seam in `internal/h264/qpel.go` uses `[]uint8`, native signed pointer-width strides/offsets, and 32-bit C-int selectors. Verified against `TestH264Qpel*`, `qpel_oracle_test.go`, cross-arch symbol builds, focused qpel `-benchmem`, and real-vector `caba3`/`canl4` in default and `purego` lanes. |
| done | Chroma MC 8-bit width 8/4/2 put and avg | verified | verified | Widths 8/4/2/1 copy and fractional x/y put/avg leaves are implemented in `internal/h264/chroma_mc_amd64.s`, `internal/h264/chroma_mc_arm64.s`, `internal/h264/chroma_fractional_amd64.s`, and `internal/h264/chroma_fractional_arm64.s`. Kernel seam uses `[]uint8`, native signed pointer-width strides, and 32-bit C-int selectors/weights. Verified against `TestH264Chroma*`, `chroma_oracle_test.go`, focused `-benchmem`, real-vector `caba3`/`canl4`, and the public benchmark canary. |
| done | Qpel high-bit-depth sizes 16/8/4/2 | verified | verified | `mc00` copy/avg, one-axis `mc10/20/30/01/02/03`, center HV `mc22`, odd/odd HV `mc11/mc31/mc13/mc33`, and HV-blend `mc21/mc12/mc32/mc23` filters are implemented in `internal/h264/qpel_high_amd64.s` and `internal/h264/qpel_high_arm64.s` through `h264QpelMCHigh00ASM`, `h264QpelMCHighX0ASM`, `h264QpelMCHigh0YASM`, `h264QpelMCHigh22ASM`, `h264QpelMCHighHVXYASM`, and `h264QpelMCHighHVBlendASM`, which accept `*uint8` buffers and native byte strides after the Go `[]uint16` wrapper validates sample geometry. Verified against high-bit dispatch parity with separate strides, `TestH264QpelUpstreamOracle` default and `purego`, cross-arch symbols, focused High10 qpel `-benchmem`, full default and `purego` test gates, the public benchmark canary, and high-bit real-vector decode. |
| done | Chroma MC high-bit-depth width 8/4/2/1 | verified | verified | Implemented in `internal/h264/chroma_high_amd64.s` and `internal/h264/chroma_high_arm64.s`. `h264ChromaMCHighASM` accepts `*uint8` buffers, native byte strides, 32-bit C-int selectors/weights, and a native byte `step`; Go dispatch validates the `[]uint16` public/internal helper shape and narrows to FFmpeg-shaped byte pointers. Verified against high-bit dispatch parity with separate strides, `TestH264ChromaMCUpstreamOracle` default and `purego`, cross-arch symbols, focused high-bit chroma `-benchmem`, and real-vector `frext-hi422fr13-sony-b` in default and `purego` lanes. |
