# goh264

`goh264` is a pure-Go, decoder-only H.264/AVC package. It imports no cgo and no
third-party Go modules.

The decoder is source-shaped from the FFmpeg `n8.0.1` H.264 decoder path,
pinned at `894da5ca7d742e4429ffb2af534fcda0103ef593`. Current evidence covers
public Annex B, AVC, avcC, packet, raw-output, side-data, delayed-output,
corpus, FATE, and FFmpeg-oracle surfaces.

## Install

```sh
go get github.com/thesyncim/goh264
```

Requires Go 1.24 or newer.

## Patent Notice

`goh264` is decoder-only and does not include an H.264 encoder. H.264/AVC
technology may be covered by patent rights in some jurisdictions. This project
does not grant patent rights; users and distributors are responsible for their
own licensing analysis. See [PATENTS.md](PATENTS.md).

## API At A Glance

| Need | Use |
| --- | --- |
| Stateful Annex B access units, stored configured-AVC packets, avcC records, and empty-input delayed-output flush | `NewDecoder`, then `DecodeFrames` |
| Packet bytes plus side data such as `NEW_EXTRADATA` | `DecodePacketFrames` |
| Complete Annex B bytestream with no retained stream state | `DecodeAnnexBFrames` |
| Complete length-prefixed AVC packet stream with known 1-, 2-, 3-, or 4-byte NAL lengths | `DecodeAVCFrames` |
| Stored avcC/configured-AVC stream packets | `ConfigureAVCC`, then `DecodeConfiguredAVCFrames` |
| Update avcC and decode one packet as an in-stream configured-AVC unit | `DecodeAVCCFrames` |
| End-of-stream delayed B-frame output | `FlushDelayedFrames` |
| Header/config metadata without changing decoder state | `InspectAnnexBHeaders`, `InspectAVCHeaders`, or `InspectAVCC` |
| Retain caller-owned packets, decoded frames, or side data | `Packet`, `Frame`, `FrameSideData`, and their `Clone`, `Validate`, or append helpers |

## Quick Start

Use `DecodeAnnexBFrames` for a complete Annex B buffer when no decoder state
must be retained. For streaming access units, keep one `Decoder` and call
`DecodeFrames` or `DecodePacketFrames`; call `FlushDelayedFrames` or pass empty
input at end of stream.

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
	frames, err := dec.DecodeAnnexBFrames(data)
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

## Decoder Surfaces

The recommended decoder path is intentionally small:

```go
dec := goh264.NewDecoder()
frames, err := dec.DecodeFrames(packetData)
frames, err = dec.DecodePacketFrames(goh264.Packet{
	Data:     packetData,
	SideData: sideData,
})
frames, err = dec.FlushDelayedFrames()
err = dec.Reset()
```

Use format-specific helpers when the packet format is already known:

```go
frames, err := dec.DecodeAnnexBFrames(annexB)
frames, err := dec.DecodeAVCFrames(packet, lengthSize)
cfg, err := dec.ConfigureAVCC(avcc)
frames, err := dec.DecodeConfiguredAVCFrames(packet)
frames, err := dec.DecodeAVCCFrames(avcc, packet)
```

Use the single-frame helpers only when exactly one output frame is expected.
They return `ErrUnsupported` for zero-frame or multi-frame packets that otherwise
decode successfully.

## Output And Ownership

Decoded `Frame` values expose Y/Cb/Cr planes, crop metadata, VUI/timing fields,
selected high-bit-depth planes, raw YUV helpers, and side data. `RawYUVBytesLE`
returns a caller-owned byte buffer for one frame. `RawYUV16` returns a
caller-owned uint16 sample buffer for high-bit-depth frames. `AppendRawYUV`,
`AppendRawYUVBytesLE`, and `AppendRawYUV16` append into caller-provided storage.

`Frame.Validate`, `Frame.Clone`, `FrameSideData.Validate`,
`FrameSideData.Clone`, `Packet.Validate`, `Packet.Clone`,
`PacketSideData.Validate`, and `PacketSideData.Clone` are provided for callers
that need explicit ownership and storage checks.

## Current Evidence

The checked-in public-vector decoder manifest currently has 225 selected green
oracle rows and no known-red rows. Rerun the evidence gates for the current
checkout before treating that state as fresh:

| Set | Count |
| --- | ---: |
| Imported public H.264 vector refs | 226 |
| Pinned FFmpeg FATE refs in imported inventory | 224 |
| Selected public H.264 vectors | 225 |
| Green oracle rows | 225 |
| Known-red rows in `failures.jsonl` | 0 |
| Explicitly excluded upstream H.264-ish rows | 1 |

The selected manifest represents 225 imported decoder-facing refs; the
remaining imported ref is the documented non-H.264 MKV exclusion.

```sh
go test ./...
scripts/h264-decoder-quality-evidence.sh
scripts/h264-real-vector-strict.sh
```

FFmpeg is not required to import the package. FFmpeg is used by the oracle,
corpus-fetch, extraction, and benchmark scripts.

## Boundaries

The decoder intentionally does not support FMO, 11/13-bit luma depths,
`chroma_format_idc > 3`, separate color planes, or mixed chroma/luma bit depths
at the pinned FFmpeg parity boundary. Guarded areas still include broader
field/MBAFF/PIC-AFF motion behavior, damaged-slice edge cases, threading/SIMD,
bulk allocation hardening, and exact libavcodec delayed-output edge behavior.

The detailed worklist lives in:

- [docs/production-readiness.md](docs/production-readiness.md)
- [docs/source-truth.md](docs/source-truth.md)
- [docs/translation-ledger.md](docs/translation-ledger.md)
