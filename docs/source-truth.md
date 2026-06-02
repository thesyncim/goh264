# Source Truth

Scope: FFmpeg `n8.0.1` H.264 decoder path only.

Proved today: progressive Annex B/AVC IDR/P/B subsets, selected High10/High12
fixtures, public FATE vector harness, frame-MD5 diagnostics, and CLI benchmark
comparison.

Public vectors: 19/26 green. Matrix mode is the safe-point gate; red-queue mode
is the intentionally failing fix queue from `testdata/h264/realvectors/failures.jsonl`.
Use `scripts/h264-real-vector-red-queue.sh <filter>` to make a lane go red.

Still guarded: MBAFF/PIC-AFF, broad PAFF, FMO, broad slice-boundary high modes,
12/14-bit public high streams, full error resilience, threading/SIMD, and full
libavcodec delayed-output behavior.

Canonical detail lives in manifests and tests, not Markdown.
