# High-Bit-Depth Roadmap

High-bit-depth decode is fixture-gated, not generally admitted.

Proved: selected High10 IDR/I/P/B plus public High10/High422 intra conformance, weighted/
partitioned/direct-sub lanes, CAVLC direct-sub residual, CABAC fixture
direct-sub residual, CABAC B16x16 direct residual, CABAC direct-sub residual
handoff/full slice plus implicit/deblock full slice, selected deblock lanes
plus fixture-backed high-B residual filter, slice-boundary IDR/P, and
unweighted 4:2:2/4:4:4 I/P chroma no-deblock plus weighted
4:2:2/4:4:4 luma-only/chroma P frame deblock modes 0/1 and slice-boundary mode-2 deblock,
public High12 IntraPCM plus mixed no-residual Intra4x4/Intra16x16 and
Intra16x16 luma, chroma, and combined luma/chroma residual plus P-skip/P16x16,
internal High12 Intra16x16 luma-AC, luma-DC+AC, and luma/chroma-DC plus
chroma-AC/DC+AC and combined luma/chroma residual plus no-deblock
weighted P-skip and weighted P16x16/partitioned P plus P16x16 residual
handoff plus mode-1 I/P deblock, 4:2:0 I/P
slice-boundary mode-2 deblock, including unweighted 4:2:2/4:4:4
I/P chroma no-deblock/mode-1 deblock plus unweighted 4:2:2/4:4:4 I/P chroma
slice-boundary mode-2 deblock, plus
CAVLC High14 IntraPCM and mixed no-residual Intra4x4/Intra16x16 plus
Intra16x16 luma-DC/luma-AC/DC+AC and chroma-DC/chroma-AC/DC+AC and
combined luma/chroma residual plus P-skip/P16x16, and High10 frame-MBAFF
field-coded CAVLC IntraPCM entropy/reconstruct pairing.

Next: high-bit-depth frame-MBAFF deblock for public field-coded rows, more
public 12/14-bit streams, then broader field/MBAFF.

Hashes live in `testdata/h264/corpus/manifest.jsonl` and
`testdata/h264/realvectors/manifest.jsonl`.
