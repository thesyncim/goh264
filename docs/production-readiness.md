# Production Readiness

Harness-first status:

```sh
GOH264_REAL_VECTOR_MATRIX=1 GOH264_CORPUS_FETCH=1 go test . -run TestH264RealVectorFailureMatrix
go run ./cmd/goh264bench -manifest testdata/h264/realvectors/manifest.jsonl -filter canl4 -iters 10 -repeats 5 -warmup 2 -ffmpeg -ffmpeg-threads 1 -strict-pix-fmt -json
```

Pending: JVT/FATE bulk manifests, allocation gates, benchstat/profile output,
larger performance corpus, and in-process libavcodec benchmark baseline.
