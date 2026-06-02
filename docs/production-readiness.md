# Production Readiness

Harness-first status:

```sh
scripts/h264-real-vector-strict.sh
scripts/h264-real-vector-red-queue.sh   # exits non-zero while known-red rows remain
GOH264_REAL_VECTOR_STRICT=1 GOH264_CORPUS_FETCH=1 go test . -run TestH264RealVectorStrictOracle
GOH264_REAL_VECTOR_RED=1 GOH264_CORPUS_FETCH=1 go test . -run TestH264RealVectorKnownRedStrict
GOH264_REAL_VECTOR_RED_QUEUE=1 GOH264_CORPUS_FETCH=1 go test . -run TestH264RealVectorRedQueue
GOH264_REAL_VECTOR_MATRIX=1 GOH264_CORPUS_FETCH=1 go test . -run TestH264RealVectorFailureMatrix
scripts/h264-red-vector.sh mbaff        # exits non-zero at first divergent raw-MD5 frame
go run ./cmd/goh264bench -manifest testdata/h264/realvectors/manifest.jsonl -filter canl4 -iters 10 -repeats 5 -warmup 2 -ffmpeg -ffmpeg-threads 1 -strict-pix-fmt -json
```

Pending: JVT/FATE bulk manifests, allocation gates, benchstat/profile output,
larger performance corpus, and in-process libavcodec benchmark baseline.
