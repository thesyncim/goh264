# Realtime/WebRTC Encoder Roadmap

This repo is decoder-first today, but encoder support is now explicitly in
scope. The encoder target is realtime/WebRTC H.264, not an archival offline
encoder. The implementation should land in source-shaped, test-driven slices
with the same rule as the decoder: no production claim until controls,
bitstream validity, oracle behavior, and allocation/performance evidence are
proved.

## Target

Initial encoder support should prioritize 8-bit 4:2:0 realtime video for WebRTC:

- Low-latency P/IDR operation with B-frames disabled by default.
- Constrained Baseline/Baseline first, then Main/High only after WebRTC-safe
  behavior is proved.
- Annex B and AVC output surfaces, with SPS/PPS emission controls.
- RTP/WebRTC packetization support for packetization-mode 1, FU-A fragmentation,
  optional STAP-A aggregation, MTU/max-payload sizing, marker-bit boundaries,
  and timestamp/keyframe metadata.
- No cgo and no mandatory external codec dependency.

## Controls

The public API must expose controls before the implementation depends on hidden
defaults:

- Source format: width, height, strides, input pixel format, crop, frame rate,
  time base, timestamps, and color/VUI metadata.
- Profile/level: profile, constraint flags, level, entropy mode, deblock mode,
  transform size, reference count, and SPS/PPS cadence.
- Rate control: CBR/VBR-like mode, target bitrate, max bitrate, buffer/VBV
  size, max frame size, initial/min/max QP, quality/speed preset, and bitrate
  update while running.
- Realtime latency: zero-lookahead mode, frame dropping policy, maximum encode
  time budget, slice count/slice byte target, worker count, and deterministic
  single-thread mode.
- Keyframes and recovery: GOP length, IDR interval, force-IDR, PLI/FIR handling,
  recovery point signaling, intra refresh when supported, and SPS/PPS before
  keyframes.
- WebRTC packetization: maximum RTP payload size, packetization mode, SPS/PPS
  out-of-band versus in-band, aggregation policy, DON-disabled mode, and
  per-packet metadata callbacks.
- Runtime reconfiguration: bitrate, frame rate, resolution reset, keyframe
  request, packetization limits, and latency/quality preset changes.

Current safe point: the public control contract is present in `encoder.go` and
covered by `tests/encoder_webrtc_controls_test.go`. Valid 8-bit I420
constrained-baseline realtime/WebRTC configs can be constructed, invalid
controls are rejected, and runtime bitrate, framerate, payload-size, PLI/FIR,
force-IDR, and partial reconfiguration controls are tested. `ParameterSets`
generates SPS/PPS NALs, Annex B sequence headers, and avcC records accepted by
the decoder parsers. `RecoveryPointSEI` generates Annex B and AVC recovery-point
SEI NAL surfaces and is proved by injecting the encoder output before a P-frame
and checking the public decoder recovery side data. `Encode` and `EncodeInto`
now validate frame shape and emit the first admitted frame bitstream paths:
8-bit I420 Constrained Baseline IDR IntraPCM access units with Annex B, AVC,
and RTP packetization-mode 1 output, plus guarded CAVLC P-skip slices for
identical frames after a reference when deblocking is disabled. Tests prove
local raw-frame decode, FFmpeg rawvideo decode, RTP FU-A reassembly, STAP-A
parameter-set aggregation, payload-type, SSRC, and sequence-number packet
metadata. RTP packets also carry complete 12-byte RTP headers plus payload
bytes. Changed frames and queued IDR requests still fall back to IDR until
motion search/residual coding land.

Bitstream-writer safe point: `internal/h264/bitwriter.go` now contains the
source-shaped MSB-first writer primitives for raw bits, unsigned/signed
Exp-Golomb codes, RBSP trailing bits, EBSP emulation-prevention, Annex B/AVC
NAL packaging, and AVC decoder configuration records. The writer round-trips
through the existing decoder readers/parsers in `internal/h264/bitwriter_test.go`.
`internal/h264/encoder_headers.go` adds baseline SPS/PPS syntax writers in the
same source-shaped style and round-trips through `DecodeSPS`, `DecodePPS`,
`SplitAnnexB`, and the avcC parser. `internal/h264/encoder_sei.go` adds the
FFmpeg CBS-shaped recovery-point SEI writer, including extended SEI header
encoding and Annex B/AVC parser round trips. `internal/h264/encoder_slice.go`
adds the first Baseline IDR slice writer using CAVLC I_PCM macroblocks, with
edge padding and deblock-control syntax kept explicit, plus a parse-proved
Baseline P-skip writer that emits a single `mb_skip_run` covering the picture.

## Implementation Order

1. Done: add the public encoder configuration and control contract with tests
   that reject invalid WebRTC configurations.
2. Done: add bitstream writer primitives. Done for raw NAL/RBSP,
   Exp-Golomb, Annex B/AVC packaging, AVC configuration records, and baseline
   SPS/PPS plus recovery-point SEI syntax.
3. Done: add an intra-only IDR path for I420 input and prove that local decode,
   FFmpeg decode, AVC decode, and RTP FU-A reassembly produce matching raw
   frames.
4. In progress: add P-frame prediction, reference management, CAVLC residual
   coding, deblock policy, and rate-control feedback in small oracle-backed
   slices. Done for identical-reference P-skip with deblock disabled and IDR
   fallback for changed frames/forced keyframes.
5. In progress: add RTP packetization and WebRTC control handling with
   packet-level tests. Done for packetization-mode 1 single NAL/FU-A output and
   STAP-A parameter-set aggregation with marker-bit boundaries plus
   payload-type, SSRC, sequence-number packet metadata, and complete RTP header
   bytes; callback-style packet metadata remains pending.
6. Add realtime allocation budgets, encode timing benchmarks, and control-loop
   stress tests.

## Oracles And Gates

Encoder tests need independent evidence, not only local decode:

- Round-trip decode through `goh264` and FFmpeg CLI for every encoded fixture.
- Bitstream admission through FFmpeg/ffprobe for SPS/PPS/profile/level and
  packetized AVC output.
- WebRTC packetization tests for FU-A, STAP-A, MTU boundaries, marker bits, and
  keyframe parameter-set behavior.
- Rate-control tests that verify bitrate and frame-size envelopes across a
  deterministic source corpus.
- Reconfiguration tests for bitrate, framerate, force-IDR, resolution reset, and
  max-payload changes.
- Allocation gates for `EncodeInto`/packetization hot paths with caller-owned
  buffers.

## Production Bar

Encoder support is not production-ready until:

- The decoder production bar remains green.
- The encoder controls above are represented by tests.
- Encoded streams pass local and FFmpeg decode oracles.
- WebRTC packetization tests cover every exposed packetization control.
- Allocation and realtime budget gates are checked into the normal release
  evidence.
