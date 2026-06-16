// Package goh264 provides a pure-Go, decoder-only H.264/AVC implementation.
//
// Decoder API shape:
//
//   - Use NewDecoder plus DecodeFrames for stateful Annex B or configured-AVC
//     packet streams. Passing an avcC record stores configured-AVC state;
//     passing empty data flushes delayed output.
//   - Use DecodePacketFrames when packet side data such as NEW_EXTRADATA needs
//     to travel with the compressed packet.
//   - Use DecodeAnnexBFrames and DecodeAVCFrames for complete, known-format
//     inputs where stateful packet streaming is not needed.
//   - Use InspectAnnexBHeaders, InspectAVCHeaders, and InspectAVCC for metadata
//     inspection without decoder-state mutation.
//   - Use the single-frame helpers only when exactly one output frame is
//     expected; they return ErrUnsupported for zero-frame or multi-frame
//     packets that otherwise decode successfully.
//
// This package does not provide H.264 bitstream generation and does not grant
// patent rights. See PATENTS.md in the repository for the user-facing patent
// notice.
package goh264
