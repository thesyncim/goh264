# goh264

Pure-Go H.264 decoder with a guarded realtime/WebRTC encoder API,
source-shaped from FFmpeg `libavcodec`.

This repository ports the FFmpeg `n8.0.1` H.264 decoder path, pinned at
`894da5ca7d742e4429ffb2af534fcda0103ef593`. Decoder evidence covers public
Annex B, AVC, avcC, packet, raw-output, side-data, delayed-output, corpus, FATE,
and FFmpeg-oracle surfaces.

The encoder surface targets realtime/WebRTC integration and admits a guarded
Constrained Baseline I420 subset: IDR IntraPCM, identical-reference P-skip,
bounded exact P16x16 no-residual prediction, bounded residual-P admission for
exact luma-DC, chroma-only, and combined luma/chroma CAVLC residuals,
changed P IntraPCM recovery frames, AVC/Annex B output, configured multi-slice
output, and RTP packetization modes 0 and 1. Outside the current encoder
contract: general motion search, broader residual macroblock generation,
rate-control decisions, wider packetizer/control breadth, and reviewed
allocation/performance evidence.

## API At A Glance

| Job | Preferred surface |
| --- | --- |
| Decode complete Annex B bytes with no retained stream state | `NewDecoder().DecodeAnnexBFrames(data)` |
| Decode stateful Annex B packets, stored configured-AVC packets, or avcC records | `dec.DecodeFrames(data)` |
| Decode packet bytes plus packet side data such as `NEW_EXTRADATA` | `dec.DecodePacketFrames(Packet{Data: data, SideData: sideData})` |
| Decode known length-prefixed AVC with 1-, 2-, 3-, or 4-byte NAL lengths | `dec.DecodeAVCFrames(data, nalLengthSize)` |
| Store avcC once, then feed configured AVC packets | `dec.ConfigureAVCC(avcc)`, then `dec.DecodeConfiguredAVCFrames(packet)` |
| Update avcC and decode one packet as an in-stream configured-AVC unit | `dec.DecodeAVCCFrames(avcc, packet)` |
| Encode guarded realtime I420 to Annex B, AVC, or RTP | start from `DefaultRTPEncoderConfig`, `DefaultAnnexBEncoderConfig`, or `DefaultAVCEncoderConfig`; call `Normalize`, `NewEncoder`, then `Encode` or `EncodeInto` |
| Retain caller-owned packets, frames, headers, encoded output, or RTP packets | use the relevant `Clone`, `Validate`, or `Append...` helper |

Use the detailed Decoder API and Encoder API sections below for state,
ownership, error, and admission rules. Use the Trust And Verification gates
before treating a checkout as production evidence.

## Quality And Parity Evidence

| Area | Evidence shape | Covered surfaces | Outside current contract / evidence targets |
| --- | --- | --- | --- |
| Decoder | Parity-driven port from the pinned FFmpeg path | Public Annex B/AVC/avcC/packet decode surfaces, delayed output, raw output, side data, corpus/FATE rows, FFmpeg-oracle rows | Broader field/MBAFF/damaged-edge behavior, fresh artifact evidence, allocation/performance review |
| Encoder | Guarded realtime subset | Baseline I420 IDR IntraPCM, P-skip, bounded exact P16x16 no-residual, bounded pixel-derived residual-P luma-DC, chroma-only, and combined luma/chroma admission, P IntraPCM recovery, Annex B/AVC/RTP output, ownership/transactional API guards | General motion search, broader residual generation, adaptive rate control, wider packetizer/control breadth, broader/full bitstream parity beyond admitted oracle rows, allocation/performance review |

Examples compiled in `examples_test.go` are API smoke tests only. README
snippets are API orientation, not codec quality, bitstream parity, acceptance,
or performance evidence.

## Capabilities

- **Decoder:** pure Go, no cgo, no module dependencies.
- **Inputs:** Annex B bytestreams, length-prefixed AVC packets, avcC decoder
  configuration records, packet `NEW_EXTRADATA`, and auto-detected packets.
- **Output:** decoded Y/Cb/Cr planes, crop metadata, VUI/timing fields,
  selected high-bit-depth planes, raw YUV byte/sample helpers, frame cloning, and
  side-data cloning.
- **State:** streaming decode keeps references and delayed B-frame output across
  calls; empty decode calls flush delayed output.
- **Encoder:** realtime/WebRTC integration surface for the guarded Baseline paths
  listed above.
- **Verification:** the checked-in public-vector decoder manifest has 225 green
  oracle rows and no known-red rows. Rerun the Trust And Verification gates for
  the current checkout before treating that state as fresh evidence.

## Worklist

The remaining work is mainly quality evidence, fresh artifact evidence, API
surface review, allocation/performance evidence, and broader encoder coverage. The
detailed worklist lives in:

- [docs/production-readiness.md](docs/production-readiness.md)
- [docs/source-truth.md](docs/source-truth.md)
- [docs/translation-ledger.md](docs/translation-ledger.md)
- [docs/encoder-webrtc-roadmap.md](docs/encoder-webrtc-roadmap.md)

## Install

```sh
go get github.com/thesyncim/goh264
```

Requires Go 1.24 or newer.

FFmpeg is not required to import the package. FFmpeg is used by the oracle,
corpus-fetch, extraction, and benchmark scripts.

## Decoder Evidence Snapshot

Public-vector matrix:

| Set | Count |
| --- | ---: |
| Imported public H.264 vector refs | 226 |
| Pinned FFmpeg FATE refs in imported inventory | 224 |
| Selected public H.264 vectors | 225 |
| Green oracle rows | 225 |
| Known-red rows in `failures.jsonl` | 0 |
| Explicitly excluded upstream H.264-ish rows | 1 |

The selected manifest represents 225 imported decoder-facing refs; the remaining
imported ref is the documented non-H.264 MKV exclusion. No known-red
public-vector rows are listed. The executable ledger at
`testdata/h264/realvectors/failures.jsonl` stays in place for future red rows
and is checked by the freshness/matrix gates when populated.
`TestH264DecoderTDDContractClassifiesEveryImportedPublicVector` is the always-on
contract that keeps the inventory, manifest, exclusions, and failure ledger in
lockstep.

Decoder coverage includes compact Baseline/Main/High rows, selected FRext and
high-bit-depth fixtures, I/P/B slices, CAVLC and CABAC, weighted and direct
motion paths, selected field/PAFF/MBAFF rows, lossless High444
transform-bypass rows, configured AVC surfaces, container-extracted Annex B
vectors, malformed packet recovery, side-data surfaces, and bounded public
no-panic fuzz coverage.

Guarded decoder areas: unselected MBAFF/PIC-AFF/PAFF motion paths, broader
high-bit-depth field/inter streams, broader damaged-slice error resilience,
threading/SIMD, bulk allocation hardening, and exact libavcodec delayed-output
edge behavior. Intentionally unsupported at the pinned FFmpeg parity boundary:
FMO, 11/13-bit luma depths, `chroma_format_idc > 3`, separate color planes, and
mixed chroma/luma bit depths.

## Quick Start: Decode

Use `DecodeAnnexBFrames` for a complete Annex B buffer when no decoder state must
be retained. For streaming access units, keep one decoder and call `DecodeFrames`
or `DecodePacketFrames`; call `FlushDelayedFrames` or pass empty input at end of
stream. For 1-, 2-, or 3-byte length-prefixed AVC, call `DecodeAVCFrames` or
configure avcC first. The raw-output helpers append pixels in FFmpeg-compatible
plane order:

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

`RawYUVBytesLE` returns a caller-owned rawvideo byte buffer for one frame.
`RawYUV16` returns a caller-owned uint16 sample buffer for high-bit-depth frames.
`Frame.Validate` checks decoded-frame plane and side-data storage for
caller-owned preflight. `Frame.Clone` returns a deep-owned decoded-frame
snapshot, including planes and side data, and rejects overflowed public frame or
side-data storage; `FrameSideData.Validate` provides the same storage check for
caller-constructed side data.
`AppendRawYUV` is available for 8-bit output. `AppendRawYUVBytesLE` handles both
8-bit and high-bit-depth output, using little-endian samples for 9-bit and
higher formats. `AppendRawYUV16` is the caller-buffer form for high-bit-depth
uint16 output. Raw-output append helpers isolate output when the caller
destination overlaps frame plane storage. `RawPixelFormat` returns names such
as `yuv420p`, `yuv422p10le`, or `yuv444p`.

## Decoder API

The recommended decoder path is intentionally small:

```go
dec := goh264.NewDecoder()
frames, err := dec.DecodeFrames(packetData) // stateful Annex B, stored configured AVC, or avcC records
frames, err = dec.DecodePacketFrames(goh264.Packet{
	Data:     packetData,
	SideData: sideData,
}) // packet side data and NEW_EXTRADATA
frames, err = dec.FlushDelayedFrames() // end-of-stream delayed output
err = dec.Reset()                      // clear decoder state
```

Choose the entry point by ownership and packet shape:

| Need | Use |
| --- | --- |
| Stateful Annex B access units, stored configured-AVC packets, avcC records, and empty-input delayed-output flush; unconfigured AVC auto-sniffing is 4-byte only | `DecodeFrames` |
| Same stream path plus packet side data such as `NEW_EXTRADATA` | `DecodePacketFrames` |
| Complete Annex B bytestream with no streaming state needed | `DecodeAnnexBFrames` |
| Complete length-prefixed AVC packet stream with known 1-, 2-, 3-, or 4-byte NAL length size | `DecodeAVCFrames` |
| Stateless header/config metadata | `InspectAnnexBHeaders`, `InspectAVCHeaders`, or `InspectAVCC` |
| Stored avcC/configured-AVC stream packets | `ConfigureAVCC`, then `DecodeConfiguredAVCFrames` |
| Exactly one expected output frame or one delayed flush | `Decode`, `DecodePacket`, `DecodeAnnexB`, `DecodeAVC`, `DecodeConfiguredAVC`, `DecodeAVCC`, or `FlushDelayedFrame` |

Use the format-specific helpers when the packet format is already known:

```go
frames, err := dec.DecodeAnnexBFrames(annexB)          // complete Annex B bytestream
frames, err := dec.DecodeAVCFrames(packet, lengthSize) // complete length-prefixed AVC packet stream
cfg, err := dec.ConfigureAVCC(avcc)                    // store avcC for configured AVC
frames, err := dec.DecodeConfiguredAVCFrames(packet)   // stateful AVC after avcC
frames, err := dec.DecodeAVCCFrames(avcc, packet)      // update avcC, decode, then drain
cfg, err = dec.AVCConfig()                             // current configured-AVC metadata
```

Single-frame helpers (`Decode`, `DecodePacket`, `DecodeAnnexB`, `DecodeAVC`,
`DecodeConfiguredAVC`, `DecodeAVCC`, and `FlushDelayedFrame`) return
`ErrUnsupported` when a packet produces zero or multiple frames. If a damaged
packet produces exactly one valid frame before a later decode error, the helper
returns that frame with the error. Empty-input delayed-output calls through
single-frame helpers consume delayed output only when exactly one frame is
available; on zero or multiple delayed frames they return `ErrUnsupported` and
the queued delayed output remains available to `FlushDelayedFrames`. For stream
processing, prefer `DecodeFrames` or
`DecodePacketFrames`; they retain decoder reference state across packets, select
Annex B or the stored configured-AVC length size, store avcC records when
encountered, preserve valid leading SEI from SEI-only packets until the next
decoded frame, and flush delayed output when called with empty data. `DecodeConfiguredAVCFrames`
uses the stored avcC length size directly. Bare length-prefixed AVC packets
with 1-, 2-, or 3-byte NAL length fields should use `DecodeAVCFrames` or a
configured-AVC path after `ConfigureAVCC`/`ParseHeadersAVC`; unconfigured
`DecodeFrames` only auto-sniffs 4-byte AVC. `DecodeAnnexBFrames` and
`DecodeAVCFrames` are complete-stream helpers for callers that already know the
format and length-size.

`DecodeAVCCFrames` updates the decoder's AVC configuration, decodes the
supplied AVC packet, and drains delayed output before returning. Compatible
in-stream avcC updates retain references; incompatible active SPS changes reset
picture state before the packet is decoded so prior references are not used
across the incompatible boundary. Passing an empty AVC packet with a
configuration record drains delayed output before an incompatible configuration
can reset picture state, then stores the new configuration without reporting an
invalid packet.
Use this for ordinary in-stream avcC updates and IDR-bound stream switches. For
an unrelated stream where the decoder cannot infer the boundary from avcC,
call `Reset` before storing the new avcC. `PacketSideDataNewExtradata` uses the
same stateful update rule when it carries avcC or Annex B parameter-set data:
compatible updates retain references; incompatible active SPS changes reset
picture state before decoding, so prior references are not used across the
incompatible boundary. When an update carries multiple SPS/PPS entries, the
reset decision follows the packet's slice-selected PPS/SPS when that packet can
be parsed. `AVCConfig` reports the packet-active SPS after a successful
configured-AVC packet identifies one; before that it reports the first SPS from
the stored avcC record. Standalone multi-SPS avcC records accepted through
`DecodeFrames` reset picture state conservatively when any SPS/PPS candidate
could be incompatible with current references.

Configure or inspect headers without decoding full frames. The decoder methods
are stateful: `ParseHeadersAnnexB` stores SPS/PPS state, and `ParseHeadersAVC`
also stores the AVC NAL length size used by later
`DecodeConfiguredAVCFrames` calls. Use the package-level inspect functions for
stateless Annex B, AVC, and avcC metadata:

```go
info, err := dec.ParseHeadersAnnexB(data)
info, err = dec.ParseHeadersAVC(packet, nalLengthSize)
info, err = goh264.InspectAnnexBHeaders(data)
info, err = goh264.InspectAVCHeaders(packet, nalLengthSize)
cfg, err := goh264.InspectAVCC(avcc)
```

Malformed `ParseHeadersAnnexB` and `ParseHeadersAVC` calls are transactional:
partially parsed SPS/PPS state is not committed over a previous valid
configuration, and delayed configured-AVC B-frame output remains available for
flush after the rejected parse.
Use `ConfigureAVCC` to store avcC metadata for later configured-AVC decode.
Use package-level `InspectAVCC` when the caller only needs avcC metadata and
does not want to mutate decoder state.
Malformed avcC records, including invalid reserved bits or
caller-constructed impossible-size inputs, are rejected before replacing the
previous stored configuration.

avcC name map:

| Need | Helper | Single-frame helper |
| --- | --- | --- |
| Stateless avcC metadata inspection | `InspectAVCC` | n/a |
| Store avcC for configured-AVC streaming | `ConfigureAVCC` | n/a |
| Decode with already stored avcC | `DecodeConfiguredAVCFrames` | `DecodeConfiguredAVC` |
| Update avcC, decode one packet, then drain delayed output | `DecodeAVCCFrames` | `DecodeAVCC` |

Packet side-data support mirrors FFmpeg-facing surfaces used by the port:

```go
sideData := []goh264.PacketSideData{
	{Type: goh264.PacketSideDataNewExtradata, Data: avcc},
	{Type: goh264.PacketSideDataA53ClosedCaptions, Data: cc},
}
frames, err := dec.DecodePacketFrames(goh264.Packet{
	Data:     packet,
	SideData: sideData,
})
ownedPacket, err := (goh264.Packet{Data: packet, SideData: sideData}).Clone()
if err != nil {
	log.Fatal(err)
}
```

`Frame` includes dimensions, crop, chroma format, bit depth, SAR/VUI fields,
timing fields, keyframe/interlace flags, raw planes, and parsed SEI/packet side
data such as A53 captions, S12M timecode, stereo 3D, spherical video, mastering
display metadata, content light metadata, display orientation, film grain, ICC
profile, HDR10+, and LCEVC side data.
`Packet.Clone`, `PacketSideData.Clone`, `Frame.Clone`, and
`FrameSideData.Clone` validate public storage and return deep-owned snapshots
for retained packets and decoded output metadata. `Packet.Validate`,
`PacketSideData.Validate`, `Frame.Validate`, and `FrameSideData.Validate` expose
the same checks for preflight before handoff or retention. `Packet.AppendData`,
`Packet.AppendSideData`, and `PacketSideData.AppendData` append caller-owned
copies for retained compressed packets, packet side-data lists, and packet
side-data payloads. `FrameSideData` also provides `AppendUserDataUnregistered`,
`AppendA53ClosedCaptions`, `AppendICCProfile`, `AppendDynamicHDR10Plus`,
`AppendLCEVC`, and `AppendS12MTimecodes` for retaining individual decoded
side-data payloads in caller-managed buffers.
`PictureTiming.Validate`/`Clone` and `ReferenceDisplaysInfo.Validate`/`Clone`
provide retained snapshots for structured side-data containers, while
`PictureTiming.AppendTimecodes` and `ReferenceDisplaysInfo.AppendDisplays`
provide the caller-buffer/error contract for individual structured lists.
Structured side-data entries are decoded only when their payload validates;
byte-oriented packet side data such as A53 captions, ICC profile, HDR10+, and
LCEVC is copied into frame side data for caller-owned retention.
Packet side-data lists or payloads beyond public storage limits are treated as
malformed packet side data during decode; the compressed packet data still
decodes.
Empty `DecodePacket` or `DecodePacketFrames` calls are flush-only and do not
apply `NEW_EXTRADATA` or any other packet side data.
Duplicate packet side data follows first-entry semantics across packet
`NEW_EXTRADATA` configuration updates, scalar active-format and S12M timecode
values, structured layouts, and A53 captions, ICC profile, HDR10+, and
LCEVC byte payloads. Empty, malformed, or oversized first entries suppress
later duplicates.

## State And Ownership Boundaries

`Decoder` and `Encoder` values are stateful per stream. Use one instance per
concurrent stream, or protect shared instances with external synchronization.

| Surface | State behavior |
| --- | --- |
| `Decoder.DecodeFrames` / `DecodePacketFrames` | Retain decoder references and delayed output across stream packets; empty input flushes delayed frames. |
| `Decoder.ConfigureAVCC` | Stores avcC metadata and resets decoder picture state for the configured-AVC boundary. |
| `Decoder.DecodeAVCCFrames` / packet `NEW_EXTRADATA` / in-band SPS/PPS | Compatible parameter-set updates retain references; incompatible active SPS changes reset picture state before decoding. |
| `Decoder.Reset` | Clears stored SPS/PPS, avcC length-size, references, delayed output, and parsed slice state. |
| `Encoder.Reset` | Clears coding/reference/rate state and queued IDR state while preserving configuration and RTP callback. |
| `Encoder.SetQP` / `SetResolution` / `SetOutputFormat` | Apply validated live changes and queue an IDR boundary for the next emitted access unit. |
| Invalid encoder setters or `Reconfigure` updates | Leave configuration, queued IDR state, RTP sequence/callback state, frame number, timestamp, and references unchanged. |

## Encoder API

The encoder surface is intentionally split into a small recommended realtime
path and grouped update helpers. Prefer the explicit setters for live
controls; use `Reconfigure` only when a grouped update needs fields that do not
have a dedicated helper.

Choose the encoder surface by what the caller owns:

| Need | Use |
| --- | --- |
| Start from a supported RTP, Annex B, or AVC template | `DefaultRTPEncoderConfig`, `DefaultAnnexBEncoderConfig`, or `DefaultAVCEncoderConfig`; `DefaultRealtimeEncoderConfig` and `DefaultEncoderConfig` return the RTP template |
| Validate setup before construction | `EncoderConfig.Validate` |
| View exact setup before construction | `EncoderConfig.Normalize` |
| Read the exact live setup after accepted setters | `Encoder.Config` |
| Validate one input frame without mutating encoder state | `EncoderConfig.ValidateFrame` or `Encoder.ValidateFrame` |
| Validate input-frame retention storage | `EncoderFrame.Validate` |
| Generate SPS/PPS or recovery SEI without a live encoder | `EncoderConfig.ParameterSets` or `EncoderConfig.RecoveryPointSEIMessage` |
| Encode with encoder-owned result storage | `Encode` |
| Encode into caller-owned result byte storage where supported | `EncodeInto` |
| Request or inspect an IDR boundary | `ForceIDR`, `HandlePLI`, `HandleFIR`, and `PendingIDR` |
| Change one live control | Explicit setters such as `SetBitrate`, `SetQP`, `SetRTPMaxPayloadSize`, `SetOutputFormat`, and `SetRTPMetadata` |
| Apply a bundled low-level update | `Reconfigure` |
| Retain input/output beyond the call | `Clone` or `Append...` helpers |

Accepted encoder setup values:

| Area | Accepted values | Rejected/admission-limited values |
| --- | --- | --- |
| Input | 8-bit I420, even width/height, valid I420 crop and strides | Other pixel formats, odd I420 dimensions, invalid crop/stride geometry |
| Profile/tools | `EncoderProfileConstrainedBaseline` or `EncoderProfileBaseline`, `EncoderEntropyCAVLC`, `Transform8x8=false`, `MaxReferenceFrames=1`, `BFrames=0` | Main/High profiles, CABAC, 8x8 transform, multiple refs, B-frames |
| Runtime | `Workers=1` for deterministic mode; `Workers>1` only with `Deterministic=false` and no parallel throughput guarantee; `SliceCount` from 1 through coded macroblock count; `IntraRefresh=false` | Deterministic multi-worker encode, too many slices, enabled intra refresh |
| Rate/budget | CBR or ConstantQP, QP range 0..51, non-negative VBV/frame/slice/time budgets | VBR mode; invalid bitrate ordering, QP outside 0..51, negative budgets |
| Preset | `EncoderPresetRealtime` | Balanced/Quality presets; only `EncoderPresetRealtime` drives current mode selection |
| Output | Annex B, AVC samples, or RTP | Unknown output formats |
| Timing | `TimeBaseNum=1`; `TimeBaseDen>0`; `RTPTimestampIncrement>0`, or zero to derive cadence from `TimeBaseDen` and frame rate | Non-1 time-base numerator, non-positive time-base denominator, impossible derived RTP timestamp increment |
| RTP | packetization-mode 0 with payload size >= 2; packetization-mode 1 with payload size >= 3; STAP-A only in mode 1; DON disabled; payload type 1..127, with zero selecting the dynamic default 96 | Mode-0 STAP-A, DON/interleaved mode, payload type >127, undersized RTP payloads |

For setup-time QP, zero scalar QP fields normally select derived defaults; set
`EncoderConfig.ExplicitQP=true` when QP 0 is an intentional setup value. Runtime
`SetQP` and pointer QP fields in `EncoderReconfigure` treat zero as an explicit
value.
For RTP, `RTPPayloadType` zero selects the dynamic default 96 during config
normalization, `SetRTPMetadata`, and pointer-based `EncoderReconfigure`; use
1..127 to emit a specific payload type.
When `EncoderReconfigure` supplies both `FrameRateNum`/`FrameRateDen` and
`RTPTimestampIncrement`, the frame rate is validated and stored while the
explicit timestamp increment controls subsequent automatic RTP cadence. For
zero-derived setup and `SetFrameRate`, automatic timestamps carry fractional
frame-rate remainders forward instead of repeating the floored integer
increment forever.
Annex B and AVC configs normalize `DONDisabled=true` so later
`SetOutputFormat(EncoderOutputRTP)` uses admitted RTP defaults; direct RTP
configs with `DONDisabled=false` return `ErrUnsupported`.

`EncoderConfig` owns encoded crop/color metadata. `Crop` and `Color` are written
into SPS/VUI headers from the normalized encoder config. `EncoderFrame.Color` is
validated input metadata and does not rewrite SPS/VUI per frame.

```go
cfg := goh264.DefaultRealtimeEncoderConfig(640, 480)
cfg.TargetBitrate = 800_000
cfg.MaxBitrate = 1_000_000
cfg.SliceCount = 2
cfg, err := cfg.Normalize()
if err != nil {
	log.Fatal(err)
}
headers, err := cfg.ParameterSets()
if err != nil {
	log.Fatal(err)
}
headerAVCC, err := headers.AVCCData()
if err != nil {
	log.Fatal(err)
}
_ = headerAVCC
sei, err := cfg.RecoveryPointSEIMessage(0)
if err != nil {
	log.Fatal(err)
}
_ = sei.AnnexB
_ = sei.AVC

enc, err := goh264.NewEncoder(cfg)
if err != nil {
	log.Fatal(err)
}
must := func(err error) {
	if err != nil {
		// Invalid controls return ErrInvalidData; unsupported tools return ErrUnsupported.
		log.Fatal(err)
	}
}
enc.HandlePLI() // queues the next frame as an IDR request
must(enc.SetBitrate(700_000, 900_000))
must(enc.SetRateControl(goh264.EncoderRateControlCBR))
must(enc.SetVBVBufferSize(1_000_000))
must(enc.SetFrameDropMode(goh264.EncoderFrameDropToBitrate))
must(enc.SetQP(26, 10, 42))
must(enc.SetFrameRate(30, 1))
must(enc.SetRTPTimestampIncrement(3000))
must(enc.SetGOP(60, 60))
must(enc.SetResolution(640, 480))
must(enc.SetDeblockMode(goh264.EncoderDeblockDisabled))
must(enc.SetRTPMaxPayloadSize(1200))
must(enc.SetLimits(goh264.EncoderLimits{
	MaxFrameSize:    0, // disable the access-unit byte budget
	SliceMaxBytes:   0, // disable the per-slice byte budget
	MaxEncodeTimeUS: 0, // disable the late-frame time budget
}))
must(enc.SetPreset(goh264.EncoderPresetRealtime))
must(enc.SetSliceCount(2))
must(enc.SetSPSPPSMode(goh264.EncoderSPSPPSOutOfBand))
must(enc.SetSPSPPSBeforeIDR(false))
must(enc.SetIntraRefresh(false)) // true returns ErrUnsupported
must(enc.SetRecoveryPointSEI(true))
must(enc.SetRTPPacketizationMode(goh264.EncoderRTPPacketizationSingleNAL, false))
must(enc.SetRTPMetadata(110, 0x11223344))
must(enc.SetOutputFormat(goh264.EncoderOutputAVC)) // queues an IDR boundary
headers, err = enc.ParameterSets() // SPS/PPS NALs plus Annex B and avcC headers
if err != nil {
	log.Fatal(err)
}
headerAVCC, err = headers.AVCCData()
if err != nil {
	log.Fatal(err)
}
_ = headerAVCC
sei, err = enc.RecoveryPointSEI(0) // Annex B/AVC recovery-point SEI NALs
if err != nil {
	log.Fatal(err)
}
_ = sei.AnnexB
_ = sei.AVC
liveCfg := enc.Config()
y := make([]byte, liveCfg.StrideY*liveCfg.Height)
cb := make([]byte, liveCfg.StrideCb*(liveCfg.Height/2))
cr := make([]byte, liveCfg.StrideCr*(liveCfg.Height/2))
pts := int64(0)
frame := enc.I420Frame(y, cb, cr, pts)
// PTS zero is explicit; set frame.TimestampMode = goh264.EncoderTimestampAuto
// for encoder-managed RTP time.
must(liveCfg.ValidateFrame(frame))
must(enc.ValidateFrame(frame))
out, err := enc.Encode(frame) // admitted path: IDR/P-skip/P16x16/residual-P/P IntraPCM
if err != nil {
	log.Fatal(err)
}
if out.Dropped {
	// Realtime budget drop: no bytes or RTP packets were emitted.
}
accessUnit, err := out.AccessUnitData()
if err != nil {
	log.Fatal(err)
}
nal0, err := out.NALData(0) // clipped raw NAL bytes from EncodedFrame.Data
if err != nil {
	log.Fatal(err)
}
_ = accessUnit
_ = nal0
owned, err := out.Clone()   // deep-owned snapshot for async retention
if err != nil {
	log.Fatal(err)
}
_ = owned
must(enc.Reset()) // clear encoder coding state, keep config/callback
```

For RTP output, set the RTP output format before encoding and use RTP packet
helpers for network send. Access-unit helpers remain available for local
inspection of the Annex B view:

```go
must(enc.SetOutputFormat(goh264.EncoderOutputRTP))
must(enc.SetRTPPacketizationMode(goh264.EncoderRTPPacketizationSingleNAL, false))
must(enc.SetRTPMetadata(110, 0x11223344))

out, err := enc.Encode(frame)
if err != nil {
	log.Fatal(err)
}
packet0, err := out.RTPPacketData(0)
if err != nil {
	log.Fatal(err)
}
payload0, err := out.RTPPayloadData(0)
if err != nil {
	log.Fatal(err)
}
_ = packet0
_ = payload0
```

The admitted encoder contract is deliberately narrow, and these are the pieces
with the strongest public API coverage for integration work:

- `EncoderConfig.Validate` reports whether setup can be accepted without
  returning normalized values. Use `EncoderConfig.Normalize` when the caller
  needs the exact setup.
- `EncoderConfig.Normalize` exposes the exact validated configuration stored by
  `NewEncoder`.
- `EncoderConfig.LevelIDC = 0` selects the smallest admitted Baseline level
  that fits the normalized geometry, frame rate, reference count, and maximum
  bitrate. Explicit nonzero levels are rejected when they are too small for
  that envelope.
- `Encoder.Config` returns the exact normalized live configuration after
  accepted runtime setters and `Reconfigure` updates.
- `EncoderConfig.ParameterSets` and `EncoderConfig.RecoveryPointSEIMessage`
  generate caller-owned helper surfaces without constructing a live encoder.
  Header and SEI results include error-returning `Clone` and append helpers for
  validating and retaining individual byte surfaces in caller-managed buffers.
- `EncoderConfig.ValidateFrame` and `Encoder.ValidateFrame` validate frame shape
  before bitstream work. Invalid frames return empty output without advancing
  RTP sequence, callback, frame-number, timestamp, or reference state. The next
  valid input resumes as P-skip, or as the queued IDR when an IDR request was
  already pending.
- `SetLimits` updates the access-unit byte budget, per-slice byte budget, and
  late-frame time budget atomically; passing zero disables the corresponding
  budget. `SetMaxFrameSize`, `SetSliceMaxBytes`, and `SetMaxEncodeTimeUS`
  remain explicit single-limit setters. For grouped updates through
  `EncoderReconfigure`, prefer `Limits` when budget updates must be applied
  atomically with other runtime controls or explicitly set a budget to zero.
  `MaxFrameSizeLimit`, `SliceMaxBytesLimit`, and `MaxEncodeTimeUSLimit` are
  zero-capable single-budget update fields. The encoder API-surface gate covers
  scalar, pointer, and grouped budget reconfigure precedence and rollback.
- `SetRateControl`, `SetVBVBufferSize`, `SetFrameDropMode`, `SetQP`,
  `SetFrameRate`, `SetRTPTimestampIncrement`, `SetGOP`, `SetResolution`,
  `SetDeblockMode`, `SetRTPMaxPayloadSize`, `SetPreset`, `SetSliceCount`,
  `SetSPSPPSMode`, `SetSPSPPSBeforeIDR`, `SetIntraRefresh`,
  `SetRecoveryPointSEI`, `SetOutputFormat`, `SetRTPPacketizationMode`, and
  `SetRTPMetadata` cover admitted control, budget, geometry, output, cadence,
  packetization, and RTP header changes without constructing an
  `EncoderReconfigure` value. `SetQP`, `SetResolution`, and output-format
  changes queue an IDR boundary after a valid update. RTP packetization and
  metadata are validated even before switching output to RTP, so invalid RTP
  state cannot be parked behind Annex B or AVC output. `SetIntraRefresh(true)`
  returns `ErrUnsupported` because intra refresh is not part of the current
  encoder contract.
- `EncoderReconfigure` remains the grouped low-level update surface for
  bundled multi-field changes, grouped `Limits`, and explicit force-IDR
  requests. Zero scalar fields in `EncoderReconfigure` mean unchanged; use
  pointer fields, grouped `Limits`, or dedicated setters when zero is the value
  to apply. `FrameRateNum`/`FrameRateDen` and `Width`/`Height` must be supplied
  as pairs. When `Limits` is non-nil, it is applied after the individual budget
  fields and their pointer zero-value forms.
- `EncoderFrame.Validate` checks input-frame plane storage before retention;
  `EncoderFrame.Clone` uses the same checks and returns a deep-owned input
  snapshot for retry queues or async handoff. Use `EncoderConfig.ValidateFrame`
  or `Encoder.ValidateFrame` for config-specific encode-shape validation.
- Parameter-set, SEI, encoded-frame, NAL, access-unit, RTP packet, and RTP
  payload helpers use one byte-access pattern: direct `SPSData`, `PPSData`,
  `AnnexBData`, `AVCCData`, `NALData`, `AVCData`, `AccessUnitData`,
  `RTPPacketData`, `RTPPayloadData`, `PacketData`, and `PayloadData` methods
  return checked clipped views; explicit append forms copy into caller-owned
  retention buffers; `Clone` forms produce async snapshots.
  `EncoderParameterSets.Validate` and `EncoderSEI.Validate` check public
  storage sizes before retention or async handoff; `Clone` uses the same checks
  before copying. `AppendSPS`, `AppendPPS`, `AppendAnnexB`, `AppendAVCC`,
  `AppendNAL`, `AppendAVC`, `AppendNALData`, `AppendAccessUnitData`,
  `AppendRTPPacketData`, `AppendRTPPayloadData`, `AppendPacketData`, and
  `AppendPayloadData` validate their source surfaces and caller-managed append
  buffers.
  Invalid or overflowed-destination append calls return the original destination
  unchanged. If a caller-managed append destination overlaps the helper source
  bytes, the helpers return isolated output storage instead of aliasing the
  source. `EncodedFrame.Validate` checks public result shape, frame-level
  keyframe/IDR metadata, RTP packet-list metadata, payload byte parity against
  the access-unit NAL list, and FU-A fragment start/continuation/end
  consistency before retention or async handoff.
  `EncodedFrame.Clone` uses the same checks and rejects dropped results that
  still carry emitted byte, NAL, or RTP packet storage, non-RTP results that
  carry RTP packets, and RTP results that lack RTP packets.
- `EncodedFrame.OutputFormat` records the emitted result format, including
  dropped frames, so callers do not need to infer format from packet presence.
  Caller-constructed `EncodedFrame` values must set `OutputFormat` and keep RTP
  packet storage matched to that format before using access-unit/RTP helper
  methods, `Validate`, or `Clone`.
- `AccessUnitRange` and `AccessUnitFormat` make the access-unit byte range and
  access-unit container explicit; RTP results report an Annex B access-unit view
  while RTP packet bytes stay under `RTPPackets`.
- For RTP output, send `RTPPackets`, `RTPPacketData`, or `RTPPayloadData`.
  `EncodedFrame.Data` is retained only as an Annex B access-unit view for local
  inspection through `AccessUnitData` and `NALData`. Packet-level helpers
  `PacketData`, `PayloadData`, `AppendPacketData`, `AppendPayloadData`,
  `Validate`, and `Clone` validate the encoder-emitted 12-byte RTP header shape,
  exported packet metadata, RTP payload view, and admitted single-NAL, STAP-A,
  and FU-A payload syntax before returning packet bytes. STAP-B, MTAP, FU-B,
  nested STAP-A units, and FU-A fragments whose reconstructed NAL type is
  another packetization unit are rejected. `PacketData`, payload helpers, packet
  validation, and packet clones require `Payload` to be exactly `Data[12:]`.
- Overflowed caller-owned `EncodeInto` destination growth is rejected across
  Annex B, AVC, and RTP without consuming queued IDR state or advancing
  RTP/callback state. The same hard-error path preserves P-frame reference and
  frame-number state before the next P-skip.

Emitted frame types in the guarded encoder subset are IDR IntraPCM,
identical-reference P-skip, exact macroblock-aligned frame-wide or
per-macroblock P16x16 no-residual, bounded luma-DC/chroma-only/combined
luma-chroma residual-P, and changed-frame P IntraPCM. Output can be split into
configured multi-slice VCL NALs. Exact P16x16 is admitted for
disabled-deblock frames and for
chroma-aligned uniform-motion enabled/slice-boundary deblock frames, including
multi-macroblock frames. Guarded mixed-vector and odd-pixel deblock cases fall
back to P IntraPCM recovery across Annex B, configured AVC, RTP reassembly, and
RTP packetization-mode 0 single-NAL output. Changed-frame P IntraPCM recovery
pictures carry recovery-point SEI when enabled.

RTP output covers:

- payload bytes plus complete RTP packet bytes;
- `SetRTPPacketCallback` per-packet metadata callbacks for RTP output; Annex B
  and AVC output return data through `EncodedFrame` helpers instead;
- packetization-mode 0 single-NAL output;
- packetization-mode 1 FU-A/STAP-A output, including small-payload STAP-A
  fallback to non-aggregated mode-1 packets;
- accurate fallback-IDR and post-fallback P-skip callback payload metadata;
- RTP packet storage isolated from `EncodedFrame.Data`;
- public `EncodedFrame.RTPPacketData`, `EncodedFrame.RTPPayloadData`, and
  packet-level `EncoderRTPPacket` byte helpers;
- caller-owned append helpers for access-unit, NAL, RTP packet, and RTP payload
  bytes, including isolated overlapping source/destination appends and unchanged
  destinations on invalid appends;
- deep-owned `EncodedFrame.Clone` snapshots for retained results, with malformed
  metadata and overflowed public result storage rejected;
- optional per-packet callback metadata for mode 0/1 IDR/P-frame single-NAL
  packets, including multi-slice IDR, P-skip, exact P16x16, odd-pixel constant
  chroma, and P IntraPCM fallback rows;
- callback packet storage isolated from returned RTP packets;
- mode-0 oversize rejection with live-state rollback for queued-IDR and P-frame
  paths;
- RTP timestamping uses `EncoderFrame.PTS` directly, including zero. Set
  `EncoderFrame.TimestampMode = EncoderTimestampAuto` to use the encoder RTP
  timeline advanced by `EncoderFrame.Duration` or config-derived cadence.
  Explicit positive `RTPTimestampIncrement` values remain fixed increments.

SPS/PPS cadence modes separate in-band keyframe headers, out-of-band headers,
and every-IDR emission. Runtime reconfiguration can switch output format and RTP
packetization controls, including RTP-to-configured-AVC forced IDR/P-skip decode
with out-of-band parameter sets and paused RTP packets/callbacks, and
configured-AVC-to-RTP forced IDR/P-skip packetization with sequence/callback
start. Rejected rate-control, QP, GOP, deblock, output, and RTP updates preserve
live state.

Bitrate-budget drops use the configured `MaxBitrate` refill rate and
`VBVBufferSize` burst capacity, then surface through `EncodedFrame.Dropped` when
`FrameDropToBitrate` is active. Caller-buffer `EncodeInto` drops return empty
output without RTP packets or callbacks before the next valid frame resumes as
P-skip.

Outside the documented realtime encoder contract: motion search beyond the
bounded 8-pixel exact macroblock-aligned inter path, broader quantized residual
coding beyond the guarded luma-DC/chroma-only/combined luma-chroma paths, and
adaptive rate-control feedback.

## Decoder Coverage At A Glance

| Area | Status |
| --- | --- |
| Annex B bytestream | Supported on green corpus rows |
| AVC length-prefixed packets | Supported, including explicit 1-, 2-, 3-, and 4-byte NAL length sizes |
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

- `h264-real-vector-strict.sh` runs the green public-vector oracle set,
  including expected decode-error rows, and excludes only rows listed
  in the failure ledger.
- `FailureLedgerFreshness` runs only known-red rows when the ledger is populated
  and requires each failure class/detail to remain current.
- `FailureMatrix` runs the full 225-row manifest and requires all 225 rows to
  match oracle output.
- `TestH264DecoderTDDContractClassifiesEveryImportedPublicVector` runs in
  normal `go test ./tests` and fails if any imported public ref is not
  classified as executable or explicitly excluded.
- `h264-real-vector-upstream-audit.sh` fetches the pinned FFmpeg source and
  verifies that the checked-in inventory matches all decoder-facing upstream
  H.264 FATE sample references, except documented non-decoder rows, and that
  public-vector count claims in the quality docs match the checked-in
  manifests.
  Normal `go test ./tests` also checks that every imported public ref is either
  represented by the manifest or listed in the exclusion file.

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

## Trust And Verification

Production use should be backed by a fresh quality-evidence pass proving:

- `scripts/h264-quality-evidence.sh` is green as the combined decoder and
  admitted-encoder quality gate.
- `scripts/h264-decoder-quality-evidence.sh` is green, including
  decoder API-surface gates, decoder output-ownership gates,
  ref-modification gates, delayed-output rollback gates, and
  native/FFmpeg oracle smoke gates.
- `go vet ./...` is green.
- `go test ./...` is green.
- `go test -race ./...` is green.
- `scripts/h264-real-vector-strict.sh` is green.
- `GOH264_REAL_VECTOR_MATRIX=1 GOH264_CORPUS_FETCH=1 go test ./tests -run '^TestH264RealVectorFailureMatrix$' -count=1 -v` is green.
- `scripts/h264-real-vector-upstream-audit.sh` represents all pinned
  decoder-facing FFmpeg H.264 FATE sample references in
  `testdata/h264/realvectors/upstream-inventory.jsonl`, except documented
  non-decoder exclusions, and quality-doc public-vector counts match the
  checked-in manifests.
- `scripts/h264-decoder-fuzz-smoke.sh` is green for the bounded public decoder
  no-panic fuzz target.
- Known-red rows, if any, are current in `testdata/h264/realvectors/failures.jsonl`.
- `scripts/h264-real-vector-quality-alloc.sh` is green with the checked-in Go
  allocation canary budget.
- `scripts/h264-benchstat-canary.sh` runs decoder and admitted encoder rows
  with stable `-benchmem` output for trend comparison. `GOH264_BENCHSTAT_TIME`
  controls the effective `-benchtime`; `GOH264_BENCHSTAT_BENCHTIME` is also
  accepted when `GOH264_BENCHSTAT_TIME` is unset.
- `scripts/h264-performance-evidence.sh` creates the local performance bundle
  with JSON benchmark output plus CPU/heap profiles.
- `scripts/h264-encoder-quality-evidence.sh` is green for the admitted
  realtime/WebRTC encoder vet, contract, API-surface, output-ownership,
  bitstream-oracles, residual-boundary, writer, allocation, and benchmark gates. This runner
  requires `ffmpeg` for admitted encoder bitstream-oracle rows; set
  `GOH264_FFMPEG_BIN` when the oracle binary is not named `ffmpeg`.
- Allocation and performance evidence expectations and local bundle paths are
  described in [docs/production-readiness.md](docs/production-readiness.md);
  checked-in reviewed profile artifacts remain pending there.
- The documented encoder contract is limited to the listed realtime subset.
  Broader support requires matching motion-search, residual, rate-control,
  packetizer, control, and oracle evidence in
  [docs/encoder-webrtc-roadmap.md](docs/encoder-webrtc-roadmap.md).
- The source-truth and translation-ledger docs match the committed tests.

The combined quality-evidence runner writes logs under
`.artifacts/h264-full-quality-evidence/` by default, drives the race, decoder,
and admitted encoder runners, and requires a clean worktree unless
`GOH264_FULL_QUALITY_ALLOW_DIRTY=1` is set for diagnostics.
Diagnostic dirty-worktree runs are labeled `worktree-clean: allowed-dirty` in
the evidence summary rather than `worktree-clean: pass`.
The decoder quality-evidence runner writes logs under
`.artifacts/h264-quality-evidence/` by default and fails while
`testdata/h264/realvectors/failures.jsonl` contains known-red rows unless
`GOH264_QUALITY_ALLOW_KNOWN_RED=1` is set for a local diagnostic run. It
also requires a clean worktree unless `GOH264_QUALITY_ALLOW_DIRTY=1` is set for
diagnostics. It requires `ffmpeg`, `ffprobe`, `cc`, and the pinned upstream
oracle sources under `.upstream/ffmpeg-n8.0.1`; run
`scripts/fetch-upstream.sh` when that upstream tree is missing.
The encoder quality-evidence runner writes logs under
`.artifacts/h264-encoder-quality-evidence/` and likewise requires a clean
worktree unless `GOH264_ENCODER_QUALITY_ALLOW_DIRTY=1` is set for diagnostics.

## Performance

Benchmarks are only useful after the Trust And Verification gates for the same
checkout pass.

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
flags, comparison lane, raw MD5 parity, oracle quality status, Go allocation
totals plus per-iteration/per-frame allocation rates, and FFmpeg-vs-Go
peer quality status. Diagnostic mode also treats expected `decode-error` rows
as oracle rows and marks them green only when the decoder error matches
`expected_error`.

Use `-max-go-alloc-bytes-per-iter` and `-max-go-allocs-per-iter` to turn those
Go allocation rates into failing benchmark budgets. The real-vector benchmark
script exposes the same gate through `GOH264_BENCH_MAX_GO_ALLOC_BYTES_PER_ITER`
and `GOH264_BENCH_MAX_GO_ALLOCS_PER_ITER`.
Use `-cpuprofile` and `-memprofile` to write Go CPU and heap profiles around
the oracle-checked benchmark run; `GOH264_BENCH_CPU_PROFILE` and
`GOH264_BENCH_MEM_PROFILE` forward those paths through the real-vector
benchmark script.
For repeated `go test -benchmem` samples covering one-shot Annex B decode,
stateful Annex B access-unit streaming, isolated raw-output export, and selected
admitted realtime encoder IDR/P-frame Annex B/AVC/RTP paths, including RTP
P-IntraPCM and packetization-mode 0 IDR/P-frame rows, suitable for `benchstat`,
run:

```sh
scripts/h264-benchstat-canary.sh
```

Use `GOH264_BENCHSTAT_COUNT` and `GOH264_BENCHSTAT_TIME` to control sample
count and `-benchtime`; `GOH264_BENCHSTAT_BENCHTIME` is accepted when
`GOH264_BENCHSTAT_TIME` is unset.

To create a local quality-evidence bundle with benchstat samples, the JSON
real-vector benchmark report, CPU/heap profiles, and run metadata:

```sh
scripts/h264-performance-evidence.sh canl4
```

The bundle is written under `.artifacts/h264-performance-evidence/` by default;
override the destination with `GOH264_PERF_DIR`.

Performance status is intentionally conservative: the benchmark harness exists
and rejects quality drift before timing, and public raw-output helpers have
caller-buffer zero-allocation guards. A checked-in public-vector allocation
canary, profile-output hooks, a benchstat-compatible decoder/encoder canary,
and a local performance-evidence bundle runner exist. Missing evidence includes
checked-in reviewed profile artifacts, a larger performance corpus, and an
in-process libavcodec baseline. For throughput-sensitive use, back decisions
with the gates in [docs/production-readiness.md](docs/production-readiness.md)
and the evidence artifacts they describe.

## Project Layout

| Path | Purpose |
| --- | --- |
| `decoder.go` | Public decoder API, frames, raw output helpers, side-data mapping |
| `internal/h264/` | Source-shaped parser, syntax, prediction, transform, DPB, reconstruct, and loop-filter code |
| `tests/decoder_*_test.go` | Public and package-level fixture/oracle coverage |
| `testdata/h264/corpus/` | Small local corpus manifest |
| `testdata/h264/realvectors/` | Public FFmpeg FATE manifest, exclusions, and known-red ledger |
| `scripts/` | Upstream fetch, oracle probes, public-vector gates, diagnostics, benchmarks |
| `cmd/goh264bench/` | JSON benchmark and FFmpeg comparison CLI |
| `docs/source-truth.md` | Compact parity snapshot |
| `docs/translation-ledger.md` | Upstream-to-Go translation ledger |
| `docs/production-readiness.md` | Verification and performance gates |
| `docs/high-bitdepth-roadmap.md` | High-bit-depth parity plan |
| `docs/encoder-webrtc-roadmap.md` | Realtime/WebRTC encoder target, controls, and gates |

## Contributing

Work in closed topics:

- Add or keep the failing vector first.
- Port the smallest source-shaped FFmpeg behavior that should make it green.
- Run the focused oracle, then the relevant public-vector gate.
- Keep known-red rows in `failures.jsonl` until they genuinely match oracle
  output, then update the failure ledger atomically with the fix.
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
