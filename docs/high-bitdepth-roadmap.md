# High-Bit-Depth Roadmap

High-bit-depth decode is fixture-gated, not generally admitted.

Proved: selected High10 IDR/I/P/B plus public High10 intra, weighted/
partitioned/direct-sub lanes, CAVLC direct-sub residual, CABAC fixture
direct-sub residual, CABAC B16x16 direct residual, CABAC direct-sub residual
handoff/full slice plus implicit/deblock full slice, selected deblock lanes
plus fixture-backed high-B residual filter, slice-boundary IDR/P, public
High12 IntraPCM plus mixed no-residual Intra4x4/Intra16x16 and
Intra16x16 luma, chroma, and combined luma/chroma residual, internal
High12 Intra16x16 luma-AC, luma-DC+AC, and luma/chroma-DC plus
chroma-AC/DC+AC and combined luma/chroma residual plus no-deblock
unweighted/weighted P-skip/P16x16/partitioned P and P16x16 residual
handoff plus mode-1 I/P deblock, plus
CAVLC High14 IntraPCM and mixed no-residual Intra4x4/Intra16x16 plus
Intra16x16 luma-DC/luma-AC/DC+AC and chroma-DC/chroma-AC/DC+AC and
combined luma/chroma residual.

Next: more public 12/14-bit streams, then field/MBAFF.

Hashes live in `testdata/h264/corpus/manifest.jsonl` and
`testdata/h264/realvectors/manifest.jsonl`.
