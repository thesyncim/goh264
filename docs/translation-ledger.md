# Translation Ledger

| Upstream area | Local area | State | Proof |
| --- | --- | --- | --- |
| NAL/extradata/SPS/PPS/SEI | `nal.go`, `sps.go`, `pps.go`, `sei.go` | translated subset | unit, corpus |
| slice headers/ref lists/DPB | `slice.go`, `simple_dpb.go` | partial | unit, MD5 |
| CAVLC/CABAC macroblocks | `cavlc*.go`, `cabac*.go` | partial | unit, MD5 |
| prediction/MC/IDCT/deblock | `pred*.go`, `motion_comp*.go`, `idct.go`, `loop_filter.go` | partial; MBAFF direct field maps + field-ref deblock | unit, C oracle, MD5 |
| public decoder/output | `decoder.go`, `simple_decode.go` | partial | corpus, FATE, FFmpeg oracle fixtures including complete FFmpeg FRext FATE row coverage, public High10/High422 intra conformance, monochrome-to-yuv420p output, High12 no-residual/P-skip/P16x16, High14 no-residual/P-skip/P16x16, internal High12 weighted P-skip and weighted P16x16/partitioned P plus no-deblock/mode-1 deblock scope with unweighted 4:2:2/4:4:4 I/P chroma deblock, High10 weighted 4:2:2/4:4:4 luma-only/chroma P frame deblock modes 0/1, 4:2:0 I/P slice-boundary mode-2 deblock, High10/High12 unweighted 4:2:2/4:4:4 I/P chroma slice-boundary mode-2 deblock, High10 frame-MBAFF field-coded CAVLC IntraPCM entropy/reconstruct pairing, public High10/High422 field-coded frame-MBAFF deblock rows, and High12/High14 luma/chroma residual |
| benchmark/oracle harness | `decoder_corpus_test.go`, `cmd/goh264bench` | replaced | FATE, smoke |

Known deviations are in `testdata/h264/realvectors/failures.jsonl` when present.
