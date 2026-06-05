# goh264

Pure-Go H.264 decoder, source-shaped from FFmpeg `libavcodec`.

`goh264` is a decoder-only Go port of the FFmpeg `n8.0.1` H.264 path, pinned at
`894da5ca7d742e4429ffb2af534fcda0103ef593`. The goal is not a loose rewrite:
the internal decoder keeps FFmpeg's state machines, syntax handling, math, and
edge cases recognizable, then proves behavior against FFmpeg oracle vectors.

- **Pure Go decoder path** - no cgo and no Go module dependencies.
- **Annex B and AVC input surfaces** - automatic packet splitting, explicit
  Annex B / length-prefixed AVC APIs, and AVC decoder configuration records.
- **Raw frame output** - `Frame` exposes Y/Cb/Cr planes, crop, strides, VUI
  fields, high-bit-depth planes, and raw YUV helpers.
- **Harness-first parity** - public FFmpeg FATE vectors are executable, with a
  red ledger kept for any future known-failing rows instead of hiding them.
- **Active port, not v1** - the public decoder-compliance matrix is green, with
  broader unselected codec lanes still guarded.

## Install

```sh
go get github.com/thesyncim/goh264
```

Requires Go 1.24 or newer.

FFmpeg is not required to import the package. FFmpeg is used by the oracle,
corpus-fetch, extraction, and benchmark scripts.

## Status Snapshot (2026-06-05)

Current public-vector matrix:

| Set | Count |
| --- | ---: |
| Selected public FFmpeg H.264 vectors | 224 |
| Green oracle rows | 224 |
| Known-red rows in `failures.jsonl` | 0 |
| Explicitly excluded upstream H.264-ish rows | 2 |

No known-red public-vector rows currently remain. The executable ledger at
`testdata/h264/realvectors/failures.jsonl` stays in place for future red rows
and is checked by the freshness/matrix gates when populated.

Green coverage includes compact Baseline/Main/High conformance rows, selected
FRext and high-bit-depth fixtures, High12/High14 CAVLC and CABAC B deblock
rows including implicit and explicit weighted B, High10 4:2:2/4:4:4
CAVLC/CABAC implicit and explicit weighted B frame and slice-boundary deblock, I/P/B slices, CAVLC and
CABAC, weighted and direct motion paths, deblock modes, selected field/PAFF/MBAFF rows
including High10 4:2:2 explicit weighted-B top/bottom field guards,
reinit metadata rows, lossless High444
transform-bypass rows, configured AVC surfaces, container-extracted Annex B
vectors, and SEI side-data surfaces.

Still guarded: unselected MBAFF/PIC-AFF/PAFF motion paths, broader high-bit-depth
field/inter streams including remaining 4:4:4 weighted chroma field variants and broader slice-boundary modes,
broader damaged-slice error resilience, threading/SIMD and
bulk allocation hardening, and exact libavcodec delayed-output behavior.
Intentionally unsupported at the pinned FFmpeg parity boundary: FMO, which
FFmpeg n8.0.1 compiles out and rejects while parsing PPS slice groups, and
11/13-bit luma depths, which FFmpeg rejects at PPS admission. SPS admission also
mirrors FFmpeg's unsupported boundaries for `chroma_format_idc > 3`, separate
color planes, and mixed chroma/luma bit depths.

## Quick Start

Decode an Annex B or automatically detected H.264 packet and append raw YUV
bytes in FFmpeg-compatible plane order:

```go
package main

import (
	"log"
	"os"

	"github.com/thesyncim/goh264"
)

func main() {
	data, err := os.ReadFile("input.h264")
	if err != nil {
		log.Fatal(err)
	}

	dec := goh264.NewDecoder()
	frames, err := dec.DecodeFrames(data)
	if err != nil {
		log.Fatal(err)
	}

	var raw []byte
	for _, frame := range frames {
		pixFmt, err := frame.RawPixelFormat()
		if err != nil {
			log.Fatal(err)
		}
		raw, err = frame.AppendRawYUVBytesLE(raw)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%dx%d %s key=%v", frame.Width, frame.Height, pixFmt, frame.KeyFrame)
	}

	if err := os.WriteFile("out.yuv", raw, 0o644); err != nil {
		log.Fatal(err)
	}
}
```

`AppendRawYUV` is available for 8-bit output. `AppendRawYUVBytesLE` handles both
8-bit and high-bit-depth output, using little-endian samples for 9-bit and
higher formats. `RawPixelFormat` returns names such as `yuv420p`, `yuv422p10le`,
or `yuv444p`.

## API Surface

Create a decoder with:

```go
dec := goh264.NewDecoder()
```

Common decode entry points:

```go
frames, err := dec.DecodeFrames(data)                  // auto Annex B / AVC / config record
frames, err := dec.DecodeAnnexBFrames(annexB)          // Annex B bytestream
frames, err := dec.DecodeAVCFrames(packet, lengthSize) // length-prefixed NAL units
frames, err := dec.DecodeConfiguredAVCFrames(packet)   // after parsing avcC
frames, err := dec.FlushDelayedFrames()                // delayed B-frame output
```

Single-frame helpers (`Decode`, `DecodeAnnexB`, `DecodeAVC`,
`DecodeConfiguredAVC`) return `ErrUnsupported` when a packet produces zero or
multiple frames. For stream processing, prefer the `*Frames` methods.

Parse headers without decoding full frames:

```go
info, err := dec.ParseHeadersAnnexB(data)
info, err := dec.ParseHeadersAVC(packet, nalLengthSize)
cfg, err := dec.ParseAVCDecoderConfigurationRecord(avcc)
```

Packet side-data support mirrors FFmpeg-facing surfaces used by the port:

```go
frames, err := dec.DecodePacketFrames(goh264.Packet{
	Data: packet,
	SideData: []goh264.PacketSideData{
		{Type: goh264.PacketSideDataNewExtradata, Data: avcc},
	},
})
```

`Frame` includes dimensions, crop, chroma format, bit depth, SAR/VUI fields,
timing fields, keyframe/interlace flags, raw planes, and parsed SEI/packet side
data such as A53 captions, S12M timecode, stereo 3D, spherical video, mastering
display metadata, content light metadata, display orientation, film grain, ICC
profile, HDR10+, and LCEVC side data.

## Supported Inputs

| Area | Status |
| --- | --- |
| Annex B bytestream | Supported on green corpus rows |
| AVC length-prefixed packets | Supported, including explicit NAL length size |
| AVC decoder configuration record (`avcC`) | Supported for configured AVC decode |
| Baseline/Main/High progressive rows | Broad public-vector coverage |
| High10/High422/High444 | Selected public and generated coverage |
| CAVLC and CABAC | Covered by unit, fixture, and public vectors |
| I/P/B slices | Covered across the current public-vector matrix |
| SEI and packet side data | Parsed for the public side-data surfaces |
| Containers | Not a demuxer; container FATE rows are extracted to Annex B for decode |

## Parity and Testing

The pinned FFmpeg source is the spec. When behavior is uncertain, port the
FFmpeg branch shape first, then prove it with an oracle or fixture. Do not
delete, skip, or widen a failing vector to make a gate pass.

Fast local gate:

```sh
go test ./...
```

Public vector gates:

```sh
scripts/h264-real-vector-strict.sh
GOH264_REAL_VECTOR_FAILURES=1 GOH264_CORPUS_FETCH=1 go test ./tests -run '^TestH264RealVectorFailureLedgerFreshness$' -count=1 -v
GOH264_REAL_VECTOR_MATRIX=1 GOH264_CORPUS_FETCH=1 go test ./tests -run '^TestH264RealVectorFailureMatrix$' -count=1 -v
scripts/h264-real-vector-upstream-audit.sh
```

What those gates mean:

- `h264-real-vector-strict.sh` runs the green public-vector set and excludes
  only rows currently listed in the failure ledger.
- `FailureLedgerFreshness` runs only known-red rows when the ledger is populated
  and requires each failure class/detail to remain current.
- `FailureMatrix` runs the full 224-row manifest, currently requiring all 224
  rows to match oracle output.
- `h264-real-vector-upstream-audit.sh` fetches the pinned FFmpeg source and
  verifies that the public-vector manifest represents all decoder-facing
  upstream H.264 FATE sample references, except the documented non-decoder rows.

Focused red-lane tools:

```sh
scripts/h264-real-vector-red-queue.sh field
scripts/h264-real-vector-red-each.sh
scripts/h264-red-vector.sh direct
GOH264_REAL_VECTOR_RAWDIFF=1 GOH264_CORPUS_FILTER=mbaff GOH264_CORPUS_FETCH=1 go test ./tests -run '^TestH264RealVectorRawDiffDiagnostics$' -count=1 -v
GOH264_REAL_VECTOR_FRAMEMD5=1 GOH264_CORPUS_FILTER=mbaff GOH264_CORPUS_FETCH=1 go test ./tests -run '^TestH264RealVectorFrameMD5Diagnostics$' -count=1 -v
```

`GOH264_CORPUS_FILTER` accepts feature tags or id fragments such as `field`,
`direct`, `high10`, `container`, `reinit`, or `mbaff`.

## Benchmarks

`cmd/goh264bench` validates oracle parity before timing selected manifest rows
and can compare Go against FFmpeg lanes:

```sh
go run ./cmd/goh264bench \
  -manifest testdata/h264/realvectors/manifest.jsonl \
  -filter canl4 \
  -iters 10 \
  -repeats 5 \
  -warmup 2 \
  -ffmpeg \
  -fair-cpu-lanes \
  -ffmpeg-threads 1 \
  -strict-pix-fmt \
  -json
```

The JSON report includes selected/green/known-red counts, backend kind, CPU
flags, comparison lane, raw MD5 parity, oracle quality status, and FFmpeg-vs-Go
peer quality status.

## Project Layout

| Path | Purpose |
| --- | --- |
| `decoder.go` | Public decoder API, frames, raw output helpers, side-data mapping |
| `internal/h264/` | Source-shaped parser, syntax, prediction, transform, DPB, reconstruct, and loop-filter code |
| `tests/decoder_*_test.go` | Public and package-level fixture/oracle coverage |
| `testdata/h264/corpus/` | Small local corpus manifest |
| `testdata/h264/realvectors/` | Public FFmpeg FATE manifest and known-red ledger |
| `scripts/` | Upstream fetch, oracle probes, public-vector gates, diagnostics, benchmarks |
| `cmd/goh264bench/` | JSON benchmark and FFmpeg comparison CLI |
| `docs/source-truth.md` | Compact current parity snapshot |
| `docs/translation-ledger.md` | Upstream-to-Go translation ledger |
| `docs/production-readiness.md` | Current verification and performance gates |
| `docs/high-bitdepth-roadmap.md` | High-bit-depth parity plan |

## Contributing

Work in closed topics:

- Add or keep the failing vector first.
- Port the smallest source-shaped FFmpeg behavior that should make it green.
- Run the focused oracle, then the relevant public-vector gate.
- Keep known-red rows in `failures.jsonl` until they genuinely match oracle
  output, then remove them in the same fix commit.
- Stage only intended files and leave unrelated worktree changes alone.

Good safe-point gates are usually:

```sh
git diff --check
go test ./...
scripts/h264-real-vector-strict.sh
GOH264_REAL_VECTOR_FAILURES=1 GOH264_CORPUS_FETCH=1 go test ./tests -run '^TestH264RealVectorFailureLedgerFreshness$' -count=1 -v
GOH264_REAL_VECTOR_MATRIX=1 GOH264_CORPUS_FETCH=1 go test ./tests -run '^TestH264RealVectorFailureMatrix$' -count=1 -v
scripts/h264-real-vector-upstream-audit.sh
```

## License

`goh264` is licensed under LGPL-2.1-or-later. See [LICENSE](LICENSE).

## References

- FFmpeg `n8.0.1`, pinned at `894da5ca7d742e4429ffb2af534fcda0103ef593`
- ITU-T H.264 / ISO/IEC 14496-10
- FFmpeg FATE H.264 sample suite
