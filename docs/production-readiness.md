# Production Evidence

`goh264` is a decoder-only H.264/AVC package. It does not provide H.264
bitstream generation or send-side controls.

The package can be imported without FFmpeg, cgo, or third-party Go modules.
FFmpeg is only part of the local evidence workflow. Patent and distribution
analysis remains the caller's responsibility, including for any surrounding
capture, transmit, storage, or playback pipeline.

| Path | Admitted Scope | Evidence Shape | Remaining Risk |
| --- | --- | --- | --- |
| Decoder | FFmpeg `n8.0.1` H.264 decoder path; public Annex B, AVC, avcC, packet, raw-output, side-data, and delayed-output surfaces | Unit tests, fixture tests, public-vector tests, strict FFmpeg-oracle rows, fuzz smoke, benchmark canaries, and quality-evidence runners | Broader field/MBAFF/PIC-AFF motion behavior, damaged-slice edge cases, allocation/performance review, threading/SIMD, exact libavcodec delayed-output edge behavior |

## Quality Gates

```sh
go test ./...
scripts/h264-quality-evidence.sh
scripts/h264-decoder-quality-evidence.sh
scripts/h264-real-vector-strict.sh
scripts/h264-decoder-fuzz-smoke.sh
scripts/h264-real-vector-quality-alloc.sh
scripts/h264-benchstat-canary.sh
scripts/h264-performance-evidence.sh
```

FFmpeg is required for oracle, corpus-fetch, extraction, and benchmark evidence
paths. It is not required to import the Go package.

## Public Vector Snapshot

The checked-in public-vector inventory currently imports 226 public H.264 refs:
224 generated from pinned FFmpeg `n8.0.1` FATE makefiles and 2 auxiliary public
fate-suite H.264/LCEVC container samples. The selected decoder manifest
currently has 225 green oracle rows and no known-red rows; the remaining
imported ref is the documented non-H.264 MKV exclusion.
`TestH264DecoderTDDContractClassifiesEveryImportedPublicVector` keeps the
inventory, manifest, exclusions, and failure ledger in lockstep.

## Ownership And Allocation Evidence

Public packet, frame, and side-data helpers validate caller-constructed storage
before clone, append, or raw-output operations. Exact-capacity append paths cover
8-bit and selected high-bit-depth raw output. Overflowed public storage is
rejected with `ErrInvalidData` instead of returning truncated snapshots, wrapped
sizes, or panicking during plane slicing.

`cmd/goh264bench` records Go benchmark allocation totals and reports allocation
rates for timed Go lanes. The benchmark scripts can turn those rates into local
budget gates through their documented environment variables.

For claim-grade FFmpeg comparisons, build the in-process libavcodec helper with
`scripts/build-libavcodec-bench.sh`, then set
`GOH264_BENCH_FAIR_LIBAVCODEC=1`. The matched compute lane checks Go and FFmpeg
rawvideo against the manifest oracle once per backend before timing. Each repeat
uses fresh decoder contexts, preloaded Annex B input, equal work counts, and one
decoder thread per independent worker. Process startup, context construction,
worker launch, file I/O, CLI setup, raw materialization, and hashing are excluded
on both sides; wakeup, parse, decode, drain/reset, and completion synchronization
are included. Measurement orders are balanced and recorded. Run
`GOH264_BENCH_WORKERS=1` for the single-thread lane and the same explicit worker
count on both sides for multicore throughput. `-fair-cpu-lanes` keeps libavcodec
pure-C and native C+assembly results separate. The native lane is claim-eligible
for the default assembly-enabled Go build; pure C is diagnostic. A win requires
the two-sided 95% confidence interval of the paired geometric elapsed ratio to
be entirely below 1, exact rawvideo parity, and `build_pgo=off` for a default
consumer-performance claim.
