# goh264

Pure-Go H.264 codec, decoder-first and source-shaped from FFmpeg `libavcodec`.

`goh264` currently implements an active Go port of the FFmpeg `n8.0.1` H.264
decoder path, pinned at `894da5ca7d742e4429ffb2af534fcda0103ef593`. Encoder
support is now in scope for realtime/WebRTC use, tracked in
[docs/encoder-webrtc-roadmap.md](docs/encoder-webrtc-roadmap.md). The encoder
API currently exposes a tested realtime/WebRTC control contract and valid
SPS/PPS parameter-set plus recovery-point SEI generation. The first admitted
bitstream paths cover 8-bit I420 Constrained Baseline IDR IntraPCM, P-skip for
identical references, and changed-frame P IntraPCM recovery pictures with
Annex B, AVC, configured multi-slice output, RTP packetization-mode 0
single-NAL output, and RTP packetization-mode 1 output, proved by local decode,
FFmpeg rawvideo decode, recovery-point side data, RTP mode-0 reassembly, RTP
FU-A reassembly, STAP-A parameter-set aggregation tests, and encode-time
`MaxFrameSize`/`SliceMaxBytes` budget guards with hard-error and dropped-frame
paths plus runtime RTP/output, rate-control, QP, GOP/IDR, and deblock
reconfiguration gates.
The goal is not a loose rewrite: internal codec paths keep upstream state
machines, syntax handling, math, and edge cases recognizable, then prove
behavior against oracle vectors.

- **Pure Go decoder path** - no cgo and no Go module dependencies.
- **Realtime/WebRTC encoder scope** - tested encoder controls cover explicit
  bitrate, latency, keyframe, packetization, profile/level, runtime
  reconfiguration controls including rate-control/QP/GOP/deblock updates,
  out-of-band SPS/PPS/avcC headers, and crop-aware SPS/encoded visible output
  plus recovery-point SEI packaging and
  `SliceCount`-backed multi-slice output plus frame/slice byte-budget reject
  and realtime drop guards, with first IDR IntraPCM, P-skip, and P IntraPCM
  Annex B/AVC/RTP output paths.
- **Annex B and AVC input surfaces** - automatic packet splitting, explicit
  Annex B / length-prefixed AVC APIs, and AVC decoder configuration records.
- **Raw frame output** - `Frame` exposes Y/Cb/Cr planes, crop, strides, VUI
  fields, high-bit-depth planes, and raw YUV helpers.
- **Harness-first parity** - public FFmpeg FATE and auxiliary H.264 vectors are
  imported as an explicit inventory, executable where decoder-facing, with a
  red ledger kept for any future known-failing rows instead of hiding them.
  The `tests` package contains the all-at-once decoder TDD contract: every
  imported public ref must be in the executable manifest or in the documented
  exclusion list.
- **Active port, not v1** - the public decoder-compliance matrix is green, with
  broader unselected codec lanes still guarded.
- **Release evidence over claims** - no production tag is planned until the
  public vector gates, upstream audit, allocation/performance evidence, and
  translation ledger all agree.

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
| Imported public H.264 vector refs | 226 |
| Pinned FFmpeg FATE refs in imported inventory | 224 |
| Selected public H.264 vectors | 225 |
| Green oracle rows | 225 |
| Known-red rows in `failures.jsonl` | 0 |
| Explicitly excluded upstream H.264-ish rows | 1 |

The selected manifest represents 225 imported decoder-facing refs; the remaining
imported ref is the documented non-H.264 MKV exclusion. No known-red
public-vector rows currently remain. The executable ledger at
`testdata/h264/realvectors/failures.jsonl` stays in place for future red rows
and is checked by the freshness/matrix gates when populated.
`TestH264DecoderTDDContractClassifiesEveryImportedPublicVector` is the always-on
contract that keeps the inventory, manifest, exclusions, and failure ledger in
lockstep.

Encoder status: `DefaultEncoderConfig`, `NewEncoder`, `ParameterSets`,
`RecoveryPointSEI`, `Encode`/`EncodeInto`, PLI/FIR/force-IDR,
bitrate/framerate/payload/slice reconfiguration, runtime rate-control, QP,
frame-drop, GOP/IDR, deblock, SPS/PPS cadence modes, runtime output-format and
RTP packetization reconfiguration, and the WebRTC control fields are public and
covered by
`tests/encoder_webrtc_controls_test.go`. Valid 8-bit I420 constrained-baseline
realtime configs are admitted as control state; SPS/PPS parameter sets, Annex B
sequence headers, avcC records, crop metadata,
in-band/out-of-band/every-IDR cadence, and recovery-point SEI Annex B/AVC NAL
surfaces are generated and parser-proved.
`Encode`/`EncodeInto` now emit source-shaped IDR IntraPCM access units for
Annex B, AVC, configured `SliceCount` multi-slice VCL output, RTP
packetization-mode 0 single-NAL packets, and RTP packetization-mode 1,
including FU-A fragmentation and STAP-A parameter-set aggregation,
payload-type/SSRC/sequence metadata, full RTP packet headers, marker-bit
boundaries, oversize mode-0 rejection, and optional RTP packet callbacks with
packet index/count, frame timing, payload form, NAL type/count, FU-A start/end,
and parameter-set metadata. RTP timestamps honor explicit frame PTS and advance
zero-PTS frames from frame duration or `RTPTimestampIncrement`. `MaxFrameSize`
and `SliceMaxBytes` are enforced before frame/reference/packet state advances:
`FrameDropDisabled` preserves the hard-error path, while
`FrameDropToBitrate` returns `EncodedFrame.Dropped` without emitted bytes or RTP
packets and advances the RTP timestamp timeline.
Runtime reconfiguration now covers SPS/PPS cadence, Annex B/AVC/RTP output
format, RTP packetization mode 0/1, STAP-A aggregation, payload type, SSRC, and
custom RTP timestamp increments plus rate-control mode, VBV size,
initial/min/max QP, frame-drop mode, GOP/IDR cadence, and deblock mode without
mutating state on invalid updates. QP updates queue an IDR/PPS refresh so the
emitted parameter sets match the active slice QP.
A combined RTP/Annex B/RTP control-loop stress test now proves QP refresh, late
drop recovery, packet metadata retargeting, paused RTP sequence/callback state
while no RTP packets are emitted, and local decode after RTP re-entry.
Identical frames after a decoded reference can use a guarded CAVLC P-skip slice
across disabled, enabled, and slice-boundary deblock controls; changed frames
can use a guarded CAVLC P IntraPCM slice in the same admitted deblock scope
with recovery-point SEI emission when enabled,
while forced keyframe requests still emit IDR; cropped I420 input emits SPS
crop metadata and local/FFmpeg decode sees the cropped visible frame. Internal
writer primitives cover raw bit/Exp-Golomb
writing, RBSP trailing bits, EBSP escaping, Annex B/AVC NAL packaging, AVC
configuration records, baseline SPS/PPS, recovery-point SEI syntax, and the
first Baseline IDR, P-skip, and P IntraPCM slice payloads. Motion-search P
prediction, residual CAVLC coding, rate-control feedback, and realtime
allocation/performance evidence remain pending.

Green coverage includes compact Baseline/Main/High conformance rows, selected
FRext and high-bit-depth fixtures, High12/High14 CAVLC and CABAC B deblock
rows including implicit and explicit weighted B, High12/High14 CAVLC/CABAC 4:2:2/4:4:4
unweighted I/P plus CAVLC/CABAC luma-only/luma+chroma weighted-P no-deblock, frame-deblock,
and slice-boundary rows, High10 4:2:2/4:4:4
CAVLC/CABAC implicit and explicit weighted B frame and slice-boundary deblock plus weighted-P frame and slice-boundary rows, I/P/B slices, CAVLC and
CABAC, weighted and direct motion paths including High12/High14 CAVLC/CABAC
direct-sub residual, deblock modes, selected field/PAFF/MBAFF rows
including High10 4:2:2/4:4:4 unweighted top/bottom field I/P/B guards for deblock modes 0/1/2,
High10 4:2:2/4:4:4 weighted-B and weighted-P top/bottom field guards for deblock modes 0/1/2,
internal High12/High14 4:2:2/4:4:4 weighted-B plus luma-weighted, luma+chroma-weighted, and source-normalized chroma-only weighted-P top/bottom field guards for deblock modes 0/1/2,
internal High9 4:2:2/4:4:4 weighted-P frame-deblock modes 0/1 plus weighted-P, unweighted-B, implicit/explicit weighted-B, and I/P slice-boundary mode-2 validation and loop-filter guards,
public High12/High14 4:2:0 frame-MBAFF CAVLC IntraPCM, P-skip, and field-coded/frame-coded P16x16/P16x8/P8x16/P8x8 no-residual, luma-residual, and luma+chroma-residual rows plus P-skip and field-coded/frame-coded P16x16/P16x8/P8x16/P8x8 mode-1/mode-2 deblock rows,
reinit metadata rows, lossless High444
transform-bypass rows, configured AVC surfaces, container-extracted Annex B
vectors, and SEI side-data surfaces.
Public malformed-input safety coverage includes deterministic corrupt packet
rows plus a bounded no-panic fuzz target over Annex B, AVC, configured AVC,
auto-detect, and packet side-data decode surfaces.
Stateful damaged-packet recovery guards prove configured AVC, AVC with a
configuration record, packet `NEW_EXTRADATA`, and auto-detected Annex B
valid-damaged-valid sequences return an error for the damaged packet without
poisoning the next valid decode, and valid frames decoded before a later
damaged slice in the same packet are returned alongside that error, including
the sole valid frame on single-frame decode helpers and delayed B-frame prefix
output from configuration-record one-shot decode. Packet
`NEW_EXTRADATA` recovery also proves malformed AVC and Annex B extradata is
non-fatal, does not replace the last good decoder configuration or reference
state, and lets the current valid packet decode. Malformed in-band SPS/PPS NALs
are skipped without replacing the last good parameter sets on configured AVC and
mixed configured-AVC/Annex B decode paths.

Still guarded: unselected MBAFF/PIC-AFF/PAFF motion paths, broader high-bit-depth
field/inter streams beyond the public High12/High14 frame-MBAFF IntraPCM/P-skip and field-coded/frame-coded P16x16/P16x8/P8x16/P8x8 no-residual, luma-residual, luma+chroma-residual, and P16x16/P16x8/P8x16/P8x8 mode-1/mode-2 deblock rows plus internal High10/High12/High14 field weighted-B/weighted-P guard matrices,
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
frames, err := dec.DecodeConfiguredAVCFrames(nil)      // delayed configured-AVC output
frames, err := dec.DecodeAVCFramesWithConfigurationRecord(avcc, packet)
frames, err := dec.DecodeAVCFramesWithConfigurationRecord(avcc, nil) // delayed config-record AVC output
frames, err := dec.FlushDelayedFrames()                // delayed B-frame output
```

Single-frame helpers (`Decode`, `DecodePacket`, `DecodeAnnexB`, `DecodeAVC`,
`DecodeConfiguredAVC`, and `DecodeAVCWithConfigurationRecord`) return
`ErrUnsupported` when a packet produces zero or multiple frames. If a damaged
packet produces exactly one valid frame before a later decode error, the helper
returns that frame with the error. For stream processing, prefer `DecodeFrames` or
`DecodePacketFrames`; they retain decoder reference state across packets and
flush delayed output when called with empty data. `DecodeConfiguredAVCFrames`
does the same after an AVC configuration record has been parsed. Annex B
access-unit streams use the same retained reference and delayed B-frame output path.
`DecodeAVCFramesWithConfigurationRecord` updates the decoder's AVC
configuration without resetting retained references, then drains delayed output
for the supplied AVC packet. Passing an empty AVC packet with a configuration
record drains delayed output without reporting an invalid packet.

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

The encoder API is a WebRTC/realtime control contract while implementation is
still landing:

```go
cfg := goh264.DefaultEncoderConfig(640, 480)
cfg.TargetBitrate = 800_000
cfg.MaxBitrate = 1_000_000
cfg.SliceCount = 2

enc, err := goh264.NewEncoder(cfg)
if err != nil {
	// Invalid controls return ErrInvalidData; unsupported future tools return ErrUnsupported.
}
enc.HandlePLI() // queues the next frame as an IDR request
err = enc.SetRTPMaxPayloadSize(1200)
mode0 := goh264.EncoderRTPPacketizationSingleNAL
stapa := false
err = enc.Reconfigure(goh264.EncoderReconfigure{
	RTPPacketizationMode: &mode0,
	STAPA:                &stapa,
	ForceIDR:             true,
})
enc.SetRTPPacketCallback(func(pkt goh264.EncoderRTPPacket, meta goh264.EncoderRTPPacketMetadata) {
	// Optional per-packet WebRTC metadata hook.
})
headers, err := enc.ParameterSets() // SPS/PPS NALs plus Annex B and avcC headers
sei, err := enc.RecoveryPointSEI(0) // Annex B/AVC recovery-point SEI NALs
out, err := enc.Encode(frame)       // admitted path: IDR/P-skip/P IntraPCM
if out.Dropped {
	// Realtime budget drop: no bytes or RTP packets were emitted.
}
```

`Encode` and `EncodeInto` validate frame shape and caller-owned output buffers,
then emit the admitted IDR IntraPCM, identical-reference P-skip, or
changed-frame P IntraPCM frame path, optionally split into configured
multi-slice VCL NALs. Changed-frame P IntraPCM recovery pictures carry
recovery-point SEI when enabled. RTP output includes payloads plus complete RTP
packet bytes, packetization-mode 0 single-NAL output, packetization-mode 1
FU-A/STAP-A output, optional per-packet callback metadata, and automatic
timestamp progression when frames omit explicit PTS. SPS/PPS cadence modes now
separate in-band keyframe headers, out-of-band headers, and every-IDR emission,
and runtime reconfiguration can switch output format and RTP packetization
controls plus rate-control/QP/GOP/deblock controls while preserving state on
rejected updates. Bitrate-budget drops surface through `EncodedFrame.Dropped`
when `FrameDropToBitrate` is active.
Motion-search inter prediction, quantized residual coding, and rate-control
decisions are still future encoder slices.

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

- `h264-real-vector-strict.sh` runs the green public-vector oracle set,
  including expected decode-error rows, and excludes only rows currently listed
  in the failure ledger.
- `FailureLedgerFreshness` runs only known-red rows when the ledger is populated
  and requires each failure class/detail to remain current.
- `FailureMatrix` runs the full 225-row manifest, currently requiring all 225
  rows to match oracle output.
- `TestH264DecoderTDDContractClassifiesEveryImportedPublicVector` runs in
  normal `go test ./tests` and fails if any imported public ref is not
  classified as executable or explicitly excluded.
- `h264-real-vector-upstream-audit.sh` fetches the pinned FFmpeg source and
  verifies that the checked-in inventory still matches all decoder-facing
  upstream H.264 FATE sample references, except documented non-decoder rows.
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

## Performance

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
stateful Annex B access-unit streaming, and the admitted realtime encoder
IDR/P-frame/RTP paths, suitable for `benchstat`, run:

```sh
scripts/h264-benchstat-canary.sh
```

To create a local release-evidence bundle with benchstat samples, the JSON
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
and a local performance-evidence bundle runner now exist, while checked-in
reviewed profile artifacts, a larger performance corpus, and an in-process
libavcodec baseline are still pending. Treat the decoder as
pre-production for throughput-sensitive use until
[docs/production-readiness.md](docs/production-readiness.md) has those release
artifacts.

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
| `docs/source-truth.md` | Compact current parity snapshot |
| `docs/translation-ledger.md` | Upstream-to-Go translation ledger |
| `docs/production-readiness.md` | Current verification and performance gates |
| `docs/high-bitdepth-roadmap.md` | High-bit-depth parity plan |
| `docs/encoder-webrtc-roadmap.md` | Realtime/WebRTC encoder target, controls, and gates |

## Trust And Verification

Released version: none yet.

No tag should be treated as production until a release-evidence pass proves:

- `scripts/h264-decoder-release-evidence.sh` is green.
- `go test ./...` is green.
- `scripts/h264-real-vector-strict.sh` is green.
- `GOH264_REAL_VECTOR_MATRIX=1 GOH264_CORPUS_FETCH=1 go test ./tests -run '^TestH264RealVectorFailureMatrix$' -count=1 -v` is green.
- `scripts/h264-real-vector-upstream-audit.sh` still represents all pinned
  decoder-facing FFmpeg H.264 FATE sample references in
  `testdata/h264/realvectors/upstream-inventory.jsonl`, except documented
  non-decoder exclusions.
- `scripts/h264-decoder-fuzz-smoke.sh` is green for the bounded public decoder
  no-panic fuzz target.
- Known-red rows, if any, are current in `testdata/h264/realvectors/failures.jsonl`.
- `scripts/h264-real-vector-release-alloc.sh` is green with the checked-in Go
  allocation canary budget.
- `scripts/h264-benchstat-canary.sh` runs decoder and admitted encoder rows
  with stable `-benchmem` output for trend comparison.
- `scripts/h264-performance-evidence.sh` creates the local performance bundle
  with JSON benchmark output plus CPU/heap profiles.
- Allocation and performance evidence is recorded in
  [docs/production-readiness.md](docs/production-readiness.md).
- Encoder support remains non-production until
  [docs/encoder-webrtc-roadmap.md](docs/encoder-webrtc-roadmap.md) has matching
  motion-search P prediction, residual bitstream implementation, rate-control
  behavior, remaining packetizer breadth, controls, and oracle evidence.
- The source-truth and translation-ledger docs match the committed tests.

The release-evidence runner writes logs under
`.artifacts/h264-release-evidence/` by default and fails while
`testdata/h264/realvectors/failures.jsonl` contains known-red rows unless
`GOH264_RELEASE_ALLOW_KNOWN_RED=1` is set for a non-release diagnostic run. It
also requires a clean worktree unless `GOH264_RELEASE_ALLOW_DIRTY=1` is set for
diagnostics.

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
