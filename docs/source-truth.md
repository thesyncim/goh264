# Source Truth

Scope: FFmpeg `n8.0.1` H.264 decoder path only.

Proved today: progressive Annex B/AVC IDR/P/B subsets, selected High10/High12/High14
fixtures including public High10/High422 intra conformance and High10 unweighted 4:2:2/4:4:4 I/P chroma
no-deblock plus weighted 4:2:2/4:4:4 luma-only/chroma P frame deblock modes 0/1 and
slice-boundary mode-2 deblock, High12 CAVLC IntraPCM, mixed no-residual
intra, plus
Intra16x16 luma, chroma, and combined luma/chroma residual plus public
P-skip/P16x16, internal High12 no-deblock weighted P-skip and
weighted P16x16/partitioned P plus
P16x16 residual handoff plus no-deblock/mode-1 I/P deblock including
unweighted 4:2:2/4:4:4 I/P chroma deblock plus 4:2:0 I/P
slice-boundary mode-2 deblock and unweighted 4:2:2/4:4:4 I/P chroma
slice-boundary mode-2 deblock, High14 CAVLC IntraPCM, mixed no-residual
intra, separate and combined Intra16x16 luma/chroma residual plus
P-skip/P16x16, High10 frame-MBAFF field-coded CAVLC IntraPCM
entropy/reconstruct pairing plus public High10/High422 field-coded
frame-MBAFF deblock rows, complete FFmpeg FRext FATE row coverage including
monochrome-to-yuv420p output, selected compact baseline/main conformance rows
including AUD, SVA/Sony/MW/FT/JVC/Sand/Toshiba/VTC baseline/main, CABAC,
no-deblock, reference, field, and slice-boundary rows,
CABAC delayed-output reorder plus weighted/field compact rows, CAVLC
field/QP/crop/weighted/no-deblock compact rows, compact
long-slice/multiple-reference rows, frame-num-gap multiple-reference rows, and
MMCO-reset multiple-reference reorder rows,
non-conformance Annex B FATE rows covering extreme plane prediction,
reference-list-mod overflow recovery, missing-frame recovery, and
large-to-small SPS reinit metadata, with
packetized/configured surfaces proven for reference-list-mod overflow,
missing-frame, timecode, and BBC2 recovery/PAFF,
lossless High444 transform-bypass Annex B plus 4-byte AVC/configuration-record surfaces,
public FATE vector harness,
raw-diff/frame-MD5 diagnostics, and CLI benchmark comparison.

Public vectors: 193/193 green. Matrix mode is the safe-point gate. Red-queue
scripts intentionally exit non-zero only while rows in
`testdata/h264/realvectors/failures.jsonl` remain red.
Use `scripts/h264-real-vector-red-queue.sh <filter>` or
`scripts/h264-red-vector.sh <filter>` to hand agents a failing public lane; the
single-lane script exits at the first divergent raw byte for raw-MD5 rows.
Use `scripts/h264-real-vector-red-each.sh` for per-known-red logs plus a TSV
summary of first-divergence evidence.

Still guarded: unselected MBAFF/PIC-AFF/PAFF motion paths, FMO, broad
slice-boundary high modes, public 12/14-bit high streams beyond the current
FFmpeg FATE 8-bit/10-bit set, full error
resilience, threading/SIMD, and full libavcodec delayed-output behavior.

Canonical detail lives in manifests and tests, not Markdown.
