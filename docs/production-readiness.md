# Production Readiness

The decoder is the current implemented path. Realtime/WebRTC encoder support is
now in scope, with a tested public control contract in `encoder.go`. Encoder
bitstream generation now has a first admitted 8-bit I420 Constrained Baseline
IDR/IntraPCM path with Annex B, AVC, RTP packetization-mode 0 single-NAL
output, and RTP packetization-mode 1 output, plus guarded identical-reference
CAVLC P-skip, bounded exact macroblock-aligned P16x16 no-residual prediction
for frame-wide and per-macroblock integer-pel shifts up to 8 pixels under
disabled-deblock multi-macroblock frames plus single-macroblock
enabled/slice-boundary deblock, with odd-pixel
luma motion admitted only when both 4:2:0 chroma planes are constant,
and changed-frame
P IntraPCM across disabled, enabled, and slice-boundary deblock controls.
Encoder
production gates live in `docs/encoder-webrtc-roadmap.md` until broader
P prediction, residual coding, rate control, remaining packetizer breadth,
allocation budgets, and oracle evidence land.
The encoder bitstream-writer residual work is still deliberately bounded, but
the CAVLC single-level and single-level-plus-trailing-ones residual writers now
round-trip both short and decoder-supported prefix-14/prefix-15 first-level
codes, and two-, three-, four-, five-, six-, seven-, eight-, nine-, ten-, eleven-, twelve-, and thirteen-non-trailing-level writers now round-trip subsequent
suffix-length transitions plus the totalCoeff>10 first-level suffix transition, before broader residual macroblock admission.

Harness-first status:

```sh
GOH264_REAL_VECTOR_FAILURES=1 GOH264_CORPUS_FETCH=1 go test ./tests -run TestH264RealVectorFailureLedgerFreshness
GOH264_REAL_VECTOR_MATRIX=1 GOH264_CORPUS_FETCH=1 go test ./tests -run TestH264RealVectorFailureMatrix
scripts/h264-real-vector-strict.sh      # strict green public-vector oracle
scripts/h264-real-vector-red-queue.sh   # exits non-zero only while known-red rows remain
scripts/h264-real-vector-red-each.sh    # per-row red queue report when the ledger is populated
scripts/h264-real-vector-upstream-audit.sh # pinned FFmpeg H.264 FATE coverage
scripts/h264-decoder-fuzz-smoke.sh # bounded public decode/packet no-panic fuzz smoke
scripts/h264-real-vector-bench.sh canl4 # set GOH264_BENCH_FFMPEG=1 GOH264_BENCH_FAIR_CPU_LANES=1 for pure C vs pure Go and native C+asm vs Go+asm lanes
scripts/h264-real-vector-release-alloc.sh # checked-in Go allocation canary budget
scripts/h264-benchstat-canary.sh      # benchstat-compatible decoder/encoder canary
scripts/h264-performance-evidence.sh  # local benchstat, JSON, CPU, and heap profile bundle
scripts/h264-release-evidence.sh # combined decoder and admitted encoder release runner
scripts/h264-decoder-release-evidence.sh # full decoder release-evidence runner
scripts/h264-encoder-release-evidence.sh # admitted encoder contract/writer/bench runner
go test ./tests -run TestEncoder # realtime/WebRTC encoder control contract
GOH264_REAL_VECTOR_STRICT=1 GOH264_CORPUS_FETCH=1 go test ./tests -run TestH264RealVectorStrictOracle
GOH264_REAL_VECTOR_RED=1 GOH264_CORPUS_FETCH=1 go test ./tests -run TestH264RealVectorKnownRedStrict
GOH264_REAL_VECTOR_RED_QUEUE=1 GOH264_CORPUS_FETCH=1 go test ./tests -run TestH264RealVectorRedQueue
GOH264_REAL_VECTOR_RAWDIFF=1 GOH264_CORPUS_FILTER=mbaff GOH264_CORPUS_FETCH=1 go test ./tests -run TestH264RealVectorRawDiffDiagnostics
scripts/h264-red-vector.sh mbaff        # exits non-zero at first divergent raw byte
go run ./cmd/goh264bench -manifest testdata/h264/realvectors/manifest.jsonl -filter canl4 -iters 10 -repeats 5 -warmup 2 -ffmpeg -fair-cpu-lanes -ffmpeg-threads 1 -strict-pix-fmt -json
```

`scripts/h264-real-vector-strict.sh` runs the green public-vector oracle set,
including expected decode-error rows, and excludes only rows currently listed
in the failure ledger. Use
`GOH264_REAL_VECTOR_FAILURES=1` or `GOH264_REAL_VECTOR_MATRIX=1` for the gates
that execute and verify known-red rows when present.

The checked-in public-vector inventory at
`testdata/h264/realvectors/upstream-inventory.jsonl` currently imports 226
public H.264 refs: 224 generated from pinned FFmpeg `n8.0.1` FATE makefiles and
2 auxiliary public fate-suite H.264/LCEVC container samples. Normal
`go test ./tests` requires every imported ref to be represented by the manifest
or by `testdata/h264/realvectors/exclusions.jsonl` through
`TestH264DecoderTDDContractClassifiesEveryImportedPublicVector`; any future
failing decoder-facing row belongs in `failures.jsonl` with a current failure
signature until it is fixed. The upstream-audit script also verifies the 224
generated FATE rows against the pinned FFmpeg source and checks that the
release-doc public-vector counts match the checked-in manifests.

Benchmark JSON reports selected/green/known-red counts, backend kind, CPU flags,
comparison lane, oracle `quality_status`, Go allocation totals plus
per-iteration/per-frame allocation rates, and FFmpeg-vs-Go
`peer_quality_status`. Diagnostic mode includes expected decode-error rows as
oracle rows and requires the observed decoder error to contain `expected_error`.
`-cpuprofile` and `-memprofile` write CPU and heap profiles around the
oracle-checked benchmark run; `scripts/h264-real-vector-bench.sh` forwards
`GOH264_BENCH_CPU_PROFILE` and `GOH264_BENCH_MEM_PROFILE` to those flags.
Use `-fair-cpu-lanes` for both `pure-c-vs-pure-go` and
`native-c+asm-vs-go+asm`; extracted container rows require FFmpeg on `PATH`
when the cache does not already contain the `.h264-annexb` derived stream.
Result `backend_kind` records the backend actually
measured, so current no-asm Go builds still report `go-pure`.

Allocation evidence: `tests/decoder_high_output_test.go` guards
`Frame.AppendRawYUV`, `Frame.AppendRawYUVBytesLE`, and `Frame.AppendRawYUV16`
with exact-capacity caller-owned buffers and requires zero steady-state
allocations for 8-bit and high-bit-depth output paths, including caller-buffer
preservation on invalid 8-bit chroma geometry and high-bit-depth luma/chroma
sample errors. `Frame.RawYUVSize` and the raw-output appenders reject
overflowed caller-constructed frame geometry instead of returning wrapped byte
counts or panicking during plane slicing. Internal low/high motion and
reconstruction plane-span checks reject overflowed geometry before indexing.
Public SEI `FrameSideData` byte slices are guarded as caller-owned by mutating
decoded unregistered-user-data, A53, and LCEVC slices and re-decoding the same
input bytes.
Malformed `ParseHeadersAnnexB` and `ParseHeadersAVC` calls are guarded as
transactional: a partially valid foreign SPS followed by malformed PPS returns
an error without replacing the previous configured-AVC state or draining
delayed B-frame output.
`cmd/goh264bench`
records Go benchmark allocation totals and reports `alloc_bytes_per_iter`,
`allocs_per_iter`, `alloc_bytes_per_frame`, and `allocs_per_frame` for each
timed Go lane; `-max-go-alloc-bytes-per-iter` and
`-max-go-allocs-per-iter` turn those rates into failing budget gates. The
real-vector benchmark script forwards
`GOH264_BENCH_MAX_GO_ALLOC_BYTES_PER_ITER` and
`GOH264_BENCH_MAX_GO_ALLOCS_PER_ITER` to those flags.
`scripts/h264-real-vector-release-alloc.sh` is the checked-in release canary:
it runs the CANL4 public vector with defaults of 64,000,000 Go allocation
bytes/iteration and 10,000 Go allocations/iteration.
`scripts/h264-benchstat-canary.sh` runs the package-level decoder benchmarks
for one-shot Annex B decode, stateful Annex B access-unit streaming, and isolated
raw-output export plus
the admitted encoder Annex B/AVC/RTP IDR/P-skip/exact-P16
edge-search/P-IntraPCM and RTP packetization benchmarks with `-benchmem`; its
output is suitable for `benchstat` trend comparisons. `GOH264_BENCHSTAT_TIME`
sets the effective `-benchtime`; `GOH264_BENCHSTAT_BENCHTIME` is accepted as an
alias when `GOH264_BENCHSTAT_TIME` is unset, and the performance/release
evidence metadata records the effective value.
`scripts/h264-performance-evidence.sh` writes a local evidence bundle under
`.artifacts/h264-performance-evidence/` containing benchstat samples, the JSON
real-vector benchmark report, CPU and heap profiles, and run metadata. Pending:
checked-in reviewed profile artifacts, larger performance corpus, and
in-process libavcodec baseline.
`scripts/h264-release-evidence.sh` is the top-level release gate. It requires a
clean worktree by default, writes summary/log output under
`.artifacts/h264-full-release-evidence/`, and runs the decoder release-evidence
runner and the admitted encoder release-evidence runner into separate child
directories.
`scripts/h264-decoder-release-evidence.sh` runs the decoder release checklist
as one gate and writes per-command logs plus the local performance bundle under
`.artifacts/h264-release-evidence/`. It fails by default if the known-red
failure ledger is non-empty or the worktree is dirty.
`scripts/h264-encoder-release-evidence.sh` runs the admitted encoder local
release checklist as one gate: full tests, the external encoder control
contract, the explicit `EncodeInto` allocation canary, internal
writer/header/SEI slice syntax tests, and admitted encoder `-benchmem` rows.
It writes per-command logs under
`.artifacts/h264-encoder-release-evidence/` and fails by default if the
worktree is dirty.

Public API delayed-output coverage includes AVC configured samples and Annex B
access-unit streaming through `DecodeFrames`, with end-of-stream flush via an
empty packet and an empty second flush. Configured AVC stateful decode also
drains delayed B-frame output through `DecodeConfiguredAVCFrames(nil)`,
including the single-frame helper when the empty packet releases exactly one
frame. AVC-with-configuration-record decode follows the same empty-packet
delayed flush rule after updating parameter sets, with fixture-matrix coverage
for 2/3/4-byte AVC length sizes, and zero-frame empty configuration-record
calls store configuration while returning no frames or `ErrUnsupported` from the
single-frame helper without retaining caller-owned avcC storage. Packet decode with repeated valid
`NEW_EXTRADATA` also preserves B-frame reorder state through delayed flush, and
frame-list and single-frame empty-packet flush ignore malformed replacement
`NEW_EXTRADATA` without blocking delayed output or replacing stored AVC
configuration, including when no delayed frame is available and the single-frame
helper reports `ErrUnsupported`.
Successful and malformed Annex B/AVC header parses preserve delayed
configured-AVC B-frame flush state.

Malformed-input safety evidence now includes deterministic public-surface
corruption rows plus `FuzzDecodePublicSurfacesNoPanic`, a bounded fuzz target
for Annex B, AVC, configured AVC, auto-detect, and packet side-data decode
surfaces. `scripts/h264-decoder-fuzz-smoke.sh` runs that target for a short
smoke window and is part of the release-evidence gate.

Damaged-packet recovery evidence includes valid-damaged-valid stateful decode
guards for configured AVC samples, AVC samples decoded with a configuration
record, packet `NEW_EXTRADATA`, and auto-detected Annex B access units. A
truncated VCL packet must return an error without preventing the next valid
packet from matching the raw-MD5 oracle. Configured AVC, AVC with a
configuration record, bare AVC, packet AVC, stateful Annex B, and one-shot
Annex B also return valid frames decoded before a later damaged slice in the
same packet alongside the damaged-slice error, including the sole valid frame on
single-frame decode helpers and delayed B-frame prefix output from
configuration-record one-shot decode. Packet `NEW_EXTRADATA` recovery also guards
malformed AVC and Annex B extradata as non-fatal side data: the current valid
packet still decodes against the last good configuration and reference state,
including when Annex B side data stages a valid foreign SPS before a malformed
PPS.
Direct avcC parse, configuration-record decode, valid packet AVC and Annex B
`NEW_EXTRADATA`, and auto-detected avcC admission also prove the stored decoder
configuration state does not retain caller-owned side-data, configuration, or
packet buffers after return.
Malformed standalone AVC decoder configuration records are also guarded: failed
`ParseAVCDecoderConfigurationRecord` and auto-detected `DecodeFrames` config
updates leave the last stored configuration usable for configured AVC decode.
In-band malformed SPS/PPS NALs are also guarded as non-fatal: they do not
replace the last good parameter sets before the next valid slice on configured
AVC or mixed configured-AVC/Annex B public decode paths.
Packet side-data byte payloads are copied before delayed B-frame storage:
mutating caller-owned packet side-data immediately after each decode call does
not affect immediate or flushed `FrameSideData`.
Packet side-data duplicate handling follows first-entry semantics for scalar
values, structured layouts, and byte payloads: empty or malformed first
active-format and S12M entries plus empty first ICC, HDR10+, and LCEVC entries
suppress later duplicates.
Configured AVC and Annex B delayed-output guards also overwrite caller-owned
compressed packet buffers after each decode call before flushing delayed frames,
proving delayed output does not retain input packet storage.

Encoder readiness evidence currently covers controls, parameter-set headers,
recovery-point SEI packaging, and the first IDR frame writer:
`tests/encoder_webrtc_controls_test.go` proves the default WebRTC config,
guards public input/result/callback surfaces for integration-facing encoder
structs, rejects invalid or not-yet-admitted realtime controls, validates runtime
bitrate, framerate, payload-size, SPS/PPS cadence, PLI/FIR, force-IDR, and
partial reconfiguration paths, proves invalid frame-rate helper/reconfigure,
runtime rate, latency/slice, output/header/preset, RTP re-entry payload-size,
RTP metadata, and packetization updates leave config, queued-IDR state, RTP
cadence, packets, and callbacks intact across Annex B, AVC, and RTP output
surfaces where the controls apply,
proves accepted bitrate, frame-rate, RTP payload-size, and deblock setter or
reconfigure updates, including SPS/PPS cadence and recovery-point toggles,
preserve a queued IDR until the next successful IDR encode,
proves runtime resolution reset rejects stale-size frames without
consuming the queued IDR before emitting/decoding a new-size IDR and resuming
P-skip references at the new dimensions, proves
`SetRTPMaxPayloadSize` live RTP FU-A retargeting before the next P-frame with
sequence/decode continuity and invalid-update rollback, proves runtime
`RecoveryPointSEI` toggles add, suppress, and restore changed-P recovery side
data without forcing IDR, proves runtime SPS/PPS cadence switches control
forced-IDR header emission across out-of-band, every-IDR, suppressed in-band,
and restored in-band modes while the stream remains decodable, proves
RTP-to-configured-AVC output switching forces an out-of-band IDR, stops RTP
packets/callbacks, preserves RTP timestamp cadence, and decodes the AVC
IDR/P-skip sequence through the emitted avcC, proves
configured-AVC-to-RTP output switching forces an every-IDR RTP frame, starts RTP
sequence numbers and callbacks from the first emitted packet, carries retargeted
payload metadata, and decodes the RTP IDR/P-skip sequence, proves
`ParameterSets` emits SPS/PPS NALs,
Annex B headers, and avcC records accepted by the public decoder parsers,
with caller mutation/append isolation proved across repeated helper calls,
proves in-band keyframe, suppressed in-band, out-of-band, and every-IDR header
modes across Annex B, AVC, and RTP, proves force-IDR, PLI, FIR, and per-frame
keyframe requests deliver IDR output across those surfaces, proves
`RecoveryPointSEI` emits caller-owned Annex B/AVC recovery-point SEI NAL
surfaces accepted by public decode paths with the same mutation/append
isolation, and proves valid and invalid header/SEI helper calls preserve queued
IDR, config, callback, RTP packet metadata, and frame-number state across Annex
B, AVC, and RTP,
proves encoded input-frame planes are not retained after `Encode` returns by
mutating caller-owned first-frame storage before a matching second frame across
Annex B, AVC, and RTP output,
proves returned `Encode` results remain stable after later encodes across
access-unit bytes, NAL metadata, and RTP packets, and proves returned RTP
packets do not alias caller-backed `EncodeInto` access-unit data,
verifies frame-shape validation returns
empty output and leaves RTP sequence, callback, frame-number, timestamp, and
reference state untouched before the next valid P-skip or queued IDR, proves
overflowed caller-owned `EncodeInto` destination growth returns an empty hard
error across Annex B, AVC, and RTP without consuming queued IDR state or
advancing RTP/callback state, and preserves P-frame reference/frame-number state
before the next P-skip, rejects overflowed configured and per-frame input-plane
geometry without wrapped sizes or panics, and
proves hard-error and `FrameDropToBitrate` dropped-frame behavior for
`MaxFrameSize`/`SliceMaxBytes` budget misses and VBV-backed `MaxBitrate` bucket
misses, including credit consumption/refill across transmitted IDR/P-skip and
dropped changed-P frames, caller-buffer `EncodeInto` budget drops with empty
returned output, no callbacks, stable reference/frame/packet state, and P-skip
recovery, checked rejection of overflowed bitrate budget derivation, runtime frame-drop mode switches that toggle the derived bitrate
budget before the next frame, runtime max-bitrate/VBV lowering that resets stale
credit before the next frame, `SetBitrate` lowering that
resets stale frame-budget credit before the next frame, `SetFrameRate` changes
that reset frame-budget credit and apply the updated RTP cadence across
drop/recovery, including rollback when derived RTP timestamp arithmetic
overflows or underflows to zero, `FrameDropLate` bypass of the derived bitrate
budget when the encode-time budget admits the frame, plus ConstantQP bypass of the derived
bitrate budget before and after runtime rate-control mode switches. `FrameDropLate`
`MaxEncodeTimeUS` budget misses advance RTP time without advancing reference,
frame-number,
packet-sequence, or callback state, including after a transmitted reference
frame. A combined control-loop stress row now switches
RTP to Annex B and back through QP refresh, late drop, P-skip recovery, packet
metadata retargeting, and local decode after RTP re-entry while proving RTP
sequence numbers and callbacks pause when no RTP packets are emitted. It proves
`Encode`/`EncodeInto`
emit IDR IntraPCM access units that round-trip through local Annex B/AVC decode,
FFmpeg rawvideo decode, RTP FU-A reassembly, RTP packetization-mode 0 single-NAL
IDR/P-frame reassembly and oversize rejection, STAP-A parameter-set aggregation
with callback metadata and packet isolation, and RTP
small-payload STAP-A fallback to non-aggregated mode-1 packets with decode and
sequence continuity and accurate non-STAP-A callback payload metadata plus
callback packet isolation for the fallback IDR and next P-skip, plus RTP
public NAL-unit metadata indexing back into encoded access-unit bytes for Annex
B, AVC, and RTP output,
including `EncodeInto` calls that append after an existing caller-buffer prefix,
and preserves caller-owned `EncodeInto` storage on RTP mode-0 rejection and
proves mode-0 oversize queued-IDR and P-frame packetization failures leave
pending-IDR, reference, RTP sequence, and callback state recoverable, plus
Annex B/AVC/RTP bitrate-drop and late-drop non-output paths,
packet payload-type/SSRC/sequence metadata plus full RTP header bytes with
clipped packet payload views over packet data, packet storage isolated from
`EncodedFrame.Data` including caller-backed `EncodeInto` output buffers, shared
packet metadata guards for header fields and clipped packet slices, checked
access-unit and RTP packet storage-size overflow rejection, and optional RTP callback metadata including mode 0/1
IDR/P-frame single-NAL packets for multi-slice IDR, P-skip, exact P16x16,
odd-pixel constant chroma, and P IntraPCM fallback rows with callback packet storage isolated from
returned RTP packets while preserving the same clipped payload-over-packet-data
shape. It also proves identical
second frames can emit
CAVLC P-skip slices through stateful local decode and FFmpeg rawvideo decode,
exact shifted macroblock-aligned frames up to 8 pixels, including mixed
per-macroblock vectors and odd-pixel luma motion when chroma is constant, can
emit predictive P16x16 no-residual slices through stateful local decode, FFmpeg
rawvideo decode, configured AVC decode, RTP reassembly decode, and
single-macroblock enabled/slice-boundary deblock syntax plus configured AVC/RTP
decode and RTP mode-0 single-NAL packetization, while patterned chroma is
proved to fall back to P IntraPCM
across Annex B, configured AVC, RTP reassembly, and RTP mode-0 single-NAL output,
changed second frames can emit P IntraPCM recovery pictures across disabled,
enabled, and slice-boundary deblock controls, and queued IDR requests emit IDR.
It now includes `EncodeInto` allocation canaries for caller-buffer Annex B
forced IDR, Annex B steady P-skip, Annex B exact P16x16 including
single-macroblock deblock controls, Annex B odd-pixel constant-chroma exact
P16x16, Annex B odd-pixel patterned-chroma P IntraPCM fallback, Annex B
macroblock-aligned exact P16x16 including 8-pixel edge search, Annex B
per-macroblock exact P16x16, Annex B changed P IntraPCM, AVC forced IDR, AVC
steady P-skip, AVC odd-pixel constant-chroma exact P16x16, AVC odd-pixel
patterned-chroma P IntraPCM fallback, AVC exact P16x16 including 8-pixel edge
search, AVC per-macroblock exact P16x16, AVC changed P IntraPCM, RTP forced
IDR/FU-A, RTP forced IDR with STAP-A, RTP odd-pixel constant-chroma exact P16x16, RTP odd-pixel
patterned-chroma P IntraPCM fallback, RTP exact P16x16 including 8-pixel edge
search, RTP per-macroblock exact P16x16, RTP steady P-skip, RTP changed
P IntraPCM, and RTP packetization-mode 0 IDR/P-frame paths including odd-pixel
constant-chroma, odd-pixel patterned-chroma fallback, per-macroblock exact
P16x16, and exact-P16 edge search. The current tightened budgets guard <=8 allocations for
Annex B/AVC forced IDR, <=10 for RTP forced IDR/FU-A and STAP-A, <=6 for Annex B/AVC
odd-patterned P IntraPCM fallback, <=8 for RTP odd-patterned fallback, <=5 for
Annex B/AVC per-macroblock exact P16x16, <=7 for RTP per-macroblock exact
P16x16, <=12 for Annex B/AVC changed P IntraPCM, and <=16 for RTP changed
P IntraPCM, plus <=7 for Annex B/AVC/RTP `EncodeInto` max-frame-size and
slice-max-bytes drops and <=8 for Annex B/AVC/RTP late drops; the
forced-IDR and recovery-point fallback hot paths build raw SPS/PPS and SEI NALs
directly instead of constructing unused public packaging;
the live encode path writes RBSP plus raw NAL output directly instead of building
discarded Annex B/AVC copies, with common one-slice NAL and slice-range
planning backed by stack storage, and stamps full RTP packet data from
access-unit buffers while exposing clipped payload views, keeping packet storage
separate from `EncodedFrame.Data`, and preserving append isolation for each public
packet slice. Package-level benchmark canary rows now
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
exact-P16 edge search with `-benchmem`.
Internal encoder writer evidence now covers raw bit/Exp-Golomb writing, RBSP
trailing bits, EBSP emulation-prevention, CAVLC coeff-token/total-zeros/run-before
VLC writer table round-trips through the existing decoder tables, bounded CAVLC
trailing-ones, single-level, and single-level-plus-trailing-ones residual block
writer round-trips through `decodeCAVLCResidual` across luma and chroma-DC CAVLC
table contexts, Annex B/AVC NAL packaging, AVC decoder configuration records,
baseline SPS/PPS, recovery-point SEI syntax, and Baseline IDR, P-skip, and
P16x16 no-residual slice syntax including
per-macroblock MVD emission, with the P16x16 writer and public exact-motion
encoder path proved through local Annex B decode, FFmpeg rawvideo decode, and
mixed per-macroblock Annex B/AVC/RTP decode after an IDR reference frame.
Broader motion-search
P prediction, residual CAVLC coding, rate-control feedback, broader allocation
budgets, and realtime performance gates remain pending.
