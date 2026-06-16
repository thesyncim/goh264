# Translation Ledger

This ledger tracks the decoder-facing parity boundary against the pinned FFmpeg
`n8.0.1` H.264 decoder path.

| Upstream Area | Local Area | State | Proof |
| --- | --- | --- | --- |
| NAL, extradata, SPS, PPS, SEI | `internal/h264/nal.go`, `internal/h264/sps.go`, `internal/h264/pps.go`, `internal/h264/sei.go` | Translated subset | Unit tests, avcC rejection tests, SPS/PPS boundary tests, SEI side-data tests |
| Slice headers, reference lists, DPB | `internal/h264/slice.go`, `internal/h264/simple_dpb.go` | Translated subset | Unit tests, delayed-output guards, configured-AVC and Annex B recovery tests |
| CAVLC and CABAC syntax | `internal/h264/cavlc*.go`, `internal/h264/cabac*.go` | Translated subset | Unit tests, fixture decode tests, FFmpeg raw-MD5 rows |
| Prediction, motion compensation, IDCT, deblock, reconstruction | `internal/h264/pred*.go`, `internal/h264/motion_comp*.go`, `internal/h264/idct.go`, `internal/h264/loop_filter.go`, `internal/h264/reconstruct*.go` | Translated subset | Unit tests, oracle tests, high-bit-depth and field/MBAFF fixture rows |
| Public decoder API and output ownership | `decoder.go`, `internal/h264/simple_decode.go`, `tests/decoder*.go` | Guarded subset | Public-surface tests for Annex B, AVC, avcC, packets, delayed flush, raw output, side data, clone/validate/append helpers, and malformed-input recovery |
| Public-vector and benchmark harness | `tests/decoder_corpus_test.go`, `tests/decoder_fuzz_test.go`, `cmd/goh264bench`, `scripts/` | Active | Strict real-vector oracle, fuzz smoke, allocation canary, benchmark JSON/profiling scripts |

## Unsupported At The Current Boundary

FMO, 11/13-bit luma depths, `chroma_format_idc > 3`, separate color planes, and
mixed chroma/luma bit depths are intentionally outside the admitted parity
boundary. Broader field/MBAFF/PIC-AFF motion behavior and damaged-slice edges
remain areas for focused expansion.
