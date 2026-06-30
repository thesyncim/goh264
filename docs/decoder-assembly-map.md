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
luma and chroma deblock ports, `caba3` moved from 122.6 ms/sample before the
deblock work to 106.8 ms/sample on the current default partial-asm build, a
12.9% end-to-end reduction. The same run is still about 1.81x slower than
FFmpeg pure-C and 2.39x slower than FFmpeg native C+asm, while `-tags=purego`
is about 2.41x slower than FFmpeg pure-C and 3.14x slower than FFmpeg native
C+asm. Current default partial assembly is about 1.33x faster than `purego` on
this vector.
The arm64 chroma deblock port keeps focused chroma vertical/horizontal
microbenchmarks at 0 allocs/op and moves their medians from about 16.5/16.2
ns/op in `purego` to about 9.2/13.9 ns/op in default builds. The follow-up
arm64 chroma 4:2:2 horizontal port moves the focused 8-bit 4:2:2 row from
about 27.1 ns/op in `purego` to about 20.4 ns/op in default builds, and the
8-bit high-422 `frext-hi422fr10-sony-b` real-vector gate passes raw-MD5 with
the zero-allocation Go gate in both lanes. The arm64 vertical/horizontal luma
intra ports move focused rows from about 57.8 ns/op in `purego` to about
10.0/17.4 ns/op in default builds, with 0 allocs/op. The matching `caba3`
real-vector gate passes raw-MD5 and the zero-allocation Go gate with default
partial asm around 106.8 ms/sample versus about 142.0 ms/sample for `purego`.
The arm64 4:2:0 chroma intra vertical/horizontal ports move focused rows from
about 12.2/12.1 ns/op in `purego` to about 6.0/9.3 ns/op in default builds,
with 0 allocs/op. The arm64 4:2:2 chroma intra horizontal port moves its
focused row from about 23.5 ns/op in `purego` to about 10.4 ns/op in default
builds. Chroma intra MBAFF remains scalar because the attempted assembly shape
did not beat the scalar path. The arm64 High10 chroma normal
vertical/horizontal/4:2:2 ports keep focused rows at 0 allocs/op and move
their medians from about 16.4/16.6/27.5 ns/op in `purego` to about
9.4/14.5/20.5 ns/op in default builds. The matching
10-bit `frext-hi422fr13-sony-b` real-vector gate passes raw-MD5 and the
zero-allocation Go gate with default partial asm around 81.3 ms/sample versus
about 96.6 ms/sample for `purego`. The arm64 High10 chroma intra
vertical/horizontal/4:2:2/4:2:2-MBAFF ports keep focused rows at 0 allocs/op
and move medians from about 14.2/14.2/24.7/14.3 ns/op in `purego` to about
6.2/10.3/11.1/10.3 ns/op in default builds. The tiny High10 4:2:0 chroma
intra MBAFF leaf is mapped but kept scalar because the direct assembly path did
not beat scalar; both default and `purego` stay around 9.0 ns/op there. The
intra-heavy 10-bit `frext-pph422i1-panasonic-a`
real-vector gate passes raw-MD5 and the zero-allocation Go gate with default
partial asm around 4385 ms/sample versus about 4685 ms/sample for `purego`;
that is a 6.4% reduction, but still about 4.1x slower than FFmpeg pure-C and
4.6x slower than FFmpeg native.
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
| Loop/deblock filter | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_deblock.asm`, `h264_deblock_10bit.asm`, `h264dsp_init.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/h264dsp_neon.S`, `h264dsp_init_aarch64.c` | wrappers in `internal/h264/dsp.go`, seam in `internal/h264/deblock_dispatch.go`, asm in `internal/h264/deblock_amd64.s` and `internal/h264/deblock_arm64.s`, integration in `internal/h264/loop_filter.go` | partial | Visible on deblocked `caba3`. The Go dispatch seam narrows FFmpeg C `int` thresholds to checked `int32` values and keeps strides as native signed `int`. Normal 8-bit luma and chroma vertical/horizontal, horizontal chroma 4:2:2, luma intra vertical/horizontal, 4:2:0 chroma intra vertical/horizontal, 4:2:2 chroma intra horizontal, High10 chroma vertical/horizontal/4:2:2, and High10 chroma intra vertical/horizontal/4:2:2 now use real FFmpeg-shaped arm64 NEON leaves; amd64 remains scalar-equivalent for luma and scalar for chroma. Chroma NEON is enabled only for non-negative `tc0` lanes; mixed negative lanes keep the scalar/oracle path. High10 assembly keeps FFmpeg's `uint8_t *` byte-pointer ABI at the assembly boundary and is gated to `bitDepth == 10`; 12/14-bit, High10 4:2:0 chroma intra MBAFF, luma MBAFF, luma high-bit, and high-bit luma intra variants still fall back to scalar. |
| Intra prediction | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_intrapred.asm`, `h264_intrapred_10bit.asm`, `h264_intrapred_init.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/h264pred_neon.S`, `h264pred_init.c` | `internal/h264/pred.go`, `internal/h264/pred_high.go`, dispatch by mode in `internal/h264/intra_prediction.go` and reconstruction files | mapped | Many small mode-specific kernels. Port after motion-comp and deblock unless profiles show intra-heavy workloads dominating. |
| CABAC helper optimizations | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/cabac.h`, `h264_cabac.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/cabac.h` | `internal/h264/cabac.go`, `internal/h264/cabac_mb.go`, `internal/h264/cabac_residual.go`, `internal/h264/cabac_frame.go` | mapped | Treat as a later micro-optimization lane. CABAC is profile-visible, but FFmpeg's asm/C helpers are tightly coupled to context layout and refill behavior. |
| VideoDSP support | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/videodsp.asm`, `videodsp_init.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/videodsp.S`, `videodsp_init.c` | edge-emulation and prefetch call sites under `internal/h264/motion_comp.go` and `internal/h264/motion_comp_high.go` | mapped | Edge emulation is profile-visible on `caba3`; the scalar helper now mirrors FFmpeg's valid-rectangle copy plus left/right extension instead of clipping every pixel. Dedicated amd64/arm64 asm remains todo; the pinned upstream arm64 file only wires prefetch, while x86 wires 8-bit SSE2/AVX2 edge emulation. |

## Implementation Order

1. Replace the remaining scalar-equivalent amd64 8-bit luma deblock asm leaves
   with FFmpeg-shaped SSE2/AVX math for normal vertical/horizontal non-intra
   edges.
2. Extend deblock coverage to luma/chroma intra, MBAFF, and high-bit leaves
   once the normal arm64 luma/chroma SIMD paths are verified.
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
| 1 | Loop filter 8-bit luma/chroma SIMD | partial | partial | arm64 now mirrors FFmpeg `ff_h264_v/h_loop_filter_luma_neon`, `ff_h264_v/h_loop_filter_chroma_neon`, `ff_h264_h_loop_filter_chroma422_neon`, `ff_h264_v/h_loop_filter_luma_intra_neon`, 4:2:0 `ff_h264_v/h_loop_filter_chroma_intra_neon`, and `ff_h264_h_loop_filter_chroma422_intra_neon` with raw NEON instruction words behind Go ABI wrappers. The focused luma deblock benchmark improved from roughly 55-57 ns/op purego to about 11 ns/op vertical and 18 ns/op horizontal on Apple arm64, with 0 allocs/op. Focused chroma deblock medians are about 9.2 ns/op vertical and 13.9 ns/op horizontal in default builds versus about 16.5 and 16.2 ns/op in `purego`; the focused 4:2:2 horizontal row is about 20.4 ns/op default versus 27.1 ns/op `purego`; focused luma intra vertical/horizontal rows are about 10.0/17.4 ns/op default versus 58.0/58.2 ns/op `purego`; focused chroma intra vertical/horizontal rows are about 6.0/9.3 ns/op default versus 12.2/12.1 ns/op `purego`; and focused chroma intra 4:2:2 is about 10.4 ns/op default versus 23.5 ns/op `purego`, all 0 allocs/op. Chroma intra MBAFF remains scalar because the attempted assembly shape did not beat scalar. The 8-bit high-422 `frext-hi422fr10-sony-b` real-vector benchmark passes raw-MD5 and the Go zero-allocation gate in both default and `purego` lanes, with default partial asm about 224.9 ms/sample versus about 266.6 ms/sample for `purego` in the sampled run. The `caba3` real-vector benchmark also passes after the chroma intra ports, with default partial asm about 106.8 ms/sample versus about 142.0 ms/sample for `purego`; before today's deblock work, the same default lane was about 122.6 ms/sample. amd64 still needs real FFmpeg `ff_deblock_v_luma_8_*`/`ff_deblock_h_luma_8_*` SSE2/AVX math instead of the current scalar-equivalent leaf, plus chroma leaves. ABI remains `*uint8` pixels, native signed `int` stride, `int32` alpha/beta, and `*int8` tc0 where the upstream prototype has tc0. |
| 2 | Loop filter intra/MBAFF/high | todo | partial | High10 arm64 chroma normal vertical/horizontal/4:2:2 now mirrors `ff_h264_v/h_loop_filter_chroma_neon_10` and `ff_h264_h_loop_filter_chroma422_neon_10`, gated to `bitDepth == 10`; focused medians are about 9.4/14.5/20.5 ns/op default versus 16.4/16.6/27.5 ns/op `purego`, all 0 allocs/op. The 10-bit `frext-hi422fr13-sony-b` real-vector benchmark passes raw-MD5 and the Go zero-allocation gate with default partial asm about 81.3 ms/sample versus about 96.6 ms/sample for `purego`. High10 arm64 chroma intra vertical/horizontal/4:2:2/4:2:2-MBAFF now mirrors `ff_h264_v/h_loop_filter_chroma_intra_neon_10`, `ff_h264_h_loop_filter_chroma_intra_neon_10`, and `ff_h264_h_loop_filter_chroma422_intra_neon_10`; focused medians are about 6.2/10.3/11.1/10.3 ns/op default versus 14.2/14.2/24.7/14.3 ns/op `purego`, all 0 allocs/op. The direct High10 4:2:0 chroma intra MBAFF leaf is mapped but not dispatched because it did not beat scalar; both default and `purego` stay around 9.0 ns/op there. The intra-heavy 10-bit `frext-pph422i1-panasonic-a` real-vector benchmark passes raw-MD5 and the Go zero-allocation gate with default partial asm about 4385 ms/sample versus about 4685 ms/sample for `purego`, still about 4.1x slower than FFmpeg pure-C and 4.6x slower than FFmpeg native. Cover luma high-bit, high-bit luma intra, `ff_deblock_h_luma_mbaff_8_*`, and 12/14-bit scalar-only lanes later. Keep high-bit assembly boundaries byte-pointer/byte-stride shaped where mirroring FFmpeg. |
| 3 | VideoDSP edge emulation and prefetch | todo | todo | `h264EmulatedEdgeMC`/`h264EmulatedEdgeMCHigh` now use FFmpeg-shaped scalar rectangle copy plus border extension with 0-alloc focused benchmarks. Add dedicated asm only after default and `purego` real-vector benches show the remaining edge helper cost is still material. |
| 4 | Weighted prediction 8/high | todo | todo | Source-equivalent families are `ff_h264_weight_*` and `ff_h264_biweight_*` on x86 plus NEON weighted-pixels leaves on arm64. Preserve signed offsets, implicit B weighting, high-bit-depth clipping, and C-int width at the dispatch boundary. |
| 5 | IDCT/add-pixels 4x4/8x8/DC | todo | todo | Must choose the local ABI first because FFmpeg uses `int16_t *block` while the current scalar code uses `[]int32`. Verify block clearing and transform-bypass exclusions. |
| 6 | Intra prediction 4x4/8x8/16x16/chroma | todo | todo | Many small kernels; enable in batches by mode only when intra-heavy profiles justify it. |
| 7 | CABAC helpers | todo | todo | Evidence-gated; do not start before the larger DSP/motion/deblock families stop dominating. |
| done | Qpel 8-bit sizes 16/8/4/2 all positions | verified | verified | All qpel positions `mc00` through `mc33` are assembly-backed for sizes 16/8/4/2 in `internal/h264/qpel_mc_amd64.s`, `internal/h264/qpel_mc_arm64.s`, `internal/h264/qpel_hvblend_amd64.s`, and `internal/h264/qpel_hvblend_arm64.s`. Kernel seam in `internal/h264/qpel.go` uses `[]uint8`, native signed pointer-width strides/offsets, and 32-bit C-int selectors. Verified against `TestH264Qpel*`, `qpel_oracle_test.go`, cross-arch symbol builds, focused qpel `-benchmem`, and real-vector `caba3`/`canl4` in default and `purego` lanes. |
| done | Chroma MC 8-bit width 8/4/2 put and avg | verified | verified | Widths 8/4/2/1 copy and fractional x/y put/avg leaves are implemented in `internal/h264/chroma_mc_amd64.s`, `internal/h264/chroma_mc_arm64.s`, `internal/h264/chroma_fractional_amd64.s`, and `internal/h264/chroma_fractional_arm64.s`. Kernel seam uses `[]uint8`, native signed pointer-width strides, and 32-bit C-int selectors/weights. Verified against `TestH264Chroma*`, `chroma_oracle_test.go`, focused `-benchmem`, real-vector `caba3`/`canl4`, and the public benchmark canary. |
| done | Qpel high-bit-depth sizes 16/8/4/2 | verified | verified | `mc00` copy/avg, one-axis `mc10/20/30/01/02/03`, center HV `mc22`, odd/odd HV `mc11/mc31/mc13/mc33`, and HV-blend `mc21/mc12/mc32/mc23` filters are implemented in `internal/h264/qpel_high_amd64.s` and `internal/h264/qpel_high_arm64.s` through `h264QpelMCHigh00ASM`, `h264QpelMCHighX0ASM`, `h264QpelMCHigh0YASM`, `h264QpelMCHigh22ASM`, `h264QpelMCHighHVXYASM`, and `h264QpelMCHighHVBlendASM`, which accept `*uint8` buffers and native byte strides after the Go `[]uint16` wrapper validates sample geometry. Verified against high-bit dispatch parity with separate strides, `TestH264QpelUpstreamOracle` default and `purego`, cross-arch symbols, focused High10 qpel `-benchmem`, full default and `purego` test gates, the public benchmark canary, and high-bit real-vector decode. |
| done | Chroma MC high-bit-depth width 8/4/2/1 | verified | verified | Implemented in `internal/h264/chroma_high_amd64.s` and `internal/h264/chroma_high_arm64.s`. `h264ChromaMCHighASM` accepts `*uint8` buffers, native byte strides, 32-bit C-int selectors/weights, and a native byte `step`; Go dispatch validates the `[]uint16` public/internal helper shape and narrows to FFmpeg-shaped byte pointers. Verified against high-bit dispatch parity with separate strides, `TestH264ChromaMCUpstreamOracle` default and `purego`, cross-arch symbols, focused high-bit chroma `-benchmem`, and real-vector `frext-hi422fr13-sony-b` in default and `purego` lanes. |
