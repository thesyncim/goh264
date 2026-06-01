# Production Readiness Notes

This tracks proof/reporting work, not decoder widening.

## Corpus

Seed manifest: `testdata/h264/corpus/manifest.jsonl`.

```sh
GOH264_CORPUS_MANIFEST=/corpora/h264/jvt-avc/manifest.jsonl go test . -run TestH264CorpusManifest
GOH264_CORPUS_MANIFESTS="/corpora/h264/jvt-avc/manifest.jsonl:/corpora/h264/fate-h264/manifest.jsonl" go test . -run TestH264CorpusManifest
```

External manifests to add:

| Path | Purpose |
| --- | --- |
| `/corpora/h264/jvt-avc/manifest.jsonl` | ITU/JVT conformance |
| `/corpora/h264/ffmpeg-fate-h264/manifest.jsonl` | FFmpeg FATE through pinned oracle |
| `/corpora/h264/bench-large/manifest.jsonl` | larger decode-ok performance clips |

Decode-ok rows need path, pixel format, frame count/size, bitstream MD5,
rawvideo MD5, frame MD5s, and surfaces. Unsupported rows need guard tags and
`ErrUnsupported`.

## Benchmark

```sh
go run ./cmd/goh264bench -manifest testdata/h264/corpus/manifest.jsonl -iters 10 -repeats 5 -warmup 2 -ffmpeg -ffmpeg-threads 1 -strict-pix-fmt -json
```

Pending: `-profile`, `-benchstat`, allocation gates, larger corpus budgets, and
an in-process libavcodec baseline. Current FFmpeg timing launches the CLI per
iteration.
