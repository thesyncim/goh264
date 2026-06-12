# Source Truth

Decoder scope: FFmpeg `n8.0.1` H.264 decoder path only.

Expanded product scope: realtime/WebRTC H.264 encoder support is now planned in
`docs/encoder-webrtc-roadmap.md`. The public encoder control contract now lives
in `encoder.go` and is tested from the external `tests` package. Encoder
parameter-set and recovery-point SEI writers follow FFmpeg `n8.0.1`
`libavcodec/cbs_h264_syntax_template.c` syntax order; recovery-point SEI
defaults mirror the FFmpeg VAAPI/Vulkan encoder shape for I-picture recovery
points (`recovery_frame_cnt=0`, exact-match set, broken-link only when B-frame
chains exist). The first encoder frame bitstream path is an 8-bit I420
Constrained Baseline IDR IntraPCM slice writer following FFmpeg CBS H.264 slice
header and reference-marking syntax order; the first P path is guarded CAVLC
P-skip for identical frames after a reference when deblocking is disabled.
RTP packet header emission follows FFmpeg `libavformat/rtpenc.c`
`ff_rtp_send_data()` field order and H.264 RTP payload boundaries follow
`libavformat/rtpenc_h264_hevc.c`; changed-frame P prediction, rate-control, and
production encoder claims do not exist yet. Encoder work must land behind its
own controls, oracles, and release evidence while the decoder production bar
stays green.

Proved today: progressive Annex B/AVC IDR/P/B subsets, selected High10/High12/High14
fixtures including public High10/High422 intra conformance and High10 unweighted 4:2:2/4:4:4 I/P chroma
no-deblock plus weighted 4:2:2/4:4:4 luma-only and luma+chroma P frame deblock modes 0/1 and
slice-boundary mode-2 deblock plus CAVLC/CABAC 4:2:2/4:4:4 implicit and
explicit weighted B frame deblock mode 1 and slice-boundary mode-2 deblock, High12 CAVLC IntraPCM, public CAVLC
Intra16x16 no-residual and luma-DC/luma-AC/luma-DC+AC/chroma-DC/chroma-AC/
chroma-DC+AC/luma+chroma residual plus two-frame CAVLC P-skip/P16x16
no-residual and P16x16 luma-residual/luma+chroma residual plus
P16x8/P8x16/P8x8 luma+chroma residual Annex B/AVC/configuration-record
fixtures, mixed no-residual intra, plus
Intra16x16 luma, chroma, and combined luma/chroma residual plus public
P-skip/P16x16, internal High12 no-deblock weighted P-skip and
weighted P16x16/partitioned P plus
P16x16 residual handoff plus no-deblock/mode-1 I/P deblock including
public CAVLC/CABAC 4:2:2/4:4:4 unweighted I/P no-deblock/mode-1/mode-2
deblock and CAVLC/CABAC luma-only and luma+chroma weighted-P no-deblock/mode-1/mode-2 deblock,
CAVLC/CABAC unweighted B no-deblock/mode-1/mode-2 deblock including
B-skip/direct-sub no-deblock and mode-1/mode-2 deblock plus partitioned
B16x8/B8x16/B8x8 no-deblock and mode-1/mode-2 deblock plus CAVLC/CABAC
direct-sub residual,
4:2:0 implicit weighted B no-deblock/mode-1/mode-2 deblock including
direct-sub and partitioned B16x8/B8x16/B8x8 shapes,
4:2:0 explicit weighted B no-deblock/mode-1/mode-2 deblock including
direct-sub and partitioned B16x8/B8x16/B8x8 shapes,
unweighted 4:2:2/4:4:4 I/P chroma deblock plus 4:2:0 I/P
slice-boundary mode-2 deblock and unweighted 4:2:2/4:4:4 I/P chroma
slice-boundary mode-2 deblock, High14 CAVLC/CABAC IntraPCM plus
CAVLC/CABAC no-residual Intra4x4/Intra16x16 and CAVLC/CABAC
Intra16x16 luma-DC/luma-AC/luma-DC+AC/chroma-DC/chroma-AC/
chroma-DC+AC/luma+chroma residual plus public two-frame CAVLC P-skip/P16x16
no-residual and P16x16 luma-residual/luma+chroma residual plus
P16x8/P8x16/P8x8 luma+chroma residual Annex B/AVC/configuration-record
fixtures, mixed no-residual
intra, separate and combined Intra16x16 luma/chroma residual plus
CAVLC/CABAC unweighted and weighted P-skip/P16x16 plus CAVLC mode-1/mode-2
I/P and weighted-P deblock plus public CAVLC/CABAC 4:2:2/4:4:4 unweighted I/P
no-deblock/mode-1/mode-2 deblock and CAVLC/CABAC luma-only and luma+chroma weighted-P
no-deblock/mode-1/mode-2 deblock
and CAVLC/CABAC unweighted B no-deblock/mode-1/mode-2 deblock
including B-skip/direct-sub no-deblock and mode-1/mode-2 deblock plus partitioned
B16x8/B8x16/B8x8 no-deblock and mode-1/mode-2 deblock plus CAVLC/CABAC
direct-sub residual
plus 4:2:0 implicit weighted B no-deblock/mode-1/mode-2 deblock including
direct-sub and partitioned B16x8/B8x16/B8x8 shapes
plus 4:2:0 explicit weighted B no-deblock/mode-1/mode-2 deblock including
direct-sub and partitioned B16x8/B8x16/B8x8 shapes
plus mode-1/mode-2 I/P and weighted-P deblock, High10 frame-MBAFF field-coded CAVLC IntraPCM
entropy/reconstruct pairing plus public High12/High14 4:2:0 frame-MBAFF CAVLC
IntraPCM, P-skip, and field-coded/frame-coded P16x16/P16x8/P8x16/P8x8 no-residual, luma-residual, and luma+chroma-residual rows plus P-skip and field-coded/frame-coded P16x16/P16x8/P8x16/P8x8 mode-1/mode-2 deblock rows plus explicit weighted-P field-coded/frame-coded P16x16/P16x8/P8x16/P8x8 no-residual, luma-residual, and luma+chroma-residual rows across no-deblock/mode-1/mode-2 plus temporal/spatial direct B-skip no-deblock/mode-1/mode-2 rows plus temporal/spatial direct-sub B8x8 field-coded/frame-coded no-deblock/mode-1/mode-2 rows plus field-coded/frame-coded partitioned B16x8/B8x16/B8x8 no-deblock/mode-1/mode-2 rows plus implicit weighted temporal/spatial direct B-skip, direct-sub B8x8 field-coded/frame-coded, and field-coded/frame-coded partitioned B16x8/B8x16/B8x8 no-deblock/mode-1/mode-2 rows plus explicit weighted temporal/spatial direct B-skip, direct-sub B8x8 field-coded/frame-coded, and field-coded/frame-coded partitioned B16x8/B8x16/B8x8 no-deblock/mode-1/mode-2 rows plus public High10/High422 field-coded
frame-MBAFF deblock rows plus public derived High10 4:2:2/4:4:4 luma-only
and luma+chroma weighted-P slice-boundary mode-2 rows plus internal High10 4:2:2/4:4:4 top/bottom field
explicit/implicit weighted B luma/chroma deblock modes 0/1/2 plus internal High10 4:2:2/4:4:4 top/bottom field
luma-only, luma+chroma, and source-normalized chroma-only weighted P deblock modes 0/1/2 plus
internal High12/High14 4:2:2/4:4:4 top/bottom field explicit/implicit weighted B plus luma-only, luma+chroma,
and source-normalized chroma-only weighted P deblock modes 0/1/2, complete FFmpeg FRext FATE row coverage including
monochrome-to-yuv420p output, selected compact baseline/main conformance rows
including AUD, SVA/Sony/MW/FT/JVC/Sand/Toshiba/VTC baseline/main, CABAC,
no-deblock, reference, field, and slice-boundary rows,
CABAC delayed-output reorder plus weighted/field compact rows, CAVLC
field/QP/crop/weighted/no-deblock compact rows, compact
long-slice/multiple-reference rows, frame-num-gap multiple-reference rows, and
MMCO-reset multiple-reference reorder rows,
internal DPB delayed-output boundary guards for IDR and MMCO reset pictures,
non-conformance Annex B FATE rows covering extreme plane prediction,
reference-list-mod overflow recovery, missing-frame recovery, and
large-to-small SPS reinit metadata, with
packetized/configured surfaces proven for reference-list-mod overflow,
missing-frame, timecode, and BBC2 recovery/PAFF,
lossless High444 transform-bypass Annex B plus 4-byte AVC/configuration-record surfaces,
public High9 4:2:0 SPS reinit metadata from 9-bit to 8-bit output,
High444 10-bit SPS reinit metadata, XAVC High422 terminal damaged
first-field recovery, container-backed FATE H.264 vectors extracted to Annex B
with both source and extracted bitstream MD5s, auxiliary API/MOV/MPEG-TS/MKV/CBS H.264 sample rows,
public FATE vector harness,
bounded public decode/packet no-panic fuzz smoke,
stateful valid-damaged-valid packet recovery guards across configured AVC,
AVC configuration-record, packet `NEW_EXTRADATA`, and auto Annex B surfaces,
valid-prefix damaged-packet recovery across configured AVC, AVC
configuration-record, bare AVC, packet AVC, stateful Annex B, and one-shot
Annex B surfaces including single-frame helpers and delayed B-frame
configuration-record one-shot flush,
configured AVC and AVC configuration-record empty-packet delayed B-frame flush
matching `h264_decode_frame()` `buf_size == 0` delayed-output behavior,
repeated valid packet `NEW_EXTRADATA` preserving delayed B-frame reorder state,
compressed packet input ownership across configured AVC and Annex B delayed
B-frame output,
malformed standalone AVC decoder configuration records preserving the last
stored configured-AVC state,
malformed AVC/Annex B `NEW_EXTRADATA` non-fatal packet-side-data guards,
including partially valid Annex B side data rollback before commit,
transactional public `ParseHeadersAnnexB`/`ParseHeadersAVC` rollback when a
foreign SPS is followed by malformed PPS,
malformed in-band SPS/PPS recovery across configured AVC and mixed
configured-AVC/Annex B public decode surfaces,
packet side-data byte-slice ownership across delayed B-frame output,
first-entry duplicate packet side-data semantics for scalar values, structured
layouts, and ICC/HDR10+/LCEVC byte payloads, including empty or malformed first
entries,
raw-output caller-buffer preservation on invalid 8-bit chroma geometry and
high-bit-depth luma/chroma sample errors, `RawYUVSize` overflow rejection, and
raw-output appender no-panic rejection for overflowed caller-constructed frame
geometry, checked low/high internal motion and reconstruction plane-span
overflow rejection,
public SEI side-data byte-slice ownership against caller mutation,
raw-diff/frame-MD5 diagnostics, and CLI benchmark comparison.

Realtime/WebRTC encoder proof currently covers control admission and
parameter-set headers: default 8-bit I420 constrained-baseline RTP config,
invalid-control rejection including I420 crop-unit bounds and crop-sum overflow, bitrate, framerate,
RTP payload-size, SPS/PPS cadence modes, PLI/FIR/force-IDR, partial
reconfiguration, public SPS/PPS/Annex B/avcC header generation with crop
metadata, recovery-point SEI packaging, IDR IntraPCM Annex B/AVC frame
generation, cropped visible-frame decode, identical-reference CAVLC P-skip,
bounded exact macroblock-aligned P16x16 no-residual prediction with
8-pixel integer-pel search, including mixed per-macroblock vectors and
odd-pixel luma motion only when both 4:2:0 chroma planes are constant,
Annex B/configured AVC/RTP plus mode-0
packetization proof for both admitted constant-chroma motion and patterned-chroma
P IntraPCM fallback, and single-macroblock enabled/slice-boundary deblock proof,
changed-frame P IntraPCM
recovery pictures, forced-keyframe IDR, FFmpeg rawvideo decode,
runtime resolution reset that rejects stale-size frames without consuming the
queued IDR before emitting/decoding a new-size IDR and resuming P-skip
references at the new dimensions,
invalid frame-rate helper/reconfigure plus invalid runtime rate,
latency/slice, output/header/preset, RTP re-entry payload-size, and
packetization updates that leave config, queued-IDR state, RTP cadence, packets,
and callbacks intact,
accepted bitrate, frame-rate, RTP payload-size, and deblock setter or
reconfigure updates, including SPS/PPS cadence and recovery-point toggles,
preserving a queued IDR until an IDR is emitted,
`SetRTPMaxPayloadSize` live RTP FU-A retargeting before the next P-frame with
sequence/decode continuity and invalid-update rollback,
runtime `RecoveryPointSEI` toggles that add, suppress, and restore changed-P
recovery side data without forcing IDR,
runtime SPS/PPS cadence switches that control forced-IDR header emission across
out-of-band, every-IDR, suppressed in-band, and restored in-band modes while the
stream remains decodable,
runtime RTP-to-configured-AVC output switching that forces an out-of-band IDR,
stops RTP packets/callbacks, preserves RTP timestamp cadence, and decodes the
AVC IDR/P-skip sequence through the emitted avcC,
runtime configured-AVC-to-RTP output switching that forces an every-IDR RTP
frame, starts RTP sequence numbers and callbacks from the first emitted packet,
carries retargeted payload metadata, and decodes the RTP IDR/P-skip sequence,
caller-owned parameter-set and recovery-point SEI surfaces with mutation/append
isolation across repeated helper calls,
public input/result/callback surface guards for integration-facing encoder
structs,
input-frame plane ownership guards proving post-call caller mutation does not
change the next Annex B, AVC, or RTP encode,
returned `Encode` result lifetime guards across later Annex B, AVC, and RTP
encodes,
invalid-frame validation that returns empty output and leaves RTP sequence,
callback, frame-number, timestamp, and reference state untouched before the next
valid P-skip,
`MaxFrameSize`/`SliceMaxBytes` hard-error and `FrameDropToBitrate`
dropped-frame budget behavior including VBV-backed `MaxBitrate` bucket misses,
deterministic max-bitrate credit consumption/refill across transmitted and
dropped frames, caller-buffer `EncodeInto` budget drops with empty returned
output, paused callbacks, preserved reference/frame/packet state, overflowed
`EncodeInto` destination growth hard-error rollback with queued-IDR preservation
and P-frame reference/frame-number preservation across Annex B/AVC/RTP,
overflowed bitrate budget derivation rejection, overflowed input-plane geometry
rejection, and P-skip
recovery, runtime frame-drop mode switches that toggle the derived bitrate
budget before the next frame, runtime max-bitrate/VBV lowering that resets stale
credit before the next frame, `SetBitrate` lowering that resets stale
frame-budget credit before the next frame, `SetFrameRate` changes that reset
frame-budget credit and apply the updated RTP cadence across drop/recovery
while rolling back derived RTP timestamp overflow or zero-increment underflow,
`FrameDropLate` bypass of the derived bitrate budget when the encode-time budget
admits the frame,
ConstantQP bitrate-budget bypass across runtime rate-control mode switches,
`FrameDropLate` `MaxEncodeTimeUS` budget drops
without reference/frame/packet/callback advancement before and after an existing
reference, RTP/Annex B/RTP control-loop stress across QP refresh, late drop
recovery, packet metadata retargeting, paused RTP sequence/callback state, and
local decode after RTP re-entry, public NAL-unit metadata indexing back into
encoded access-unit bytes for Annex B/AVC/RTP output including non-empty
caller-buffer prefixes, caller-buffer preservation on RTP mode-0 rejection and
mode-0 oversize queued-IDR/P-frame packetization rollback, Annex B/AVC/RTP
bitrate-drop and late-drop non-output paths, checked
access-unit and RTP packet storage-size overflow rejection, RTP packetization-mode 0 single-NAL IDR/P-frame
reassembly and oversize rejection, RTP packetization-mode 1 FU-A reassembly,
STAP-A parameter-set aggregation plus small-payload fallback to non-aggregated
mode-1 packets, RTP payload-type/SSRC/sequence metadata, full
RTP header bytes with clipped packet payload views over packet data and packet
storage isolated from `EncodedFrame.Data` including caller-backed `EncodeInto`
output buffers, shared packet metadata guards for
header fields and clipped packet slices, optional
RTP callback metadata including mode 0/1 IDR/P-frame single-NAL packets with
P-skip, exact P16x16, odd-pixel constant chroma, and P IntraPCM fallback rows
and callback packet storage isolated from returned RTP packets while preserving
the clipped payload-over-packet-data shape, initial
`EncodeInto` allocation canaries for caller-buffer Annex B/AVC/RTP IDR,
P-skip, exact-P16x16 including odd-pixel constant-chroma across Annex B/AVC/RTP
and 8-pixel edge search, per-macroblock exact P16x16 across Annex B/AVC/RTP,
odd-pixel patterned-chroma P IntraPCM fallback across Annex B/AVC/RTP,
P-IntraPCM including AVC/RTP changed P IntraPCM, and RTP packetization-mode 0
IDR/P-frame paths including odd-pixel constant-chroma, per-macroblock exact
P16x16, odd-pixel patterned-chroma fallback, and exact-P16 edge search, and a
live encode path that avoids discarded Annex B/AVC copies when only raw NAL
output is needed. The admitted forced-IDR and fallback paths build raw SPS/PPS
and recovery-point SEI NALs directly, with tightened budgets of <=8 allocations
for Annex B/AVC forced IDR, <=10 for RTP forced IDR/FU-A and STAP-A, <=6 for Annex B/AVC
odd-patterned P IntraPCM fallback, <=8 for RTP odd-patterned fallback, <=5 for
Annex B/AVC per-macroblock exact P16x16, and <=7 for RTP per-macroblock exact
P16x16, <=12 for Annex B/AVC changed P IntraPCM, and <=16 for RTP changed
P IntraPCM, plus <=7 for Annex B/AVC/RTP `EncodeInto` max-frame-size and
slice-max-bytes drops, <=8 for Annex B/AVC/RTP late drops, and stack-backed common
one-slice NAL and slice-range planning,
and access-unit full RTP packet buffer stamping with clipped payload views and
per-packet append isolation plus frame-data ownership separation.
Package-level benchmark canaries cover admitted Annex B/AVC/RTP IDR and
P-frame encode paths, including odd-pixel constant-chroma exact P16x16 across
Annex B/AVC/RTP plus RTP mode 0, odd-pixel patterned-chroma P IntraPCM fallback
across Annex B/AVC/RTP plus RTP mode 0, per-macroblock exact P16x16 across
Annex B/AVC/RTP plus RTP mode 0, 8-pixel exact-P16 edge search, AVC/RTP changed
P IntraPCM, RTP STAP-A IDR/changed-P, RTP max-frame-size/late drop paths, and
RTP packetization-mode 0 IDR/P-frame rows including exact-P16 edge search, with `-benchmem`.
Internal writer proof covers raw
bit/Exp-Golomb
writing, RBSP trailing bits, EBSP emulation-prevention, Annex B/AVC NAL
packaging, AVC decoder configuration records, baseline SPS/PPS syntax with
4:2:0 crop-unit emission, recovery-point SEI syntax, and Baseline IDR plus
P-skip/P16x16 no-residual/P IntraPCM slice syntax via decoder-parser and
encoded-frame round trips, including per-macroblock MVD syntax for the P16x16
writer.

Public vectors: 226 imported public refs, 225 selected decoder-facing manifest
rows, 225 green oracle rows, 0 known-red, and one explicit non-decoder
exclusion. The always-on
`TestH264DecoderTDDContractClassifiesEveryImportedPublicVector` test treats
that inventory as the decoder TDD backlog: every imported ref must be executable
in the manifest, documented as an exclusion, or recorded as a known-red
manifest row. Matrix mode is the safe-point gate. Green oracle rows include
decode-ok, metadata-ok, and expected decode-error rows. The strict script
excludes only rows in
`testdata/h264/realvectors/failures.jsonl`, which is currently empty; the
failure-ledger freshness and matrix gates execute known-red rows when present
and require them to remain current. The red-queue script intentionally exits
non-zero while selected known-red rows remain red.
Use `scripts/h264-real-vector-red-queue.sh <filter>` or
`scripts/h264-red-vector.sh <filter>` to hand agents a failing public lane; the
single-lane script exits at the first divergent raw byte for raw-MD5 rows.
Use `scripts/h264-real-vector-red-each.sh` for per-known-red logs plus a TSV
summary of first-divergence evidence.

Imported public-vector inventory: 224 refs are generated from the pinned FFmpeg
`n8.0.1` FATE/CBS scan and 2 additional public fate-suite H.264/LCEVC auxiliary
container refs are represented in the manifest. The FATE-derived subset has 223
refs represented in the public-vector manifest and one explicitly excluded
non-decoder row. `h264-conformance/FM1_BT_B.h264` is a negative decode-error
manifest row; `mkv/h264_tta_undecodable.mkv` remains excluded because it has no
H.264 video stream.

Still guarded: unselected MBAFF/PIC-AFF/PAFF motion paths, broad public field/MBAFF
high modes, public 12/14-bit
high streams beyond the current FFmpeg FATE 8-bit/10-bit set, broader
damaged-slice error resilience, threading/SIMD, and full libavcodec
delayed-output behavior.

Intentionally unsupported at the pinned FFmpeg parity boundary: FMO, 11/13-bit
luma depths, `chroma_format_idc > 3`, separate color planes, and mixed
chroma/luma bit depths. FFmpeg n8.0.1 has `FMO` compiled out in
`libavcodec/h264dec.h`, and `ff_h264_decode_picture_parameter_set` returns
PATCHWELCOME as soon as `num_slice_groups_minus1 > 0` or `bit_depth_luma` is
11 or 13; local PPS parsing mirrors both with `ErrUnsupported`. FFmpeg
`ff_h264_decode_seq_parameter_set` also fails SPS admission for unsupported
chroma formats, separate color planes, and different chroma/luma bit depths;
local SPS parsing mirrors those with `ErrUnsupported`.

Frame-MBAFF direct-sub B4x4 rows are not listed as missing proof: non-frame-only
SPS syntax must set `direct_8x8_inference_flag`, so the valid generated
frame-MBAFF direct-sub B surface is B8x8-only.

Canonical detail lives in manifests and tests, not Markdown.
