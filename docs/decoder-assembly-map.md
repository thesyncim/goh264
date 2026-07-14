# H.264 Decoder Assembly Map

This file tracks the decoder SIMD/assembly port from the FFmpeg n8.0.1
upstream snapshot pinned in `docs/source-truth.md` into the Go decoder. A row
is not complete until the Go entrypoint has an oracle test, architecture
dispatch, `purego` scalar fallback, and real-vector benchmark coverage.

Current profile evidence on Apple arm64 is dominated by CABAC, loop filtering,
and motion-comp orchestration rather than a single missing leaf. Benchmark
commands build with PGO disabled unless an explicitly labeled profile is
provided, so the default lane represents ordinary library consumers.

The corrected `caba3-sva-b` baseline uses 100 complete decodes per worker, 12
paired balanced-order repeats, fresh decoder contexts per repeat, preloaded
identical Annex B bytes, and one decoder thread per worker. Context construction,
worker launch, file I/O, output materialization, and hashing are outside timing;
wakeup, parsing, decoding, drain/reset work, and completion synchronization are
inside timing on both sides. Against native libavcodec C+assembly, the default
Go build is currently slower: the single-worker candidate-over-baseline paired
geometric elapsed ratio is 1.6874 with a two-sided 95% confidence interval of
[1.6689, 1.7061], and the 12-worker ratio is 1.7326 with a confidence interval
of [1.6961, 1.7700]. Both lanes produce raw-video MD5
`63a0f8fdcbb87b0dff330acdd10905c0`, and the Go zero-steady-state-allocation
gate passes. The native C+assembly lane is the claim-eligible baseline for the
assembly-enabled Go build; libavcodec pure C is retained only as a diagnostic
lane. A performance win is not recorded until the paired confidence interval is
entirely below 1 in both the one-worker and multicore gates.
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
partial asm around 104.8 ms/sample versus about 141.4 ms/sample for `purego`.
The arm64 4:2:0 chroma intra vertical/horizontal ports move focused rows from
about 12.2/12.1 ns/op in `purego` to about 6.0/9.3 ns/op in default builds,
with 0 allocs/op. The arm64 4:2:2 chroma intra horizontal port moves its
focused row from about 23.5 ns/op in `purego` to about 10.4 ns/op in default
builds. The 8-bit 4:2:0 chroma intra MBAFF NEON leaf is implemented and has
direct scalar parity coverage, but remains undispatched because it did not beat
the scalar path. The arm64 High10 chroma normal
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
The amd64 8-bit normal luma vertical leaf now uses FFmpeg-shaped SSE2 math, and
the matching horizontal leaf uses the FFmpeg transpose/call/store wrapper shape
around that shared vertical core. On the local virtualized amd64 lane, focused
default rows are about 20 ns/op vertical and 48-49 ns/op horizontal, versus
about 78 ns/op vertical and 79-80 ns/op horizontal in `purego`, all
0 allocs/op. The amd64 8-bit luma MBAFF horizontal leaf now mirrors FFmpeg's
8x8 transpose path with an 8-lane `tc0` expansion; focused medians are about
37.7 ns/op default versus about 40.3 ns/op in `purego`, all 0 allocs/op. The
amd64 8-bit luma intra vertical and horizontal leaves now use SSE2 word
arithmetic with the same `uint8_t`/`ptrdiff_t`/`int` ABI shape; the horizontal
leaf transposes a 16x8 stack scratch tile, calls the shared vertical core, and
transposes back. Focused medians are about 32.4/47.8 ns/op default versus
about 69.6/69.0 ns/op in `purego`, all 0 allocs/op. The clean amd64
`caba3-sva-b` real-vector gate at
`08fbd927f449d03384d5841939c147c162024a92` passes raw-MD5 and the
zero-allocation Go gate with default partial asm at 157.1 ms/sample
(12.485 ns/raw-byte), versus 178.7 ms/sample (14.285 ns/raw-byte) in
`purego`. The installed FFmpeg binary on this host is arm64-only, so these
amd64 rows are Go default-vs-`purego`, parity, and zero-allocation evidence;
the paired FFmpeg arm64 CLI lanes measured 63.1 ms/sample pure-C and
48.5 ms/sample native C+asm, but they are not an x86-vs-x86 ISA-fair baseline.
The amd64 8-bit chroma vertical, chroma 4:2:2 horizontal, chroma intra
vertical, and chroma intra 4:2:2 horizontal leaves now use FFmpeg-shaped SSE2
math with `uint8_t`/`ptrdiff_t`/`int`/`int8_t*` width parity at the assembly
boundary. Focused local medians are about 16.0/25.6/10.0/21.4 ns/op in default
builds versus about 18.4/32.1/15.8/28.6 ns/op in `purego`, all 0 allocs/op.
The normal 4:2:0 chroma horizontal and intra horizontal wrappers are mapped and
implemented, but stay undispatched on amd64 because this local lane did not
show a win over scalar.
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
| Chroma motion compensation | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_chromamc.asm`, `h264_chromamc_10bit.asm`, `h264chroma_init.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/h264cmc_neon.S`, `h264chroma_init_aarch64.c` | `internal/h264/chroma.go`, `internal/h264/chroma_dispatch_asm.go`, `internal/h264/chroma_dispatch_purego.go`, `internal/h264/chroma_dispatch_high_asm.go`, `internal/h264/chroma_dispatch_high.go`, `internal/h264/chroma_mc_amd64.s`, `internal/h264/chroma_mc_arm64.s`, `internal/h264/chroma_fractional_amd64.s`, `internal/h264/chroma_fractional_arm64.s`, `internal/h264/chroma_fractional_neon_arm64.s`, `internal/h264/chroma_high_amd64.s`, `internal/h264/chroma_high_arm64.s`, call sites in `internal/h264/motion_comp.go` and `internal/h264/motion_comp_high.go` | verified | Widths 1/2/4/8, put/avg, 8-bit and high-bit-depth. The arm64 8-bit fractional width-8/4/2 leaves use NEON with independent source/destination strides; width-4 moved from roughly 26-31 ns/op to 12-14 ns/op and width-2 from 18-21 ns/op to 12-14 ns/op locally, all 0 allocs/op. Width-1 and high-bit-depth arm64 remain scalar assembly. The high-bit-depth seam keeps the FFmpeg `uint8_t*` byte-pointer ABI and native byte strides. |
| Luma qpel motion compensation | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_qpel_8bit.asm`, `h264_qpel_10bit.asm`, `h264_qpel.c`, plus `x86/fpel.asm` and `x86/qpel.asm` support | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/h264qpel_neon.S`, `h264qpel_init_aarch64.c`, plus mc00 helpers in `aarch64/hpeldsp_neon.S` | `internal/h264/qpel.go`, `internal/h264/qpel_dispatch_asm.go`, `internal/h264/qpel_dispatch_purego.go`, `internal/h264/qpel_dispatch_high_asm.go`, `internal/h264/qpel_dispatch_high.go`, `internal/h264/qpel_mc_amd64.s`, `internal/h264/qpel_mc_arm64.s`, `internal/h264/qpel_hvblend_amd64.s`, `internal/h264/qpel_hvblend_arm64.s`, `internal/h264/qpel_h_neon_arm64.s`, `internal/h264/qpel_v_neon_arm64.s`, `internal/h264/qpel_hv_neon_words_arm64.s`, `internal/h264/qpel_high_amd64.s`, `internal/h264/qpel_high_arm64.s`, call sites in `internal/h264/motion_comp.go` and `internal/h264/motion_comp_high.go` | partial | All positions are assembly-backed and oracle-verified. On arm64, the hot width-16/8 horizontal, vertical, odd/odd, center-22, and HV-blend fractional shapes now use pinned-upstream-shaped NEON; focused leaves improve by roughly 10-16x. Width-4/2 fractional and high-bit-depth entrypoints remain scalar assembly. The refreshed `caba3` profile attributes about 5% to the qpel family, down from about 21% before this port. |
| Weighted prediction | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_weight.asm`, `h264_weight_10bit.asm` | covered by DSP/init paths in the arm64 snapshot when present upstream | `internal/h264/dsp.go`, `internal/h264/weight_arm64.s`, weighted call sites in `internal/h264/motion_comp.go` and `internal/h264/motion_comp_high.go` | partial | The arm64 8-bit width-16 even-height leaf uses NEON with direct scalar differential coverage, zero-allocation focused benchmarks, and a guard for the signed-16-bit overflow corner. It improves the weighted `cawp1-toshiba-e` vector by about 1%. Other widths, biweight, and high-bit-depth remain scalar. |
| IDCT and add-pixels | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_idct.asm`, `h264_idct_10bit.asm`, `h264dsp_init.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/h264idct_neon.S`, `h264dsp_init_aarch64.c` | `internal/h264/idct.go`, `internal/h264/dsp.go`, reconstruction call sites in `internal/h264/reconstruct.go` and `internal/h264/reconstruct_high.go` | mapped | Includes 4x4, 8x8, DC-only, chroma DC, and add-pixels clear variants. Needs exact block clearing semantics. |
| Loop/deblock filter | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_deblock.asm`, `h264_deblock_10bit.asm`, `h264dsp_init.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/h264dsp_neon.S`, `h264dsp_init_aarch64.c` | wrappers in `internal/h264/dsp.go`, seam in `internal/h264/deblock_dispatch.go`, asm in `internal/h264/deblock_amd64.s` and `internal/h264/deblock_arm64.s`, integration in `internal/h264/loop_filter.go` | partial | Visible on deblocked `caba3`. The Go dispatch seam narrows FFmpeg C `int` thresholds to checked `int32` values and keeps strides as native signed `int`. Normal 8-bit luma and chroma vertical/horizontal, horizontal chroma 4:2:2, luma intra vertical/horizontal, 4:2:0 chroma intra vertical/horizontal, 4:2:2 chroma intra horizontal, High10 chroma vertical/horizontal/4:2:2, and High10 chroma intra vertical/horizontal/4:2:2 now use real FFmpeg-shaped arm64 NEON leaves; the 8-bit 4:2:0 chroma intra MBAFF NEON leaf is mapped and directly parity-tested but deliberately undispatched because it did not beat scalar. Amd64 normal 8-bit luma vertical/horizontal/MBAFF, luma intra vertical/horizontal, and the enabled amd64 8-bit chroma vertical, chroma 4:2:2 horizontal, chroma intra vertical, and chroma intra 4:2:2 horizontal leaves now use real FFmpeg-shaped SSE2 math. Amd64 4:2:0 chroma horizontal and intra horizontal wrappers are implemented but not dispatched because they did not beat scalar on the local lane. Chroma assembly is enabled only for non-negative `tc0` lanes; mixed negative lanes keep the scalar/oracle path. High10 assembly keeps FFmpeg's `uint8_t *` byte-pointer ABI at the assembly boundary and is gated to `bitDepth == 10`; 12/14-bit, High10 4:2:0 chroma intra MBAFF, arm64 luma MBAFF, luma high-bit, and high-bit luma intra variants still fall back to scalar. |
| Intra prediction | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/h264_intrapred.asm`, `h264_intrapred_10bit.asm`, `h264_intrapred_init.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/h264pred_neon.S`, `h264pred_init.c` | `internal/h264/pred.go`, `internal/h264/pred_high.go`, dispatch by mode in `internal/h264/intra_prediction.go` and reconstruction files | mapped | Many small mode-specific kernels. Port after motion-comp and deblock unless profiles show intra-heavy workloads dominating. |
| CABAC helper optimizations | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/cabac.h`, `h264_cabac.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/cabac.h` | `internal/h264/cabac.go`, `internal/h264/cabac_mb.go`, `internal/h264/cabac_residual.go`, `internal/h264/cabac_frame.go` | partial | The concrete slice decoder hand-inlines get/bypass/sign arithmetic, specializes MVD, residual, and intra-prediction bins without changing the scripted interface seam, and masks the pinned 0..9 normalization shift so arm64 emits native register shifts without redundant guards. A 256-step differential state test covers the primitive. The corrected no-PGO profile attributes about 70% of samples to CABAC slice decoding, including about 13% to residual decoding and about 10% directly to unchecked table access. Native libavcodec uses an inlined arm64 CABAC arithmetic/refill block, so the next experiments must batch enough CABAC work to amortize a Go-to-assembly call. |
| VideoDSP support | `.upstream/ffmpeg-n8.0.1/libavcodec/x86/videodsp.asm`, `videodsp_init.c` | `.upstream/ffmpeg-n8.0.1/libavcodec/aarch64/videodsp.S`, `videodsp_init.c` | `internal/h264/edge_arm64.s`, `internal/h264/motion_comp.go`, and high-bit fallback in `internal/h264/motion_comp_high.go` | partial | The arm64 8-bit H.264 21- and 9-pixel scratch shapes have an exact fixed-width assembly row kernel; exhaustive clamped-pixel coverage includes 21x21, 9x9, and 9x17 shapes. Focused 21x21 rows move from roughly 58-82 ns/op to 31-42 ns/op, with 0 allocs/op. High-bit-depth and non-arm64 lanes retain the FFmpeg-shaped scalar rectangle-copy fallback. |

## Implementation Order

1. Reduce CABAC hot-call overhead and loop-filter cache construction; both
   remain larger than any unported leaf in the refreshed profile.
2. Finish arm64 width-4/2 qpel only when a representative vector proves a
   connected win; width-16/8 hot shapes are already NEON.
3. Continue 8-bit deblock coverage with any remaining perf-positive chroma
   shapes now that normal luma vertical/horizontal/MBAFF, luma intra
   vertical/horizontal, and the enabled chroma leaves have FFmpeg-shaped amd64
   assembly.
4. Extend high-bit deblock coverage after the remaining 8-bit hot leaves are
   verified.
5. Extend weighted prediction to remaining widths, biweight, and high-bit lanes.
6. Add IDCT/add-pixels 4x4, 8x8, DC-only, and clear variants after choosing the
   local block ABI.
7. Add intra prediction mode batches for intra-heavy workloads.
8. Prototype batched CABAC assembly only behind differential state and
   real-vector parity tests; a per-bin assembly call is unlikely to amortize its
   boundary cost.

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
| 1 | CABAC and loop-cache hot paths | n/a | partial | CABAC get/residual work remains about 15% of the refreshed profile and loop filtering about 16%. Continue evidence-gated direct-call/layout work; the compact loop-only motion/NNZ cache and direct residual specialization are verified safe points. |
| 2 | Luma qpel arm64 NEON | verified scalar asm | partial | Width-16/8 horizontal, vertical, odd/odd, center-22, and HV-blend shapes now use NEON and reduce the qpel family from about 21% to about 5% of `caba3`. Width-4/2 fractional leaves remain scalar assembly and need connected evidence before porting. |
| 1 | Loop filter 8-bit luma/chroma SIMD | partial | partial | arm64 now mirrors FFmpeg `ff_h264_v/h_loop_filter_luma_neon`, `ff_h264_v/h_loop_filter_chroma_neon`, `ff_h264_h_loop_filter_chroma422_neon`, `ff_h264_v/h_loop_filter_luma_intra_neon`, 4:2:0 `ff_h264_v/h_loop_filter_chroma_intra_neon`, `ff_h264_h_loop_filter_chroma422_intra_neon`, and the directly tested but undispatched `ff_h264_h_loop_filter_chroma_mbaff_intra_neon` shape with raw NEON instruction words behind Go ABI wrappers. The focused luma deblock benchmark improved from roughly 55-57 ns/op purego to about 11 ns/op vertical and 18 ns/op horizontal on Apple arm64, with 0 allocs/op. Focused chroma deblock medians are about 9.2 ns/op vertical and 13.9 ns/op horizontal in default builds versus about 16.5 and 16.2 ns/op in `purego`; the focused 4:2:2 horizontal row is about 20.4 ns/op default versus 27.1 ns/op `purego`; focused luma intra vertical/horizontal rows are about 10.0/17.4 ns/op default versus 58.0/58.2 ns/op `purego`; focused chroma intra vertical/horizontal rows are about 6.0/9.3 ns/op default versus 12.2/12.1 ns/op `purego`; focused chroma intra 4:2:2 is about 10.4 ns/op default versus 23.5 ns/op `purego`; and 4:2:0 chroma intra MBAFF stays scalar in default builds, around 7.4 ns/op versus about 7.0 ns/op in `purego`, all 0 allocs/op. The amd64 normal luma vertical leaf now mirrors FFmpeg `ff_deblock_v_luma_8_sse2`, the horizontal leaf mirrors FFmpeg `ff_deblock_h_luma_8_sse2`, and the MBAFF leaf mirrors FFmpeg `ff_deblock_h_luma_mbaff_8_sse2`. Focused local virtualized-amd64 luma medians are about 20 ns/op vertical, 48-49 ns/op horizontal, and 37.7 ns/op MBAFF in default builds versus about 78 ns/op vertical, 79-80 ns/op horizontal, and 40.3 ns/op MBAFF in `purego`, all 0 allocs/op. The amd64 luma intra vertical/horizontal leaves now mirror FFmpeg's SSE2 vertical core plus horizontal 16x8 transpose wrapper; focused medians are about 32.4/47.8 ns/op default versus about 69.6/69.0 ns/op `purego`, all 0 allocs/op. The enabled amd64 chroma SSE2 leaves cover normal vertical, normal 4:2:2 horizontal, intra vertical, and intra 4:2:2 horizontal with focused medians about 16.0/25.6/10.0/21.4 ns/op default versus about 18.4/32.1/15.8/28.6 ns/op `purego`, all 0 allocs/op; normal 4:2:0 horizontal and intra horizontal are implemented but stay scalar on amd64 because the SSE2 wrappers did not beat scalar. The clean amd64 `caba3-sva-b` real-vector row is 157.1 ms/sample default versus 178.7 ms/sample `purego` on clean commit `08fbd927f449d03384d5841939c147c162024a92`; the installed FFmpeg is arm64-only on this host, so paired FFmpeg arm64 CLI lanes are tracked separately and not presented as x86 ISA-fair. The 8-bit high-422 `frext-hi422fr10-sony-b` real-vector benchmark passes raw-MD5 and the Go zero-allocation gate in both default and `purego` lanes, with default partial asm about 224.9 ms/sample versus about 266.6 ms/sample for `purego` in the sampled run. The Apple arm64 `caba3` real-vector benchmark also passes on `d0272d01abf3f7e500443fa76c30732719a25a80`, with default partial asm about 104.8 ms/sample versus about 141.4 ms/sample for `purego`; before today's deblock work, the same default lane was about 122.6 ms/sample. ABI remains `*uint8` pixels, native signed `int` stride, `int32` alpha/beta, and `*int8` tc0 where the upstream prototype has tc0. |
| 2 | Loop filter intra/MBAFF/high | partial | partial | amd64 now has 8-bit luma intra vertical/horizontal and normal luma MBAFF SSE2 coverage; arm64 luma MBAFF remains scalar because the pinned arm64 init has no separate luma MBAFF symbol. High10 arm64 chroma normal vertical/horizontal/4:2:2 now mirrors `ff_h264_v/h_loop_filter_chroma_neon_10` and `ff_h264_h_loop_filter_chroma422_neon_10`, gated to `bitDepth == 10`; focused medians are about 9.4/14.5/20.5 ns/op default versus 16.4/16.6/27.5 ns/op `purego`, all 0 allocs/op. The 10-bit `frext-hi422fr13-sony-b` real-vector benchmark passes raw-MD5 and the Go zero-allocation gate with default partial asm about 81.3 ms/sample versus about 96.6 ms/sample for `purego`. High10 arm64 chroma intra vertical/horizontal/4:2:2/4:2:2-MBAFF now mirrors `ff_h264_v/h_loop_filter_chroma_intra_neon_10`, `ff_h264_h_loop_filter_chroma_intra_neon_10`, and `ff_h264_h_loop_filter_chroma422_intra_neon_10`; focused medians are about 6.2/10.3/11.1/10.3 ns/op default versus 14.2/14.2/24.7/14.3 ns/op `purego`, all 0 allocs/op. The direct High10 4:2:0 chroma intra MBAFF leaf is mapped but not dispatched because it did not beat scalar; both default and `purego` stay around 9.0 ns/op there. The intra-heavy 10-bit `frext-pph422i1-panasonic-a` real-vector benchmark passes raw-MD5 and the Go zero-allocation gate with default partial asm about 4385 ms/sample versus about 4685 ms/sample for `purego`, still about 4.1x slower than FFmpeg pure-C and 4.6x slower than FFmpeg native. Cover luma high-bit, high-bit luma intra, and 12/14-bit scalar-only lanes later. Keep high-bit assembly boundaries byte-pointer/byte-stride shaped where mirroring FFmpeg. |
| 3 | VideoDSP edge emulation and prefetch | todo | partial | Arm64 8-bit fixed 21/9-pixel edge rows are assembly-backed and oracle-covered. High-bit-depth and other architectures remain scalar; add more only when profiles justify them. |
| 4 | Weighted prediction 8/high | todo | partial | Arm64 width-16 even-height weighting is NEON and parity-covered. Extend widths, biweight, and high-bit-depth only with weighted-vector evidence. |
| 5 | IDCT/add-pixels 4x4/8x8/DC | todo | todo | Must choose the local ABI first because FFmpeg uses `int16_t *block` while the current scalar code uses `[]int32`. Verify block clearing and transform-bypass exclusions. |
| 6 | Intra prediction 4x4/8x8/16x16/chroma | todo | todo | Many small kernels; enable in batches by mode only when intra-heavy profiles justify it. |
| functional | Qpel 8-bit sizes 16/8/4/2 all positions | verified | verified mixed SIMD/scalar asm | All qpel positions are assembly-backed. Arm64 width-16/8 hot fractional shapes are NEON; width-4/2 fractional shapes remain scalar assembly. Kernel seam retains separate native-width strides and 32-bit selectors. |
| done | Chroma MC 8-bit width 8/4/2 put and avg | verified | verified | Widths 8/4/2/1 copy and fractional x/y put/avg are implemented and oracle-covered. Arm64 fractional width-8/4/2 uses NEON; width-1 remains scalar assembly. |
| done | Qpel high-bit-depth sizes 16/8/4/2 | verified | verified | `mc00` copy/avg, one-axis `mc10/20/30/01/02/03`, center HV `mc22`, odd/odd HV `mc11/mc31/mc13/mc33`, and HV-blend `mc21/mc12/mc32/mc23` filters are implemented in `internal/h264/qpel_high_amd64.s` and `internal/h264/qpel_high_arm64.s` through `h264QpelMCHigh00ASM`, `h264QpelMCHighX0ASM`, `h264QpelMCHigh0YASM`, `h264QpelMCHigh22ASM`, `h264QpelMCHighHVXYASM`, and `h264QpelMCHighHVBlendASM`, which accept `*uint8` buffers and native byte strides after the Go `[]uint16` wrapper validates sample geometry. Verified against high-bit dispatch parity with separate strides, `TestH264QpelUpstreamOracle` default and `purego`, cross-arch symbols, focused High10 qpel `-benchmem`, full default and `purego` test gates, the public benchmark canary, and high-bit real-vector decode. |
| done | Chroma MC high-bit-depth width 8/4/2/1 | verified | verified | Implemented in `internal/h264/chroma_high_amd64.s` and `internal/h264/chroma_high_arm64.s`. `h264ChromaMCHighASM` accepts `*uint8` buffers, native byte strides, 32-bit C-int selectors/weights, and a native byte `step`; Go dispatch validates the `[]uint16` public/internal helper shape and narrows to FFmpeg-shaped byte pointers. Verified against high-bit dispatch parity with separate strides, `TestH264ChromaMCUpstreamOracle` default and `purego`, cross-arch symbols, focused high-bit chroma `-benchmem`, and real-vector `frext-hi422fr13-sony-b` in default and `purego` lanes. |
