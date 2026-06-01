# goh264

`goh264` is a source-shaped Go port of FFmpeg `libavcodec`'s H.264 decoder path.

The port is intentionally decoder-only. Encoder, muxer, filter, and unrelated codec code are out of scope unless an H.264 decoder file directly depends on a shared primitive.

Current safe point:

- Upstream source truth is pinned to FFmpeg `n8.0.1`, commit `894da5ca7d742e4429ffb2af534fcda0103ef593`.
- The active Go slice covers Annex B and AVC/NALFF length-prefixed NAL splitting, AVC decoder configuration record (`avcC`) SPS/PPS extradata parsing, FFmpeg-shaped packet auto-detection for public `Decode`/`DecodeFrames` entrypoints, public packet side-data plumbing for `AV_PKT_DATA_NEW_EXTRADATA`, display matrix, Stereo3D, spherical mapping, ICC profile, Dynamic HDR10+ bytes, LCEVC bytes, A53 captions, AFD, S12M timecode, mastering-display metadata, content-light metadata, ambient-viewing-environment metadata, and 3D reference display metadata, simple 8-bit progressive frame-picture decode through IDR/P/B output with DPB/reorder/weighted/direct-motion support, CAVLC/CABAC frame-MB entropy-to-state handoff, 8-bit reconstruction/deblocking, high-bit-depth 9/10/12/14 IDCT/prediction/qpel/chroma/weight/deblock kernels, source-shaped 8-bit and high-bit-depth frame-MB motion-compensation dispatch, internal high-bit-depth frame storage and IntraPCM/intra/inter macroblock reconstruction helpers over uint16 planes, and public raw high-depth helper methods for value-preserving sample or little-endian rawvideo output once a high frame is available.
- This safe point adds high-bit-depth decoded-frame storage and DPB high-plane reference views, keeps the simple high slice decode boundary explicitly unsupported, and exposes value-preserving public raw helpers (`Y16` planes plus `AppendRawYUV16`/`AppendRawYUVBytesLE`) for future FFmpeg rawvideo parity.
- Actual public high-bit-depth H.264 bitstream decode remains explicitly unsupported: the simple decode path still returns `ErrUnsupported` before high slice reconstruction reaches frame output. Row-threaded/border-exchange deblocking for complex slice scheduling, high-bit-depth slice/frame integration, high-bit-depth deblocking integration, field/MBAFF motion-compensation remapping and reference-progress waits, field/MBAFF implicit B weights, field/gap/error-resilience MMCO behavior, SIMD transform and DSP variants, complete SEI timing/interlace behavior, remaining packet side-data types, full libavcodec delayed-output semantics beyond the simple progressive public path, threading, and broad public decode beyond the simple 8-bit progressive frame-picture subset are not yet implemented.

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
