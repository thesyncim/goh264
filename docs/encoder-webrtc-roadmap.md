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
- RTP/WebRTC packetization support for packetization-mode 0 single-NAL output,
  packetization-mode 1 FU-A fragmentation, optional STAP-A aggregation,
  MTU/max-payload sizing, marker-bit boundaries, and timestamp/keyframe
  metadata.
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
  request, SPS/PPS cadence, output format, packetization mode, packetization
  limits, RTP payload metadata, timestamp increment, rate-control/QP,
  frame-drop, GOP/IDR, deblock, and latency/quality preset changes.

Current safe point: the public control contract is present in `encoder.go` and
covered by `tests/encoder_webrtc_controls_test.go`. Valid 8-bit I420
constrained-baseline realtime/WebRTC configs can be constructed, invalid
controls are rejected, including I420 crop offsets that H.264 cannot represent,
and runtime bitrate, framerate, payload-size, slice-count/byte-target,
rate-control mode, VBV size, initial/min/max QP, frame-drop mode, GOP/IDR
cadence, deblock mode, SPS/PPS cadence, RTP output format, packetization-mode
0/1, STAP-A, payload type, SSRC, timestamp increment, PLI/FIR, force-IDR, and
partial reconfiguration controls are tested, including rejected runtime
rate-control, output/header/preset, RTP re-entry payload-size, and packetization
updates that leave the prior config and queued-IDR state intact.
Runtime resolution reset rejects stale-size frames without consuming the queued
IDR, then emits/decodes a new-size IDR and resumes P-skip references at the new
dimensions.
`SetRTPMaxPayloadSize` retargets live RTP FU-A fragmentation before the next
P-frame while preserving sequence and decode state, including invalid-update
rollback.
Runtime `RecoveryPointSEI` toggles add, suppress, and restore changed-P
recovery side data without forcing IDR.
Runtime SPS/PPS cadence switches control forced-IDR header emission across
out-of-band, every-IDR, suppressed in-band, and restored in-band modes while the
stream remains decodable.
Runtime RTP-to-configured-AVC output switching forces an out-of-band IDR, stops
RTP packets/callbacks, preserves RTP timestamp cadence, and decodes the AVC
IDR/P-skip sequence through the emitted avcC.
Runtime configured-AVC-to-RTP output switching forces an every-IDR RTP frame,
starts RTP sequence numbers and callbacks from the first emitted packet, carries
retargeted payload metadata, and decodes the RTP IDR/P-skip sequence.
QP updates queue an IDR/PPS refresh. `MaxFrameSize` and `SliceMaxBytes` are now
enforced as encode-time guards before frame/reference/packet state advances:
`FrameDropDisabled` keeps the hard-error path, while `FrameDropToBitrate`
returns `EncodedFrame.Dropped` without emitted bytes or RTP packets for explicit
byte-budget misses or VBV-backed `MaxBitrate` bucket misses, and advances the RTP
timestamp timeline, with deterministic proof that transmitted frames consume
credit, per-frame refill resumes after dropped frames, and reference/packet
state remains stable; caller-buffer `EncodeInto` drops return empty output,
pause callbacks, and recover through the next P-skip without advancing
reference/frame/packet state. Runtime frame-drop mode switches toggle the
derived budget before the next frame, runtime max-bitrate/VBV lowering resets
stale credit, `SetBitrate` lowering resets stale frame-budget credit,
`SetFrameRate` changes reset frame-budget credit and apply the updated RTP
cadence across drop/recovery, `FrameDropLate` bypasses the derived bitrate
budget when the encode-time budget
admits the frame, and ConstantQP mode is proved to bypass that derived budget
before and after runtime switches through CBR. `FrameDropLate` now uses
`MaxEncodeTimeUS` as an
encode-time budget only when that mode is selected; late frames return dropped
metadata, advance the RTP timestamp timeline, and leave reference, frame-number,
packet-sequence, and callback state untouched, including after an existing
transmitted reference.
A combined control-loop stress proof now walks RTP IDR, Annex B forced IDR with
QP refresh, Annex B late drop, Annex B P-skip recovery, and RTP re-entry with
retargeted packet metadata while preserving decode, reference, callback, RTP
sequence, and timestamp semantics.
`ParameterSets`
generates SPS/PPS NALs, crop metadata, Annex B sequence headers, and avcC
records accepted by the decoder parsers. IDR header cadence is explicit for
in-band keyframes, out-of-band headers, and every-IDR emission.
`RecoveryPointSEI` generates Annex B and AVC recovery-point
SEI NAL surfaces and is proved by injecting the encoder output before a P-frame
and checking the public decoder recovery side data. `Encode` and `EncodeInto`
now validate frame shape before bitstream work, returning empty output for
invalid frames without advancing RTP sequence, callback, frame-number,
timestamp, or reference state, then emit the first admitted frame bitstream
paths: 8-bit I420 Constrained Baseline IDR IntraPCM access units with Annex B,
AVC, RTP packetization-mode 0 single-NAL output, and RTP packetization-mode 1
output, plus configured `SliceCount` multi-slice VCL output, guarded CAVLC
P-skip slices for identical frames, a guarded exact macroblock-aligned CAVLC P16x16
no-residual path for bounded even integer-pel shifted references under
disabled-deblock multi-macroblock frames plus single-macroblock
enabled/slice-boundary deblock, and guarded
CAVLC P IntraPCM slices for changed frames after a reference across disabled,
enabled, and slice-boundary deblock controls. Changed-frame P IntraPCM recovery
pictures carry recovery-point SEI when enabled, across Annex B, configured AVC,
and RTP packetization-mode 1 reassembly. Tests prove
local raw-frame decode, FFmpeg rawvideo decode, configured AVC and RTP
exact-P16 decode, recovery-point side data, multi-slice `first_mb_in_slice`
ordering, RTP packetization-mode 0 single-NAL IDR/P-frame reassembly and
oversize rejection, RTP FU-A reassembly, STAP-A parameter-set aggregation,
payload-type, SSRC, and sequence-number packet metadata. RTP
packets also carry complete 12-byte RTP headers plus payload bytes with clipped
per-packet payload views over packet data, and
`SetRTPPacketCallback` reports callback-style packet metadata for
packet index/count, frame PTS/DTS/RTP time, keyframe/IDR flags, STAP-A/FU-A/
single-NAL payload form including mode 0/1 IDR/P frames, NAL type/count, FU-A start/end, and parameter-set
packets. RTP timestamps honor explicit frame PTS and advance zero-PTS frames
from frame duration or `RTPTimestampIncrement`, including after runtime
timestamp-increment reconfiguration. `EncodeInto` now has checked allocation
canaries for caller-buffer Annex B forced IDR, Annex B steady P-skip, Annex B
exact P16x16 including single-macroblock deblock controls, Annex B
macroblock-aligned exact P16x16, Annex B changed
P IntraPCM, RTP forced IDR/FU-A, RTP exact P16x16, RTP steady P-skip, RTP changed
P IntraPCM, and RTP packetization-mode 0 IDR/P-skip/exact-P16x16/P-IntraPCM paths so
admitted packetization/output paths cannot
silently regress while broader allocation budgets are still pending; the live
encode path builds RBSP plus raw NAL output directly instead of constructing
discarded Annex B/AVC copies, with common one-slice NAL and slice-range
planning backed by stack storage. Package-level `-benchmem` canary rows now
cover Annex B IDR IntraPCM, Annex B steady P-skip, Annex B exact P16x16,
Annex B changed P IntraPCM, RTP FU-A IDR IntraPCM, RTP exact P16x16, RTP
steady P-skip, and RTP changed P IntraPCM plus RTP packetization-mode 0
IDR/P-skip/exact-P16x16/P-IntraPCM.
Cropped I420 IDR output is
proved through local decode and FFmpeg rawvideo decode of the cropped visible
frame. Queued IDR requests still emit IDR, and motion-search prediction,
residual coding, and adaptive rate-control feedback remain pending beyond the bounded exact
macroblock-aligned P16x16 admission.

Bitstream-writer safe point: `internal/h264/bitwriter.go` now contains the
source-shaped MSB-first writer primitives for raw bits, unsigned/signed
Exp-Golomb codes, RBSP trailing bits, EBSP emulation-prevention, Annex B/AVC
NAL packaging, and AVC decoder configuration records. The writer round-trips
through the existing decoder readers/parsers in `internal/h264/bitwriter_test.go`.
`internal/h264/encoder_headers.go` adds baseline SPS/PPS syntax writers,
including 4:2:0 crop-unit emission, in the same source-shaped style and
round-trips through `DecodeSPS`, `DecodePPS`, `SplitAnnexB`, and the avcC
parser. `internal/h264/encoder_sei.go` adds the
FFmpeg CBS-shaped recovery-point SEI writer, including extended SEI header
encoding and Annex B/AVC parser round trips. `internal/h264/encoder_slice.go`
adds the first Baseline IDR slice writer using CAVLC I_PCM macroblocks, with
edge padding and deblock-control syntax kept explicit, plus a parse-proved
Baseline P-skip writer that emits `mb_skip_run` for the selected slice range
and a parse-proved Baseline P16x16 no-residual writer that emits explicit
P_L0_16x16 macroblocks with constant or per-macroblock signed MVD syntax and
zero CBP. A public decode oracle proves SPS/PPS + IDR IntraPCM + P16x16
no-residual output through local Annex B decode and FFmpeg rawvideo decode. A
parse-proved Baseline P IntraPCM writer emits `mb_skip_run=0` plus P-slice
`mb_type=30` macroblocks. The current IDR, P-skip, P16x16 no-residual, and P
IntraPCM writers accept explicit
raster-scan macroblock ranges so public `SliceCount` can emit multiple VCL NALs
in one access unit.

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
   slices. Done for identical-reference P-skip, exact macroblock-aligned P16x16
   no-residual prediction with single-macroblock enabled/slice-boundary deblock
   proof, and changed-frame P IntraPCM across disabled, enabled, and
   slice-boundary deblock controls, configured multi-slice ranges,
   and recovery-point SEI emission on changed-frame P IntraPCM recovery
   pictures; forced keyframes still emit IDR.
5. In progress: add RTP packetization and WebRTC control handling with
   packet-level tests. Done for packetization-mode 0 single-NAL IDR, P-skip,
   exact-P16x16, and P IntraPCM output with oversize rejection,
   packetization-mode 1 single NAL/FU-A output, and
   STAP-A parameter-set aggregation with marker-bit boundaries plus
   payload-type, SSRC, sequence-number packet metadata, complete RTP header
   bytes, clipped per-packet payload views over packet data, callback-style
   packet metadata including mode 0/1 IDR/P-frame single-NAL packets,
   and automatic timestamp progression for frames without explicit PTS, plus explicit SPS/PPS in-band,
   out-of-band, and every-IDR cadence semantics. Runtime reconfiguration now
   switches output format, RTP mode 0/1, STAP-A, payload type, SSRC, SPS/PPS
   mode, RTP timestamp increments, rate-control mode, VBV size,
   initial/min/max QP, frame-drop mode, GOP/IDR cadence, and deblock mode with
   rollback on invalid updates, including invalid output/header/preset controls
   and invalid RTP re-entry payload sizing. Runtime resolution reset rejects
   stale-size frames without consuming the queued IDR, then emits/decodes a
   new-size IDR and resumes P-skip references at the new dimensions.
   `SetRTPMaxPayloadSize`
   retargets live RTP FU-A fragmentation before the next P-frame while
   preserving sequence/decode state and rolling back invalid updates. Runtime
   `RecoveryPointSEI` toggles add, suppress, and restore changed-P recovery
   side data without forcing IDR. Runtime SPS/PPS cadence switches control
   forced-IDR header emission across out-of-band, every-IDR, suppressed in-band,
   and restored in-band modes while the stream remains decodable.
   Runtime RTP-to-configured-AVC output switching forces an out-of-band IDR,
   stops RTP packets/callbacks, preserves RTP timestamp cadence, and decodes the
   AVC IDR/P-skip sequence through the emitted avcC.
   Runtime configured-AVC-to-RTP output switching forces an every-IDR RTP frame,
   starts RTP sequence numbers and callbacks from the first emitted packet,
   carries retargeted payload metadata, and decodes the RTP IDR/P-skip sequence.
   Configured `SliceCount` output now feeds RTP mode 1 as separate VCL NAL
   packets when each slice fits the payload limit, and configured
   `MaxFrameSize`/`SliceMaxBytes` budgets now reject oversized encoded output
   without advancing encoder state when frame dropping is disabled, or return
   dropped-frame metadata without emitted packets when `FrameDropToBitrate` is
   active; low VBV-backed `MaxBitrate` budgets now use the same dropped-frame
   state path, including proof of credit consumption/refill across transmitted
   and dropped frames plus stale-credit reset after runtime max-bitrate/VBV lowering,
   while runtime frame-drop mode switches toggle the derived budget before the
   next frame, `SetBitrate` and `SetFrameRate` also reset stale frame-budget
   credit, and `SetFrameRate` applies the updated RTP cadence across
   drop/recovery. `FrameDropLate` bypasses that derived bitrate budget when the
   encode-time budget admits the frame, and ConstantQP mode bypasses the same
   budget across runtime rate-control mode switches. `FrameDropLate` now drops frames
   that exceed `MaxEncodeTimeUS`
   without advancing reference/frame/packet state, including after a transmitted
   reference frame.
6. In progress: add realtime allocation budgets, encode timing benchmarks, and
   control-loop stress tests. Done for the first RTP/Annex B/RTP control-loop
   stress proof, initial `EncodeInto` allocation canaries on Annex B and RTP
   admitted IDR/P-frame paths including RTP P-IntraPCM and packetization-mode 0
   IDR/P frames with tightened RTP allocation budgets, and package-level
   benchmark canaries for admitted IDR/P-frame and RTP paths.

## Oracles And Gates

Encoder tests need independent evidence, not only local decode:

- Round-trip decode through `goh264` and FFmpeg CLI for every encoded fixture.
- Bitstream admission through FFmpeg/ffprobe for SPS/PPS/profile/level and
  packetized AVC output.
- WebRTC packetization tests for mode-0 single NAL IDR/P-frames, FU-A, STAP-A, MTU
  boundaries, marker bits, and keyframe parameter-set behavior.
- Rate-control tests that verify bitrate and frame-size envelopes across a
  deterministic source corpus.
- Reconfiguration tests for bitrate, framerate, force-IDR, resolution reset,
  max-payload, RTP, rate-control/QP, frame-drop, GOP/IDR, and deblock changes.
- Allocation gates for `EncodeInto`/packetization hot paths with caller-owned
  buffers; current canaries cover Annex B forced IDR, Annex B steady P-skip,
  Annex B exact P16x16, Annex B changed P IntraPCM, RTP forced IDR/FU-A, RTP
  exact P16x16, RTP steady P-skip, RTP changed P IntraPCM, and RTP
  packetization-mode 0 IDR/P-frame paths.

## Production Bar

Encoder support is not production-ready until:

- The decoder production bar remains green.
- The encoder controls above are represented by tests.
- Encoded streams pass local and FFmpeg decode oracles.
- WebRTC packetization tests cover every exposed packetization control.
- Allocation and realtime budget gates are checked into the normal release
  evidence.
