# Translation Ledger

Status values: `copied`, `translated`, `stubbed`, `optimized`, `replaced`, `intentionally unsupported`.

| Upstream path/function | Local path/function | Status | Parity proof | Known deviations and source-cited reason |
| --- | --- | --- | --- | --- |
| `libavcodec/h264.h` NAL constants | `internal/h264/nal.go` `NALUnitType` constants | translated | unit | H.264-only constants; shared FFmpeg codec constants outside decoder scope are omitted. |
| `libavcodec/h2645_parse.c` `ff_h2645_extract_rbsp` | `internal/h264/nal.go` `AppendRBSP` | translated | unit | Keeps H.264 RBSP unescaping behavior; HEVC/VVC naming, FFmpeg padding, and skipped-byte position bookkeeping are deferred until a decoder slice needs them. |
| `libavcodec/get_bits.h` `GetBitContext` read/skip helpers | `internal/h264/bitreader.go` `bitReader` | translated | unit | MSB-first bounded reader only; FFmpeg cached-reader macros and unaligned fast paths are intentionally not optimized yet. |
| `libavcodec/golomb.h` `get_ue_golomb*`, `get_se_golomb_long` | `internal/h264/golomb.go` | translated | unit | Table-accelerated VLC paths are not ported yet; semantics are preserved with direct bit reads before optimization. |
| `libavcodec/h264_ps.c` `ff_h264_decode_seq_parameter_set` through cropping | `internal/h264/sps.go` `DecodeSPS` | translated | unit, `ffprobe` oracle | SPS parsing currently proves metadata through dimensions/crop and records the VUI presence bit. Full VUI/HRD and scaling matrix storage are deferred; scaling lists are consumed to preserve bit position. |
| `libavcodec/h264_ps.c` `ff_h264_decode_picture_parameter_set` syntax fields | `internal/h264/pps.go` `DecodePPS` | translated | unit, `ffprobe` fixture exercised through header parse | PPS syntax fields through chroma QP offsets are parsed against active SPS state. QP/dequant table construction is deferred until residual decode needs it. FMO remains intentionally unsupported, matching FFmpeg's missing-feature failure path here. |
| `libavcodec/h264_parse.c` slice header helpers | none | stubbed | not yet proved | Pending after PPS because slice parsing depends on active SPS/PPS state. |
| `libavcodec/h264_cabac.c` | none | stubbed | not yet proved | Pending after parser state and native trace oracle exist. |
| `libavcodec/h264_cavlc.c` | none | stubbed | not yet proved | Pending after parser state and native trace oracle exist. |
| `libavcodec/h264_slice.c` | none | stubbed | not yet proved | Pending after entropy slices. |
| `libavcodec/h264_mb.c`, `h264_direct.c`, `h264_refs.c`, `h264_picture.c` | none | stubbed | not yet proved | Pending after slice header and entropy parity. |
| `libavcodec/h264dsp*`, `h264idct*`, `h264pred*`, `h264qpel*`, `h264chroma*` | none | stubbed | not yet proved | Reference Go kernels first; SIMD/architecture dispatch only after byte parity. |
| `libavcodec/h264dec.c` public decoder integration | none | stubbed | not yet proved | Final raw-port integration layer after core state machines are translated. |
