# goh264

`goh264` is a source-shaped Go port of FFmpeg `libavcodec`'s H.264 decoder path.

The port is intentionally decoder-only. Encoder, muxer, filter, and unrelated codec code are out of scope unless an H.264 decoder file directly depends on a shared primitive.

Current safe point:

- Upstream source truth is pinned to FFmpeg `n8.0.1`, commit `894da5ca7d742e4429ffb2af534fcda0103ef593`.
- The active Go slice covers Annex B and AVC/NALFF length-prefixed NAL splitting, AVC decoder configuration record (`avcC`) SPS/PPS extradata parsing, RBSP emulation-prevention removal, bit reading, unsigned/signed Exp-Golomb reads, SPS/PPS metadata parsing including source-shaped SPS VUI/HRD timing, SAR, color, chroma-location, and bitstream-restriction fields, source-shaped H.264 SEI RBSP message framing for buffering period, picture timing, recovery point, x264 user-data, display orientation, frame packing, green metadata, alternative transfer, mastering display, and content light messages, scaling/QP/dequant table construction, MB-type/CBP/scan/IntraPCM-size tables, CAVLC VLC/residual coefficient primitives including transform-bypass qscale-0 scan selection, luma/chroma residual cache updates including monochrome chroma-cache clearing, intra prediction mode checks, intra CAVLC macroblock syntax, CAVLC 8-bit IntraPCM byte alignment and frame-MB table write-back, P/B-slice inter reference/MVD syntax, CAVLC frame-MB entropy-to-state handoff with P-skip `mb_skip_run`, and state write-back for intra/inter/P-skip frame macroblocks, CABAC arithmetic primitives/context initialization, CABAC macroblock type/CBP/ref/MVD syntax helpers, CABAC frame-MB entropy-to-state handoff with P-skip flags and 8-bit IntraPCM byte-stream handoff/reinitialization, CABAC inter motion-cache/MVD write-back and intra/inter/P-skip/IntraPCM macroblock state persistence, CABAC residual CBF/significance/level syntax, CABAC luma/chroma residual orchestration including `chroma_format_idc == 0`, 4:4:4 luma-shaped chroma planes, and qscale-0 transform-bypass scans, CABAC dQP syntax, frame-MB macroblock cache/write-back tables for residual/intra/motion/ref/MVD/direct state, frame-MB slice cursor/neighbor/cache orchestration, slice-header parsing with live macroblock-payload bitreader handoff, CAVLC/CABAC slice-data dispatch with FFmpeg-style CABAC byte realignment/context-state initialization, simple 8-bit frame-MB slice decode/reconstruct loops for CAVLC and CABAC frame pictures, source-shaped 8-bit frame-picture loop-filter strength/call-site integration for the simple path including 4:4:4 luma-shaped Cb/Cr edge filtering, public `DecodeAnnexB`/`DecodeAVC` output for simple 8-bit IDR/I frame pictures, public `DecodeAnnexBFrames`/`DecodeAVCFrames` output for simple 8-bit IDR/P packets and explicit non-direct B packets with planar raw-YUV extraction, configured AVC packet decode where SPS/PPS live only in `avcC` extradata, simple progressive frame-picture DPB/default P-list construction with short and long refs, POC-backed B-list construction, list0/list1 short/long reordering, implicit B-weight table construction, delayed display-order output with FFmpeg-style POC-gap reorder inference and signaled VUI reorder-depth handling, public delayed-frame flush for configured sample streams, sliding-window marking, and MMCO short/long operations for the simple public path, explicit weighted P prediction for the simple frame path, frame-MB motion-vector prediction and CAVLC inter motion-cache fill, 8-bit H.264 inverse transform/add and luma/chroma DC dequant reference kernels, 8-bit transform-bypass add-pixels and weighted/biweighted prediction DSP kernels, 8-bit luma/chroma deblocking filter DSP kernels, 8-bit luma qpel and chroma motion-compensation DSP kernels, frame-MB standard and weighted motion-compensation dispatch over 16x16, 16x8, 8x16, and 8x8 subpartition shapes with source-shaped edge emulation using independent scratch/source strides, simple 8-bit frame-MB reconstruction call sites for IntraPCM, intra16x16, intra4x4, intra8x8, inter residual add, and qscale-0 transform-bypass add/pred-add over monochrome/4:2:0/4:2:2/4:4:4, and 8-bit 16x16/4x4/8x8l luma and 8x8/8x16 chroma intra prediction DSP kernels.
- This safe point proves H.264 SEI message framing and decoder-relevant SEI state with targeted unit fixtures, wires leading SEI NAL parsing into the simple decoder with FFmpeg's non-exploding error policy, and keeps SPS VUI/HRD FFprobe coverage plus the existing CAVLC/CABAC frame-MD5 oracle corpus green.
- Row-threaded/border-exchange deblocking for complex slice scheduling, high-bit-depth IntraPCM, high-bit-depth transform-bypass reconstruction, field/MBAFF motion-compensation remapping and reference-progress waits, B-direct/B-skip motion, field/MBAFF implicit B weights, field/gap/error-resilience MMCO behavior, high-bit-depth/SIMD transform and DSP variants, SEI side-data export/full timing application, full libavcodec delayed-frame draining outside the simple progressive public path, threading, and general `Decode` output are not yet implemented.

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
