# Production Readiness Corpus And Bench Plan

This file tracks the next non-decoder work needed before claiming broader H.264
decoder production readiness. It is intentionally about proof surfaces, reporting,
and guardrails, not widening decoder support.

## Corpus Manifests

The committed seed corpus remains `testdata/h264/corpus/manifest.jsonl`. It is a
regression ladder, not a conformance suite. External manifests should live next
to the downloaded testvectors and run through the same runner:

```sh
GOH264_CORPUS_MANIFEST=/corpora/h264/jvt-avc/manifest.jsonl go test . -run TestH264CorpusManifest
GOH264_CORPUS_MANIFESTS="/corpora/h264/jvt-avc/manifest.jsonl:/corpora/h264/fate-h264/manifest.jsonl" go test . -run TestH264CorpusManifest
```

Use `GOH264_CORPUS_MANIFEST` for one external set and
`GOH264_CORPUS_MANIFESTS` for a path-list of independent manifests. Relative
`path` values resolve from each manifest's directory.

Recommended external manifest artifacts:

- `/corpora/h264/jvt-avc/manifest.jsonl`: JVT/ITU conformance vectors, keeping
  decode-ok rows limited to features already public and unsupported rows tagged
  with the exact guard.
- `/corpora/h264/ffmpeg-fate-h264/manifest.jsonl`: FFmpeg FATE H.264 vectors
  with rawvideo MD5s regenerated through the pinned FFmpeg oracle.
- `/corpora/h264/bench-large/manifest.jsonl`: stable larger decode-ok clips for
  throughput/allocation reports. Do not mix unsupported rows into benchmark
  comparison summaries.

Decode-ok rows must include `id`, `path`, `format`, `expect`, `pix_fmt`,
`frame_count`, `frame_size`, `bitstream_md5`, `rawvideo_md5`, `frame_md5`, and
`surfaces`. Unsupported rows must include `guard_tags` and should set
`expected_error` to `ErrUnsupported`. Do not turn an unsupported row into
`decode-ok` until the rawvideo oracle is exact and the remaining unsupported
boundaries are still named in the docs.

Useful guard tags for the next external rows: `field`, `mbaff`,
`fmo`, `data-partitioning`, `interlaced`, `high-422`, `high-444`, `high-12bit`,
`high-14bit`, `high-partitioned-p`, `high-b8x8`, `high-implicit-weight-b`,
`high-partitioned-b`, `high-slice-boundary-deblock`, and `high-row-threaded-deblock`.

## Benchmark Profiles

`cmd/goh264bench` already exposes the raw ingredients: warmup, repeats, manifest
mode, raw MD5 parity, FFmpeg CLI comparison, host/build metadata, and allocation
counters. Until first-class presets are added, use these command profiles:

```sh
# Smoke: quick local sanity, seed corpus only.
go run ./cmd/goh264bench -manifest testdata/h264/corpus/manifest.jsonl -max-entries 4 -iters 3 -repeats 3 -warmup 1 -json

# Steady local profile: enough repeats for trend watching.
go run ./cmd/goh264bench -manifest testdata/h264/corpus/manifest.jsonl -iters 20 -repeats 7 -warmup 3 -json

# FFmpeg rawvideo parity profile: CLI baseline with explicit comparator caveat.
go run ./cmd/goh264bench -manifest testdata/h264/corpus/manifest.jsonl -iters 10 -repeats 5 -warmup 2 -ffmpeg -ffmpeg-threads 1 -strict-pix-fmt -json

# Larger external corpus profile.
go run ./cmd/goh264bench -manifest /corpora/h264/bench-large/manifest.jsonl -iters 20 -repeats 7 -warmup 3 -ffmpeg -ffmpeg-threads 1 -strict-pix-fmt -json
```

Recommended future CLI presets:

- `-profile=smoke`: `-max-entries 4 -iters 3 -repeats 3 -warmup 1`.
- `-profile=steady`: `-iters 20 -repeats 7 -warmup 3`.
- `-profile=ffmpeg-cli`: steady settings plus `-ffmpeg -ffmpeg-threads 1
  -strict-pix-fmt`.
- `-profile=alloc`: steady settings plus allocation gates once those flags exist.

## Benchstat-Friendly Output

JSON should remain the archival report. Add a separate `-benchstat` mode or
subcommand that emits one line per result in Go benchmark shape so `benchstat`
can compare runs without a JSON adapter:

```text
BenchmarkH264/<entry_id>/goh264-10        123456 ns/op        2048 B/op        12 allocs/op
BenchmarkH264/<entry_id>/ffmpeg_cli-10    456789 ns/op           0 B/op         0 allocs/op
```

The derived fields should be machine-stable:

- `ns/op`: mean elapsed nanoseconds per decode iteration for the result.
- `B/op`: Go allocation bytes per decode iteration for `goh264`, zero or omitted
  for the FFmpeg CLI baseline.
- `allocs/op`: Go allocation count per decode iteration for `goh264`, zero or
  omitted for the FFmpeg CLI baseline.
- Benchmark name components: corpus entry id, engine name, raw/no-raw mode, and
  optional pixel format when the same bitstream can emit multiple formats.

## Allocation Gates

The JSON report already exposes `alloc_bytes` and `allocs` totals plus per-repeat
samples. The next guard should fail only after oracle parity passes, then compare
normalized Go allocation rates against explicit budgets:

- `-max-go-allocs-per-iter`.
- `-max-go-alloc-bytes-per-iter`.
- `-max-go-allocs-per-frame`.
- `-max-go-alloc-bytes-per-frame`.

Manifest-level optional budgets can follow later for stable larger clips:
`max_allocs_per_frame`, `max_alloc_bytes_per_frame`, `max_allocs_per_iter`, and
`max_alloc_bytes_per_iter`.

Keep the FFmpeg comparator caveat visible in every mixed report: the current
baseline launches the FFmpeg CLI once per timed iteration and therefore includes
process startup, CLI demux/parser setup, file reads, and stdout pipe cost. It is
a useful correctness and rough timing reference, not a fair in-process libavcodec
throughput baseline.
