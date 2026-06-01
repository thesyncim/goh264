# High-Bit-Depth Roadmap

High-bit-depth public decode stays fixture-gated. Internal 9/10/12/14-bit DSP
does not by itself admit public streams.

## Proved Lanes

- High10 4:2:0 IDR/I, P16x16 residual/weighted, mixed-P intra.
- High10 CAVLC/CABAC partitioned P16x8/P8x16/P8x8, now including explicit P
  weights when deblocking is disabled.
- High10 selected B/direct/implicit-weighted lanes and selected deblock-enabled
  IDR/P/B lanes.
- High10 CAVLC temporal B8x8 direct-sub with visible luma residual.
- High10 CAVLC/CABAC slice-boundary IDR/P.
- High12 CAVLC IDR/I IntraPCM.

Hashes and row IDs live in `testdata/h264/corpus/manifest.jsonl`.

## Next Lanes

| Priority | Lane |
| --- | --- |
| 1 | CABAC/implicit/deblock direct-sub residual variants |
| 2 | High B deblock residual partitions |
| 3 | Broader 12-bit / first 14-bit streams |
| 4 | MBAFF/field pictures |

Keep deblocking, weighting, direct motion, residual, chroma, and bit depth as
separate guard axes unless one oracle proves the combination.

## Still Guarded

P IntraPCM, P 8x8-DCT intra, deblock-enabled weighted partitioned P, mixed
direct/explicit B8x8, CABAC/implicit/deblock direct-sub residual variants,
unproved partitioned implicit B variants, chroma/B-slice slice-boundary modes,
broader 12/14-bit, GBR/RGB, field/MBAFF, FMO, threading/SIMD, and row border
exchange.
