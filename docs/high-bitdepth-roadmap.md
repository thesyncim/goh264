# High-Bit-Depth Roadmap

High-bit-depth decode is fixture-gated, not generally admitted.

Proved: selected High10 IDR/I/P/B plus public High10 intra, weighted/
partitioned/direct-sub lanes, CAVLC direct-sub residual, CABAC B16x16 direct
residual, CABAC direct-sub residual handoff/full slice plus implicit/deblock
full slice, selected deblock lanes,
slice-boundary IDR/P, and High12 IntraPCM plus no-residual Intra4x4/
Intra16x16 and Intra16x16 luma-AC, luma-DC+AC, and luma/chroma-DC
plus chroma-AC/DC+AC and combined luma/chroma residual.

Next: CABAC fixture direct-sub residual, high B deblock residual, broader
12/14-bit public streams, then field/MBAFF.

Hashes live in `testdata/h264/corpus/manifest.jsonl` and
`testdata/h264/realvectors/manifest.jsonl`.
