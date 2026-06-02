# goh264

Source-shaped Go port of FFmpeg `libavcodec` H.264 decoder code. Decoder only.

Upstream: FFmpeg `n8.0.1` (`894da5ca7d742e4429ffb2af534fcda0103ef593`).

Current public-vector gate: 26 selected FATE vectors, 13 green, 13 known-red.
Known failures are explicit in `testdata/h264/realvectors/failures.jsonl`.

```sh
go test ./...
GOH264_REAL_VECTOR_MATRIX=1 GOH264_CORPUS_FETCH=1 go test . -run TestH264RealVectorFailureMatrix
GOH264_REAL_VECTOR_FAILURES=1 GOH264_CORPUS_FETCH=1 go test . -run TestH264RealVectorFailureLedgerFreshness
GOH264_REAL_VECTOR_FRAMEMD5=1 GOH264_CORPUS_FILTER=canl4 GOH264_CORPUS_FETCH=1 go test . -run TestH264RealVectorFrameMD5Diagnostics
go run ./cmd/goh264bench -manifest testdata/h264/realvectors/manifest.jsonl -filter canl4 -iters 1 -repeats 1 -warmup 0 -ffmpeg -json
```

Compact state: `docs/source-truth.md`, `docs/translation-ledger.md`.
