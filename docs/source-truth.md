# Source Truth

Upstream is FFmpeg `n8.0.1`, commit
`894da5ca7d742e4429ffb2af534fcda0103ef593`. This repo ports only the
`libavcodec` H.264 decoder path; encoders, muxers, filters, hardware backends,
and non-H.264 codecs are out of scope unless a decoder file directly depends on
their primitive behavior.

## Current Proved Surface

- Packet/input: Annex B, AVC/NALFF, `avcC`, auto packet detection, configured
  sample decode, delayed flush, and the translated packet/frame side-data subset.
- Core 8-bit path: progressive frame-picture IDR/P/B simple decoder with
  CAVLC/CABAC macroblock handoff, DPB/reorder, weighted/direct motion,
  reconstruction, and frame deblocking for the proved fixtures.
- High storage/DSP: 9/10/12/14-bit scalar primitives and uint16 frame/ref/output
  surfaces exist; public decode is intentionally narrower than the primitives.
- Public High10 4:2:0, deblock disabled: IDR/I, P-skip/P16x16 no-residual,
  exact P16x16 residual, explicit weighted P16x16, mixed-P Intra4x4/Intra16x16,
  CAVLC/CABAC unweighted and explicit weighted partitioned P16x8/P8x16/P8x8,
  non-direct B16x16, temporal/spatial direct B16x16, temporal/spatial B-skip,
  B 8x8/B_SUB_4x4 direct-sub, explicit partitioned B16x8/B8x16/B8x8, implicit
  weighted B16x16, and partitioned implicit weighted B16x8/B8x16/B8x8.
- Public High10 deblock-enabled: 32x32 IDR/P for 4:2:0/4:2:2/4:4:4, CAVLC-only
  4:2:0 slice-boundary IDR/P, narrow CAVLC/CABAC B16x16 non-direct/direct,
  implicit-weighted B16x16, neutral/implicit partitioned B, and neutral/implicit
  direct-sub B with the documented CBP-zero limits.
- Public High12: only the narrow yuv420p12le CAVLC IDR/I IntraPCM fixture.

## Oracle Authority

Detailed bitstream MD5s, per-frame MD5s, rawvideo MD5s, packet surfaces, and
pixel formats live in `testdata/h264/corpus/manifest.jsonl`. That manifest is
the canonical fixture ledger for tests and `cmd/goh264bench`; Markdown should
summarize it, not duplicate it.

Primary verification surfaces:

- `go test ./...`
- `GOH264_ORACLE=1 go test ./...`
- `go test . -run TestH264CorpusManifest`
- `go run ./cmd/goh264bench -manifest testdata/h264/corpus/manifest.jsonl -iters 1 -repeats 1 -warmup 0 -ffmpeg -json`

## Latest Safe Point

Weighted partitioned High10 P is now proved for frame-only, 4:2:0,
deblock-disabled, CAVLC/CABAC P16x8/P8x16/P8x8 streams with explicit P weights.

Files:

- `testdata/h264/high10_weighted_partitioned_p_cavlc.h264`
- `testdata/h264/high10_weighted_partitioned_p_cabac.h264`
- `decoder_high10_partitioned_p_test.go`
- `internal/h264/simple_decode_high_weighted_partitioned_p_fixture_test.go`

Manifest rows:

- `local/high10/high10-weighted-partitioned-p-cavlc`
- `local/high10/high10-weighted-partitioned-p-cabac`

## Still Unsupported

Keep these guarded until a matching oracle fixture and narrow admission test
land: P IntraPCM, P 8x8-DCT intra, deblock-enabled weighted partitioned P,
mixed direct/explicit B8x8, residual-bearing direct-sub B, broader partitioned
implicit B, residual-bearing direct-sub high B deblocking, CABAC/chroma/B-slice
public high slice-boundary modes, broader 12-bit and all 14-bit public streams,
GBR/RGB output, field/MBAFF, FMO, full error resilience, threading/SIMD, and
full libavcodec delayed-output semantics.
