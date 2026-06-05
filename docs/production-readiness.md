# Production Readiness

Harness-first status:

```sh
GOH264_REAL_VECTOR_FAILURES=1 GOH264_CORPUS_FETCH=1 go test ./tests -run TestH264RealVectorFailureLedgerFreshness
GOH264_REAL_VECTOR_MATRIX=1 GOH264_CORPUS_FETCH=1 go test ./tests -run TestH264RealVectorFailureMatrix
scripts/h264-real-vector-strict.sh      # strict green public-vector oracle
scripts/h264-real-vector-red-queue.sh   # exits non-zero only while known-red rows remain
scripts/h264-real-vector-red-each.sh    # per-row red queue report when the ledger is populated
scripts/h264-real-vector-upstream-audit.sh # pinned FFmpeg H.264 FATE coverage
scripts/h264-real-vector-bench.sh canl4 # set GOH264_BENCH_FFMPEG=1 GOH264_BENCH_FAIR_CPU_LANES=1 for pure C vs pure Go and native C+asm vs Go+asm lanes
GOH264_REAL_VECTOR_STRICT=1 GOH264_CORPUS_FETCH=1 go test ./tests -run TestH264RealVectorStrictOracle
GOH264_REAL_VECTOR_RED=1 GOH264_CORPUS_FETCH=1 go test ./tests -run TestH264RealVectorKnownRedStrict
GOH264_REAL_VECTOR_RED_QUEUE=1 GOH264_CORPUS_FETCH=1 go test ./tests -run TestH264RealVectorRedQueue
GOH264_REAL_VECTOR_RAWDIFF=1 GOH264_CORPUS_FILTER=mbaff GOH264_CORPUS_FETCH=1 go test ./tests -run TestH264RealVectorRawDiffDiagnostics
scripts/h264-red-vector.sh mbaff        # exits non-zero at first divergent raw byte
go run ./cmd/goh264bench -manifest testdata/h264/realvectors/manifest.jsonl -filter canl4 -iters 10 -repeats 5 -warmup 2 -ffmpeg -fair-cpu-lanes -ffmpeg-threads 1 -strict-pix-fmt -json
```

`scripts/h264-real-vector-strict.sh` runs the green public-vector oracle set,
including expected decode-error rows, and excludes only rows currently listed
in the failure ledger. Use
`GOH264_REAL_VECTOR_FAILURES=1` or `GOH264_REAL_VECTOR_MATRIX=1` for the gates
that execute and verify known-red rows when present.

The checked-in public-vector inventory at
`testdata/h264/realvectors/upstream-inventory.jsonl` currently imports 226
public H.264 refs: 224 generated from pinned FFmpeg `n8.0.1` FATE makefiles and
2 auxiliary public fate-suite H.264/LCEVC container samples. Normal
`go test ./tests` requires every imported ref to be represented by the manifest
or by `testdata/h264/realvectors/exclusions.jsonl` through
`TestH264DecoderTDDContractClassifiesEveryImportedPublicVector`; any future
failing decoder-facing row belongs in `failures.jsonl` with a current failure
signature until it is fixed. The upstream-audit script also verifies the 224
generated FATE rows against the pinned FFmpeg source.

Benchmark JSON reports selected/green/known-red counts, backend kind, CPU flags,
comparison lane, oracle `quality_status`, Go allocation totals plus
per-iteration/per-frame allocation rates, and FFmpeg-vs-Go
`peer_quality_status`. Diagnostic mode includes expected decode-error rows as
oracle rows and requires the observed decoder error to contain `expected_error`.
Use `-fair-cpu-lanes` for both `pure-c-vs-pure-go` and
`native-c+asm-vs-go+asm`; extracted container rows require FFmpeg on `PATH`
when the cache does not already contain the `.h264-annexb` derived stream.
Result `backend_kind` records the backend actually
measured, so current no-asm Go builds still report `go-pure`.

Allocation evidence: `tests/decoder_high_output_test.go` guards
`Frame.AppendRawYUV`, `Frame.AppendRawYUVBytesLE`, and `Frame.AppendRawYUV16`
with exact-capacity caller-owned buffers and requires zero steady-state
allocations for 8-bit and high-bit-depth output paths. `cmd/goh264bench`
records Go benchmark allocation totals and reports `alloc_bytes_per_iter`,
`allocs_per_iter`, `alloc_bytes_per_frame`, and `allocs_per_frame` for each
timed Go lane. Pending: bulk decode allocation budget gates,
benchstat/profile output, larger performance corpus, and in-process libavcodec
baseline.
