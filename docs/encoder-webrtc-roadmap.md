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
covered by `tests/encoder_webrtc_controls_test.go`, including public
input/result/callback surface guards for integration-facing encoder structs.
Valid 8-bit I420 constrained-baseline realtime/WebRTC configs can be
constructed, invalid controls are rejected, including I420 crop offsets that
H.264 cannot represent, and runtime bitrate, framerate, payload-size,
slice-count/byte-target,
rate-control mode, VBV size, initial/min/max QP, frame-drop mode, GOP/IDR
cadence, deblock mode, SPS/PPS cadence, RTP output format, packetization-mode
0/1, STAP-A, payload type, SSRC, timestamp increment, PLI/FIR, force-IDR, and
partial reconfiguration controls are tested, including rejected runtime
frame-rate/rate-control, latency/slice, output/header/preset,
RTP re-entry payload-size, RTP metadata, and packetization updates that leave
the prior config, queued-IDR state, callbacks, packet metadata, RTP cadence, and
VCL frame-number continuity intact across Annex B, AVC, and RTP where the
controls apply.
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
state remains stable for changed-P frame-size and slice-size drops;
caller-buffer `EncodeInto` drops return empty output,
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
generates caller-owned SPS/PPS NALs, crop metadata, Annex B sequence headers,
and avcC records accepted by the decoder parsers. IDR header cadence is explicit
for in-band keyframes, suppressed in-band headers, out-of-band headers, and
every-IDR emission across Annex B, AVC, and RTP.
`RecoveryPointSEI` generates caller-owned Annex B and AVC recovery-point
SEI NAL surfaces, and caller mutation/append isolation is proved across repeated
header and SEI helper calls. Valid and invalid header/SEI helper calls preserve
queued IDR, config, RTP packet/callback state, and frame-number continuity across
Annex B, AVC, and RTP. SEI side data is also proved by injecting the
encoder output before a P-frame and checking the public decoder recovery side
data. Force-IDR, PLI, FIR, and per-frame keyframe requests all bypass P-skip
references and deliver IDR output across Annex B, AVC, and RTP. `Encode` and `EncodeInto`
now validate frame shape before bitstream work, returning empty output for
invalid frames without advancing RTP sequence, callback, frame-number,
timestamp, or reference state. Overflowed caller-owned `EncodeInto` destination
growth also returns an empty hard error across Annex B, AVC, and RTP without
consuming queued IDR state or advancing RTP/callback state, then valid input
resumes as the queued IDR. The same hard-error path preserves P-frame reference
and frame-number state before the next P-skip. They emit the first admitted
frame bitstream
paths: 8-bit I420 Constrained Baseline IDR IntraPCM access units with Annex B,
AVC, RTP packetization-mode 0 single-NAL output, and RTP packetization-mode 1
output, plus configured `SliceCount` multi-slice VCL output, guarded CAVLC
P-skip slices for identical frames, a guarded exact macroblock-aligned CAVLC P16x16
no-residual path for bounded frame-wide and per-macroblock integer-pel shifted
references up to 8 pixels under disabled-deblock multi-macroblock frames plus
single-macroblock
enabled/slice-boundary deblock. Odd-pixel luma shifts are admitted only when
both 4:2:0 chroma planes are constant, with Annex B, configured AVC, RTP
reassembly, and RTP mode-0 single-NAL proof; patterned chroma is guarded to
fall back to P IntraPCM across the same output surfaces.
Guarded CAVLC P IntraPCM slices handle changed frames after a reference across
disabled, enabled, and slice-boundary deblock controls. Changed-frame P
IntraPCM recovery pictures carry recovery-point SEI when enabled, across Annex
B, configured AVC, and RTP packetization-mode 1 reassembly. Tests prove
local raw-frame decode, FFmpeg rawvideo decode, configured AVC and RTP
exact-P16 decode, recovery-point side data, multi-slice `first_mb_in_slice`
ordering, RTP packetization-mode 0 single-NAL IDR/P-frame reassembly and
oversize rejection, public NAL-unit metadata indexing back into encoded
access-unit bytes for Annex B/AVC/RTP output including non-empty caller-buffer
prefixes, caller-buffer preservation on RTP mode-0 rejection, live-state
rollback for mode-0 oversize queued-IDR and P-frame packetization failures, and
Annex B/AVC/RTP bitrate-drop and late-drop non-output paths, RTP FU-A
reassembly, STAP-A parameter-set aggregation with callback metadata and packet isolation,
small-payload STAP-A fallback to non-aggregated mode-1 packets with decode and
sequence continuity plus non-STAP-A callback payload metadata and callback
packet isolation for the fallback IDR and next P-skip,
changed-P recovery SEI single-NAL output when STAP-A is enabled,
payload-type, SSRC, and sequence-number packet metadata. RTP
packets also carry complete 12-byte RTP headers plus payload bytes with clipped
per-packet payload views over packet data and packet storage isolated from
`EncodedFrame.Data`, including caller-backed `EncodeInto` output buffers, with
shared metadata guards for header fields and clipped
packet slices, input-frame plane ownership after `Encode` returns across
Annex B/AVC/RTP, returned `Encode` result stability across later encodes, and
`SetRTPPacketCallback` reports callback-style packet metadata for
packet index/count, frame PTS/DTS/RTP time, keyframe/IDR flags, STAP-A/FU-A/
single-NAL payload form including mode 0/1 IDR/P frames with P-skip, exact
P16x16, multi-slice IDR, odd-pixel constant chroma, P IntraPCM fallback rows, and changed-P
STAP-A recovery SEI single-NAL packets, NAL type/count,
FU-A start/end, parameter-set packets, and callback packet storage isolated from
returned RTP packets while preserving the clipped payload-over-packet-data
shape. RTP timestamps honor explicit frame PTS and advance zero-PTS frames
from frame duration or `RTPTimestampIncrement`, including after runtime
timestamp-increment reconfiguration. `EncodeInto` now has checked allocation
canaries for caller-buffer Annex B forced IDR, Annex B steady P-skip, Annex B
exact P16x16 including single-macroblock deblock controls, Annex B odd-pixel
constant-chroma exact P16x16, Annex B odd-pixel patterned-chroma P IntraPCM
fallback, Annex B macroblock-aligned exact P16x16 including 8-pixel edge search,
Annex B per-macroblock exact P16x16, Annex B changed P IntraPCM, AVC forced IDR,
AVC steady P-skip, AVC odd-pixel constant-chroma exact P16x16, AVC odd-pixel
patterned-chroma P IntraPCM fallback, AVC exact P16x16 including 8-pixel edge
search, AVC per-macroblock exact P16x16, AVC changed P IntraPCM, RTP forced
IDR/FU-A, RTP forced IDR with STAP-A, RTP odd-pixel constant-chroma exact P16x16, RTP odd-pixel
patterned-chroma P IntraPCM fallback, RTP exact P16x16 including 8-pixel edge
search, RTP per-macroblock exact P16x16, RTP steady P-skip, RTP changed
P IntraPCM, and RTP packetization-mode 0 IDR/P-skip/exact-P16x16/P-IntraPCM
paths including odd-pixel constant-chroma, odd-pixel patterned-chroma fallback,
per-macroblock exact P16x16, and 8-pixel exact-P16 edge search, so admitted
packetization/output paths cannot
silently regress while broader allocation budgets are still pending; the live
encode path builds RBSP plus raw NAL output directly instead of constructing
discarded Annex B/AVC copies, including raw SPS/PPS NALs for forced IDR and raw
recovery-point SEI NALs for P IntraPCM fallback. Current budgets are tightened
to <=8 allocations for Annex B/AVC forced IDR, <=10 for RTP forced IDR/FU-A
and STAP-A, <=6 for
Annex B/AVC odd-patterned fallback, <=8 for RTP odd-patterned fallback, <=5 for
Annex B/AVC per-macroblock exact P16x16, <=7 for RTP per-macroblock exact
P16x16, <=12 for Annex B/AVC changed P IntraPCM, and <=16 for RTP changed
P IntraPCM, plus <=7 for Annex B/AVC/RTP `EncodeInto` max-frame-size and
slice-max-bytes drops and <=8 for Annex B/AVC/RTP late drops, with common one-slice NAL and slice-range planning backed by stack
storage.
Package-level `-benchmem` canary rows now
cover Annex B IDR IntraPCM, Annex B steady P-skip, Annex B exact P16x16,
Annex B odd-pixel constant-chroma exact P16x16, Annex B odd-pixel
patterned-chroma P IntraPCM fallback, Annex B per-macroblock exact P16x16,
Annex B exact P16x16 including 8-pixel edge search, Annex B changed P IntraPCM,
AVC IDR IntraPCM, AVC steady P-skip, AVC odd-pixel constant-chroma exact P16x16,
AVC odd-pixel patterned-chroma P IntraPCM fallback, AVC per-macroblock exact
P16x16, AVC exact P16x16 including 8-pixel edge search, AVC changed P IntraPCM,
RTP FU-A IDR IntraPCM, RTP STAP-A IDR IntraPCM, RTP odd-pixel constant-chroma exact P16x16, RTP
odd-pixel patterned-chroma P IntraPCM fallback, RTP per-macroblock exact P16x16,
RTP exact P16x16 including 8-pixel edge search, RTP steady P-skip, RTP
changed P IntraPCM, RTP STAP-A changed P IntraPCM, and RTP max-frame-size/late
drop paths plus RTP packetization-mode 0
IDR/P-skip/exact-P16x16/P-IntraPCM paths including odd-pixel constant-chroma,
per-macroblock exact P16x16, odd-pixel patterned-chroma fallback, and 8-pixel
exact-P16 edge search.
Cropped I420 IDR output is
proved through local decode and FFmpeg rawvideo decode of the cropped visible
frame. Queued IDR requests still emit IDR, and motion-search prediction,
residual coding, and adaptive rate-control feedback remain pending beyond the
bounded 8-pixel exact macroblock-aligned P16x16 admission, including the
per-macroblock exact-vector subset.

Bitstream-writer safe point: `internal/h264/bitwriter.go` now contains the
source-shaped MSB-first writer primitives for raw bits, unsigned/signed
Exp-Golomb codes, RBSP trailing bits, EBSP emulation-prevention, Annex B/AVC
NAL packaging, and AVC decoder configuration records. The writer round-trips
through the existing decoder readers/parsers in `internal/h264/bitwriter_test.go`.
`internal/h264/cavlc.go` now also contains CAVLC coeff-token, total-zeros, and
run-before VLC write primitives, with every emitted table code round-tripped
through the existing CAVLC decoder tables, plus bounded trailing-ones,
single-level, and single-level-plus-trailing-ones residual block writers
round-tripped through `decodeCAVLCResidual` across luma and chroma-DC table
contexts. The first non-trailing residual level writer covers the short code
path plus the decoder-supported prefix-14 and prefix-15 forms, while larger
levels remain rejected until the next source-shaped residual slice. Bounded two-
and three-non-trailing-level residual writers now also round-trip
subsequent-level suffix-length state transitions while still rejecting
trailing-one, higher-count, and out-of-range cases outside that admitted shape.
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
zero CBP. An internal Baseline P16x16 luma-residual slice writer now carries
per-macroblock signed MVD and nonzero coefficient inputs through parser and
CAVLC macroblock decode proof while preserving stateful `mb_qp_delta` emission
across consecutive residual macroblocks. A public decode oracle proves SPS/PPS + IDR IntraPCM + P16x16
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
   SPS/PPS plus recovery-point SEI syntax, with CAVLC residual VLC write
   primitives plus trailing-ones, single-level, and
   single-level-plus-trailing-ones residual block writing now round-tripped
   through the decoder.
3. Done: add an intra-only IDR path for I420 input and prove that local decode,
   FFmpeg decode, AVC decode, and RTP FU-A reassembly produce matching raw
   frames.
4. In progress: add P-frame prediction, reference management, CAVLC residual
   coding, deblock policy, and rate-control feedback in small oracle-backed
   slices. Done for identical-reference P-skip, exact macroblock-aligned P16x16
   no-residual prediction for frame-wide and per-macroblock integer-pel shifts
   up to 8 pixels, including odd-pixel luma motion only with constant chroma,
   with single-macroblock
   enabled/slice-boundary deblock proof, and changed-frame P
   IntraPCM across disabled, enabled, and
   slice-boundary deblock controls, configured multi-slice ranges,
   and recovery-point SEI emission on changed-frame P IntraPCM recovery
   pictures; forced keyframes still emit IDR.
5. In progress: add RTP packetization and WebRTC control handling with
   packet-level tests. Done for packetization-mode 0 single-NAL IDR, P-skip,
   exact-P16x16, and P IntraPCM output with oversize rejection,
   packetization-mode 1 single NAL/FU-A output, and
   STAP-A parameter-set aggregation with marker-bit boundaries plus changed-P
   recovery SEI left as single-NAL output plus
   payload-type, SSRC, sequence-number packet metadata, complete RTP header
   bytes, clipped per-packet payload views over packet data, callback-style
   packet metadata including mode 0/1 IDR/P-frame single-NAL packets and
   changed-P STAP-A recovery SEI single-NAL packets,
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
   active, including changed-P frame-size and slice-size drops without reference
   or packet-state advancement; low VBV-backed `MaxBitrate` budgets now use the
   same dropped-frame state path, including proof of credit consumption/refill
   across transmitted and dropped frames plus stale-credit reset after runtime max-bitrate/VBV lowering,
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
   stress proof, initial `EncodeInto` allocation canaries on Annex B, AVC, and
   RTP admitted IDR/P-frame paths including Annex B odd-pixel constant-chroma
   exact P16x16, AVC/RTP/RTP mode-0 odd-pixel constant-chroma exact P16x16,
   per-macroblock exact P16x16, 8-pixel exact-P16 edge search, RTP P-IntraPCM,
   and packetization-mode 0 IDR/P frames including per-macroblock exact P16x16
   and exact-P16 edge search with tightened IDR/fallback/exact-P16 RTP and
   Annex B/AVC allocation budgets,
   and package-level benchmark canaries for admitted IDR/P-frame
   Annex B/AVC/RTP paths plus RTP mode 0, including odd-pixel constant-chroma
   exact P16x16, per-macroblock exact P16x16, and 8-pixel exact-P16 edge search.

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
  Annex B exact P16x16 including odd-pixel constant-chroma and 8-pixel edge
  search, Annex B changed P IntraPCM, AVC forced IDR, AVC steady P-skip, AVC
  exact P16x16 including odd-pixel constant-chroma and 8-pixel edge search, AVC
  changed P IntraPCM, RTP forced IDR/FU-A, RTP exact P16x16 including odd-pixel
  constant-chroma and 8-pixel edge search, RTP steady P-skip, RTP changed P
  IntraPCM, and RTP
  packetization-mode 0 IDR/P-frame paths including odd-pixel constant-chroma
  and exact-P16 edge search.

## Production Bar

Encoder support is not production-ready until:

- The decoder production bar remains green.
- The encoder controls above are represented by tests.
- Encoded streams pass local and FFmpeg decode oracles.
- WebRTC packetization tests cover every exposed packetization control.
- Allocation and realtime budget gates are checked into the normal release
  evidence. The admitted local contract, writer, allocation, and `-benchmem`
  rows are now bundled by `scripts/h264-encoder-release-evidence.sh`; broader
  production status still requires the combined
  `scripts/h264-release-evidence.sh` pass plus the remaining motion-search,
  residual, rate-control, and packetizer breadth above.
