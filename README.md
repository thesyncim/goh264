# goh264

Source-shaped Go port of FFmpeg `libavcodec` H.264 decoder code. Decoder only.

Upstream: FFmpeg `n8.0.1` (`894da5ca7d742e4429ffb2af534fcda0103ef593`).

Current public-vector gate: 120 selected FATE vectors, 120 green, 0 known-red.
Known failures are explicit in `testdata/h264/realvectors/failures.jsonl` when present.

```sh
go test ./...
scripts/h264-real-vector-strict.sh
scripts/h264-real-vector-red-queue.sh   # exits non-zero while known-red rows remain
scripts/h264-real-vector-red-each.sh    # per-known-red logs + first divergence TSV
scripts/h264-real-vector-bench.sh canl4 # JSON: selected/green/known-red/skipped + timings
GOH264_REAL_VECTOR_STRICT=1 GOH264_CORPUS_FETCH=1 go test . -run TestH264RealVectorStrictOracle
GOH264_REAL_VECTOR_RED=1 GOH264_CORPUS_FETCH=1 go test . -run TestH264RealVectorKnownRedStrict
GOH264_REAL_VECTOR_RED_QUEUE=1 GOH264_CORPUS_FETCH=1 go test . -run TestH264RealVectorRedQueue
GOH264_REAL_VECTOR_MATRIX=1 GOH264_CORPUS_FETCH=1 go test . -run TestH264RealVectorFailureMatrix
GOH264_REAL_VECTOR_FAILURES=1 GOH264_CORPUS_FETCH=1 go test . -run TestH264RealVectorFailureLedgerFreshness
GOH264_REAL_VECTOR_RAWDIFF=1 GOH264_CORPUS_FILTER=mbaff GOH264_CORPUS_FETCH=1 go test . -run TestH264RealVectorRawDiffDiagnostics
GOH264_REAL_VECTOR_FRAMEMD5=1 GOH264_CORPUS_FILTER=mbaff GOH264_CORPUS_FETCH=1 go test . -run TestH264RealVectorFrameMD5Diagnostics
scripts/h264-red-vector.sh mbaff        # exits non-zero at first divergent raw byte
GOH264_BENCH_FFMPEG=1 GOH264_BENCH_FAIR_CPU_LANES=1 scripts/h264-real-vector-bench.sh canl4 # oracle + peer raw-MD5 quality in pure C vs pure Go and native C+asm vs Go+asm lanes
go run ./cmd/goh264bench -manifest testdata/h264/realvectors/manifest.jsonl -filter canl4 -iters 1 -repeats 1 -warmup 0 -ffmpeg -fair-cpu-lanes -json
```

Compact state: `docs/source-truth.md`, `docs/translation-ledger.md`.
