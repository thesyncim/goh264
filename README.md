# goh264

`goh264` is a source-shaped Go port of FFmpeg `libavcodec`'s H.264 decoder
path. The scope is decoder-only: no encoder, muxer, filter, hardware backend,
or unrelated codec code unless the H.264 decoder directly depends on it.

## Current Safe Point

- Upstream: FFmpeg `n8.0.1`, commit
  `894da5ca7d742e4429ffb2af534fcda0103ef593`.
- Packet/API surface: Annex B, AVC/NALFF, `avcC`, packet auto-detection,
  configured sample decode, delayed flush, and the documented packet/frame
  side-data subset.
- 8-bit surface: progressive frame-picture IDR/P/B simple decode with
  CAVLC/CABAC macroblock handoff, DPB/reorder, weighted/direct motion,
  reconstruction, and fixture-proved deblocking.
- High-bit-depth internals: 9/10/12/14-bit scalar kernels and uint16
  frame/ref/output storage.
- Public High10/High12 decode is fixture-gated. Proved High10 lanes include
  IDR/I, P16x16 residual/weighted, mixed-P intra, unweighted and explicit
  weighted partitioned P16x8/P8x16/P8x8, selected B/direct/implicit-weighted
  lanes, selected deblock-enabled IDR/P/B lanes, and one CAVLC slice-boundary
  IDR/P lane. Proved High12 is the narrow yuv420p12le CAVLC IDR/I IntraPCM row.
- Still guarded: P IntraPCM, P 8x8-DCT intra, deblock-enabled weighted
  partitioned P, mixed direct/explicit B8x8, residual direct-sub B, broader
  12/14-bit public streams, field/MBAFF, FMO, broad error resilience,
  threading/SIMD, and full libavcodec delayed-output behavior.

The detailed lane list lives in [docs/source-truth.md](docs/source-truth.md).
Fixture hashes, frame counts, packet surfaces, and oracle rawvideo MD5s live in
`testdata/h264/corpus/manifest.jsonl`.

## Verify

```sh
go test ./...
GOH264_ORACLE=1 go test ./...
go test . -run TestH264CorpusManifest
```

External manifests can be supplied with `GOH264_CORPUS_MANIFEST` or the
path-list form `GOH264_CORPUS_MANIFESTS`.

## Benchmark

```sh
go run ./cmd/goh264bench -input testdata/h264/cavlc_b8x8_spatial_direct_sub.h264 -iters 5 -repeats 5 -ffmpeg -json
go run ./cmd/goh264bench -manifest testdata/h264/corpus/manifest.jsonl -iters 5 -repeats 5 -ffmpeg -json
```

The benchmark report includes host/VCS metadata, raw pixel format, timing
scope, parity checks, and per-repeat stats. The current FFmpeg comparator is a
CLI baseline, so it is a correctness and rough timing reference rather than an
in-process libavcodec throughput number.

## Port Ledger

Fetch upstream into the ignored local cache:

```sh
scripts/fetch-upstream.sh
```

File-by-file status lives in [docs/translation-ledger.md](docs/translation-ledger.md).
Corpus and benchmark readiness notes live in
[docs/production-readiness.md](docs/production-readiness.md).
