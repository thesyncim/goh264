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
force-IDR, and partial reconfiguration controls are tested. `Encode` and
`EncodeInto` validate frame shape but still return `ErrUnsupported` for
bitstream generation.

## Implementation Order

1. Done: add the public encoder configuration and control contract with tests
   that reject invalid WebRTC configurations.
2. Next: add bitstream writer primitives for NAL/RBSP, Exp-Golomb, SPS, PPS, SEI,
   slice headers, and AVC configuration records.
3. Add an intra-only IDR path for I420 input and prove that local decode and
   FFmpeg decode produce matching raw frames.
4. Add P-frame prediction, reference management, CAVLC residual coding, deblock
   policy, and rate-control feedback in small oracle-backed slices.
5. Add RTP packetization and WebRTC control handling with packet-level tests.
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
