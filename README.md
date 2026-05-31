# goh264

`goh264` is a source-shaped Go port of FFmpeg `libavcodec`'s H.264 decoder path.

The port is intentionally decoder-only. Encoder, muxer, filter, and unrelated codec code are out of scope unless an H.264 decoder file directly depends on a shared primitive.

Current safe point:

- Upstream source truth is pinned to FFmpeg `n8.0.1`, commit `894da5ca7d742e4429ffb2af534fcda0103ef593`.
- The active Go slice covers Annex B NAL splitting, RBSP emulation-prevention removal, bit reading, unsigned/signed Exp-Golomb reads, SPS/PPS metadata parsing, scaling/QP/dequant table construction, MB-type/CBP/scan tables, CAVLC VLC/residual coefficient primitives, luma/chroma residual cache updates, intra prediction mode checks, intra CAVLC macroblock syntax, P/B-slice inter reference/MVD syntax, CAVLC frame-MB entropy-to-state handoff with P-skip `mb_skip_run`, and state write-back for intra/inter/P-skip frame macroblocks, CABAC arithmetic primitives/context initialization, CABAC macroblock type/CBP/ref/MVD syntax helpers, CABAC frame-MB entropy-to-state handoff with P-skip flags, CABAC inter motion-cache/MVD write-back and intra/inter/P-skip macroblock state persistence, CABAC residual CBF/significance/level syntax, CABAC luma/chroma residual orchestration, CABAC dQP syntax, frame-MB macroblock cache/write-back tables for residual/intra/motion/ref/MVD/direct state, frame-MB slice cursor/neighbor/cache orchestration, frame-MB motion-vector prediction and CAVLC inter motion-cache fill, 8-bit H.264 inverse transform/add and luma/chroma DC dequant reference kernels, 8-bit transform-bypass add-pixels and weighted/biweighted prediction DSP kernels, and slice-header parsing up to macroblock payload.
- Frame reconstruction call-site wiring, full slice-loop CAVLC/CABAC macroblock integration, B-direct/B-skip motion, high-bit-depth/SIMD transform and DSP variants, deblocking, DPB, threading, and public `Decode` output are not yet implemented.

Run the default tests:

```sh
go test ./...
```

Run tests that call the pinned native oracle tools available on this machine:

```sh
GOH264_ORACLE=1 go test ./...
```

Fetch the pinned upstream source snapshot into the ignored local cache:

```sh
scripts/fetch-upstream.sh
```

The file-by-file status lives in [docs/translation-ledger.md](docs/translation-ledger.md).
