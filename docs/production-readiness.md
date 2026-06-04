# Production Readiness

Harness-first status:

```sh
GOH264_REAL_VECTOR_FAILURES=1 GOH264_CORPUS_FETCH=1 go test ./tests -run TestH264RealVectorFailureLedgerFreshness
GOH264_REAL_VECTOR_MATRIX=1 GOH264_CORPUS_FETCH=1 go test ./tests -run TestH264RealVectorFailureMatrix
scripts/h264-real-vector-strict.sh      # strict green public-vector oracle
scripts/h264-real-vector-red-queue.sh   # exits non-zero while known-red rows remain
scripts/h264-real-vector-red-each.sh    # per-row red queue report
scripts/h264-real-vector-upstream-audit.sh # pinned FFmpeg H.264 FATE coverage
scripts/h264-real-vector-bench.sh canl4 # set GOH264_BENCH_FFMPEG=1 GOH264_BENCH_FAIR_CPU_LANES=1 for pure C vs pure Go and native C+asm vs Go+asm lanes
GOH264_REAL_VECTOR_STRICT=1 GOH264_CORPUS_FETCH=1 go test ./tests -run TestH264RealVectorStrictOracle
GOH264_REAL_VECTOR_RED=1 GOH264_CORPUS_FETCH=1 go test ./tests -run TestH264RealVectorKnownRedStrict
GOH264_REAL_VECTOR_RED_QUEUE=1 GOH264_CORPUS_FETCH=1 go test ./tests -run TestH264RealVectorRedQueue
GOH264_REAL_VECTOR_RAWDIFF=1 GOH264_CORPUS_FILTER=mbaff GOH264_CORPUS_FETCH=1 go test ./tests -run TestH264RealVectorRawDiffDiagnostics
scripts/h264-red-vector.sh mbaff        # exits non-zero at first divergent raw byte
go run ./cmd/goh264bench -manifest testdata/h264/realvectors/manifest.jsonl -filter canl4 -iters 10 -repeats 5 -warmup 2 -ffmpeg -fair-cpu-lanes -ffmpeg-threads 1 -strict-pix-fmt -json
```

`scripts/h264-real-vector-strict.sh` runs the green public-vector set and logs
the known-red ids excluded from strict mode. Use
`GOH264_REAL_VECTOR_FAILURES=1` or `GOH264_REAL_VECTOR_MATRIX=1` for the gates
that execute and verify the known-red rows.

Benchmark JSON reports selected/green/known-red counts, backend kind, CPU flags,
comparison lane, oracle `quality_status`, and FFmpeg-vs-Go
`peer_quality_status`. Use `-fair-cpu-lanes` for both `pure-c-vs-pure-go` and
`native-c+asm-vs-go+asm`; extracted container rows require FFmpeg on `PATH`
when the cache does not already contain the `.h264-annexb` derived stream.
Result `backend_kind` records the backend actually
measured, so current no-asm Go builds still report `go-pure`. Pending: bulk
allocation gates, benchstat/profile output, larger performance corpus, and
in-process libavcodec baseline.
