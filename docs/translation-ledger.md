# Translation Ledger

Status values: `translated`, `partially translated`, `stubbed`, `replaced`,
or `intentionally unsupported`. Fixture hashes and packet surfaces live in
`testdata/h264/corpus/manifest.jsonl`.

| Upstream path/function | Local path/function | Status | Parity proof | Known deviations |
| --- | --- | --- | --- | --- |
| `h2645_parse.c`, `h264_parse.c`, `h264dec.c` packet/extradata intake | `internal/h264/nal.go`, `internal/h264/simple_decode.go`, `decoder.go` packet APIs | translated | unit, corpus, rawvideo MD5 | H.264 decoder-only; non-video packet side data is out of scope. |
| `get_bits.h`, `golomb.h` | `internal/h264/bitreader.go`, `internal/h264/golomb.go` | translated | unit | Fast cached-reader optimizations deferred. |
| `h264data.c`, `h264_parse.h` constants/tables | `internal/h264/data.go` | translated | unit | H.264-relevant tables only. |
| `h264_ps.c`, `h2645_vui.c` SPS/PPS/VUI | `internal/h264/sps.go`, `internal/h264/pps.go` | translated | unit, ffprobe/oracle fixtures | FMO intentionally unsupported. |
| `h264_sei.c`, selected `h2645_sei.c`, frame side-data export | `internal/h264/sei.go`, `decoder.go` side-data projection | translated | unit, native layout oracle, rawvideo fixtures | Late-SEI and field/MBAFF timing effects pending. |
| `h264_slice.c` slice header/ref-count/ref-list syntax | `internal/h264/slice.go` | translated | unit | Field/MBAFF behavior pending. |
| `h264_refs.c`, `h264_slice.c` simple DPB/reorder/implicit B weights | `internal/h264/simple_dpb.go` | partially translated | unit, rawvideo MD5 | Progressive frame subset only; gap/error-resilience and field ownership pending. |
| `ff_h264_pred_weight_table` | `internal/h264/slice.go` `predWeightTable` | translated | unit, rawvideo MD5 | MBAFF field copies pending. |
| CAVLC/CABAC entropy and macroblock handoff | `internal/h264/cavlc*.go`, `internal/h264/cabac*.go` | partially translated | unit, rawvideo MD5 | MBAFF/field and full overread/error policy pending. |
| Macroblock caches, neighbors, write-back | `internal/h264/macroblock_*.go`, `internal/h264/slice_macroblock.go` | translated | unit, fixture syntax tests | FMO/MBAFF remapping pending. |
| Direct motion | `internal/h264/direct_motion.go` | partially translated | unit, rawvideo MD5 | Progressive frame direct motion only. |
| Prediction, IDCT, qpel, chroma MC, weighting, deblocking DSP | `internal/h264/pred*.go`, `idct.go`, `qpel.go`, `chroma.go`, `dsp.go` | translated | unit, compiled C oracle | SIMD dispatch pending. |
| Frame-MB motion and reconstruction | `internal/h264/motion_comp*.go`, `reconstruct*.go` | partially translated | unit, compiled C oracle, rawvideo MD5 | Public surface remains fixture-gated by profile/chroma/depth/MB shape. |
| Simple frame slice loop | `internal/h264/slice_decode.go` | partially translated | unit, corpus, rawvideo MD5 | Row-threaded deblocking, error resilience, threading, field/MBAFF pending. |
| Public simple decoder/output | `internal/h264/simple_decode.go`, `decoder.go` | partially translated | corpus, FFmpeg rawvideo MD5 | Full libavcodec delayed-output semantics beyond the simple progressive path pending. |
| High10 weighted partitioned P safe point | `decoder_high10_partitioned_p_test.go`, `internal/h264/simple_decode_high_weighted_partitioned_p_fixture_test.go`, `testdata/h264/high10_weighted_partitioned_p_*.h264` | translated | internal macroblock-table proof, corpus MD5, FFmpeg rawvideo MD5 | Only frame-only 4:2:0 deblock-disabled CAVLC/CABAC P16x8/P8x16/P8x8 with explicit P weights. |
| Corpus/oracle/benchmark harness | `decoder_corpus_test.go`, `cmd/goh264bench`, `testdata/h264/corpus/manifest.jsonl` | replaced | corpus MD5, benchmark smoke | Full FATE/JVT import and in-process native benchmark baseline pending. |
| Remaining `h264_slice.c`/`h264_mb.c`/`h264dec.c` behavior | guarded public paths | stubbed | not yet proved | P IntraPCM, P 8x8-DCT intra, deblock-enabled weighted partitioned P, mixed direct/explicit B8x8, residual-bearing direct-sub B, broader 12/14-bit public streams, field/MBAFF, FMO, threading, SIMD, full error resilience. |
