# Translation Ledger

Status values: `translated`, `partially translated`, `stubbed`, `replaced`, or
`intentionally unsupported`. Fixture hashes live in the corpus manifest.

| Upstream | Local | Status | Proof | Deviation |
| --- | --- | --- | --- | --- |
| packet/extradata intake | `nal.go`, `simple_decode.go`, `decoder.go` | translated | unit, corpus, MD5 | H.264 decoder-only |
| bit/golomb readers, tables, SPS/PPS/VUI | `bitreader.go`, `golomb.go`, `data.go`, `sps.go`, `pps.go` | translated | unit, oracle | FMO guarded |
| SEI and frame side-data subset | `sei.go`, `decoder.go` | translated | unit, oracle | late-SEI/MBAFF effects pending |
| slice headers, ref lists, pred weights | `slice.go` | translated | unit, MD5 | field copies pending |
| DPB/reorder/direct/implicit B weights | `simple_dpb.go`, `direct_motion.go` | partially translated | unit, MD5 | progressive frame subset |
| CAVLC/CABAC macroblock handoff | `cavlc*.go`, `cabac*.go` | partially translated | unit, MD5 | field/overread policy pending |
| MB caches, prediction, MC, IDCT, deblock DSP | `macroblock_*.go`, `pred*.go`, `motion_comp*.go`, `idct.go`, `dsp.go` | translated | unit, C oracle | SIMD pending |
| frame reconstruction and slice loop | `reconstruct*.go`, `slice_decode.go` | partially translated | unit, corpus, MD5 | fixture-gated public surface |
| public decoder/output | `decoder.go`, `simple_decode.go` | partially translated | corpus, MD5 | full delayed-output pending |
| High10 weighted partitioned P | weighted P fixtures/tests | translated | macroblock proof, corpus, MD5 | frame-only 4:2:0 deblock-disabled only |
| High10 CABAC slice-boundary IDR/P | slice-boundary fixture/tests | translated | corpus, FFmpeg MD5 | frame-only 4:2:0 IDR/P only |
| High10 CAVLC direct-sub residual B | direct-sub residual fixture/tests | translated | macroblock proof, corpus, FFmpeg MD5 | temporal B8x8 4:2:0 only |
| corpus/benchmark harness | `decoder_corpus_test.go`, `cmd/goh264bench`, manifest | replaced | corpus, bench smoke | FATE/JVT import and in-process baseline pending |
| remaining H.264 decoder behavior | guards | stubbed | none | P IntraPCM, MBAFF, FMO, threading, broad ER/SIMD, unproved high lanes |
