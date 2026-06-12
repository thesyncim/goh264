# goh264

Pure-Go H.264 codec workbench, decoder-first and source-shaped from FFmpeg
`libavcodec`.

This repository is an active port of the FFmpeg `n8.0.1` H.264 decoder path,
pinned at `894da5ca7d742e4429ffb2af534fcda0103ef593`. The decoder is the mature
side of the project: public Annex B, AVC, avcC, packet, raw-output, side-data,
and delayed-output surfaces are covered by unit, corpus, FATE, and FFmpeg-oracle
tests.

The encoder is intentionally narrower. It exposes a tested realtime/WebRTC API
and admits a guarded Constrained Baseline I420 subset today: IDR IntraPCM,
identical-reference P-skip, bounded exact P16x16 no-residual prediction, changed
P IntraPCM recovery frames, AVC/Annex B output, configured multi-slice output,
and RTP packetization modes 0 and 1. Broader motion search, residual coding,
rate-control behavior, and production performance evidence are still in flight.

## What Works Today

- **Decoder:** pure Go, no cgo, no module dependencies.
- **Inputs:** Annex B bytestreams, length-prefixed AVC packets, avcC decoder
  configuration records, packet `NEW_EXTRADATA`, and auto-detected packets.
- **Output:** decoded Y/Cb/Cr planes, crop metadata, VUI/timing fields,
  high-bit-depth planes, raw YUV byte/sample helpers, frame cloning, and
  side-data cloning.
- **State:** streaming decode keeps references and delayed B-frame output across
  calls; empty decode calls flush delayed output.
- **Encoder:** usable as an experimental realtime/WebRTC surface for the
  admitted Baseline paths listed above.
- **Verification:** the selected public-vector decoder matrix is green with no
  known-red rows.

## Not Yet Production

No release tag should be treated as production. The remaining work is mainly
quality hardening, API cleanup, allocation/performance evidence, and broader
encoder coverage. The detailed status lives in:

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

## Status Snapshot

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

Decoder coverage includes compact Baseline/Main/High rows, selected FRext and
high-bit-depth fixtures, I/P/B slices, CAVLC and CABAC, weighted and direct
motion paths, selected field/PAFF/MBAFF rows, lossless High444
transform-bypass rows, configured AVC surfaces, container-extracted Annex B
vectors, malformed packet recovery, side-data surfaces, and bounded public
no-panic fuzz coverage.

Still guarded: unselected MBAFF/PIC-AFF/PAFF motion paths, broader
high-bit-depth field/inter streams, broader damaged-slice error resilience,
threading/SIMD, bulk allocation hardening, and exact libavcodec delayed-output
edge behavior. Intentionally unsupported at the pinned FFmpeg parity boundary:
FMO, 11/13-bit luma depths, `chroma_format_idc > 3`, separate color planes, and
mixed chroma/luma bit depths.

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

`RawYUVBytesLE` returns a caller-owned rawvideo byte buffer for one frame.
`RawYUV16` returns a caller-owned uint16 sample buffer for high-bit-depth frames.
`Frame.Clone` returns a deep-owned decoded-frame snapshot, including planes and
side data; `FrameSideData.Clone` snapshots side data without copying planes.
`AppendRawYUV` is available for 8-bit output. `AppendRawYUVBytesLE` handles both
8-bit and high-bit-depth output, using little-endian samples for 9-bit and
higher formats. `AppendRawYUV16` is the caller-buffer form for high-bit-depth
uint16 output. `RawPixelFormat` returns names such as `yuv420p`,
`yuv422p10le`, or `yuv444p`.

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
frames, err := dec.DecodeAVCCFrames(avcc, packet)      // parse avcC, decode packet
frames, err := dec.DecodeAVCCFrames(avcc, nil)         // delayed avcC output
frames, err := dec.FlushDelayedFrames()                // delayed B-frame output
frame, err := dec.FlushDelayedFrame()                  // single delayed B-frame
cfg, err := dec.AVCConfig()                            // current configured-AVC metadata
err = dec.Reset()                                      // clear decoder state
```

Single-frame helpers (`Decode`, `DecodePacket`, `DecodeAnnexB`, `DecodeAVC`,
`DecodeConfiguredAVC`, `DecodeAVCC`, and `FlushDelayedFrame`) return
`ErrUnsupported` when a packet produces zero or multiple frames. If a damaged
packet produces exactly one valid frame before a later decode error, the helper
returns that frame with the error. For stream processing, prefer `DecodeFrames` or
`DecodePacketFrames`; they retain decoder reference state across packets and
flush delayed output when called with empty data. `DecodeConfiguredAVCFrames`
does the same after an AVC configuration record has been parsed. Annex B
access-unit streams use the same retained reference and delayed B-frame output path.
`DecodeAVCCFrames` updates the decoder's AVC
configuration without resetting retained references, then drains delayed output
for the supplied AVC packet. Passing an empty AVC packet with a configuration
record drains delayed output without reporting an invalid packet.

Parse headers without decoding full frames:

```go
info, err := dec.ParseHeadersAnnexB(data)
info, err := dec.ParseHeadersAVC(packet, nalLengthSize)
cfg, err := dec.ParseAVCC(avcc)
cfg, err := goh264.ParseAVCC(avcc) // stateless avcC inspection
```

Malformed `ParseHeadersAnnexB` and `ParseHeadersAVC` calls are transactional:
partially parsed SPS/PPS state is not committed over a previous valid
configuration, and delayed configured-AVC B-frame output remains available for
flush after the rejected parse.
Decoder `ParseAVCC` stores the configuration for later configured-AVC decode;
package-level `ParseAVCC` and `ParseAVCDecoderConfigurationRecord` parse the
same metadata without mutating decoder state.

Packet side-data support mirrors FFmpeg-facing surfaces used by the port:

```go
frames, err := dec.DecodePacketFrames(goh264.Packet{
	Data: packet,
	SideData: []goh264.PacketSideData{
		{Type: goh264.PacketSideDataNewExtradata, Data: avcc},
	},
})
ownedPacket := goh264.Packet{Data: packet}.Clone()
```

`Frame` includes dimensions, crop, chroma format, bit depth, SAR/VUI fields,
timing fields, keyframe/interlace flags, raw planes, and parsed SEI/packet side
data such as A53 captions, S12M timecode, stereo 3D, spherical video, mastering
display metadata, content light metadata, display orientation, film grain, ICC
profile, HDR10+, and LCEVC side data.
Duplicate packet side data follows first-entry semantics: empty or malformed
first active-format, S12M timecode, ICC profile, HDR10+, and LCEVC entries
suppress later duplicates.

The encoder API is a WebRTC/realtime control contract while implementation is
still landing:

```go
cfg := goh264.DefaultEncoderConfig(640, 480)
cfg.TargetBitrate = 800_000
cfg.MaxBitrate = 1_000_000
cfg.SliceCount = 2
cfg, err := cfg.Normalize()

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
avcc := headers.AVCC()
headersCopy := headers.Clone()
sei, err := enc.RecoveryPointSEI(0) // Annex B/AVC recovery-point SEI NALs
seiCopy := sei.Clone()
frame := enc.I420Frame(y, cb, cr, pts)
err = cfg.ValidateFrame(frame)
err = enc.ValidateFrame(frame)
out, err := enc.Encode(frame) // admitted path: IDR/P-skip/P16x16/P IntraPCM
if out.Dropped {
	// Realtime budget drop: no bytes or RTP packets were emitted.
}
accessUnit, err := out.AccessUnitData()
nal0, err := out.NALData(0) // clipped raw NAL bytes from EncodedFrame.Data
packet0, err := out.RTPPacketData(0)
payload0, err := out.RTPPayloadData(0)
owned, err := out.Clone()   // deep-owned snapshot for async retention
err = enc.Reset()           // clear encoder coding state, keep config/callback
```

`EncoderConfig.Normalize` exposes the exact validated configuration stored by
`NewEncoder`. `EncoderConfig.ValidateFrame` and `Encoder.ValidateFrame` validate
frame shape before bitstream work; invalid frames return empty output without
advancing RTP sequence, callback, frame-number, timestamp, or reference state,
then valid input resumes as the expected P-skip, or as the queued IDR when a
prior IDR request was pending.
Overflowed caller-owned `EncodeInto` destination growth is also rejected across
Annex B, AVC, and RTP without consuming queued IDR state or advancing
RTP/callback state, and the same hard-error path preserves P-frame reference
and frame-number state before the next P-skip.
They emit the admitted IDR IntraPCM, identical-reference P-skip, exact
macroblock-aligned frame-wide or per-macroblock P16x16 no-residual, or
changed-frame P IntraPCM frame path, optionally split into configured
multi-slice VCL NALs. Exact P16x16 is admitted for disabled-deblock
multi-macroblock frames and single-macroblock
enabled/slice-boundary deblock frames. Changed-frame P IntraPCM recovery
pictures carry recovery-point SEI when enabled. RTP output includes payloads
plus complete RTP
packet bytes, packetization-mode 0 single-NAL output, packetization-mode 1
FU-A/STAP-A output with small-payload STAP-A fallback to non-aggregated mode-1
packets plus accurate fallback-IDR and post-fallback P-skip callback payload
metadata and callback packet isolation, RTP packet storage isolated from `EncodedFrame.Data`,
public `EncodedFrame.RTPPacketData` and `EncodedFrame.RTPPayloadData` helpers,
deep-owned `EncodedFrame.Clone` snapshots for retained results,
optional per-packet callback metadata including mode 0/1
IDR/P-frame single-NAL packets for multi-slice IDR, P-skip, exact P16x16,
odd-pixel constant chroma, and P IntraPCM fallback rows with callback packet storage isolated from
the returned RTP packets, mode-0 oversize rejection live-state rollback for
queued-IDR and P-frame paths, and automatic
timestamp progression when frames omit explicit PTS. SPS/PPS cadence modes now
separate in-band keyframe headers, out-of-band headers, and every-IDR emission,
and runtime reconfiguration can switch output format and RTP packetization
controls, including RTP-to-configured-AVC forced IDR/P-skip decode with
out-of-band parameter sets and paused RTP packets/callbacks, plus
configured-AVC-to-RTP forced IDR/P-skip packetization with sequence/callback
start, plus rate-control/QP/GOP/deblock controls while preserving state on
rejected updates. Bitrate-budget drops use the configured `MaxBitrate` refill
rate and `VBVBufferSize` burst capacity, then surface through
`EncodedFrame.Dropped` when `FrameDropToBitrate` is active; caller-buffer
`EncodeInto` drops return empty output without RTP packets or callbacks before
the next valid frame resumes as P-skip.
Motion search beyond the bounded 8-pixel exact macroblock-aligned inter path,
quantized residual coding, and adaptive rate-control feedback are still future
encoder slices.

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
  upstream H.264 FATE sample references, except documented non-decoder rows,
  and that public-vector count claims in the release docs match the checked-in
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
stateful Annex B access-unit streaming, isolated raw-output export, and the admitted realtime encoder
IDR/P-frame Annex B/AVC/RTP paths, including RTP P-IntraPCM and
packetization-mode 0 IDR/P-frame rows, suitable for `benchstat`, run:

```sh
scripts/h264-benchstat-canary.sh
```

Use `GOH264_BENCHSTAT_COUNT` and `GOH264_BENCHSTAT_TIME` to control sample
count and `-benchtime`; `GOH264_BENCHSTAT_BENCHTIME` is accepted as a
compatibility alias when `GOH264_BENCHSTAT_TIME` is unset.

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

- `scripts/h264-release-evidence.sh` is green as the combined decoder and
  admitted-encoder release gate.
- `scripts/h264-decoder-release-evidence.sh` is green.
- `go vet ./...` is green.
- `go test ./...` is green.
- `go test -race ./...` is green.
- `scripts/h264-real-vector-strict.sh` is green.
- `GOH264_REAL_VECTOR_MATRIX=1 GOH264_CORPUS_FETCH=1 go test ./tests -run '^TestH264RealVectorFailureMatrix$' -count=1 -v` is green.
- `scripts/h264-real-vector-upstream-audit.sh` still represents all pinned
  decoder-facing FFmpeg H.264 FATE sample references in
  `testdata/h264/realvectors/upstream-inventory.jsonl`, except documented
  non-decoder exclusions, and release-doc public-vector counts match the
  checked-in manifests.
- `scripts/h264-decoder-fuzz-smoke.sh` is green for the bounded public decoder
  no-panic fuzz target.
- Known-red rows, if any, are current in `testdata/h264/realvectors/failures.jsonl`.
- `scripts/h264-real-vector-release-alloc.sh` is green with the checked-in Go
  allocation canary budget.
- `scripts/h264-benchstat-canary.sh` runs decoder and admitted encoder rows
  with stable `-benchmem` output for trend comparison. `GOH264_BENCHSTAT_TIME`
  controls the effective `-benchtime`, with `GOH264_BENCHSTAT_BENCHTIME`
  accepted as an unset-time alias.
- `scripts/h264-performance-evidence.sh` creates the local performance bundle
  with JSON benchmark output plus CPU/heap profiles.
- `scripts/h264-encoder-release-evidence.sh` is green for the admitted
  realtime/WebRTC encoder vet, contract, writer, allocation, and benchmark
  gates.
- Allocation and performance evidence is recorded in
  [docs/production-readiness.md](docs/production-readiness.md).
- Encoder support remains non-production until
  [docs/encoder-webrtc-roadmap.md](docs/encoder-webrtc-roadmap.md) has matching
  broader motion-search P prediction, residual bitstream implementation,
  rate-control behavior, remaining packetizer breadth, controls, and oracle
  evidence.
- The source-truth and translation-ledger docs match the committed tests.

The combined release-evidence runner writes logs under
`.artifacts/h264-full-release-evidence/` by default, drives the race, decoder,
and admitted encoder runners, and requires a clean worktree unless
`GOH264_FULL_RELEASE_ALLOW_DIRTY=1` is set for diagnostics.
The decoder release-evidence runner writes logs under
`.artifacts/h264-release-evidence/` by default and fails while
`testdata/h264/realvectors/failures.jsonl` contains known-red rows unless
`GOH264_RELEASE_ALLOW_KNOWN_RED=1` is set for a non-release diagnostic run. It
also requires a clean worktree unless `GOH264_RELEASE_ALLOW_DIRTY=1` is set for
diagnostics.
The encoder release-evidence runner writes logs under
`.artifacts/h264-encoder-release-evidence/` and likewise requires a clean
worktree unless `GOH264_ENCODER_RELEASE_ALLOW_DIRTY=1` is set for diagnostics.

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
