# Source Truth

Scope: FFmpeg `n8.0.1` H.264 decoder path only.

Proved today: progressive Annex B/AVC IDR/P/B subsets, selected High10/High12/High14
fixtures including High10 unweighted 4:2:2/4:4:4 I/P chroma
no-deblock plus weighted 4:2:2/4:4:4 chroma P no-deblock and
slice-boundary mode-2 deblock, High12 CAVLC IntraPCM, mixed no-residual
intra, plus
Intra16x16 luma, chroma, and combined luma/chroma residual, internal
High12 no-deblock unweighted/weighted P-skip/P16x16/partitioned P and
P16x16 residual handoff plus no-deblock/mode-1 I/P deblock including
unweighted 4:2:2/4:4:4 I/P chroma deblock plus 4:2:0 I/P
slice-boundary mode-2 deblock and unweighted 4:2:2/4:4:4 I/P chroma
slice-boundary mode-2 deblock, High14 CAVLC IntraPCM, mixed no-residual
intra, separate and combined Intra16x16 luma/chroma residual, public FATE vector harness,
raw-diff/frame-MD5 diagnostics, and CLI benchmark comparison.

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
