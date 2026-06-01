# goh264

`goh264` is a source-shaped Go port of FFmpeg `libavcodec`'s H.264 decoder path.

The port is intentionally decoder-only. Encoder, muxer, filter, and unrelated codec code are out of scope unless an H.264 decoder file directly depends on a shared primitive.

Current safe point:

- Upstream source truth is pinned to FFmpeg `n8.0.1`, commit `894da5ca7d742e4429ffb2af534fcda0103ef593`.
- The active Go slice covers Annex B and AVC/NALFF length-prefixed packet intake, `avcC` SPS/PPS extradata, FFmpeg-shaped packet auto-detection, translated packet/frame side-data plumbing for the documented subset, simple 8-bit progressive frame-picture IDR/P/B output with DPB/reorder/weighted/direct-motion support, CAVLC/CABAC frame-MB entropy-to-state handoff, 8-bit reconstruction/deblocking, high-bit-depth 9/10/12/14 scalar kernels, high-bit-depth frame storage, and public High 10 4:2:0 I/P plus selected B lanes through value-preserving raw helpers.
- Current High 10 public decode includes deblock-disabled I, P-skip/P16x16 no-residual, exact P16x16 L0 residual, explicit weighted P16x16, mixed-P Intra4x4/Intra16x16, CAVLC/CABAC partitioned P16x8/P8x16/P8x8, exact non-direct B16x16, temporal/spatial direct B16x16, temporal/spatial B-skip, CAVLC/CABAC B 8x8/B_SUB_4x4 direct-sub, explicit partitioned B16x8/B8x16/B8x8, implicit weighted B16x16, partitioned implicit weighted B16x8/B8x16/B8x8, narrow CAVLC/CABAC non-direct and top-level direct B16x16 deblock-enabled lanes, neutral and implicit-weighted partitioned B16x8/B8x16/B8x8 deblock-enabled lanes, deblock-enabled CAVLC/CABAC 32x32 IDR/P fixtures for 4:2:0 plus the narrow 4:2:2/4:4:4 I/P deblocking lane, a CAVLC-only High10 4:2:0 `disable_deblocking_filter_idc == 2` slice-boundary IDR/P lane, and a narrow yuv420p12le CAVLC IDR/I IntraPCM lane across Annex B, AVC/NALFF, configured AVC, sample-by-sample flush where applicable, and FFmpeg rawvideo oracle surfaces.
- Public high-bit-depth H.264 decode is still intentionally narrow: High10 P-slice intra is admitted only for the proved Intra4x4/Intra16x16 mixed-P lane, partitioned P is admitted only for the proved unweighted P16x8/P8x16/P8x8 lane, partitioned implicit B is admitted only for the proved High10 4:2:0 B16x8/B8x16/B8x8 lanes with and without the new implicit high-deblock fixtures, high B deblocking is admitted only for the proved CAVLC/CABAC non-direct/direct B16x16 and neutral/implicit-weighted partitioned lanes, and 12-bit decode is admitted only for the proved yuv420p12le IDR/I IntraPCM lane; P IntraPCM, P 8x8-DCT intra, weighted partitioned P, mixed direct/explicit B8x8, residual-bearing direct-sub B, direct-sub/skip high B deblocking, implicit B16x16 high B deblocking, CABAC/chroma/B-slice public high slice-boundary deblocking, broader 12-bit and all 14-bit public high bitstreams, field/MBAFF, FMO, broad error resilience, threading, SIMD dispatch, and full libavcodec delayed-output semantics remain future lanes.

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
GOH264_CORPUS_MANIFESTS="/path/to/jvt.jsonl:/path/to/fate.jsonl" go test . -run TestH264CorpusManifest
```

The seed manifest is intentionally small but now file-backs the local 8-bit
B direct-sub vectors plus the proved High 10 4:2:0 IDR/P, residual P, weighted
P, CAVLC/CABAC partitioned P, non-direct B, temporal/spatial direct B, temporal/spatial B-skip, CAVLC/CABAC
B 8x8/B_SUB_4x4 direct-sub, explicit partitioned B16x8/B8x16/B8x8, implicit
weighted B16x16, partitioned implicit weighted B16x8/B8x16/B8x8, the narrow
CAVLC/CABAC non-direct/direct B16x16 deblock rows, neutral and implicit-weighted partitioned B deblock rows, deblock-enabled IDR/P vectors
including narrow High 10 4:2:2/4:4:4 deblocking rows, the CAVLC-only High10
slice-boundary row, and the High 4:4:4 Predictive-compatible yuv420p12le
IntraPCM row.

Run the benchmark harness with repeated samples and an FFmpeg CLI rawvideo
baseline. The JSON report includes input, host, VCS, timing-scope, raw pixel
format, and per-repeat statistics. For corpus-bound reports, use `-manifest`;
that mode validates bitstream MD5, raw pixel format, frame count, raw byte
count, and rawvideo MD5 against the manifest oracle before emitting timing
results:

```sh
go run ./cmd/goh264bench -input testdata/h264/cavlc_b8x8_spatial_direct_sub.h264 -iters 5 -repeats 5 -ffmpeg -json
go run ./cmd/goh264bench -manifest testdata/h264/corpus/manifest.jsonl -iters 5 -repeats 5 -ffmpeg -json
```

The next corpus/benchmark readiness checklist, including external manifest tiers,
benchmark profile recipes, benchstat-friendly output, and allocation-gate fields,
lives in [docs/production-readiness.md](docs/production-readiness.md).

Fetch the pinned upstream source snapshot into the ignored local cache:

```sh
scripts/fetch-upstream.sh
```

The file-by-file status lives in [docs/translation-ledger.md](docs/translation-ledger.md).
