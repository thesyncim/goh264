# High-Bit-Depth Roadmap

High-bit-depth decode is fixture-gated, not generally admitted.

Proved: selected High10 IDR/I/P/B plus public High10/High422 intra conformance, weighted/
partitioned/direct-sub lanes, CAVLC direct-sub residual, CABAC fixture
direct-sub residual, CABAC B16x16 direct residual, CABAC direct-sub residual
handoff/full slice plus implicit/deblock full slice, selected deblock lanes
plus fixture-backed high-B residual filter, slice-boundary IDR/P, and
unweighted 4:2:2/4:4:4 I/P chroma no-deblock plus weighted
4:2:2/4:4:4 luma-only/chroma P frame deblock modes 0/1 and slice-boundary mode-2 deblock,
public High12 IntraPCM plus CAVLC Intra16x16 no-residual and
luma-DC/luma-AC/luma-DC+AC/chroma-DC/chroma-AC/chroma-DC+AC/luma+chroma
residual fixtures plus two-frame CAVLC P-skip/P16x16 no-residual and
P16x16 luma-residual/luma+chroma residual plus P16x8/P8x16/P8x8
luma+chroma residual public fixtures plus mixed no-residual
Intra4x4/Intra16x16 and Intra16x16 luma, chroma, and combined
luma/chroma residual plus P-skip/P16x16,
internal High12 Intra16x16 luma-AC, luma-DC+AC, and luma/chroma-DC plus
chroma-AC/DC+AC and combined luma/chroma residual plus no-deblock
weighted P-skip and weighted P16x16/partitioned P plus P16x16 residual
handoff plus CABAC unweighted B no-deblock/mode-1 deblock and
mode-1 I/P deblock, 4:2:0 I/P
slice-boundary mode-2 deblock, including unweighted 4:2:2/4:4:4
I/P chroma no-deblock/mode-1 deblock plus unweighted 4:2:2/4:4:4 I/P chroma
slice-boundary mode-2 deblock, plus
CAVLC/CABAC High14 IntraPCM plus CAVLC/CABAC no-residual Intra4x4/
Intra16x16 and CAVLC/CABAC Intra16x16
luma-DC/luma-AC/luma-DC+AC/chroma-DC/chroma-AC/chroma-DC+AC/luma+chroma
residual fixtures plus two-frame CAVLC P-skip/P16x16 no-residual and
P16x16 luma-residual/luma+chroma residual plus P16x8/P8x16/P8x8
luma+chroma residual public fixtures plus mixed no-residual Intra4x4/Intra16x16 plus
Intra16x16 luma-DC/luma-AC/DC+AC and chroma-DC/chroma-AC/DC+AC and
combined luma/chroma residual plus CAVLC/CABAC unweighted and weighted
P-skip/P16x16 plus CAVLC mode-1/mode-2 I/P and weighted-P deblock plus
CABAC unweighted B no-deblock/mode-1 deblock and mode-1/mode-2 I/P and
weighted-P deblock, High10 frame-MBAFF
field-coded CAVLC IntraPCM entropy/reconstruct pairing, and public High10/
High422 field-coded frame-MBAFF deblock rows, plus public High9 4:2:0
SPS reinit metadata from 9-bit to 8-bit output, public High444 10-bit SPS
reinit metadata, and the XAVC High422 terminal damaged top-field row that
FFmpeg conceals while draining already-complete delayed frames.

Next: public 12/14-bit streams beyond the current FFmpeg FATE 8-bit/10-bit set,
broader high-bit-depth field/MBAFF motion, PIC-AFF/PAFF, and broader
damaged-slice error resilience beyond terminal first-field recovery.

Hashes live in `testdata/h264/corpus/manifest.jsonl` and
`testdata/h264/realvectors/manifest.jsonl`.
