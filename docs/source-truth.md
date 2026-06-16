# Source Truth

`goh264` is decoder-only. The implementation follows the FFmpeg `n8.0.1`
H.264 decoder path pinned at `894da5ca7d742e4429ffb2af534fcda0103ef593`.

## In Scope

- Public Annex B decoding through `DecodeFrames` and `DecodeAnnexBFrames`.
- Public length-prefixed AVC decoding through `DecodeAVCFrames`.
- avcC inspection, storage, configured-AVC decoding, and in-stream avcC update
  through `InspectAVCC`, `ConfigureAVCC`, `DecodeConfiguredAVCFrames`, and
  `DecodeAVCCFrames`.
- Packet side data, including `NEW_EXTRADATA`, display metadata, captions,
  HDR-related payloads, and structured frame side data.
- Delayed B-frame output and explicit flushing.
- Raw output helpers for 8-bit and selected high-bit-depth frames.
- Header inspection without decoder-state mutation.

## Out Of Scope

- H.264 bitstream generation.
- Send-side controls.
- Rate control, RTP packetization for generated frames, or SPS/PPS generation
  for newly encoded video.
- FMO, 11/13-bit luma depths, `chroma_format_idc > 3`, separate color planes,
  and mixed chroma/luma bit depths at the pinned FFmpeg parity boundary.

## Evidence Shape

Decoder evidence is held in unit tests, public fixture tests, FFmpeg-backed
oracle rows, fuzz smoke tests, and the quality-evidence scripts under
`scripts/`. Public vectors: 226 imported public refs, 225 selected
decoder-facing manifest rows, 225 green oracle rows, 0 known-red. Treat that as
a repository snapshot; rerun the current checkout's gates before using it as
fresh production evidence.
