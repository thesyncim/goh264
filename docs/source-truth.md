# Source Truth

Scope: FFmpeg `n8.0.1` H.264 decoder path only.

Proved today: progressive Annex B/AVC IDR/P/B subsets, selected High10/High12
fixtures, public FATE vector harness, frame-MD5 diagnostics, and CLI benchmark
comparison.

Public vectors: 13/26 green. Red rows are not hidden; each has a known-failure
signature in `testdata/h264/realvectors/failures.jsonl`.

Still guarded: MBAFF/PIC-AFF, broad PAFF, FMO, broad slice-boundary high modes,
12/14-bit public high streams, full error resilience, threading/SIMD, and full
libavcodec delayed-output behavior.

Canonical detail lives in manifests and tests, not Markdown.
