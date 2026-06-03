# Translation Ledger

| Upstream area | Local area | State | Proof |
| --- | --- | --- | --- |
| NAL/extradata/SPS/PPS/SEI | `nal.go`, `sps.go`, `pps.go`, `sei.go` | translated subset | unit, corpus |
| slice headers/ref lists/DPB | `slice.go`, `simple_dpb.go` | partial | unit, MD5 |
| CAVLC/CABAC macroblocks | `cavlc*.go`, `cabac*.go` | partial | unit, MD5 |
| prediction/MC/IDCT/deblock | `pred*.go`, `motion_comp*.go`, `idct.go`, `loop_filter.go` | partial; MBAFF direct field maps + field-ref deblock | unit, C oracle, MD5 |
| public decoder/output | `decoder.go`, `simple_decode.go` | partial | corpus, FATE, FFmpeg oracle fixtures including High12/High14 luma/chroma residual |
| benchmark/oracle harness | `decoder_corpus_test.go`, `cmd/goh264bench` | replaced | FATE, smoke |

Known deviations are in `testdata/h264/realvectors/failures.jsonl` when present.
