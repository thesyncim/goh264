# Production Readiness

The decoder is the current implemented path. Realtime/WebRTC encoder support is
now in scope, with a tested public control contract in `encoder.go`. Encoder
bitstream generation now has a first admitted 8-bit I420 Constrained Baseline
IDR/IntraPCM path with Annex B, AVC, and RTP packetization-mode 1 output, plus
guarded identical-reference CAVLC P-skip when deblocking is disabled. Encoder
production gates live in `docs/encoder-webrtc-roadmap.md` until changed-frame
P prediction, residual coding, rate control, RTP callback metadata, allocation
budgets, and oracle evidence land.

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
scripts/h264-benchstat-canary.sh      # benchstat-compatible decoder canary
scripts/h264-performance-evidence.sh  # local benchstat, JSON, CPU, and heap profile bundle
scripts/h264-decoder-release-evidence.sh # full decoder release-evidence runner
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
generated FATE rows against the pinned FFmpeg source.

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
allocations for 8-bit and high-bit-depth output paths. `cmd/goh264bench`
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
for one-shot Annex B decode and stateful Annex B access-unit streaming with
`-benchmem`; its output is suitable for `benchstat` trend comparisons.
`scripts/h264-performance-evidence.sh` writes a local evidence bundle under
`.artifacts/h264-performance-evidence/` containing benchstat samples, the JSON
real-vector benchmark report, CPU and heap profiles, and run metadata. Pending:
checked-in reviewed profile artifacts, larger performance corpus, and
in-process libavcodec baseline.
`scripts/h264-decoder-release-evidence.sh` runs the decoder release checklist
as one gate and writes per-command logs plus the local performance bundle under
`.artifacts/h264-release-evidence/`. It fails by default if the known-red
failure ledger is non-empty or the worktree is dirty.

Public API delayed-output coverage includes AVC configured samples and Annex B
access-unit streaming through `DecodeFrames`, with end-of-stream flush via an
empty packet and an empty second flush. Configured AVC stateful decode also
drains delayed B-frame output through `DecodeConfiguredAVCFrames(nil)`,
including the single-frame helper when the empty packet releases exactly one
frame. AVC-with-configuration-record decode follows the same empty-packet
delayed flush rule after updating parameter sets, with fixture-matrix coverage
for 2/3/4-byte AVC length sizes.

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
packet still decodes against the last good configuration and reference state.
In-band malformed SPS/PPS NALs are also guarded as non-fatal: they do not
replace the last good parameter sets before the next valid slice on configured
AVC or mixed configured-AVC/Annex B public decode paths.

Encoder readiness evidence currently covers controls, parameter-set headers,
recovery-point SEI packaging, and the first IDR frame writer:
`tests/encoder_webrtc_controls_test.go` proves the default WebRTC config,
rejects invalid or not-yet-admitted realtime controls, validates runtime
bitrate, framerate, payload-size, PLI/FIR, force-IDR, and partial
reconfiguration paths, proves `ParameterSets` emits SPS/PPS NALs, Annex B
headers, and avcC records accepted by the public decoder parsers, proves
`RecoveryPointSEI` emits caller-owned Annex B/AVC recovery-point SEI NAL
surfaces accepted by public decode paths, verifies frame-shape validation, and
proves `Encode`/`EncodeInto` emit IDR IntraPCM access units that round-trip
through local Annex B/AVC decode, FFmpeg rawvideo decode, RTP FU-A reassembly,
STAP-A parameter-set aggregation, and RTP packet payload-type/SSRC/sequence
metadata plus full RTP header bytes. It also proves identical second frames can
emit CAVLC P-skip slices through stateful local decode and FFmpeg rawvideo
decode, while changed second frames and queued IDR requests fall back to IDR.
Internal encoder writer evidence now covers raw bit/Exp-Golomb writing, RBSP
trailing bits, EBSP emulation-prevention, Annex B/AVC NAL packaging, AVC
decoder configuration records, baseline SPS/PPS, recovery-point SEI syntax, and
Baseline IDR plus P-skip slice syntax. Changed-frame P prediction, residual
CAVLC coding, rate-control feedback, RTP callback metadata, and realtime
allocation/performance gates remain pending.
