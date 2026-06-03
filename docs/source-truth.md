# Source Truth

Scope: FFmpeg `n8.0.1` H.264 decoder path only.

Proved today: progressive Annex B/AVC IDR/P/B subsets, selected High10/High12/High14
fixtures including High12 CAVLC IntraPCM plus Intra16x16 luma and
combined luma/chroma residual, High14 CAVLC IntraPCM, mixed no-residual
intra, separate and combined Intra16x16 luma/chroma residual, public FATE
vector harness, raw-diff/frame-MD5 diagnostics, and CLI benchmark
comparison.

Public vectors: 36/36 green. Matrix mode is the safe-point gate. Red-queue
scripts intentionally exit non-zero only while rows in
`testdata/h264/realvectors/failures.jsonl` remain red.
Use `scripts/h264-real-vector-red-queue.sh <filter>` or
`scripts/h264-red-vector.sh <filter>` to hand agents a failing public lane; the
single-lane script exits at the first divergent raw byte for raw-MD5 rows.
Use `scripts/h264-real-vector-red-each.sh` for per-known-red logs plus a TSV
summary of first-divergence evidence.

Still guarded: unselected MBAFF/PIC-AFF/PAFF, FMO, broad slice-boundary high
modes, broader 12/14-bit public high streams, full error resilience,
threading/SIMD, and full libavcodec delayed-output behavior.

Canonical detail lives in manifests and tests, not Markdown.
