# goh264

`goh264` is a source-shaped Go port of FFmpeg `libavcodec`'s H.264 decoder path.

The port is intentionally decoder-only. Encoder, muxer, filter, and unrelated codec code are out of scope unless an H.264 decoder file directly depends on a shared primitive.

Current safe point:

- Upstream source truth is pinned to FFmpeg `n8.0.1`, commit `894da5ca7d742e4429ffb2af534fcda0103ef593`.
- The active Go slice covers Annex B and AVC/NALFF length-prefixed packet intake, `avcC` SPS/PPS extradata, FFmpeg-shaped packet auto-detection, translated packet/frame side-data plumbing for the documented subset, simple 8-bit progressive frame-picture IDR/P/B output with DPB/reorder/weighted/direct-motion support, CAVLC/CABAC frame-MB entropy-to-state handoff, 8-bit reconstruction/deblocking, high-bit-depth 9/10/12/14 scalar kernels, high-bit-depth frame storage, and public High 10 4:2:0 deblock-disabled I/P plus selected B lanes through value-preserving raw helpers.
- Current High 10 public decode includes deblock-disabled I, P-skip/P16x16 no-residual, exact P16x16 L0 residual, explicit weighted P16x16, exact non-direct B16x16, temporal/spatial direct B16x16, and temporal/spatial B-skip fixtures across Annex B, AVC/NALFF, configured AVC, sample-by-sample flush, and FFmpeg rawvideo oracle surfaces.
- Public high-bit-depth H.264 decode is still intentionally narrow: high P intra/partitioned P, high B 8x8/direct-sub, implicit weighted high B, partitioned high B, high deblocking, field/MBAFF, FMO, broad error resilience, threading, SIMD dispatch, and full libavcodec delayed-output semantics remain future lanes.

Run the default tests:

```sh
go test ./...
```

Run tests that call the pinned native oracle tools available on this machine:

```sh
GOH264_ORACLE=1 go test ./...
```

Run the manifest-driven corpus runner with the committed seed manifest, or with
an external conformance/testvector manifest:

```sh
go test . -run TestH264CorpusManifest
GOH264_CORPUS_MANIFEST=/path/to/manifest.jsonl go test . -run TestH264CorpusManifest
```

Run the benchmark harness with repeated samples and an FFmpeg CLI rawvideo
baseline. The JSON report includes input, host, VCS, timing-scope, raw pixel
format, and per-repeat statistics:

```sh
go run ./cmd/goh264bench -input testdata/h264/cavlc_b8x8_spatial_direct_sub.h264 -iters 5 -repeats 5 -ffmpeg -json
```

Fetch the pinned upstream source snapshot into the ignored local cache:

```sh
scripts/fetch-upstream.sh
```

The file-by-file status lives in [docs/translation-ledger.md](docs/translation-ledger.md).
