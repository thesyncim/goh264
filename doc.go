// Package goh264 provides a pure-Go H.264 decoder and an experimental
// realtime-oriented encoder.
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
//   - Use the single-frame helpers only when exactly one output frame is
//     expected; they return ErrUnsupported for zero-frame or multi-frame
//     packets that otherwise decode successfully.
//
// Encoder API shape:
//
//   - Start from DefaultEncoderConfig, normalize or validate it, then construct
//     an Encoder with NewEncoder.
//   - Use explicit runtime setters for ordinary bitrate, frame-rate, GOP,
//     geometry, slice, header, packetization, RTP, and recovery-SEI controls.
//   - Use Reconfigure only for grouped low-level updates or fields that do not
//     have a dedicated setter yet.
//   - Use EncodeInto when the caller owns access-unit storage, and Encode when
//     encoder-owned storage is sufficient.
//
// The decoder is the best-covered side of the module. The encoder deliberately
// admits a narrower Constrained Baseline I420 subset while its broader
// production coverage is still being hardened.
package goh264
