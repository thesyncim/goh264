# Production Readiness Notes

This file tracks proof and reporting work, not decoder widening.

## Corpus

Seed manifest: `testdata/h264/corpus/manifest.jsonl`.

External vectors should use the same JSONL schema and runner:

```sh
GOH264_CORPUS_MANIFEST=/corpora/h264/jvt-avc/manifest.jsonl go test . -run TestH264CorpusManifest
GOH264_CORPUS_MANIFESTS="/corpora/h264/jvt-avc/manifest.jsonl:/corpora/h264/fate-h264/manifest.jsonl" go test . -run TestH264CorpusManifest
```

Recommended external manifests:

| Path | Purpose |
| --- | --- |
| `/corpora/h264/jvt-avc/manifest.jsonl` | ITU/JVT conformance rows, with decode-ok only for proved public lanes. |
| `/corpora/h264/ffmpeg-fate-h264/manifest.jsonl` | FFmpeg FATE rows regenerated through the pinned oracle. |
| `/corpora/h264/bench-large/manifest.jsonl` | Larger decode-ok clips for throughput/allocation reports. |

Decode-ok rows need `id`, `path`, `format`, `expect`, `pix_fmt`,
`frame_count`, `frame_size`, `bitstream_md5`, `rawvideo_md5`, `frame_md5`, and
`surfaces`. Unsupported rows need `guard_tags` and should use
`expected_error:"ErrUnsupported"`.

## Benchmark Profiles

```sh
# Smoke
go run ./cmd/goh264bench -manifest testdata/h264/corpus/manifest.jsonl -max-entries 4 -iters 3 -repeats 3 -warmup 1 -json

# Steady
go run ./cmd/goh264bench -manifest testdata/h264/corpus/manifest.jsonl -iters 20 -repeats 7 -warmup 3 -json

# FFmpeg CLI parity baseline
go run ./cmd/goh264bench -manifest testdata/h264/corpus/manifest.jsonl -iters 10 -repeats 5 -warmup 2 -ffmpeg -ffmpeg-threads 1 -strict-pix-fmt -json
```

Future flags worth adding:

- `-profile=smoke|steady|ffmpeg-cli|alloc`
- `-benchstat` for Go benchmark-shaped output
- `-max-go-allocs-per-iter`
- `-max-go-alloc-bytes-per-iter`
- `-max-go-allocs-per-frame`
- `-max-go-alloc-bytes-per-frame`

Manifest-level allocation budgets can follow after larger fixtures stabilize.
The mixed FFmpeg comparison must continue to state that the baseline launches
the FFmpeg CLI per timed iteration.
