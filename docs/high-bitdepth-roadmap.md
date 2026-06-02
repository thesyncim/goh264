# High-Bit-Depth Roadmap

High-bit-depth decode is fixture-gated, not generally admitted.

Proved: selected High10 IDR/I/P/B, weighted/partitioned/direct-sub lanes,
selected deblock lanes, slice-boundary IDR/P, and High12 IntraPCM.

Next: CABAC/implicit/deblock direct-sub residual, high B deblock residual,
broader 12/14-bit public streams, then field/MBAFF.

Hashes live in `testdata/h264/corpus/manifest.jsonl`.
