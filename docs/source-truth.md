# Source Truth

Upstream is FFmpeg `n8.0.1`,
`894da5ca7d742e4429ffb2af534fcda0103ef593`. Only the `libavcodec` H.264
decoder path is in scope.

## Proved Surface

- Packet/API: Annex B, AVC/NALFF, `avcC`, auto detection, configured samples,
  delayed flush, and the documented side-data subset.
- Core 8-bit: progressive frame-picture IDR/P/B with CAVLC/CABAC handoff,
  DPB/reorder, weighted/direct motion, reconstruction, and fixture-proved
  deblocking.
- High internals: 9/10/12/14-bit scalar DSP and uint16 frame/ref/output planes.
- Public High10/High12: only manifest-backed lanes. Latest addition is High10
  4:2:0 frame-only `disable_deblocking_filter_idc == 2` slice-boundary IDR/P,
  CAVLC and CABAC.

Canonical fixture detail lives in `testdata/h264/corpus/manifest.jsonl`, not in
Markdown.

## Proof Commands

```sh
go test ./...
GOH264_ORACLE=1 go test ./...
go run ./cmd/goh264bench -manifest testdata/h264/corpus/manifest.jsonl -iters 1 -repeats 1 -warmup 0 -ffmpeg -json
```

## Still Guarded

P IntraPCM, P 8x8-DCT intra, deblock-enabled weighted partitioned P, mixed
direct/explicit B8x8, residual direct-sub B, broader high B deblock residual
lanes, public chroma/B-slice slice-boundary modes, broader 12-bit and all 14-bit
public streams, GBR/RGB, field/MBAFF, FMO, threading/SIMD, broad error
resilience, and full libavcodec delayed-output behavior.
