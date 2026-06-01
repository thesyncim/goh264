# High-Bit-Depth Roadmap

Source truth is FFmpeg `n8.0.1`. Keep high-bit-depth public decode gated by
bitstream oracles; internal 9/10/12/14-bit DSP support is not permission to
admit broad public streams.

## Proved Public Lanes

| Lane | Public proof |
| --- | --- |
| High10 4:2:0 IDR/I, P-skip/P16x16, exact residual P16x16 | Annex B/AVC/configured/sample surfaces plus FFmpeg rawvideo MD5 |
| High10 explicit weighted P16x16 | Public syntax tests, rawvideo MD5, manifest rows |
| High10 mixed-P Intra4x4/Intra16x16 | Public fixtures plus internal P-intra macroblock-table proof |
| High10 CAVLC/CABAC partitioned P16x8/P8x16/P8x8 | Public fixtures, internal partition tests, manifest rows |
| High10 CAVLC/CABAC explicit weighted partitioned P16x8/P8x16/P8x8 | Public fixtures, internal weighted partition macroblock-table proof, manifest rows |
| High10 non-direct/direct/B-skip/direct-sub/partitioned B lanes | Public fixtures and targeted B syntax guards for the documented subsets |
| High10 implicit weighted B16x16 and partitioned B | DPB-built implicit weights plus public rawvideo MD5 |
| High10 deblock-enabled IDR/P and narrow B lanes | Public fixtures for documented 4:2:0/4:2:2/4:4:4 and B subsets |
| High10 CAVLC-only slice-boundary IDR/P | `disable_deblocking_filter_idc == 2` public fixture |
| High12 CAVLC IDR/I IntraPCM | Narrow yuv420p12le public fixture |

The detailed row IDs, hashes, frame counts, and packet surfaces live in
`testdata/h264/corpus/manifest.jsonl`.

## Guard Rules

- Admit only the exact profile/chroma/depth/picture-structure/macroblock shape
  proved by a fixture.
- Keep deblocking, weighting, direct motion, residual, chroma format, and bit
  depth as separate guard axes unless one fixture proves the combination.
- Prefer adding an unsupported guard test before widening a public lane.
- Do not optimize or add SIMD until byte parity and manifest benchmark parity
  are stable for the lane.

## Next High-Value Lanes

| Priority | Lane | First acceptable safe point |
| --- | --- | --- |
| 1 | CABAC High10 4:2:0 slice-boundary IDR/P | True `disable_deblocking_filter_idc == 2` fixture plus narrow CABAC guard |
| 2 | Residual-bearing direct-sub B | Start with neutral CAVLC temporal B_8x8 direct-sub, CBP nonzero, high deblock off |
| 3 | High B deblock residual partitions | One shape at a time after residual B proof |
| 4 | Broader 12-bit / first 14-bit public streams | Small IDR/P oracle fixtures before B/deblock variants |
| 5 | MBAFF/field pictures | Separate architecture lane; do not mix with high-bit-depth expansion |

## Still Guarded

P IntraPCM, P 8x8-DCT intra, deblock-enabled weighted partitioned P,
mixed direct/explicit B8x8, residual-bearing direct-sub B, broader partitioned
implicit B, residual-bearing direct-sub high B deblocking, CABAC/chroma/B-slice
public high slice-boundary modes, 12-bit beyond the IntraPCM fixture, all
14-bit public streams, GBR/RGB, field/MBAFF, FMO, threading, SIMD, row-time
border exchange, and full libavcodec delayed-output behavior.
