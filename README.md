# goh264

Source-shaped Go port of FFmpeg `libavcodec`'s H.264 decoder path.
Scope is decoder-only.

## State

- Upstream: FFmpeg `n8.0.1`
  `894da5ca7d742e4429ffb2af534fcda0103ef593`.
- Decodes the fixture-proved progressive Annex B/AVC subset: 8-bit IDR/P/B
  plus selected High10/High12 lanes.
- Latest lane: High10 CAVLC temporal B8x8 direct-sub with visible luma
  residual, FFmpeg rawvideo-MD5 proved.
- Still not a general libavcodec replacement: field/MBAFF, FMO, broad error
  resilience, threading/SIMD, full delayed output, broad 12/14-bit streams, and
  unproved high-bit-depth feature combinations remain guarded.
- Public FATE vectors are wired as an oracle gate. `GOH264_ORACLE=1` runs them
  and is intentionally red until `testdata/h264/realvectors/failures.jsonl` is
  empty.

Detailed source state: [docs/source-truth.md](docs/source-truth.md).
File/function ledger: [docs/translation-ledger.md](docs/translation-ledger.md).
Fixture hashes/oracles: `testdata/h264/corpus/manifest.jsonl`.

## Verify

```sh
go test ./...
GOH264_ORACLE=1 GOH264_CORPUS_FETCH=1 go test ./...
go test . -run TestH264CorpusManifest
GOH264_REAL_VECTORS=1 GOH264_CORPUS_FETCH=1 GOH264_CORPUS_FILTER=canl4 go test . -run TestH264RealVectorManifest
GOH264_REAL_VECTOR_FAILURES=1 GOH264_CORPUS_FETCH=1 go test . -run TestH264RealVectorFailureLedgerFreshness
```

Use `GOH264_CORPUS_FILTER=frext3`, `hi422`, `hcamff1`, or any feature tag to
narrow a red public-vector lane.

## Benchmark

```sh
go run ./cmd/goh264bench -manifest testdata/h264/corpus/manifest.jsonl -iters 5 -repeats 5 -ffmpeg -json
go run ./cmd/goh264bench -manifest testdata/h264/realvectors/manifest.jsonl -filter canl4 -iters 1 -repeats 1 -warmup 0 -ffmpeg -json
```

The FFmpeg comparator is CLI-based, useful for parity and rough timing. In-process
libavcodec benchmarking is still pending.
