// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
	"unsafe"

	"github.com/thesyncim/goh264/internal/h264"
)

type EncoderPixelFormat uint8

const (
	EncoderPixelFormatI420 EncoderPixelFormat = iota + 1
)

// EncoderProfile selects H.264 profile syntax.
//
// NewEncoder admits EncoderProfileConstrainedBaseline and
// EncoderProfileBaseline. Main and High are exported control values that
// return ErrUnsupported from NewEncoder.
type EncoderProfile uint8

const (
	EncoderProfileConstrainedBaseline EncoderProfile = iota + 1
	EncoderProfileBaseline
	EncoderProfileMain
	EncoderProfileHigh
)

// EncoderEntropyMode selects entropy coding syntax.
//
// NewEncoder admits EncoderEntropyCAVLC. CABAC is an exported control value
// that returns ErrUnsupported from NewEncoder.
type EncoderEntropyMode uint8

const (
	EncoderEntropyCAVLC EncoderEntropyMode = iota + 1
	EncoderEntropyCABAC
)

type EncoderDeblockMode uint8

const (
	EncoderDeblockEnabled EncoderDeblockMode = iota + 1
	EncoderDeblockDisabled
	EncoderDeblockSliceBoundary
)

type EncoderSPSPPSMode uint8

const (
	EncoderSPSPPSInBandKeyframes EncoderSPSPPSMode = iota + 1
	EncoderSPSPPSOutOfBand
	EncoderSPSPPSEveryIDR
)

type EncoderRateControlMode uint8

const (
	EncoderRateControlCBR EncoderRateControlMode = iota + 1
	// EncoderRateControlVBR is an exported control value that returns
	// ErrUnsupported from NewEncoder.
	EncoderRateControlVBR
	EncoderRateControlConstantQP
)

// EncoderPreset selects speed/quality tradeoff policy.
//
// NewEncoder admits EncoderPresetRealtime. Balanced and Quality are exported
// control values that return ErrUnsupported from NewEncoder.
type EncoderPreset uint8

const (
	maxEncoderRawNALListLen    = maxInt / 64
	maxEncoderNALUnitListLen   = maxInt / 32
	maxEncoderRTPPacketListLen = maxInt / 64
)

const (
	EncoderPresetRealtime EncoderPreset = iota + 1
	EncoderPresetBalanced
	EncoderPresetQuality
)

type EncoderFrameDropMode uint8

const (
	EncoderFrameDropDisabled EncoderFrameDropMode = iota + 1
	EncoderFrameDropLate
	EncoderFrameDropToBitrate
)

type EncoderOutputFormat uint8

const (
	EncoderOutputAnnexB EncoderOutputFormat = iota + 1
	EncoderOutputAVC
	EncoderOutputRTP
)

func validEncoderOutputFormat(format EncoderOutputFormat) bool {
	switch format {
	case EncoderOutputAnnexB, EncoderOutputAVC, EncoderOutputRTP:
		return true
	default:
		return false
	}
}

type EncoderRTPPacketizationMode uint8

const (
	EncoderRTPPacketizationSingleNAL      EncoderRTPPacketizationMode = 0
	EncoderRTPPacketizationNonInterleaved EncoderRTPPacketizationMode = 1
)

type EncoderCrop struct {
	Left   int
	Right  int
	Top    int
	Bottom int
}

type EncoderColorConfig struct {
	SARNum                         int32
	SARDen                         int32
	VideoFormat                    int32
	FullRange                      bool
	ColorPrimaries                 int32
	ColorTransfer                  int32
	ColorMatrix                    int32
	ChromaSampleLocTypeTopField    int32
	ChromaSampleLocTypeBottomField int32
}

// EncoderConfig controls encoder setup.
//
// Start from DefaultRealtimeEncoderConfig and override the fields needed by the
// integration. DefaultEncoderConfig returns the same realtime template.
// NewEncoder applies derived defaults and rejects invalid or unsupported
// controls. Validate reports whether cfg is accepted without returning the
// normalized values; Normalize returns the exact validated setup. Crop and Color
// are encoded in SPS/VUI headers from this config; per-frame Color is validated
// but does not rewrite output headers.
// InitialQP, MinQP, and MaxQP accept 0..51. When ExplicitQP is false, zero QP
// fields select derived defaults during setup normalization; set ExplicitQP when
// zero is an intentional setup value.
// RTPPayloadType accepts 1..127; zero selects the dynamic default payload type
// 96 during setup normalization.
type EncoderConfig struct {
	Width        int
	Height       int
	StrideY      int
	StrideCb     int
	StrideCr     int
	PixelFormat  EncoderPixelFormat
	Crop         EncoderCrop
	FrameRateNum int
	FrameRateDen int
	TimeBaseNum  int
	TimeBaseDen  int
	Color        EncoderColorConfig

	Profile            EncoderProfile
	LevelIDC           uint8
	EntropyMode        EncoderEntropyMode
	DeblockMode        EncoderDeblockMode
	Transform8x8       bool
	MaxReferenceFrames int
	BFrames            int
	SPSPPSMode         EncoderSPSPPSMode

	RateControl   EncoderRateControlMode
	TargetBitrate int
	MaxBitrate    int
	VBVBufferSize int
	MaxFrameSize  int
	InitialQP     int
	MinQP         int
	MaxQP         int
	ExplicitQP    bool
	Preset        EncoderPreset

	ZeroLookahead   bool
	FrameDrop       EncoderFrameDropMode
	MaxEncodeTimeUS int
	SliceCount      int
	SliceMaxBytes   int
	Workers         int
	Deterministic   bool

	GOPSize          int
	IDRInterval      int
	SPSPPSBeforeIDR  bool
	RecoveryPointSEI bool
	IntraRefresh     bool

	OutputFormat          EncoderOutputFormat
	RTPMaxPayloadSize     int
	RTPPacketizationMode  EncoderRTPPacketizationMode
	STAPA                 bool
	DONDisabled           bool
	RTPPayloadType        uint8
	RTPSSRC               uint32
	RTPTimestampIncrement uint32
}

// EncoderFrame is one I420 input frame.
//
// Encode and EncodeInto read the plane slices during the call and do not retain
// them after the call returns. Color is validated as caller-supplied frame
// metadata; encoded SPS/VUI color metadata comes from EncoderConfig.Color.
type EncoderFrame struct {
	Y        []byte
	Cb       []byte
	Cr       []byte
	StrideY  int
	StrideCb int
	StrideCr int
	Width    int
	Height   int
	PTS      int64
	Duration int64
	ForceIDR bool
	Color    EncoderColorConfig
}

// Clone returns a deep-owned copy of the input frame.
//
// The cloned planes are independent from frame and safe to retain after the
// original caller-owned buffers are reused.
func (frame EncoderFrame) Clone() (EncoderFrame, error) {
	if err := frame.Validate(); err != nil {
		return EncoderFrame{}, err
	}
	clone := frame
	clone.Y = cloneByteSlice(frame.Y)
	clone.Cb = cloneByteSlice(frame.Cb)
	clone.Cr = cloneByteSlice(frame.Cr)
	return clone, nil
}

// Validate reports whether frame has public storage sizes accepted by Clone.
//
// Use EncoderConfig.ValidateFrame or Encoder.ValidateFrame to validate that the
// frame geometry and plane lengths can be encoded by a specific configuration.
func (frame EncoderFrame) Validate() error {
	if len(frame.Y) > maxInt/2 || len(frame.Cb) > maxInt/2 || len(frame.Cr) > maxInt/2 {
		return ErrInvalidData
	}
	return nil
}

// I420Frame returns an EncoderFrame populated from cfg dimensions, strides,
// timing defaults, and color metadata.
func (cfg EncoderConfig) I420Frame(y, cb, cr []byte, pts int64) EncoderFrame {
	return EncoderFrame{
		Y:        y,
		Cb:       cb,
		Cr:       cr,
		StrideY:  cfg.StrideY,
		StrideCb: cfg.StrideCb,
		StrideCr: cfg.StrideCr,
		Width:    cfg.Width,
		Height:   cfg.Height,
		PTS:      pts,
		Duration: int64(cfg.RTPTimestampIncrement),
		Color:    cfg.Color,
	}
}

// EncoderNALUnit describes one H.264 NAL unit inside EncodedFrame.Data.
//
// Offset points at the NAL header byte, not at the Annex B start code or AVC
// length prefix. Size is the raw NAL byte count.
type EncoderNALUnit struct {
	Type         uint8
	Offset       int
	Size         int
	KeyFrame     bool
	ParameterSet bool
}

// EncoderRTPPacket is one encoded RTP packet.
//
// Data contains the complete RTP packet, including the 12-byte header. Payload is
// the exact clipped view over Data[12:], so appending to either slice cannot
// overwrite another returned packet. Returned RTP packet storage is independent
// from EncodedFrame.Data.
type EncoderRTPPacket struct {
	Data           []byte
	Payload        []byte
	PayloadType    uint8
	SequenceNumber uint16
	Timestamp      uint32
	SSRC           uint32
	Marker         bool
}

// PacketData returns the complete RTP packet bytes, including the 12-byte RTP
// header.
//
// PacketData validates the packet metadata, payload view, and RTP payload
// syntax before returning bytes.
//
// The returned slice is clipped to its length, so appending to it cannot
// overwrite the following bytes in the packet backing store.
func (packet EncoderRTPPacket) PacketData() ([]byte, error) {
	if _, err := packet.PayloadData(); err != nil {
		return nil, err
	}
	return packet.Data[:len(packet.Data):len(packet.Data)], nil
}

// AppendPacketData appends a caller-owned copy of PacketData to dst.
//
// On error, dst is returned unchanged.
func (packet EncoderRTPPacket) AppendPacketData(dst []byte) ([]byte, error) {
	data, err := packet.PacketData()
	if err != nil {
		return dst, err
	}
	return appendPublicBytes(dst, data)
}

// PayloadData returns the RTP payload bytes.
//
// Payload must be the exact Data[12:] payload view. The returned slice is
// clipped to its length, so appending to it cannot overwrite the following bytes
// in the packet backing store.
func (packet EncoderRTPPacket) PayloadData() ([]byte, error) {
	if !encoderRTPPacketCloneStorageOK(packet) ||
		!encoderRTPPacketHeaderMetadataOK(packet) || len(packet.Payload) == 0 {
		return nil, ErrInvalidData
	}
	dataStart := unsafeSliceOffset(packet.Data, packet.Payload)
	if dataStart != 12 {
		return nil, ErrInvalidData
	}
	dataEnd, err := checkedAddInt(dataStart, len(packet.Payload))
	if err != nil || dataEnd != len(packet.Data) {
		return nil, ErrInvalidData
	}
	if err := validateEncoderRTPPayload(packet.Payload); err != nil {
		return nil, err
	}
	return packet.Payload[:len(packet.Payload):len(packet.Payload)], nil
}

// Validate reports whether packet is a well-formed RTP packet helper value.
func (packet EncoderRTPPacket) Validate() error {
	if _, err := packet.PacketData(); err != nil {
		return err
	}
	if _, err := packet.PayloadData(); err != nil {
		return err
	}
	return nil
}

func encoderRTPPacketHeaderOK(data []byte) bool {
	return len(data) >= 12 && data[0] == 0x80
}

func encoderRTPPacketHeaderMetadataOK(packet EncoderRTPPacket) bool {
	if !encoderRTPPacketHeaderOK(packet.Data) || packet.PayloadType > 127 {
		return false
	}
	if packet.Data[1]&0x7f != packet.PayloadType ||
		(packet.Data[1]&0x80 != 0) != packet.Marker {
		return false
	}
	return binary.BigEndian.Uint16(packet.Data[2:4]) == packet.SequenceNumber &&
		binary.BigEndian.Uint32(packet.Data[4:8]) == packet.Timestamp &&
		binary.BigEndian.Uint32(packet.Data[8:12]) == packet.SSRC
}

func validateEncoderRTPPayload(payload []byte) error {
	if len(payload) == 0 || payload[0]&0x80 != 0 {
		return ErrInvalidData
	}
	typ := payload[0] & 0x1f
	if typ == 0 {
		return ErrInvalidData
	}
	switch typ {
	case 24:
		pos := 1
		if pos == len(payload) {
			return ErrInvalidData
		}
		for pos < len(payload) {
			if pos+2 > len(payload) {
				return ErrInvalidData
			}
			size := int(payload[pos])<<8 | int(payload[pos+1])
			pos += 2
			if size == 0 || pos+size > len(payload) {
				return ErrInvalidData
			}
			if payload[pos]&0x80 != 0 || payload[pos]&0x1f == 0 {
				return ErrInvalidData
			}
			pos += size
		}
	case 28:
		if len(payload) < 3 {
			return ErrInvalidData
		}
		fuHeader := payload[1]
		if fuHeader&0x20 != 0 || fuHeader&0x1f == 0 || fuHeader&0xc0 == 0xc0 {
			return ErrInvalidData
		}
	}
	return nil
}

// AppendPayloadData appends a caller-owned copy of PayloadData to dst.
//
// On error, dst is returned unchanged.
func (packet EncoderRTPPacket) AppendPayloadData(dst []byte) ([]byte, error) {
	data, err := packet.PayloadData()
	if err != nil {
		return dst, err
	}
	return appendPublicBytes(dst, data)
}

// Clone returns a deep-owned RTP packet snapshot.
//
// The cloned Payload view points into the cloned Data backing store and is safe
// to retain after the original packet storage is reused or mutated.
func (packet EncoderRTPPacket) Clone() (EncoderRTPPacket, error) {
	if !encoderRTPPacketCloneStorageOK(packet) {
		return EncoderRTPPacket{}, ErrInvalidData
	}
	if err := packet.Validate(); err != nil {
		return EncoderRTPPacket{}, err
	}
	data := cloneByteSlice(packet.Data)
	payloadOffset := unsafeSliceOffset(packet.Data, packet.Payload)
	clone := packet
	clone.Data = data
	clone.Payload = data[payloadOffset : payloadOffset+len(packet.Payload) : payloadOffset+len(packet.Payload)]
	return clone, nil
}

func encoderRTPPacketCloneStorageOK(packet EncoderRTPPacket) bool {
	return len(packet.Data) <= maxInt/2 && len(packet.Payload) <= maxInt/2
}

type EncoderRTPPayloadFormat uint8

const (
	EncoderRTPPayloadSingleNAL EncoderRTPPayloadFormat = iota + 1
	EncoderRTPPayloadSTAPA
	EncoderRTPPayloadFUA
)

// EncoderRTPPacketMetadata describes a packet reported through
// EncoderRTPPacketCallback.
type EncoderRTPPacketMetadata struct {
	PacketIndex int
	PacketCount int

	FramePTS int64
	FrameDTS int64
	RTPTime  uint32
	KeyFrame bool
	IDR      bool

	PayloadFormat EncoderRTPPayloadFormat
	NALUnitType   uint8
	NALUnitCount  int
	StartOfNAL    bool
	EndOfNAL      bool
	ParameterSet  bool
}

// EncoderRTPPacketCallback observes RTP packets emitted by Encode or EncodeInto.
//
// The callback runs synchronously before Encode or EncodeInto returns. The
// packet passed to the callback is a clone and does not alias the packet storage
// returned in EncodedFrame.RTPPackets.
type EncoderRTPPacketCallback func(packet EncoderRTPPacket, metadata EncoderRTPPacketMetadata)

// EncodedFrame is the result of one encoder call.
//
// OutputFormat records the format requested for this result. For Annex B and
// AVC output, Data contains the encoded access unit in that container. For RTP
// output, Data contains an Annex B access-unit view for NAL/access-unit helpers;
// RTPPackets contains the complete RTP packets and owns storage separate from
// Data. NALUnits index into Data. When Dropped is true, no bytes, NAL units, or
// RTP packets were emitted.
type EncodedFrame struct {
	OutputFormat EncoderOutputFormat
	Data         []byte
	NALUnits     []EncoderNALUnit
	RTPPackets   []EncoderRTPPacket
	KeyFrame     bool
	IDR          bool
	PTS          int64
	DTS          int64
	RTPTime      uint32
	Dropped      bool
}

// NALData returns the raw NAL bytes described by NALUnits[index].
//
// The returned slice is clipped to its length, so appending to it cannot
// overwrite the following bytes in EncodedFrame.Data.
func (frame EncodedFrame) NALData(index int) ([]byte, error) {
	if frame.Dropped || len(frame.Data) > maxInt/2 || len(frame.NALUnits) > maxEncoderNALUnitListLen ||
		index < 0 || index >= len(frame.NALUnits) {
		return nil, ErrInvalidData
	}
	if err := frame.validateAccessUnitStorageShape(); err != nil {
		return nil, err
	}
	unit := frame.NALUnits[index]
	end, err := frame.validateNALUnitMetadata(unit)
	if err != nil {
		return nil, ErrInvalidData
	}
	return frame.Data[unit.Offset:end:end], nil
}

// AppendNALData appends a caller-owned copy of NALData(index) to dst.
//
// On error, dst is returned unchanged.
func (frame EncodedFrame) AppendNALData(dst []byte, index int) ([]byte, error) {
	data, err := frame.NALData(index)
	if err != nil {
		return dst, err
	}
	return appendPublicBytes(dst, data)
}

// AccessUnitRange returns the EncodedFrame.Data byte range for the encoded
// access-unit view described by NALUnits.
//
// For EncodeInto calls that append after an existing dst prefix, start excludes
// the caller prefix. For RTP output, the range describes the Annex B access-unit
// view in Data, not RTPPackets.
func (frame EncodedFrame) AccessUnitRange() (start int, end int, err error) {
	return frame.accessUnitRange()
}

// AccessUnitFormat returns the container format used by AccessUnitData.
//
// For RTP output, AccessUnitData is an Annex B access-unit view and this method
// returns EncoderOutputAnnexB. Use RTPPackets, RTPPacketData, or RTPPayloadData
// for RTP packet bytes.
func (frame EncodedFrame) AccessUnitFormat() (EncoderOutputFormat, error) {
	if _, _, err := frame.accessUnitRange(); err != nil {
		return 0, err
	}
	switch frame.OutputFormat {
	case EncoderOutputAnnexB, EncoderOutputRTP:
		return EncoderOutputAnnexB, nil
	case EncoderOutputAVC:
		return EncoderOutputAVC, nil
	default:
		return 0, ErrInvalidData
	}
}

// AccessUnitData returns the encoded access-unit bytes described by NALUnits.
//
// The returned slice is clipped to its length. For EncodeInto calls that append
// after an existing dst prefix, AccessUnitData excludes the caller prefix.
func (frame EncodedFrame) AccessUnitData() ([]byte, error) {
	start, end, err := frame.accessUnitRange()
	if err != nil {
		return nil, err
	}
	return frame.Data[start:end:end], nil
}

func (frame EncodedFrame) accessUnitRange() (int, int, error) {
	if frame.Dropped || len(frame.Data) > maxInt/2 || len(frame.NALUnits) > maxEncoderNALUnitListLen ||
		len(frame.RTPPackets) > maxEncoderRTPPacketListLen || len(frame.NALUnits) == 0 {
		return 0, 0, ErrInvalidData
	}
	if err := frame.validateAccessUnitStorageShape(); err != nil {
		return 0, 0, err
	}
	start := -1
	end := 0
	for _, unit := range frame.NALUnits {
		unitEnd, err := frame.validateNALUnitMetadata(unit)
		if err != nil || unit.Offset < 4 {
			return 0, 0, ErrInvalidData
		}
		prefixStart := unit.Offset - 4
		prefix := frame.Data[prefixStart:unit.Offset]
		annexBPrefix := prefix[0] == 0 && prefix[1] == 0 && prefix[2] == 0 && prefix[3] == 1
		avcPrefix := binary.BigEndian.Uint32(prefix) == uint32(unit.Size)
		switch frame.OutputFormat {
		case EncoderOutputAnnexB, EncoderOutputRTP:
			if !annexBPrefix {
				return 0, 0, ErrInvalidData
			}
		case EncoderOutputAVC:
			if !avcPrefix {
				return 0, 0, ErrInvalidData
			}
		default:
			return 0, 0, ErrInvalidData
		}
		if start < 0 {
			start = prefixStart
		} else if prefixStart != end {
			return 0, 0, ErrInvalidData
		}
		end = unitEnd
	}
	if start >= end {
		return 0, 0, ErrInvalidData
	}
	return start, end, nil
}

func (frame EncodedFrame) validateAccessUnitStorageShape() error {
	switch frame.OutputFormat {
	case EncoderOutputAnnexB, EncoderOutputAVC:
		if len(frame.RTPPackets) != 0 {
			return ErrInvalidData
		}
	case EncoderOutputRTP:
		if len(frame.RTPPackets) == 0 || len(frame.RTPPackets) > maxEncoderRTPPacketListLen {
			return ErrInvalidData
		}
	default:
		return ErrInvalidData
	}
	return nil
}

func (frame EncodedFrame) validateNALUnitMetadata(unit EncoderNALUnit) (int, error) {
	if unit.Offset < 0 || unit.Size <= 0 {
		return 0, ErrInvalidData
	}
	end, err := checkedAddInt(unit.Offset, unit.Size)
	if err != nil || end > len(frame.Data) {
		return 0, ErrInvalidData
	}
	if unit.Type == 0 {
		if frame.Data[unit.Offset]&0x80 != 0 || frame.Data[unit.Offset]&0x1f == 0 {
			return 0, ErrInvalidData
		}
		return end, nil
	}
	if frame.Data[unit.Offset]&0x80 != 0 {
		return 0, ErrInvalidData
	}
	if unit.Type != frame.Data[unit.Offset]&0x1f {
		return 0, ErrInvalidData
	}
	if unit.ParameterSet != (unit.Type == 7 || unit.Type == 8) {
		return 0, ErrInvalidData
	}
	if unit.KeyFrame != (unit.Type == 5 || unit.ParameterSet) {
		return 0, ErrInvalidData
	}
	return end, nil
}

// AppendAccessUnitData appends a caller-owned copy of AccessUnitData to dst.
//
// On error, dst is returned unchanged.
func (frame EncodedFrame) AppendAccessUnitData(dst []byte) ([]byte, error) {
	data, err := frame.AccessUnitData()
	if err != nil {
		return dst, err
	}
	return appendPublicBytes(dst, data)
}

// RTPPacketData returns the complete RTP packet bytes for RTPPackets[index].
//
// RTPPacketData validates the packet metadata, payload view, and RTP payload
// syntax before returning bytes.
//
// The returned slice is clipped to its length, so appending to it cannot
// overwrite the following bytes in the packet backing store.
func (frame EncodedFrame) RTPPacketData(index int) ([]byte, error) {
	if frame.Dropped || frame.OutputFormat != EncoderOutputRTP ||
		len(frame.RTPPackets) > maxEncoderRTPPacketListLen ||
		index < 0 || index >= len(frame.RTPPackets) {
		return nil, ErrInvalidData
	}
	return frame.RTPPackets[index].PacketData()
}

// AppendRTPPacketData appends a caller-owned copy of RTPPacketData(index) to dst.
//
// On error, dst is returned unchanged.
func (frame EncodedFrame) AppendRTPPacketData(dst []byte, index int) ([]byte, error) {
	data, err := frame.RTPPacketData(index)
	if err != nil {
		return dst, err
	}
	return appendPublicBytes(dst, data)
}

// RTPPayloadData returns the RTP payload bytes for RTPPackets[index].
//
// The returned slice is clipped to its length, so appending to it cannot
// overwrite the following bytes in the packet backing store.
func (frame EncodedFrame) RTPPayloadData(index int) ([]byte, error) {
	if frame.Dropped || frame.OutputFormat != EncoderOutputRTP ||
		len(frame.RTPPackets) > maxEncoderRTPPacketListLen ||
		index < 0 || index >= len(frame.RTPPackets) {
		return nil, ErrInvalidData
	}
	return frame.RTPPackets[index].PayloadData()
}

// AppendRTPPayloadData appends a caller-owned copy of RTPPayloadData(index) to dst.
//
// On error, dst is returned unchanged.
func (frame EncodedFrame) AppendRTPPayloadData(dst []byte, index int) ([]byte, error) {
	data, err := frame.RTPPayloadData(index)
	if err != nil {
		return dst, err
	}
	return appendPublicBytes(dst, data)
}

// Validate reports whether frame is a well-formed encoded result.
//
// Dropped results must carry only metadata. Non-dropped Annex B and AVC results
// must not carry RTP packets. Non-dropped RTP results must carry RTP packets,
// and every packet must pass EncoderRTPPacket.Validate.
func (frame EncodedFrame) Validate() error {
	if !encodedFrameCloneStorageOK(frame) {
		return ErrInvalidData
	}
	if frame.Dropped {
		if !validEncoderOutputFormat(frame.OutputFormat) ||
			len(frame.Data) != 0 || len(frame.NALUnits) != 0 || len(frame.RTPPackets) != 0 {
			return ErrInvalidData
		}
		return nil
	}
	if _, _, err := frame.accessUnitRange(); err != nil {
		return err
	}
	for i := range frame.RTPPackets {
		if err := frame.RTPPackets[i].Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Clone returns a deep-owned copy of the encoded result.
//
// The cloned Data, NALUnits, RTPPackets, and RTP packet payload views are
// independent from frame and safe to retain after caller-owned EncodeInto
// buffers are reused.
func (frame EncodedFrame) Clone() (EncodedFrame, error) {
	if err := frame.Validate(); err != nil {
		return EncodedFrame{}, err
	}
	if frame.Dropped {
		return EncodedFrame{
			OutputFormat: frame.OutputFormat,
			KeyFrame:     frame.KeyFrame,
			IDR:          frame.IDR,
			PTS:          frame.PTS,
			DTS:          frame.DTS,
			RTPTime:      frame.RTPTime,
			Dropped:      true,
		}, nil
	}
	clone := EncodedFrame{
		OutputFormat: frame.OutputFormat,
		Data:         cloneByteSlice(frame.Data),
		NALUnits:     append([]EncoderNALUnit(nil), frame.NALUnits...),
		RTPPackets:   make([]EncoderRTPPacket, len(frame.RTPPackets)),
		KeyFrame:     frame.KeyFrame,
		IDR:          frame.IDR,
		PTS:          frame.PTS,
		DTS:          frame.DTS,
		RTPTime:      frame.RTPTime,
	}
	for i, packet := range frame.RTPPackets {
		clonePacket, err := packet.Clone()
		if err != nil {
			return EncodedFrame{}, err
		}
		clone.RTPPackets[i] = clonePacket
	}
	return clone, nil
}

func encodedFrameCloneStorageOK(frame EncodedFrame) bool {
	if len(frame.Data) > maxInt/2 ||
		len(frame.NALUnits) > maxEncoderNALUnitListLen ||
		len(frame.RTPPackets) > maxEncoderRTPPacketListLen {
		return false
	}
	for _, packet := range frame.RTPPackets {
		if !encoderRTPPacketCloneStorageOK(packet) {
			return false
		}
	}
	return true
}

func unsafeSliceOffset(base []byte, sub []byte) int {
	if len(base) == 0 || len(sub) == 0 {
		return -1
	}
	baseStart := uintptr(unsafe.Pointer(unsafe.SliceData(base)))
	baseEnd := baseStart + uintptr(len(base))
	subStart := uintptr(unsafe.Pointer(unsafe.SliceData(sub)))
	subEnd := subStart + uintptr(len(sub))
	if subStart < baseStart || subEnd > baseEnd || subEnd < subStart {
		return -1
	}
	return int(subStart - baseStart)
}

// EncoderParameterSets contains caller-owned SPS/PPS helper surfaces.
//
// Each byte slice returned by ParameterSets is isolated from later calls and may
// be mutated by the caller.
type EncoderParameterSets struct {
	SPS                           []byte
	PPS                           []byte
	AnnexB                        []byte
	AVCDecoderConfigurationRecord []byte
}

// SPSData returns the SPS NAL bytes after validating public storage sizes.
//
// The returned slice is clipped to its length, so appending to it cannot
// overwrite the following bytes in the backing store.
func (sets EncoderParameterSets) SPSData() ([]byte, error) {
	return publicBytesView(sets.SPS)
}

// PPSData returns the PPS NAL bytes after validating public storage sizes.
//
// The returned slice is clipped to its length, so appending to it cannot
// overwrite the following bytes in the backing store.
func (sets EncoderParameterSets) PPSData() ([]byte, error) {
	return publicBytesView(sets.PPS)
}

// AnnexBData returns the Annex B parameter-set bytes after validating public
// storage sizes.
//
// The returned slice is clipped to its length, so appending to it cannot
// overwrite the following bytes in the backing store.
func (sets EncoderParameterSets) AnnexBData() ([]byte, error) {
	return publicBytesView(sets.AnnexB)
}

// AVCCData returns the AVC decoder configuration record bytes after validating
// public storage sizes.
//
// The returned slice is clipped to its length, so appending to it cannot
// overwrite the following bytes in the backing store.
func (sets EncoderParameterSets) AVCCData() ([]byte, error) {
	return publicBytesView(sets.AVCDecoderConfigurationRecord)
}

// AppendSPS appends a caller-owned copy of the SPS NAL to dst after validating
// public storage sizes. On error, dst is returned unchanged.
func (sets EncoderParameterSets) AppendSPS(dst []byte) ([]byte, error) {
	return appendPublicBytes(dst, sets.SPS)
}

// AppendPPS appends a caller-owned copy of the PPS NAL to dst after validating
// public storage sizes. On error, dst is returned unchanged.
func (sets EncoderParameterSets) AppendPPS(dst []byte) ([]byte, error) {
	return appendPublicBytes(dst, sets.PPS)
}

// AppendAnnexB appends a caller-owned copy of the Annex B parameter sets to dst
// after validating public storage sizes. On error, dst is returned unchanged.
func (sets EncoderParameterSets) AppendAnnexB(dst []byte) ([]byte, error) {
	return appendPublicBytes(dst, sets.AnnexB)
}

// AppendAVCC appends a caller-owned copy of the AVC decoder configuration
// record to dst after validating public storage sizes. On error, dst is
// returned unchanged.
func (sets EncoderParameterSets) AppendAVCC(dst []byte) ([]byte, error) {
	return appendPublicBytes(dst, sets.AVCDecoderConfigurationRecord)
}

// Clone returns a deep-owned copy of the parameter-set helper surfaces after
// validating public storage sizes.
func (sets EncoderParameterSets) Clone() (EncoderParameterSets, error) {
	if err := sets.Validate(); err != nil {
		return EncoderParameterSets{}, err
	}
	return EncoderParameterSets{
		SPS:                           cloneByteSlice(sets.SPS),
		PPS:                           cloneByteSlice(sets.PPS),
		AnnexB:                        cloneByteSlice(sets.AnnexB),
		AVCDecoderConfigurationRecord: cloneByteSlice(sets.AVCDecoderConfigurationRecord),
	}, nil
}

// Validate reports whether the parameter-set helper surfaces have public
// storage sizes accepted by Clone.
func (sets EncoderParameterSets) Validate() error {
	if !encoderParameterSetsCloneStorageOK(sets) {
		return ErrInvalidData
	}
	return nil
}

func encoderParameterSetsCloneStorageOK(sets EncoderParameterSets) bool {
	return len(sets.SPS) <= maxInt/2 &&
		len(sets.PPS) <= maxInt/2 &&
		len(sets.AnnexB) <= maxInt/2 &&
		len(sets.AVCDecoderConfigurationRecord) <= maxInt/2
}

// EncoderSEI contains caller-owned recovery-point SEI helper surfaces.
//
// Each byte slice returned by RecoveryPointSEI is isolated from later calls and
// may be mutated by the caller.
type EncoderSEI struct {
	NAL    []byte
	AnnexB []byte
	AVC    []byte
}

// NALData returns the SEI NAL bytes after validating public storage sizes.
//
// The returned slice is clipped to its length, so appending to it cannot
// overwrite the following bytes in the backing store.
func (sei EncoderSEI) NALData() ([]byte, error) {
	return publicBytesView(sei.NAL)
}

// AnnexBData returns the Annex B SEI bytes after validating public storage
// sizes.
//
// The returned slice is clipped to its length, so appending to it cannot
// overwrite the following bytes in the backing store.
func (sei EncoderSEI) AnnexBData() ([]byte, error) {
	return publicBytesView(sei.AnnexB)
}

// AVCData returns the AVC SEI bytes after validating public storage sizes.
//
// The returned slice is clipped to its length, so appending to it cannot
// overwrite the following bytes in the backing store.
func (sei EncoderSEI) AVCData() ([]byte, error) {
	return publicBytesView(sei.AVC)
}

// AppendNAL appends a caller-owned copy of the SEI NAL to dst after validating
// public storage sizes. On error, dst is returned unchanged.
func (sei EncoderSEI) AppendNAL(dst []byte) ([]byte, error) {
	return appendPublicBytes(dst, sei.NAL)
}

// AppendAnnexB appends a caller-owned copy of the Annex B SEI NAL to dst after
// validating public storage sizes. On error, dst is returned unchanged.
func (sei EncoderSEI) AppendAnnexB(dst []byte) ([]byte, error) {
	return appendPublicBytes(dst, sei.AnnexB)
}

// AppendAVC appends a caller-owned copy of the AVC SEI NAL to dst after
// validating public storage sizes. On error, dst is returned unchanged.
func (sei EncoderSEI) AppendAVC(dst []byte) ([]byte, error) {
	return appendPublicBytes(dst, sei.AVC)
}

// Clone returns a deep-owned copy of the SEI helper surfaces after validating
// public storage sizes.
func (sei EncoderSEI) Clone() (EncoderSEI, error) {
	if err := sei.Validate(); err != nil {
		return EncoderSEI{}, err
	}
	return EncoderSEI{
		NAL:    cloneByteSlice(sei.NAL),
		AnnexB: cloneByteSlice(sei.AnnexB),
		AVC:    cloneByteSlice(sei.AVC),
	}, nil
}

// Validate reports whether the SEI helper surfaces have public storage sizes
// accepted by Clone.
func (sei EncoderSEI) Validate() error {
	if !encoderSEICloneStorageOK(sei) {
		return ErrInvalidData
	}
	return nil
}

func encoderSEICloneStorageOK(sei EncoderSEI) bool {
	return len(sei.NAL) <= maxInt/2 &&
		len(sei.AnnexB) <= maxInt/2 &&
		len(sei.AVC) <= maxInt/2
}

func publicBytesView(src []byte) ([]byte, error) {
	if len(src) > maxInt/2 {
		return nil, ErrInvalidData
	}
	return src[:len(src):len(src)], nil
}

func appendPublicBytes(dst []byte, src []byte) ([]byte, error) {
	if len(dst) > maxInt/2 || len(src) > maxInt/2 {
		return dst, ErrInvalidData
	}
	n, err := checkedAddInt(len(dst), len(src))
	if err != nil || n > maxInt/2 {
		return dst, ErrInvalidData
	}
	if cap(dst) >= n && byteSlicesOverlap(dst[:n], src) {
		out := make([]byte, n)
		copy(out, dst)
		copy(out[len(dst):], src)
		return out, nil
	}
	return append(dst, src...), nil
}

func appendPublicSlice[T any](dst []T, src []T, maxLen int) ([]T, error) {
	if len(dst) > maxLen || len(src) > maxLen {
		return dst, ErrInvalidData
	}
	n, err := checkedAddInt(len(dst), len(src))
	if err != nil || n > maxLen {
		return dst, ErrInvalidData
	}
	if cap(dst) >= n && slicesOverlap(dst[:n], src) {
		out := make([]T, n)
		copy(out, dst)
		copy(out[len(dst):], src)
		return out, nil
	}
	return append(dst, src...), nil
}

func byteSlicesOverlap(a []byte, b []byte) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	aStart := uintptr(unsafe.Pointer(&a[0]))
	aEnd := aStart + uintptr(len(a)-1)
	bStart := uintptr(unsafe.Pointer(&b[0]))
	bEnd := bStart + uintptr(len(b)-1)
	if aEnd < aStart || bEnd < bStart {
		return true
	}
	return aStart <= bEnd && bStart <= aEnd
}

func slicesOverlap[T any](a []T, b []T) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	var zero T
	size := unsafe.Sizeof(zero)
	if size == 0 {
		return false
	}
	aStart := uintptr(unsafe.Pointer(unsafe.SliceData(a)))
	aEnd := aStart + uintptr(len(a)-1)*size
	bStart := uintptr(unsafe.Pointer(unsafe.SliceData(b)))
	bEnd := bStart + uintptr(len(b)-1)*size
	if aEnd < aStart || bEnd < bStart {
		return true
	}
	return aStart <= bEnd && bStart <= aEnd
}

// EncoderReconfigure contains optional runtime encoder updates.
//
// Scalar fields update when non-zero. Paired scalar fields, including
// FrameRateNum/FrameRateDen and Width/Height, must be supplied together.
// Pointer fields update when non-nil, including explicit false or zero values
// where valid. Limits is the zero-capable grouped budget update and is applied
// after MaxFrameSize, MaxEncodeTimeUS, SliceMaxBytes, and their pointer
// zero-value forms. RTPPayloadType uses nil to leave the live value unchanged;
// a non-nil zero selects the dynamic default payload type 96. Reconfigure
// validates the resulting configuration before changing encoder state; invalid
// updates leave the encoder unchanged. ForceIDR queues an IDR request even when
// no config field changes.
type EncoderReconfigure struct {
	TargetBitrate         int
	MaxBitrate            int
	RateControl           EncoderRateControlMode
	VBVBufferSize         *int
	InitialQP             *int
	MinQP                 *int
	MaxQP                 *int
	FrameDrop             EncoderFrameDropMode
	FrameRateNum          int
	FrameRateDen          int
	Width                 int
	Height                int
	DeblockMode           EncoderDeblockMode
	RTPMaxPayloadSize     int
	Limits                *EncoderLimits
	MaxFrameSize          int
	MaxFrameSizeLimit     *int
	MaxEncodeTimeUS       int
	MaxEncodeTimeUSLimit  *int
	SliceCount            int
	SliceMaxBytes         int
	SliceMaxBytesLimit    *int
	Preset                EncoderPreset
	ForceIDR              bool
	GOPSize               int
	IDRInterval           int
	SPSPPSMode            EncoderSPSPPSMode
	SPSPPSBeforeIDR       *bool
	RecoveryPointSEI      *bool
	OutputFormat          EncoderOutputFormat
	RTPPacketizationMode  *EncoderRTPPacketizationMode
	STAPA                 *bool
	RTPPayloadType        *uint8
	RTPSSRC               *uint32
	RTPTimestampIncrement uint32
}

// EncoderLimits groups runtime output and latency budgets.
//
// Zero disables the corresponding budget. Negative values are invalid. Use
// SetLimits when multiple limits must change together and either all updates
// should be accepted or none should be applied.
type EncoderLimits struct {
	MaxFrameSize    int
	SliceMaxBytes   int
	MaxEncodeTimeUS int
}

type Encoder struct {
	cfg                EncoderConfig
	forceIDR           bool
	frameNum           uint32
	idrPicID           uint32
	rtpSequenceNumber  uint16
	nextRTPTime        uint32
	rtpTimeInitialized bool
	reference          encoderReferenceFrame
	p16MVs             []encoderP16x16MotionVector
	p16MVDs            []h264.EncoderMotionVectorDelta
	bitrateCreditBytes int
	bitrateCreditInit  bool
	framesSinceIDR     int
	rtpPacketCallback  EncoderRTPPacketCallback
}

// DefaultRealtimeEncoderConfig returns a realtime/WebRTC-oriented 8-bit I420
// configuration template for the requested dimensions.
func DefaultRealtimeEncoderConfig(width, height int) EncoderConfig {
	strideY, strideCb, strideCr := defaultEncoderI420Strides(width)
	return EncoderConfig{
		Width:                 width,
		Height:                height,
		StrideY:               strideY,
		StrideCb:              strideCb,
		StrideCr:              strideCr,
		PixelFormat:           EncoderPixelFormatI420,
		FrameRateNum:          30,
		FrameRateDen:          1,
		TimeBaseNum:           1,
		TimeBaseDen:           90000,
		Profile:               EncoderProfileConstrainedBaseline,
		LevelIDC:              31,
		EntropyMode:           EncoderEntropyCAVLC,
		DeblockMode:           EncoderDeblockEnabled,
		MaxReferenceFrames:    1,
		SPSPPSMode:            EncoderSPSPPSInBandKeyframes,
		RateControl:           EncoderRateControlCBR,
		TargetBitrate:         1_000_000,
		MaxBitrate:            1_000_000,
		VBVBufferSize:         1_000_000,
		InitialQP:             26,
		MinQP:                 10,
		MaxQP:                 42,
		Preset:                EncoderPresetRealtime,
		ZeroLookahead:         true,
		FrameDrop:             EncoderFrameDropToBitrate,
		MaxEncodeTimeUS:       10_000,
		SliceCount:            1,
		Workers:               1,
		Deterministic:         true,
		GOPSize:               60,
		IDRInterval:           60,
		SPSPPSBeforeIDR:       true,
		RecoveryPointSEI:      true,
		OutputFormat:          EncoderOutputRTP,
		RTPMaxPayloadSize:     1200,
		RTPPacketizationMode:  EncoderRTPPacketizationNonInterleaved,
		DONDisabled:           true,
		RTPPayloadType:        96,
		RTPTimestampIncrement: 3000,
	}
}

// DefaultEncoderConfig returns the same realtime template as
// DefaultRealtimeEncoderConfig.
func DefaultEncoderConfig(width, height int) EncoderConfig {
	return DefaultRealtimeEncoderConfig(width, height)
}

// NewEncoder validates and normalizes cfg, then returns a fresh encoder.
func NewEncoder(cfg EncoderConfig) (*Encoder, error) {
	normalized, err := normalizeEncoderConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Encoder{cfg: normalized}, nil
}

// Validate reports whether cfg can be used to construct an encoder.
func (cfg EncoderConfig) Validate() error {
	_, err := normalizeEncoderConfig(cfg)
	return err
}

// Normalize returns the validated configuration after applying derived defaults.
//
// The returned value matches the configuration NewEncoder would store. cfg is
// not modified.
func (cfg EncoderConfig) Normalize() (EncoderConfig, error) {
	return normalizeEncoderConfig(cfg)
}

// ValidateFrame reports whether frame can be encoded by cfg after normalization.
//
// cfg and frame are not modified. The validation matches Encoder.ValidateFrame
// without constructing or mutating a live encoder.
func (cfg EncoderConfig) ValidateFrame(frame EncoderFrame) error {
	normalized, err := normalizeEncoderConfig(cfg)
	if err != nil {
		return err
	}
	enc := Encoder{cfg: normalized}
	return enc.validateFrame(frame)
}

// ParameterSets returns SPS/PPS headers for cfg after normalization.
//
// All returned byte slices are caller-owned. cfg is not modified.
func (cfg EncoderConfig) ParameterSets() (EncoderParameterSets, error) {
	normalized, err := normalizeEncoderConfig(cfg)
	if err != nil {
		return EncoderParameterSets{}, err
	}
	enc := Encoder{cfg: normalized}
	return enc.ParameterSets()
}

// RecoveryPointSEIMessage returns a recovery-point SEI NAL for cfg after normalization.
//
// All returned byte slices are caller-owned. cfg is not modified.
func (cfg EncoderConfig) RecoveryPointSEIMessage(recoveryFrameCount uint32) (EncoderSEI, error) {
	normalized, err := normalizeEncoderConfig(cfg)
	if err != nil {
		return EncoderSEI{}, err
	}
	enc := Encoder{cfg: normalized}
	return enc.RecoveryPointSEI(recoveryFrameCount)
}

// Reset clears encoder coding state while preserving configuration and RTP callback.
//
// After Reset, the next successfully encoded frame starts a fresh sequence for
// the current encoder configuration.
func (e *Encoder) Reset() error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	cfg := e.cfg
	callback := e.rtpPacketCallback
	*e = Encoder{
		cfg:               cfg,
		rtpPacketCallback: callback,
	}
	return nil
}

// Config returns a copy of the current normalized encoder configuration,
// including accepted runtime setter and Reconfigure updates.
func (e *Encoder) Config() EncoderConfig {
	if e == nil {
		return EncoderConfig{}
	}
	return e.cfg
}

// I420Frame returns an EncoderFrame populated from the current encoder
// configuration.
func (e *Encoder) I420Frame(y, cb, cr []byte, pts int64) EncoderFrame {
	if e == nil {
		return EncoderFrame{Y: y, Cb: cb, Cr: cr, PTS: pts}
	}
	return e.cfg.I420Frame(y, cb, cr, pts)
}

// ValidateFrame reports whether frame can be encoded with the current
// configuration without changing encoder state.
func (e *Encoder) ValidateFrame(frame EncoderFrame) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	return e.validateFrame(frame)
}

// ParameterSets returns SPS/PPS headers for the current encoder configuration.
//
// All returned byte slices are caller-owned and isolated from the encoder.
func (e *Encoder) ParameterSets() (EncoderParameterSets, error) {
	if e == nil {
		return EncoderParameterSets{}, encoderInvalid("nil encoder")
	}
	cfg, err := e.parameterSetConfig()
	if err != nil {
		return EncoderParameterSets{}, err
	}
	sets, err := h264.BuildEncoderParameterSets(cfg)
	if err != nil {
		return EncoderParameterSets{}, err
	}
	return encoderParameterSetsFromH264(sets), nil
}

func encoderParameterSetsFromH264(sets h264.EncoderParameterSets) EncoderParameterSets {
	return EncoderParameterSets{
		SPS:                           cloneByteSlice(sets.SPS),
		PPS:                           cloneByteSlice(sets.PPS),
		AnnexB:                        cloneByteSlice(sets.AnnexB),
		AVCDecoderConfigurationRecord: cloneByteSlice(sets.AVCDecoderConfigurationRecord),
	}
}

func (e *Encoder) parameterSetConfig() (h264.EncoderParameterSetConfig, error) {
	profileIDC, constraintFlags, err := encoderProfileSyntax(e.cfg.Profile)
	if err != nil {
		return h264.EncoderParameterSetConfig{}, err
	}
	return h264.EncoderParameterSetConfig{
		ProfileIDC:                     profileIDC,
		ConstraintSetFlags:             constraintFlags,
		LevelIDC:                       e.cfg.LevelIDC,
		Width:                          e.cfg.Width,
		Height:                         e.cfg.Height,
		CropLeft:                       e.cfg.Crop.Left,
		CropRight:                      e.cfg.Crop.Right,
		CropTop:                        e.cfg.Crop.Top,
		CropBottom:                     e.cfg.Crop.Bottom,
		FrameRateNum:                   e.cfg.FrameRateNum,
		FrameRateDen:                   e.cfg.FrameRateDen,
		MaxReferenceFrames:             uint32(e.cfg.MaxReferenceFrames),
		InitialQP:                      e.cfg.InitialQP,
		SARNum:                         e.cfg.Color.SARNum,
		SARDen:                         e.cfg.Color.SARDen,
		VideoFormat:                    e.cfg.Color.VideoFormat,
		FullRange:                      e.cfg.Color.FullRange,
		ColorPrimaries:                 e.cfg.Color.ColorPrimaries,
		ColorTransfer:                  e.cfg.Color.ColorTransfer,
		ColorMatrix:                    e.cfg.Color.ColorMatrix,
		ChromaSampleLocTypeTopField:    e.cfg.Color.ChromaSampleLocTypeTopField,
		ChromaSampleLocTypeBottomField: e.cfg.Color.ChromaSampleLocTypeBottomField,
		NALLengthSize:                  4,
	}, nil
}

// RecoveryPointSEI returns a recovery-point SEI NAL for the current encoder
// configuration.
//
// All returned byte slices are caller-owned and isolated from the encoder.
func (e *Encoder) RecoveryPointSEI(recoveryFrameCount uint32) (EncoderSEI, error) {
	if e == nil {
		return EncoderSEI{}, encoderInvalid("nil encoder")
	}
	sei, err := h264.BuildEncoderRecoveryPointSEI(h264.EncoderRecoveryPointSEIConfig{
		RecoveryFrameCount:    recoveryFrameCount,
		ExactMatchFlag:        true,
		BrokenLinkFlag:        e.cfg.BFrames > 0,
		ChangingSliceGroupIDC: 0,
		NALLengthSize:         4,
	})
	if err != nil {
		return EncoderSEI{}, err
	}
	return encoderSEIFromH264(sei), nil
}

func encoderSEIFromH264(sei h264.EncoderSEIMessage) EncoderSEI {
	return EncoderSEI{
		NAL:    cloneByteSlice(sei.NAL),
		AnnexB: cloneByteSlice(sei.AnnexB),
		AVC:    cloneByteSlice(sei.AVC),
	}
}

// Encode encodes one frame using encoder-owned output storage.
//
// For caller-owned access-unit storage, use EncodeInto.
func (e *Encoder) Encode(frame EncoderFrame) (EncodedFrame, error) {
	return e.EncodeInto(nil, frame)
}

// EncodeInto encodes one frame, appending access-unit bytes to dst.
//
// The returned EncodedFrame.Data may share backing storage with dst. Keep dst
// unchanged while using Data. For RTP output, returned RTP packets own storage
// separate from Data.
func (e *Encoder) EncodeInto(dst []byte, frame EncoderFrame) (EncodedFrame, error) {
	if e == nil {
		return EncodedFrame{}, encoderInvalid("nil encoder")
	}
	view, err := e.validatedFrameView(frame)
	if err != nil {
		return EncodedFrame{}, err
	}
	oldP16MVs := e.p16MVs
	oldP16MVDs := e.p16MVDs
	p16ScratchCommitted := false
	defer func() {
		if !p16ScratchCommitted {
			e.p16MVs = oldP16MVs
			e.p16MVDs = oldP16MVDs
		}
	}()
	idr := e.shouldEncodeIDR(view, frame)
	lateStart := encoderLateDropStart(e.cfg)
	var nalsBuf [4]encoderRawNAL
	nals := nalsBuf[:0]
	if e.shouldEmitParameterSets(idr) {
		parameterSetCfg, err := e.parameterSetConfig()
		if err != nil {
			return EncodedFrame{}, err
		}
		sets, err := h264.BuildEncoderParameterSetNALs(parameterSetCfg)
		if err != nil {
			return EncodedFrame{}, err
		}
		nals = append(nals,
			encoderRawNAL{typ: uint8(h264.NALSPS), raw: sets.SPS, keyFrame: true, parameterSet: true},
			encoderRawNAL{typ: uint8(h264.NALPPS), raw: sets.PPS, keyFrame: true, parameterSet: true},
		)
	}

	var sliceRangeBuf [4]encoderSliceRange
	sliceRanges := appendEncoderSliceRanges(sliceRangeBuf[:0], view.width, view.height, e.cfg.SliceCount)
	if idr {
		for _, r := range sliceRanges {
			nal, err := buildEncoderI420IntraPCMIDRNAL(h264.EncoderI420IntraPCMIDRConfig{
				Width:                      view.width,
				Height:                     view.height,
				StrideY:                    view.strideY,
				StrideCb:                   view.strideCb,
				StrideCr:                   view.strideCr,
				Y:                          view.y,
				Cb:                         view.cb,
				Cr:                         view.cr,
				FrameNum:                   e.frameNum & 0xff,
				IDRPicID:                   e.idrPicID & 0xffff,
				InitialQP:                  e.cfg.InitialQP,
				DisableDeblockingFilterIDC: encoderDeblockingFilterIDC(e.cfg.DeblockMode),
				FirstMBAddr:                uint32(r.firstMB),
				MacroblockCount:            uint32(r.macroblockCount),
				NALLengthSize:              4,
			})
			if err != nil {
				return EncodedFrame{}, err
			}
			nals = append(nals, encoderRawNAL{typ: uint8(h264.NALIDRSlice), raw: nal, keyFrame: true})
		}
	} else if e.referenceMatches(view) {
		for _, r := range sliceRanges {
			nal, err := buildEncoderI420PSkipNAL(h264.EncoderI420PSkipConfig{
				Width:                      view.width,
				Height:                     view.height,
				FrameNum:                   e.frameNum & 0xff,
				InitialQP:                  e.cfg.InitialQP,
				DisableDeblockingFilterIDC: encoderDeblockingFilterIDC(e.cfg.DeblockMode),
				FirstMBAddr:                uint32(r.firstMB),
				MacroblockCount:            uint32(r.macroblockCount),
				NALLengthSize:              4,
			})
			if err != nil {
				return EncodedFrame{}, err
			}
			nals = append(nals, encoderRawNAL{typ: uint8(h264.NALSlice), raw: nal})
		}
	} else {
		var p16MVBuf [64]encoderP16x16MotionVector
		p16MVs, ok := e.p16x16NoResidualMotion(view, p16MVBuf[:0])
		if !ok {
			residualNals, residualOK, err := e.p16x16ResidualNALs(view, sliceRanges)
			if err != nil {
				return EncodedFrame{}, err
			}
			if residualOK {
				nals = append(nals, residualNals...)
			} else {
				if e.cfg.RecoveryPointSEI {
					sei, err := h264.BuildEncoderRecoveryPointSEINAL(h264.EncoderRecoveryPointSEIConfig{
						RecoveryFrameCount:    0,
						ExactMatchFlag:        true,
						BrokenLinkFlag:        e.cfg.BFrames > 0,
						ChangingSliceGroupIDC: 0,
					})
					if err != nil {
						return EncodedFrame{}, err
					}
					nals = append(nals, encoderRawNAL{typ: uint8(h264.NALSEI), raw: sei})
				}
				for _, r := range sliceRanges {
					nal, err := buildEncoderI420IntraPCMPNAL(h264.EncoderI420IntraPCMPConfig{
						Width:                      view.width,
						Height:                     view.height,
						StrideY:                    view.strideY,
						StrideCb:                   view.strideCb,
						StrideCr:                   view.strideCr,
						Y:                          view.y,
						Cb:                         view.cb,
						Cr:                         view.cr,
						FrameNum:                   e.frameNum & 0xff,
						InitialQP:                  e.cfg.InitialQP,
						DisableDeblockingFilterIDC: encoderDeblockingFilterIDC(e.cfg.DeblockMode),
						FirstMBAddr:                uint32(r.firstMB),
						MacroblockCount:            uint32(r.macroblockCount),
						NALLengthSize:              4,
					})
					if err != nil {
						return EncodedFrame{}, err
					}
					nals = append(nals, encoderRawNAL{typ: uint8(h264.NALSlice), raw: nal})
				}
			}
		} else {
			var mvdBuf [64]h264.EncoderMotionVectorDelta
			macroblocksPerRow := view.width >> 4
			for _, r := range sliceRanges {
				mvdsBuf := mvdBuf[:0]
				if r.macroblockCount > cap(mvdsBuf) {
					e.p16MVDs = resizeEncoderP16x16MVDs(e.p16MVDs, r.macroblockCount)
					mvdsBuf = e.p16MVDs[:0]
				}
				mvds := appendEncoderP16x16NoResidualMVDs(mvdsBuf, p16MVs, r.firstMB, r.macroblockCount, macroblocksPerRow)
				nal, err := buildEncoderI420P16x16NoResidualNAL(h264.EncoderI420P16x16NoResidualConfig{
					Width:                      view.width,
					Height:                     view.height,
					FrameNum:                   e.frameNum & 0xff,
					InitialQP:                  e.cfg.InitialQP,
					DisableDeblockingFilterIDC: encoderDeblockingFilterIDC(e.cfg.DeblockMode),
					FirstMBAddr:                uint32(r.firstMB),
					MacroblockCount:            uint32(r.macroblockCount),
					MVDs:                       mvds,
					NALLengthSize:              4,
				})
				if err != nil {
					return EncodedFrame{}, err
				}
				nals = append(nals, encoderRawNAL{typ: uint8(h264.NALSlice), raw: nal})
			}
		}
	}

	outputSize, err := encoderAccessUnitOutputSize(e.cfg.OutputFormat, nals)
	if err != nil {
		return EncodedFrame{}, err
	}
	rtpTime := e.encoderRTPTime(frame)
	if miss, err := encoderOutputBudgetMiss(e.cfg, nals, outputSize); err != nil {
		if e.cfg.FrameDrop == EncoderFrameDropToBitrate && miss != encoderOutputBudgetNone {
			e.advanceEncoderRTPTime(frame, rtpTime)
			e.advanceEncoderBitrateBudget(0)
			return EncodedFrame{
				OutputFormat: e.cfg.OutputFormat,
				PTS:          frame.PTS,
				DTS:          frame.PTS,
				RTPTime:      rtpTime,
				Dropped:      true,
			}, nil
		}
		return EncodedFrame{}, err
	}
	if e.encoderBitrateBudgetMiss(outputSize) {
		e.advanceEncoderRTPTime(frame, rtpTime)
		e.advanceEncoderBitrateBudget(0)
		return EncodedFrame{
			OutputFormat: e.cfg.OutputFormat,
			PTS:          frame.PTS,
			DTS:          frame.PTS,
			RTPTime:      rtpTime,
			Dropped:      true,
		}, nil
	}
	var packets []EncoderRTPPacket
	if e.cfg.OutputFormat == EncoderOutputRTP {
		switch e.cfg.RTPPacketizationMode {
		case EncoderRTPPacketizationSingleNAL:
			packets, err = packetizeEncoderRTPSingleNAL(nals, e.cfg.RTPMaxPayloadSize, rtpTime)
		case EncoderRTPPacketizationNonInterleaved:
			packets, err = packetizeEncoderRTPMode1(nals, e.cfg.RTPMaxPayloadSize, rtpTime, e.cfg.STAPA)
		default:
			err = encoderInvalid("unknown RTP packetization mode")
		}
		if err != nil {
			return EncodedFrame{}, err
		}
		if encoderLateBudgetMiss(lateStart, e.cfg) {
			e.advanceEncoderRTPTime(frame, rtpTime)
			return EncodedFrame{
				OutputFormat: e.cfg.OutputFormat,
				PTS:          frame.PTS,
				DTS:          frame.PTS,
				RTPTime:      rtpTime,
				Dropped:      true,
			}, nil
		}
	} else if encoderLateBudgetMiss(lateStart, e.cfg) {
		e.advanceEncoderRTPTime(frame, rtpTime)
		return EncodedFrame{
			OutputFormat: e.cfg.OutputFormat,
			PTS:          frame.PTS,
			DTS:          frame.PTS,
			RTPTime:      rtpTime,
			Dropped:      true,
		}, nil
	}
	data, units, err := appendEncoderAccessUnit(dst, e.cfg.OutputFormat, nals)
	if err != nil {
		return EncodedFrame{}, err
	}
	p16ScratchCommitted = true
	if e.cfg.OutputFormat == EncoderOutputRTP {
		e.stampRTPPackets(packets)
		e.notifyRTPPacketCallback(packets, frame, rtpTime, idr, idr)
	}

	e.advanceEncoderRTPTime(frame, rtpTime)
	e.advanceEncoderBitrateBudget(outputSize)
	e.storeReference(view)
	e.forceIDR = false
	e.frameNum = (e.frameNum + 1) & 0xff
	if idr {
		e.idrPicID = (e.idrPicID + 1) & 0xffff
		e.framesSinceIDR = 1
	} else {
		e.framesSinceIDR++
	}
	return EncodedFrame{
		OutputFormat: e.cfg.OutputFormat,
		Data:         data,
		NALUnits:     units,
		RTPPackets:   packets,
		KeyFrame:     idr,
		IDR:          idr,
		PTS:          frame.PTS,
		DTS:          frame.PTS,
		RTPTime:      rtpTime,
	}, nil
}

// ForceIDR requests that the next successfully encoded frame be an IDR frame.
func (e *Encoder) ForceIDR() {
	if e != nil {
		e.forceIDR = true
	}
}

// HandlePLI handles a WebRTC Picture Loss Indication by requesting an IDR.
func (e *Encoder) HandlePLI() {
	e.ForceIDR()
}

// HandleFIR handles a WebRTC Full Intra Request by requesting an IDR.
func (e *Encoder) HandleFIR() {
	e.ForceIDR()
}

// PendingIDR reports whether an IDR request is queued.
func (e *Encoder) PendingIDR() bool {
	return e != nil && e.forceIDR
}

// SetBitrate updates the target and max bitrate and resets bitrate budget
// accounting after a successful validation.
func (e *Encoder) SetBitrate(targetBitrate, maxBitrate int) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	cfg := e.cfg
	cfg.TargetBitrate = targetBitrate
	cfg.MaxBitrate = maxBitrate
	normalized, err := normalizeEncoderConfig(cfg)
	if err != nil {
		return err
	}
	e.cfg = normalized
	e.resetEncoderBitrateBudget()
	return nil
}

// SetRateControl updates the runtime rate-control mode.
//
// Invalid updates leave the encoder configuration and coding state unchanged.
func (e *Encoder) SetRateControl(mode EncoderRateControlMode) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	if mode == 0 {
		return encoderInvalid("rate-control mode cannot be zero")
	}
	return e.Reconfigure(EncoderReconfigure{RateControl: mode})
}

// SetVBVBufferSize updates the VBV burst-capacity budget in bits.
//
// Passing zero disables the VBV cap. Invalid updates leave the encoder
// configuration and coding state unchanged.
func (e *Encoder) SetVBVBufferSize(size int) error {
	return e.Reconfigure(EncoderReconfigure{VBVBufferSize: &size})
}

// SetFrameDropMode updates runtime drop behavior.
//
// Invalid updates leave the encoder configuration and coding state unchanged.
func (e *Encoder) SetFrameDropMode(mode EncoderFrameDropMode) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	if mode == 0 {
		return encoderInvalid("frame drop mode cannot be zero")
	}
	return e.Reconfigure(EncoderReconfigure{FrameDrop: mode})
}

// SetQP updates the runtime QP range.
//
// A valid QP update queues an IDR through the same path as EncoderReconfigure.
// Invalid updates leave the encoder configuration and coding state unchanged.
func (e *Encoder) SetQP(initial, min, max int) error {
	return e.Reconfigure(EncoderReconfigure{
		InitialQP: &initial,
		MinQP:     &min,
		MaxQP:     &max,
	})
}

// SetFrameRate updates the configured frame rate, derived RTP timestamp
// increment, and bitrate budget accounting after a successful validation.
func (e *Encoder) SetFrameRate(num, den int) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	cfg := e.cfg
	cfg.FrameRateNum = num
	cfg.FrameRateDen = den
	increment, err := rtpTimestampIncrementChecked(cfg.TimeBaseDen, num, den)
	if err != nil {
		return err
	}
	cfg.RTPTimestampIncrement = increment
	normalized, err := normalizeEncoderConfig(cfg)
	if err != nil {
		return err
	}
	e.cfg = normalized
	e.resetEncoderBitrateBudget()
	return nil
}

// SetResolution updates encoded frame dimensions and default I420 strides.
//
// A valid resolution update clears the current reference state and queues an
// IDR. Invalid updates leave the encoder configuration and coding state
// unchanged.
func (e *Encoder) SetResolution(width, height int) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	if width <= 0 || height <= 0 {
		return encoderInvalid("runtime resolution update requires positive width and height")
	}
	return e.Reconfigure(EncoderReconfigure{Width: width, Height: height})
}

// SetGOP updates GOP and IDR cadence.
//
// Invalid updates leave the encoder configuration and coding state unchanged.
func (e *Encoder) SetGOP(gopSize, idrInterval int) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	if gopSize <= 0 || idrInterval <= 0 {
		return encoderInvalid("GOP size and IDR interval must be positive")
	}
	return e.Reconfigure(EncoderReconfigure{GOPSize: gopSize, IDRInterval: idrInterval})
}

// SetRTPTimestampIncrement updates the automatic RTP timestamp step.
//
// Passing zero is invalid at runtime; use SetFrameRate to derive a new
// increment from frame-rate controls. Invalid updates leave the encoder
// configuration and coding state unchanged.
func (e *Encoder) SetRTPTimestampIncrement(increment uint32) error {
	if increment == 0 {
		return encoderInvalid("RTP timestamp increment cannot be zero")
	}
	return e.Reconfigure(EncoderReconfigure{RTPTimestampIncrement: increment})
}

// SetRTPMaxPayloadSize updates the RTP packet payload limit after validating the
// resulting configuration.
func (e *Encoder) SetRTPMaxPayloadSize(size int) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	cfg := e.cfg
	cfg.RTPMaxPayloadSize = size
	normalized, err := normalizeEncoderConfig(cfg)
	if err != nil {
		return err
	}
	e.cfg = normalized
	return nil
}

// SetLimits updates output byte and late-frame time budgets atomically.
//
// Passing zero for a field disables that budget. Invalid updates leave the
// encoder configuration and coding state unchanged.
func (e *Encoder) SetLimits(limits EncoderLimits) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	cfg := e.cfg
	cfg.MaxFrameSize = limits.MaxFrameSize
	cfg.SliceMaxBytes = limits.SliceMaxBytes
	cfg.MaxEncodeTimeUS = limits.MaxEncodeTimeUS
	normalized, err := normalizeEncoderConfig(cfg)
	if err != nil {
		return err
	}
	e.cfg = normalized
	return nil
}

// SetMaxFrameSize updates the encoded access-unit byte budget.
//
// Passing zero disables the max-frame-size budget. Invalid updates leave the
// encoder configuration and coding state unchanged.
func (e *Encoder) SetMaxFrameSize(size int) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	cfg := e.cfg
	cfg.MaxFrameSize = size
	normalized, err := normalizeEncoderConfig(cfg)
	if err != nil {
		return err
	}
	e.cfg = normalized
	return nil
}

// SetSliceMaxBytes updates the per-slice encoded byte budget.
//
// Passing zero disables the slice byte budget. Invalid updates leave the
// encoder configuration and coding state unchanged.
func (e *Encoder) SetSliceMaxBytes(size int) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	cfg := e.cfg
	cfg.SliceMaxBytes = size
	normalized, err := normalizeEncoderConfig(cfg)
	if err != nil {
		return err
	}
	e.cfg = normalized
	return nil
}

// SetMaxEncodeTimeUS updates the late-frame encode-time budget.
//
// Passing zero disables the late-frame time budget. Invalid updates leave the
// encoder configuration and coding state unchanged.
func (e *Encoder) SetMaxEncodeTimeUS(us int) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	cfg := e.cfg
	cfg.MaxEncodeTimeUS = us
	normalized, err := normalizeEncoderConfig(cfg)
	if err != nil {
		return err
	}
	e.cfg = normalized
	return nil
}

// SetPreset updates the runtime encoder preset.
//
// Invalid updates leave the encoder configuration and coding state unchanged.
func (e *Encoder) SetPreset(preset EncoderPreset) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	if preset == 0 {
		return encoderInvalid("encoder preset cannot be zero")
	}
	return e.Reconfigure(EncoderReconfigure{Preset: preset})
}

// SetSliceCount updates the number of VCL slices emitted per frame.
//
// Invalid updates leave the encoder configuration and coding state unchanged.
func (e *Encoder) SetSliceCount(count int) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	if count <= 0 {
		return encoderInvalid("slice count must be positive")
	}
	return e.Reconfigure(EncoderReconfigure{SliceCount: count})
}

// SetDeblockMode updates the runtime deblocking mode.
//
// Invalid updates leave the encoder configuration and coding state unchanged.
func (e *Encoder) SetDeblockMode(mode EncoderDeblockMode) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	if mode == 0 {
		return encoderInvalid("deblock mode cannot be zero")
	}
	return e.Reconfigure(EncoderReconfigure{DeblockMode: mode})
}

// SetSPSPPSMode updates in-band/out-of-band SPS/PPS emission policy.
//
// Invalid updates leave the encoder configuration and coding state unchanged.
func (e *Encoder) SetSPSPPSMode(mode EncoderSPSPPSMode) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	if mode == 0 {
		return encoderInvalid("SPS/PPS emission mode cannot be zero")
	}
	return e.Reconfigure(EncoderReconfigure{SPSPPSMode: mode})
}

// SetSPSPPSBeforeIDR enables or disables SPS/PPS emission before IDR pictures.
//
// Invalid updates leave the encoder configuration and coding state unchanged.
func (e *Encoder) SetSPSPPSBeforeIDR(enabled bool) error {
	return e.Reconfigure(EncoderReconfigure{SPSPPSBeforeIDR: &enabled})
}

// SetIntraRefresh enables or disables intra refresh.
//
// Enabling intra refresh is outside the admitted encoder subset and returns
// ErrUnsupported.
// Disabling it is accepted as an explicit no-op for callers that mirror a
// runtime control surface. Invalid updates leave the encoder configuration and
// coding state unchanged.
func (e *Encoder) SetIntraRefresh(enabled bool) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	if enabled {
		return encoderUnsupported("intra refresh is outside the admitted encoder subset")
	}
	cfg := e.cfg
	cfg.IntraRefresh = false
	normalized, err := normalizeEncoderConfig(cfg)
	if err != nil {
		return err
	}
	e.cfg = normalized
	return nil
}

// SetRecoveryPointSEI enables or disables recovery-point SEI emission for
// non-IDR recovery pictures.
//
// Invalid updates leave the encoder configuration and coding state unchanged.
func (e *Encoder) SetRecoveryPointSEI(enabled bool) error {
	return e.Reconfigure(EncoderReconfigure{RecoveryPointSEI: &enabled})
}

// SetOutputFormat updates the emitted access-unit container.
//
// A successful output-format update queues an IDR so the next emitted access
// unit can carry a clean boundary for the new container.
func (e *Encoder) SetOutputFormat(format EncoderOutputFormat) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	if format == 0 {
		return encoderInvalid("encoder output format cannot be zero")
	}
	return e.Reconfigure(EncoderReconfigure{
		OutputFormat: format,
		ForceIDR:     true,
	})
}

// SetRTPPacketizationMode updates RTP H.264 packetization controls.
//
// stapa enables STAP-A aggregation when mode is
// EncoderRTPPacketizationNonInterleaved. Invalid updates leave the encoder
// configuration and coding state unchanged.
func (e *Encoder) SetRTPPacketizationMode(mode EncoderRTPPacketizationMode, stapa bool) error {
	return e.Reconfigure(EncoderReconfigure{
		RTPPacketizationMode: &mode,
		STAPA:                &stapa,
	})
}

// SetRTPMetadata updates the RTP payload type and SSRC metadata.
//
// A zero payload type keeps the EncoderConfig normalization rule and restores
// the dynamic default payload type. Invalid updates leave the encoder
// configuration and coding state unchanged.
func (e *Encoder) SetRTPMetadata(payloadType uint8, ssrc uint32) error {
	return e.Reconfigure(EncoderReconfigure{
		RTPPayloadType: &payloadType,
		RTPSSRC:        &ssrc,
	})
}

// SetRTPPacketCallback installs an optional synchronous callback for emitted RTP
// packets.
//
// Passing nil disables the callback.
func (e *Encoder) SetRTPPacketCallback(callback EncoderRTPPacketCallback) {
	if e != nil {
		e.rtpPacketCallback = callback
	}
}

// Reconfigure applies validated runtime updates.
//
// Invalid updates return an error without changing encoder state. Resolution and
// QP and output-format changes queue an IDR after they are accepted.
func (e *Encoder) Reconfigure(update EncoderReconfigure) error {
	if e == nil {
		return encoderInvalid("nil encoder")
	}
	if update.Width < 0 || update.Height < 0 ||
		((update.Width != 0 || update.Height != 0) && (update.Width == 0 || update.Height == 0)) {
		return encoderInvalid("runtime resolution update requires positive width and height")
	}
	if (update.FrameRateNum != 0 || update.FrameRateDen != 0) && (update.FrameRateNum == 0 || update.FrameRateDen == 0) {
		return encoderInvalid("runtime frame-rate update requires positive numerator and denominator")
	}
	if update.GOPSize < 0 || update.IDRInterval < 0 {
		return encoderInvalid("GOP size and IDR interval cannot be negative")
	}
	cfg := e.cfg
	oldWidth := cfg.Width
	oldHeight := cfg.Height
	oldOutputFormat := cfg.OutputFormat
	qpRefresh := update.InitialQP != nil || update.MinQP != nil || update.MaxQP != nil
	if update.TargetBitrate != 0 {
		cfg.TargetBitrate = update.TargetBitrate
	}
	if update.MaxBitrate != 0 {
		cfg.MaxBitrate = update.MaxBitrate
	}
	if update.RateControl != 0 {
		cfg.RateControl = update.RateControl
	}
	if update.VBVBufferSize != nil {
		cfg.VBVBufferSize = *update.VBVBufferSize
	}
	if update.InitialQP != nil {
		cfg.InitialQP = *update.InitialQP
	}
	if update.MinQP != nil {
		cfg.MinQP = *update.MinQP
	}
	if update.MaxQP != nil {
		cfg.MaxQP = *update.MaxQP
	}
	if update.FrameDrop != 0 {
		cfg.FrameDrop = update.FrameDrop
	}
	if update.FrameRateNum != 0 || update.FrameRateDen != 0 {
		cfg.FrameRateNum = update.FrameRateNum
		cfg.FrameRateDen = update.FrameRateDen
		increment, err := rtpTimestampIncrementChecked(cfg.TimeBaseDen, cfg.FrameRateNum, cfg.FrameRateDen)
		if err != nil {
			return err
		}
		cfg.RTPTimestampIncrement = increment
	}
	if update.Width != 0 || update.Height != 0 {
		strideY, strideCb, strideCr := defaultEncoderI420Strides(update.Width)
		cfg.Width = update.Width
		cfg.Height = update.Height
		cfg.StrideY = strideY
		cfg.StrideCb = strideCb
		cfg.StrideCr = strideCr
	}
	if update.DeblockMode != 0 {
		cfg.DeblockMode = update.DeblockMode
	}
	if update.RTPMaxPayloadSize != 0 {
		cfg.RTPMaxPayloadSize = update.RTPMaxPayloadSize
	}
	if update.MaxFrameSize != 0 {
		cfg.MaxFrameSize = update.MaxFrameSize
	}
	if update.MaxFrameSizeLimit != nil {
		cfg.MaxFrameSize = *update.MaxFrameSizeLimit
	}
	if update.MaxEncodeTimeUS != 0 {
		cfg.MaxEncodeTimeUS = update.MaxEncodeTimeUS
	}
	if update.MaxEncodeTimeUSLimit != nil {
		cfg.MaxEncodeTimeUS = *update.MaxEncodeTimeUSLimit
	}
	if update.SliceCount != 0 {
		cfg.SliceCount = update.SliceCount
	}
	if update.SliceMaxBytes != 0 {
		cfg.SliceMaxBytes = update.SliceMaxBytes
	}
	if update.SliceMaxBytesLimit != nil {
		cfg.SliceMaxBytes = *update.SliceMaxBytesLimit
	}
	if update.Limits != nil {
		cfg.MaxFrameSize = update.Limits.MaxFrameSize
		cfg.SliceMaxBytes = update.Limits.SliceMaxBytes
		cfg.MaxEncodeTimeUS = update.Limits.MaxEncodeTimeUS
	}
	if update.Preset != 0 {
		cfg.Preset = update.Preset
	}
	if update.GOPSize != 0 {
		cfg.GOPSize = update.GOPSize
	}
	if update.IDRInterval != 0 {
		cfg.IDRInterval = update.IDRInterval
	}
	if update.SPSPPSMode != 0 {
		cfg.SPSPPSMode = update.SPSPPSMode
	}
	if update.SPSPPSBeforeIDR != nil {
		cfg.SPSPPSBeforeIDR = *update.SPSPPSBeforeIDR
	}
	if update.RecoveryPointSEI != nil {
		cfg.RecoveryPointSEI = *update.RecoveryPointSEI
	}
	if update.OutputFormat != 0 {
		cfg.OutputFormat = update.OutputFormat
	}
	if update.RTPPacketizationMode != nil {
		cfg.RTPPacketizationMode = *update.RTPPacketizationMode
	}
	if update.STAPA != nil {
		cfg.STAPA = *update.STAPA
	}
	if update.RTPPayloadType != nil {
		cfg.RTPPayloadType = *update.RTPPayloadType
	}
	if update.RTPSSRC != nil {
		cfg.RTPSSRC = *update.RTPSSRC
	}
	if update.RTPTimestampIncrement != 0 {
		cfg.RTPTimestampIncrement = update.RTPTimestampIncrement
	}
	normalized, err := normalizeEncoderConfigWithExplicitQP(cfg, update.InitialQP != nil, update.MinQP != nil, update.MaxQP != nil)
	if err != nil {
		return err
	}
	bitrateBudgetRefresh := normalized.TargetBitrate != e.cfg.TargetBitrate ||
		normalized.MaxBitrate != e.cfg.MaxBitrate ||
		normalized.VBVBufferSize != e.cfg.VBVBufferSize ||
		normalized.FrameRateNum != e.cfg.FrameRateNum ||
		normalized.FrameRateDen != e.cfg.FrameRateDen ||
		normalized.RateControl != e.cfg.RateControl ||
		normalized.FrameDrop != e.cfg.FrameDrop
	e.cfg = normalized
	if bitrateBudgetRefresh {
		e.resetEncoderBitrateBudget()
	}
	if normalized.Width != oldWidth || normalized.Height != oldHeight {
		e.reference = encoderReferenceFrame{}
		e.framesSinceIDR = 0
		e.forceIDR = true
	}
	if normalized.OutputFormat != oldOutputFormat {
		e.forceIDR = true
	}
	if qpRefresh {
		e.forceIDR = true
	}
	if update.ForceIDR {
		e.forceIDR = true
	}
	return nil
}

type encoderFrameView struct {
	y        []byte
	cb       []byte
	cr       []byte
	width    int
	height   int
	strideY  int
	strideCb int
	strideCr int
}

type encoderRawNAL struct {
	typ          uint8
	raw          []byte
	keyFrame     bool
	parameterSet bool
}

type encoderReferenceFrame struct {
	valid  bool
	width  int
	height int
	y      []byte
	cb     []byte
	cr     []byte
}

type encoderSliceRange struct {
	firstMB         int
	macroblockCount int
}

type encoderP16x16MotionVector struct {
	x int32
	y int32
}

type encoderP16x16ResidualCandidate struct {
	coeff           int32
	chromaDCCoeffCb int32
	chromaDCCoeffCr int32
	chromaACCoeffCb int32
	chromaACCoeffCr int32
}

type encoderP16x16ResidualPlan struct {
	lumaCoefficients [][]h264.EncoderResidualCoefficient
}

var encoderP16x16ResidualCandidates = [...]encoderP16x16ResidualCandidate{
	{coeff: 4, chromaDCCoeffCb: 2, chromaDCCoeffCr: -2, chromaACCoeffCb: 1, chromaACCoeffCr: -1},
	{coeff: -4, chromaDCCoeffCb: -2, chromaDCCoeffCr: 2, chromaACCoeffCb: -1, chromaACCoeffCr: 1},
	{chromaDCCoeffCb: 2, chromaDCCoeffCr: -2, chromaACCoeffCb: 1, chromaACCoeffCr: -1},
	{chromaDCCoeffCb: -2, chromaDCCoeffCr: 2, chromaACCoeffCb: -1, chromaACCoeffCr: 1},
	{coeff: 4},
	{coeff: -4},
	{coeff: 2},
	{coeff: -2},
	{coeff: 1},
	{coeff: -1},
}

func encoderLateDropStart(cfg EncoderConfig) time.Time {
	if cfg.FrameDrop == EncoderFrameDropLate && cfg.MaxEncodeTimeUS > 0 {
		return time.Now()
	}
	return time.Time{}
}

func encoderLateBudgetMiss(start time.Time, cfg EncoderConfig) bool {
	return !start.IsZero() && time.Since(start) > time.Duration(cfg.MaxEncodeTimeUS)*time.Microsecond
}

func buildEncoderI420IntraPCMIDRNAL(cfg h264.EncoderI420IntraPCMIDRConfig) ([]byte, error) {
	rbsp, err := h264.EncodeI420IntraPCMIDRSliceRBSP(cfg)
	if err != nil {
		return nil, err
	}
	dst, err := encoderNALBuffer(rbsp)
	if err != nil {
		return nil, err
	}
	return h264.AppendNAL(dst, 3, h264.NALIDRSlice, rbsp)
}

func buildEncoderI420PSkipNAL(cfg h264.EncoderI420PSkipConfig) ([]byte, error) {
	rbsp, err := h264.EncodeI420PSkipSliceRBSP(cfg)
	if err != nil {
		return nil, err
	}
	return h264.AppendNAL(nil, 2, h264.NALSlice, rbsp)
}

func buildEncoderI420P16x16NoResidualNAL(cfg h264.EncoderI420P16x16NoResidualConfig) ([]byte, error) {
	rbsp, err := h264.EncodeI420P16x16NoResidualSliceRBSP(cfg)
	if err != nil {
		return nil, err
	}
	dst, err := encoderNALBuffer(rbsp)
	if err != nil {
		return nil, err
	}
	return h264.AppendNAL(dst, 2, h264.NALSlice, rbsp)
}

func buildEncoderI420P16x16ResidualNAL(cfg h264.EncoderI420P16x16ResidualConfig) ([]byte, error) {
	rbsp, err := h264.EncodeI420P16x16ResidualSliceRBSP(cfg)
	if err != nil {
		return nil, err
	}
	dst, err := encoderNALBuffer(rbsp)
	if err != nil {
		return nil, err
	}
	return h264.AppendNAL(dst, 2, h264.NALSlice, rbsp)
}

func buildEncoderI420IntraPCMPNAL(cfg h264.EncoderI420IntraPCMPConfig) ([]byte, error) {
	rbsp, err := h264.EncodeI420IntraPCMPSliceRBSP(cfg)
	if err != nil {
		return nil, err
	}
	dst, err := encoderNALBuffer(rbsp)
	if err != nil {
		return nil, err
	}
	return h264.AppendNAL(dst, 2, h264.NALSlice, rbsp)
}

func (e *Encoder) validateFrame(frame EncoderFrame) error {
	_, err := e.validatedFrameView(frame)
	return err
}

func (e *Encoder) validatedFrameView(frame EncoderFrame) (encoderFrameView, error) {
	if frame.PTS < 0 || uint64(frame.PTS) > uint64(^uint32(0)) {
		return encoderFrameView{}, encoderInvalid("frame PTS must fit RTP timestamp")
	}
	if frame.Duration < 0 || uint64(frame.Duration) > uint64(^uint32(0)) {
		return encoderFrameView{}, encoderInvalid("frame duration must fit RTP timestamp duration")
	}
	if err := validateEncoderColorConfig(frame.Color); err != nil {
		return encoderFrameView{}, err
	}
	width := frame.Width
	if width == 0 {
		width = e.cfg.Width
	}
	height := frame.Height
	if height == 0 {
		height = e.cfg.Height
	}
	if width != e.cfg.Width || height != e.cfg.Height {
		return encoderFrameView{}, encoderInvalid("frame dimensions do not match encoder configuration")
	}
	strideY := frame.StrideY
	if strideY == 0 {
		strideY = e.cfg.StrideY
	}
	strideCb := frame.StrideCb
	if strideCb == 0 {
		strideCb = e.cfg.StrideCb
	}
	strideCr := frame.StrideCr
	if strideCr == 0 {
		strideCr = e.cfg.StrideCr
	}
	if strideY < width {
		return encoderFrameView{}, encoderInvalid("frame luma stride is smaller than width")
	}
	chromaWidth, chromaHeight, err := encoderI420ChromaDimensions(width, height)
	if err != nil {
		return encoderFrameView{}, err
	}
	if strideCb < chromaWidth || strideCr < chromaWidth {
		return encoderFrameView{}, encoderInvalid("frame chroma stride is smaller than chroma width")
	}
	lumaSamples, err := checkedMulInt(strideY, height)
	if err != nil {
		return encoderFrameView{}, encoderInvalid("frame luma plane size overflows")
	}
	cbSamples, err := checkedMulInt(strideCb, chromaHeight)
	if err != nil {
		return encoderFrameView{}, encoderInvalid("frame chroma plane size overflows")
	}
	crSamples, err := checkedMulInt(strideCr, chromaHeight)
	if err != nil {
		return encoderFrameView{}, encoderInvalid("frame chroma plane size overflows")
	}
	if len(frame.Y) < lumaSamples {
		return encoderFrameView{}, encoderInvalid("frame luma plane is too small")
	}
	if len(frame.Cb) < cbSamples || len(frame.Cr) < crSamples {
		return encoderFrameView{}, encoderInvalid("frame chroma plane is too small")
	}
	return encoderFrameView{
		y:        frame.Y,
		cb:       frame.Cb,
		cr:       frame.Cr,
		width:    width,
		height:   height,
		strideY:  strideY,
		strideCb: strideCb,
		strideCr: strideCr,
	}, nil
}

func (e *Encoder) shouldEncodeIDR(view encoderFrameView, frame EncoderFrame) bool {
	if e.forceIDR || frame.ForceIDR || !e.reference.valid {
		return true
	}
	if e.cfg.IDRInterval > 0 && e.framesSinceIDR >= e.cfg.IDRInterval {
		return true
	}
	return false
}

func (e *Encoder) shouldEmitParameterSets(idr bool) bool {
	if !idr {
		return false
	}
	switch e.cfg.SPSPPSMode {
	case EncoderSPSPPSOutOfBand:
		return false
	case EncoderSPSPPSEveryIDR:
		return true
	default:
		return e.cfg.SPSPPSBeforeIDR
	}
}

func (e *Encoder) referenceMatches(view encoderFrameView) bool {
	ref := &e.reference
	if !ref.valid || ref.width != view.width || ref.height != view.height {
		return false
	}
	lumaSize, chromaSize, ok := encoderI420ReferencePlaneSizes(view)
	if !ok {
		return false
	}
	if len(ref.y) != lumaSize {
		return false
	}
	chromaWidth := view.width / 2
	chromaHeight := view.height / 2
	if len(ref.cb) != chromaSize || len(ref.cr) != chromaSize {
		return false
	}
	for y := 0; y < view.height; y++ {
		src := view.y[y*view.strideY : y*view.strideY+view.width]
		dst := ref.y[y*view.width : (y+1)*view.width]
		if !bytes.Equal(src, dst) {
			return false
		}
	}
	for y := 0; y < chromaHeight; y++ {
		srcCb := view.cb[y*view.strideCb : y*view.strideCb+chromaWidth]
		srcCr := view.cr[y*view.strideCr : y*view.strideCr+chromaWidth]
		dstCb := ref.cb[y*chromaWidth : (y+1)*chromaWidth]
		dstCr := ref.cr[y*chromaWidth : (y+1)*chromaWidth]
		if !bytes.Equal(srcCb, dstCb) || !bytes.Equal(srcCr, dstCr) {
			return false
		}
	}
	return true
}

func (e *Encoder) p16x16NoResidualMotion(view encoderFrameView, dst []encoderP16x16MotionVector) ([]encoderP16x16MotionVector, bool) {
	if view.height < 16 ||
		view.height&15 != 0 ||
		view.width < 16 ||
		view.width&15 != 0 {
		return nil, false
	}
	ref := &e.reference
	if !ref.valid || ref.width != view.width || ref.height != view.height {
		return nil, false
	}
	lumaSize, chromaSize, ok := encoderI420ReferencePlaneSizes(view)
	if !ok {
		return nil, false
	}
	if len(ref.y) != lumaSize {
		return nil, false
	}
	if len(ref.cb) != chromaSize || len(ref.cr) != chromaSize {
		return nil, false
	}

	macroblocksPerRow := view.width >> 4
	macroblockRows := view.height >> 4
	macroblockCount, err := checkedMulInt(macroblocksPerRow, macroblockRows)
	if err != nil {
		return nil, false
	}
	if cap(dst) < macroblockCount {
		e.p16MVs = resizeEncoderP16x16MVs(e.p16MVs, macroblockCount)
		dst = e.p16MVs[:0]
	} else {
		dst = dst[:0]
	}
	constantChromaKnown := false
	constantChroma := false

	if dx, dy, ok := encoderI420FindFrameP16x16NoResidualMotion(ref, view, &constantChromaKnown, &constantChroma); ok {
		mv := encoderP16x16MotionVector{x: int32(dx * 4), y: int32(dy * 4)}
		if e.cfg.DeblockMode != EncoderDeblockDisabled && !encoderP16x16MotionVectorChromaAligned(mv) {
			return nil, false
		}
		for mbAddr := 0; mbAddr < macroblockCount; mbAddr++ {
			dst = append(dst, mv)
		}
		return dst, true
	}

	for mbAddr := 0; mbAddr < macroblockCount; mbAddr++ {
		dx, dy, ok := encoderI420FindP16x16NoResidualMotion(ref, view, mbAddr, macroblocksPerRow, &constantChromaKnown, &constantChroma)
		if !ok {
			return nil, false
		}
		dst = append(dst, encoderP16x16MotionVector{x: int32(dx * 4), y: int32(dy * 4)})
	}
	if e.cfg.DeblockMode != EncoderDeblockDisabled && !encoderP16x16MotionVectorsDeblockSafe(dst) {
		return nil, false
	}
	return dst, true
}

func (e *Encoder) p16x16ResidualNALs(view encoderFrameView, sliceRanges []encoderSliceRange) ([]encoderRawNAL, bool, error) {
	if !e.p16x16ResidualAllowed(view, sliceRanges) {
		return nil, false, nil
	}
	if plan, planOK, err := e.p16x16ResidualPlanFromPixelDelta(view, sliceRanges); err != nil {
		return nil, false, err
	} else if planOK {
		nals, err := e.p16x16ResidualPlanNALs(view, sliceRanges, plan)
		if err != nil {
			return nil, false, err
		}
		ok, err := e.p16x16ResidualCandidateMatches(view, nals)
		if err != nil {
			return nil, false, err
		}
		if ok {
			return nals, true, nil
		}
	}
	if !encoderP16x16ResidualBruteForceAllowed(view) {
		return nil, false, nil
	}
	for _, candidate := range encoderP16x16ResidualCandidates {
		nals := make([]encoderRawNAL, 0, len(sliceRanges))
		for _, r := range sliceRanges {
			var lumaCoefficients [][]h264.EncoderResidualCoefficient
			if candidate.coeff == 0 {
				lumaCoefficients = make([][]h264.EncoderResidualCoefficient, r.macroblockCount)
			}
			nal, err := buildEncoderI420P16x16ResidualNAL(h264.EncoderI420P16x16ResidualConfig{
				Width:                      view.width,
				Height:                     view.height,
				FrameNum:                   e.frameNum & 0xff,
				InitialQP:                  e.cfg.InitialQP,
				NextQP:                     e.cfg.InitialQP,
				DisableDeblockingFilterIDC: encoderDeblockingFilterIDC(e.cfg.DeblockMode),
				FirstMBAddr:                uint32(r.firstMB),
				MacroblockCount:            uint32(r.macroblockCount),
				Coeff:                      candidate.coeff,
				LumaCoefficients:           lumaCoefficients,
				ChromaDCCoeffCb:            candidate.chromaDCCoeffCb,
				ChromaDCCoeffCr:            candidate.chromaDCCoeffCr,
				ChromaACCoeffCb:            candidate.chromaACCoeffCb,
				ChromaACCoeffCr:            candidate.chromaACCoeffCr,
				NALLengthSize:              4,
			})
			if err != nil {
				return nil, false, err
			}
			nals = append(nals, encoderRawNAL{typ: uint8(h264.NALSlice), raw: nal})
		}
		ok, err := e.p16x16ResidualCandidateMatches(view, nals)
		if err != nil {
			return nil, false, err
		}
		if ok {
			return nals, true, nil
		}
	}
	return nil, false, nil
}

func (e *Encoder) p16x16ResidualPlanNALs(view encoderFrameView, sliceRanges []encoderSliceRange, plan encoderP16x16ResidualPlan) ([]encoderRawNAL, error) {
	nals := make([]encoderRawNAL, 0, len(sliceRanges))
	for _, r := range sliceRanges {
		end := r.firstMB + r.macroblockCount
		if r.firstMB < 0 || r.macroblockCount <= 0 || end > len(plan.lumaCoefficients) {
			return nil, ErrInvalidData
		}
		nal, err := buildEncoderI420P16x16ResidualNAL(h264.EncoderI420P16x16ResidualConfig{
			Width:                      view.width,
			Height:                     view.height,
			FrameNum:                   e.frameNum & 0xff,
			InitialQP:                  e.cfg.InitialQP,
			NextQP:                     e.cfg.InitialQP,
			DisableDeblockingFilterIDC: encoderDeblockingFilterIDC(e.cfg.DeblockMode),
			FirstMBAddr:                uint32(r.firstMB),
			MacroblockCount:            uint32(r.macroblockCount),
			LumaCoefficients:           plan.lumaCoefficients[r.firstMB:end],
			NALLengthSize:              4,
		})
		if err != nil {
			return nil, err
		}
		nals = append(nals, encoderRawNAL{typ: uint8(h264.NALSlice), raw: nal})
	}
	return nals, nil
}

func (e *Encoder) p16x16ResidualPlanFromPixelDelta(view encoderFrameView, sliceRanges []encoderSliceRange) (encoderP16x16ResidualPlan, bool, error) {
	if !encoderI420ReferenceChromaMatchesView(&e.reference, view) {
		return encoderP16x16ResidualPlan{}, false, nil
	}
	qmul, err := e.p16x16ResidualLumaDCQMul()
	if err != nil {
		return encoderP16x16ResidualPlan{}, false, err
	}
	macroblockCount, err := encoderMacroblockCountChecked(view.width, view.height)
	if err != nil {
		return encoderP16x16ResidualPlan{}, false, err
	}
	lumaCoefficients := make([][]h264.EncoderResidualCoefficient, macroblockCount)
	macroblocksPerRow := view.width >> 4
	for _, r := range sliceRanges {
		end := r.firstMB + r.macroblockCount
		if r.firstMB < 0 || r.macroblockCount <= 0 || end > macroblockCount {
			return encoderP16x16ResidualPlan{}, false, nil
		}
		for mbAddr := r.firstMB; mbAddr < end; mbAddr++ {
			coeffs, ok := encoderP16x16LumaDCCoefficientsFromPixels(&e.reference, view, mbAddr, macroblocksPerRow, qmul)
			if !ok {
				return encoderP16x16ResidualPlan{}, false, nil
			}
			lumaCoefficients[mbAddr] = coeffs
		}
	}
	return encoderP16x16ResidualPlan{lumaCoefficients: lumaCoefficients}, true, nil
}

func (e *Encoder) p16x16ResidualLumaDCQMul() (uint32, error) {
	parameterSetCfg, err := e.parameterSetConfig()
	if err != nil {
		return 0, err
	}
	sets, err := h264.BuildEncoderParameterSets(parameterSetCfg)
	if err != nil {
		return 0, err
	}
	avcc, err := h264.DecodeAVCDecoderConfigurationRecord(sets.AVCDecoderConfigurationRecord)
	if err != nil {
		return 0, err
	}
	for _, pps := range avcc.PPS {
		if pps != nil {
			qmul := pps.Dequant4Buffer[3][e.cfg.InitialQP][0]
			if qmul == 0 {
				return 0, ErrInvalidData
			}
			return qmul, nil
		}
	}
	return 0, ErrInvalidData
}

func encoderI420ReferenceChromaMatchesView(ref *encoderReferenceFrame, view encoderFrameView) bool {
	if ref == nil || !ref.valid || ref.width != view.width || ref.height != view.height {
		return false
	}
	_, chromaSize, ok := encoderI420ReferencePlaneSizes(view)
	if !ok || len(ref.cb) != chromaSize || len(ref.cr) != chromaSize {
		return false
	}
	chromaWidth := view.width >> 1
	chromaHeight := view.height >> 1
	for y := 0; y < chromaHeight; y++ {
		srcCb := view.cb[y*view.strideCb : y*view.strideCb+chromaWidth]
		srcCr := view.cr[y*view.strideCr : y*view.strideCr+chromaWidth]
		dstCb := ref.cb[y*chromaWidth : (y+1)*chromaWidth]
		dstCr := ref.cr[y*chromaWidth : (y+1)*chromaWidth]
		if !bytes.Equal(srcCb, dstCb) || !bytes.Equal(srcCr, dstCr) {
			return false
		}
	}
	return true
}

func encoderP16x16LumaDCCoefficientsFromPixels(ref *encoderReferenceFrame, view encoderFrameView, mbAddr int, macroblocksPerRow int, qmul uint32) ([]h264.EncoderResidualCoefficient, bool) {
	if ref == nil || macroblocksPerRow <= 0 || mbAddr < 0 || qmul == 0 {
		return nil, false
	}
	mbX := mbAddr % macroblocksPerRow
	mbY := mbAddr / macroblocksPerRow
	baseX := mbX << 4
	baseY := mbY << 4
	if baseX+16 > view.width || baseY+16 > view.height || len(ref.y) != view.width*view.height {
		return nil, false
	}

	var coeffs []h264.EncoderResidualCoefficient
	for blockIndex := 0; blockIndex < 16; blockIndex++ {
		i8x8 := blockIndex >> 2
		i4x4 := blockIndex & 3
		blockX := ((i8x8 & 1) << 3) + ((i4x4 & 1) << 2)
		blockY := ((i8x8 >> 1) << 3) + ((i4x4 >> 1) << 2)
		delta := 0
		for y := 0; y < 4; y++ {
			viewRow := view.y[(baseY+blockY+y)*view.strideY+baseX+blockX : (baseY+blockY+y)*view.strideY+baseX+blockX+4]
			refRow := ref.y[(baseY+blockY+y)*view.width+baseX+blockX : (baseY+blockY+y)*view.width+baseX+blockX+4]
			for x := 0; x < 4; x++ {
				pixelDelta := int(viewRow[x]) - int(refRow[x])
				if x == 0 && y == 0 {
					delta = pixelDelta
					continue
				}
				if pixelDelta != delta {
					return nil, false
				}
			}
		}
		if delta == 0 {
			continue
		}
		level, ok := encoderP16x16ResidualLumaDCLevelForPixelDelta(delta, qmul)
		if !ok {
			return nil, false
		}
		coeffs = append(coeffs, h264.EncoderResidualCoefficient{Pos: blockIndex * 16, Value: level})
	}
	if len(coeffs) == 0 {
		return nil, false
	}
	return coeffs, true
}

func encoderP16x16ResidualLumaDCLevelForPixelDelta(delta int, qmul uint32) (int32, bool) {
	const maxDerivedResidualLevel = 512
	for absLevel := int32(1); absLevel <= maxDerivedResidualLevel; absLevel++ {
		if encoderP16x16ResidualPixelDeltaForDCLevel(absLevel, qmul) == delta {
			return absLevel, true
		}
		negLevel := -absLevel
		if encoderP16x16ResidualPixelDeltaForDCLevel(negLevel, qmul) == delta {
			return negLevel, true
		}
	}
	return 0, false
}

func encoderP16x16ResidualPixelDeltaForDCLevel(level int32, qmul uint32) int {
	dequantized := int32(uint32(level)*qmul+32) >> 6
	return (int(dequantized) + 32) >> 6
}

func (e *Encoder) p16x16ResidualAllowed(view encoderFrameView, sliceRanges []encoderSliceRange) bool {
	if e.cfg.DeblockMode != EncoderDeblockDisabled ||
		e.cfg.RateControl != EncoderRateControlConstantQP ||
		e.cfg.Crop != (EncoderCrop{}) ||
		view.width < 16 ||
		view.height < 16 ||
		view.width&15 != 0 ||
		view.height&15 != 0 ||
		len(sliceRanges) == 0 ||
		!e.reference.valid ||
		e.reference.width != view.width ||
		e.reference.height != view.height {
		return false
	}
	lumaSize, chromaSize, ok := encoderI420ReferencePlaneSizes(view)
	if !ok || len(e.reference.y) != lumaSize || len(e.reference.cb) != chromaSize || len(e.reference.cr) != chromaSize {
		return false
	}
	return true
}

func encoderP16x16ResidualBruteForceAllowed(view encoderFrameView) bool {
	macroblocksPerRow := view.width >> 4
	macroblockRows := view.height >> 4
	macroblockCount, err := checkedMulInt(macroblocksPerRow, macroblockRows)
	if err != nil {
		return false
	}
	const maxBruteForceResidualCandidateMacroblocks = 4
	return macroblockCount <= maxBruteForceResidualCandidateMacroblocks
}

func (e *Encoder) p16x16ResidualCandidateMatches(view encoderFrameView, nals []encoderRawNAL) (bool, error) {
	if len(nals) == 0 {
		return false, nil
	}
	parameterSetCfg, err := e.parameterSetConfig()
	if err != nil {
		return false, err
	}
	sets, err := h264.BuildEncoderParameterSetNALs(parameterSetCfg)
	if err != nil {
		return false, err
	}
	ref := &e.reference
	chromaWidth := ref.width >> 1
	refIDR, err := buildEncoderI420IntraPCMIDRNAL(h264.EncoderI420IntraPCMIDRConfig{
		Width:                      ref.width,
		Height:                     ref.height,
		StrideY:                    ref.width,
		StrideCb:                   chromaWidth,
		StrideCr:                   chromaWidth,
		Y:                          ref.y,
		Cb:                         ref.cb,
		Cr:                         ref.cr,
		FrameNum:                   (e.frameNum - 1) & 0xff,
		IDRPicID:                   0,
		InitialQP:                  e.cfg.InitialQP,
		DisableDeblockingFilterIDC: encoderDeblockingFilterIDC(e.cfg.DeblockMode),
		NALLengthSize:              4,
	})
	if err != nil {
		return false, err
	}

	stream := appendEncoderAnnexBRawNAL(nil, sets.SPS)
	stream = appendEncoderAnnexBRawNAL(stream, sets.PPS)
	stream = appendEncoderAnnexBRawNAL(stream, refIDR)
	for _, nal := range nals {
		stream = appendEncoderAnnexBRawNAL(stream, nal.raw)
	}
	frames, err := NewDecoder().DecodeFrames(stream)
	if err != nil {
		return false, err
	}
	if len(frames) != 2 {
		return false, nil
	}
	raw, err := frames[1].AppendRawYUV(nil)
	if err != nil {
		return false, err
	}
	return encoderI420RawMatchesView(raw, view), nil
}

func appendEncoderAnnexBRawNAL(dst []byte, raw []byte) []byte {
	if len(raw) == 0 {
		return dst
	}
	dst = append(dst, 0, 0, 0, 1)
	return append(dst, raw...)
}

func encoderI420RawMatchesView(raw []byte, view encoderFrameView) bool {
	lumaSize, chromaSize, ok := encoderI420ReferencePlaneSizes(view)
	if !ok || len(raw) != lumaSize+2*chromaSize {
		return false
	}
	pos := 0
	for y := 0; y < view.height; y++ {
		row := view.y[y*view.strideY : y*view.strideY+view.width]
		if !bytes.Equal(raw[pos:pos+view.width], row) {
			return false
		}
		pos += view.width
	}
	chromaWidth := view.width >> 1
	chromaHeight := view.height >> 1
	for y := 0; y < chromaHeight; y++ {
		row := view.cb[y*view.strideCb : y*view.strideCb+chromaWidth]
		if !bytes.Equal(raw[pos:pos+chromaWidth], row) {
			return false
		}
		pos += chromaWidth
	}
	for y := 0; y < chromaHeight; y++ {
		row := view.cr[y*view.strideCr : y*view.strideCr+chromaWidth]
		if !bytes.Equal(raw[pos:pos+chromaWidth], row) {
			return false
		}
		pos += chromaWidth
	}
	return pos == len(raw)
}

func encoderP16x16MotionVectorsDeblockSafe(mvs []encoderP16x16MotionVector) bool {
	if len(mvs) <= 1 {
		return len(mvs) == 0 || encoderP16x16MotionVectorChromaAligned(mvs[0])
	}
	first := mvs[0]
	if !encoderP16x16MotionVectorChromaAligned(first) {
		return false
	}
	for _, mv := range mvs[1:] {
		if mv != first {
			return false
		}
	}
	return true
}

func encoderP16x16MotionVectorChromaAligned(mv encoderP16x16MotionVector) bool {
	return mv.x%8 == 0 && mv.y%8 == 0
}

func encoderI420FindFrameP16x16NoResidualMotion(ref *encoderReferenceFrame, view encoderFrameView, constantChromaKnown *bool, constantChroma *bool) (int, int, bool) {
	primaryCandidates := [...]struct {
		dx int
		dy int
	}{
		{dx: 2, dy: 0},
		{dx: -2, dy: 0},
		{dx: 0, dy: 2},
		{dx: 0, dy: -2},
	}
	for _, candidate := range primaryCandidates {
		if encoderMotionNeedsConstantChroma(candidate.dx, candidate.dy) && !*constantChromaKnown {
			*constantChroma = encoderI420ChromaPlanesAreConstant(ref, view, view.width/2, view.height/2)
			*constantChromaKnown = true
		}
		if encoderI420MatchesIntegerMotion(ref, view, candidate.dx, candidate.dy, *constantChroma) {
			return candidate.dx, candidate.dy, true
		}
	}
	const maxExactMotion = 8
	for radius := 1; radius <= maxExactMotion; radius++ {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				if absInt(dx) != radius && absInt(dy) != radius {
					continue
				}
				if (dx == 2 && dy == 0) ||
					(dx == -2 && dy == 0) ||
					(dx == 0 && dy == 2) ||
					(dx == 0 && dy == -2) {
					continue
				}
				if encoderMotionNeedsConstantChroma(dx, dy) && !*constantChromaKnown {
					*constantChroma = encoderI420ChromaPlanesAreConstant(ref, view, view.width/2, view.height/2)
					*constantChromaKnown = true
				}
				if encoderI420MatchesIntegerMotion(ref, view, dx, dy, *constantChroma) {
					return dx, dy, true
				}
			}
		}
	}
	return 0, 0, false
}

func encoderI420FindP16x16NoResidualMotion(ref *encoderReferenceFrame, view encoderFrameView, mbAddr int, macroblocksPerRow int, constantChromaKnown *bool, constantChroma *bool) (int, int, bool) {
	primaryCandidates := [...]struct {
		dx int
		dy int
	}{
		{dx: 0, dy: 0},
		{dx: 2, dy: 0},
		{dx: -2, dy: 0},
		{dx: 0, dy: 2},
		{dx: 0, dy: -2},
	}
	for _, candidate := range primaryCandidates {
		if encoderMotionNeedsConstantChroma(candidate.dx, candidate.dy) && !*constantChromaKnown {
			*constantChroma = encoderI420ChromaPlanesAreConstant(ref, view, view.width/2, view.height/2)
			*constantChromaKnown = true
		}
		if encoderI420MacroblockMatchesIntegerMotion(ref, view, mbAddr, macroblocksPerRow, candidate.dx, candidate.dy, *constantChroma) {
			return candidate.dx, candidate.dy, true
		}
	}
	const maxExactMotion = 8
	for radius := 1; radius <= maxExactMotion; radius++ {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				if absInt(dx) != radius && absInt(dy) != radius {
					continue
				}
				if (dx == 2 && dy == 0) ||
					(dx == -2 && dy == 0) ||
					(dx == 0 && dy == 2) ||
					(dx == 0 && dy == -2) {
					continue
				}
				if encoderMotionNeedsConstantChroma(dx, dy) && !*constantChromaKnown {
					*constantChroma = encoderI420ChromaPlanesAreConstant(ref, view, view.width/2, view.height/2)
					*constantChromaKnown = true
				}
				if encoderI420MacroblockMatchesIntegerMotion(ref, view, mbAddr, macroblocksPerRow, dx, dy, *constantChroma) {
					return dx, dy, true
				}
			}
		}
	}
	return 0, 0, false
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func appendEncoderP16x16NoResidualMVDs(dst []h264.EncoderMotionVectorDelta, mvs []encoderP16x16MotionVector, firstMB int, macroblockCount int, macroblocksPerRow int) []h264.EncoderMotionVectorDelta {
	if cap(dst) < macroblockCount {
		dst = make([]h264.EncoderMotionVectorDelta, macroblockCount)
	} else {
		dst = dst[:macroblockCount]
	}
	if macroblockCount == 0 {
		return dst
	}
	for i := 0; i < macroblockCount; i++ {
		mbAddr := firstMB + i
		mv := mvs[mbAddr]
		pred := encoderP16x16NoResidualMVPredictor(mvs, mbAddr, firstMB, macroblocksPerRow)
		dst[i] = h264.EncoderMotionVectorDelta{X: mv.x - pred.x, Y: mv.y - pred.y}
	}
	return dst
}

func encoderP16x16NoResidualMVPredictor(mvs []encoderP16x16MotionVector, mbAddr int, firstMB int, macroblocksPerRow int) encoderP16x16MotionVector {
	var left, top, diagonal encoderP16x16MotionVector
	x := mbAddr % macroblocksPerRow
	y := mbAddr / macroblocksPerRow
	leftAvailable := false
	if x > 0 && mbAddr-1 >= firstMB {
		leftAvailable = true
		left = mvs[mbAddr-1]
	}
	topAvailable := false
	topAddr := mbAddr - macroblocksPerRow
	if y > 0 && topAddr >= firstMB {
		topAvailable = true
		top = mvs[topAddr]
	}
	diagonalAvailable := false
	if y > 0 {
		topRight := topAddr + 1
		if x < macroblocksPerRow-1 && topRight >= firstMB {
			diagonalAvailable = true
			diagonal = mvs[topRight]
		} else {
			topLeft := topAddr - 1
			if x > 0 && topLeft >= firstMB {
				diagonalAvailable = true
				diagonal = mvs[topLeft]
			}
		}
	}

	matchCount := 0
	if leftAvailable {
		matchCount++
	}
	if topAvailable {
		matchCount++
	}
	if diagonalAvailable {
		matchCount++
	}
	switch matchCount {
	case 0:
		return encoderP16x16MotionVector{}
	case 1:
		if leftAvailable {
			return left
		}
		if topAvailable {
			return top
		}
		return diagonal
	default:
		return encoderP16x16MotionVector{
			x: encoderMidPredInt32(left.x, top.x, diagonal.x),
			y: encoderMidPredInt32(left.y, top.y, diagonal.y),
		}
	}
}

func encoderMidPredInt32(a int32, b int32, c int32) int32 {
	if a > b {
		if c > b {
			if c > a {
				b = a
			} else {
				b = c
			}
		}
	} else if b > c {
		if c > a {
			b = c
		} else {
			b = a
		}
	}
	return b
}

func encoderMotionNeedsConstantChroma(dx int, dy int) bool {
	return dx%2 != 0 || dy%2 != 0
}

func encoderI420MatchesIntegerMotion(ref *encoderReferenceFrame, view encoderFrameView, dx int, dy int, constantChroma bool) bool {
	if encoderMotionNeedsConstantChroma(dx, dy) {
		return constantChroma &&
			encoderPlaneMatchesIntegerMotion(view.y, view.strideY, view.width, view.height, ref.y, view.width, dx, dy)
	}
	if !encoderPlaneMatchesIntegerMotion(view.y, view.strideY, view.width, view.height, ref.y, view.width, dx, dy) {
		return false
	}
	chromaWidth := view.width / 2
	chromaHeight := view.height / 2
	chromaDX := dx / 2
	chromaDY := dy / 2
	return encoderPlaneMatchesIntegerMotion(view.cb, view.strideCb, chromaWidth, chromaHeight, ref.cb, chromaWidth, chromaDX, chromaDY) &&
		encoderPlaneMatchesIntegerMotion(view.cr, view.strideCr, chromaWidth, chromaHeight, ref.cr, chromaWidth, chromaDX, chromaDY)
}

func encoderI420MacroblockMatchesIntegerMotion(ref *encoderReferenceFrame, view encoderFrameView, mbAddr int, macroblocksPerRow int, dx int, dy int, constantChroma bool) bool {
	mbX := (mbAddr % macroblocksPerRow) << 4
	mbY := (mbAddr / macroblocksPerRow) << 4
	if !encoderPlaneBlockMatchesIntegerMotion(view.y, view.strideY, view.width, view.height, ref.y, view.width, mbX, mbY, 16, 16, dx, dy) {
		return false
	}
	if encoderMotionNeedsConstantChroma(dx, dy) {
		return constantChroma
	}
	chromaX := mbX >> 1
	chromaY := mbY >> 1
	chromaWidth := view.width >> 1
	chromaHeight := view.height >> 1
	chromaDX := dx / 2
	chromaDY := dy / 2
	return encoderPlaneBlockMatchesIntegerMotion(view.cb, view.strideCb, chromaWidth, chromaHeight, ref.cb, chromaWidth, chromaX, chromaY, 8, 8, chromaDX, chromaDY) &&
		encoderPlaneBlockMatchesIntegerMotion(view.cr, view.strideCr, chromaWidth, chromaHeight, ref.cr, chromaWidth, chromaX, chromaY, 8, 8, chromaDX, chromaDY)
}

func encoderI420ChromaPlanesAreConstant(ref *encoderReferenceFrame, view encoderFrameView, chromaWidth int, chromaHeight int) bool {
	return encoderPlaneAllValue(ref.cb, chromaWidth, chromaWidth, chromaHeight, ref.cb[0]) &&
		encoderPlaneAllValue(view.cb, view.strideCb, chromaWidth, chromaHeight, ref.cb[0]) &&
		encoderPlaneAllValue(ref.cr, chromaWidth, chromaWidth, chromaHeight, ref.cr[0]) &&
		encoderPlaneAllValue(view.cr, view.strideCr, chromaWidth, chromaHeight, ref.cr[0])
}

func encoderPlaneAllValue(plane []byte, stride int, width int, height int, value byte) bool {
	for y := 0; y < height; y++ {
		row := plane[y*stride : y*stride+width]
		for _, got := range row {
			if got != value {
				return false
			}
		}
	}
	return true
}

func encoderPlaneMatchesIntegerMotion(cur []byte, curStride int, width int, height int, ref []byte, refStride int, dx int, dy int) bool {
	for y := 0; y < height; y++ {
		curRow := cur[y*curStride : y*curStride+width]
		refY := clampEncoderReferenceCoord(y+dy, height)
		for x := 0; x < width; x++ {
			refX := clampEncoderReferenceCoord(x+dx, width)
			if curRow[x] != ref[refY*refStride+refX] {
				return false
			}
		}
	}
	return true
}

func encoderPlaneBlockMatchesIntegerMotion(cur []byte, curStride int, width int, height int, ref []byte, refStride int, left int, top int, blockWidth int, blockHeight int, dx int, dy int) bool {
	for y := 0; y < blockHeight; y++ {
		curRow := cur[(top+y)*curStride+left : (top+y)*curStride+left+blockWidth]
		refY := clampEncoderReferenceCoord(top+y+dy, height)
		for x := 0; x < blockWidth; x++ {
			refX := clampEncoderReferenceCoord(left+x+dx, width)
			if curRow[x] != ref[refY*refStride+refX] {
				return false
			}
		}
	}
	return true
}

func clampEncoderReferenceCoord(v int, limit int) int {
	if v < 0 {
		return 0
	}
	if v >= limit {
		return limit - 1
	}
	return v
}

func (e *Encoder) storeReference(view encoderFrameView) {
	lumaSize, chromaSize, ok := encoderI420ReferencePlaneSizes(view)
	if !ok {
		e.reference = encoderReferenceFrame{}
		return
	}
	chromaWidth := view.width / 2
	chromaHeight := view.height / 2
	ref := &e.reference
	ref.width = view.width
	ref.height = view.height
	ref.y = resizeEncoderReferencePlane(ref.y, lumaSize)
	ref.cb = resizeEncoderReferencePlane(ref.cb, chromaSize)
	ref.cr = resizeEncoderReferencePlane(ref.cr, chromaSize)
	for y := 0; y < view.height; y++ {
		copy(ref.y[y*view.width:(y+1)*view.width], view.y[y*view.strideY:y*view.strideY+view.width])
	}
	for y := 0; y < chromaHeight; y++ {
		copy(ref.cb[y*chromaWidth:(y+1)*chromaWidth], view.cb[y*view.strideCb:y*view.strideCb+chromaWidth])
		copy(ref.cr[y*chromaWidth:(y+1)*chromaWidth], view.cr[y*view.strideCr:y*view.strideCr+chromaWidth])
	}
	ref.valid = true
}

func encoderI420ReferencePlaneSizes(view encoderFrameView) (int, int, bool) {
	if view.width <= 0 || view.height <= 0 {
		return 0, 0, false
	}
	lumaSize, err := checkedMulInt(view.width, view.height)
	if err != nil {
		return 0, 0, false
	}
	chromaWidth := view.width / 2
	chromaHeight := view.height / 2
	chromaSize, err := checkedMulInt(chromaWidth, chromaHeight)
	if err != nil {
		return 0, 0, false
	}
	return lumaSize, chromaSize, true
}

func resizeEncoderReferencePlane(buf []byte, size int) []byte {
	if cap(buf) < size {
		return make([]byte, size)
	}
	return buf[:size]
}

func encoderNALBuffer(rbsp []byte) ([]byte, error) {
	n, err := checkedAddInt(1, len(rbsp))
	if err != nil {
		return nil, err
	}
	n, err = checkedAddInt(n, len(rbsp)/2)
	if err != nil {
		return nil, err
	}
	return make([]byte, 0, n), nil
}

func resizeEncoderP16x16MVs(buf []encoderP16x16MotionVector, size int) []encoderP16x16MotionVector {
	if cap(buf) < size {
		return make([]encoderP16x16MotionVector, size)
	}
	return buf[:size]
}

func resizeEncoderP16x16MVDs(buf []h264.EncoderMotionVectorDelta, size int) []h264.EncoderMotionVectorDelta {
	if cap(buf) < size {
		return make([]h264.EncoderMotionVectorDelta, size)
	}
	return buf[:size]
}

func appendEncoderSliceRanges(dst []encoderSliceRange, width int, height int, sliceCount int) []encoderSliceRange {
	dst = dst[:0]
	total := encoderMacroblockCount(width, height)
	if sliceCount <= 0 {
		sliceCount = 1
	}
	if sliceCount > total {
		sliceCount = total
	}
	base := total / sliceCount
	extra := total % sliceCount
	first := 0
	for i := 0; i < sliceCount; i++ {
		count := base
		if i < extra {
			count++
		}
		dst = append(dst, encoderSliceRange{firstMB: first, macroblockCount: count})
		first += count
	}
	return dst
}

func encoderMacroblockCount(width int, height int) int {
	count, err := encoderMacroblockCountChecked(width, height)
	if err != nil {
		return 0
	}
	return count
}

func encoderMacroblockCountChecked(width int, height int) (int, error) {
	if width <= 0 || height <= 0 {
		return 0, encoderInvalid("width and height must be positive")
	}
	mbWidthInput, err := checkedAddInt(width, 15)
	if err != nil {
		return 0, encoderInvalid("coded width is too large")
	}
	mbHeightInput, err := checkedAddInt(height, 15)
	if err != nil {
		return 0, encoderInvalid("coded height is too large")
	}
	count, err := checkedMulInt(mbWidthInput>>4, mbHeightInput>>4)
	if err != nil {
		return 0, encoderInvalid("coded macroblock count overflows")
	}
	return count, nil
}

func encoderI420ChromaDimensions(width int, height int) (int, int, error) {
	chromaWidthInput, err := checkedAddInt(width, 1)
	if err != nil {
		return 0, 0, encoderInvalid("I420 chroma width overflows")
	}
	chromaHeightInput, err := checkedAddInt(height, 1)
	if err != nil {
		return 0, 0, encoderInvalid("I420 chroma height overflows")
	}
	return chromaWidthInput / 2, chromaHeightInput / 2, nil
}

func defaultEncoderI420Strides(width int) (int, int, int) {
	chromaWidthInput, err := checkedAddInt(width, 1)
	if err != nil {
		return width, 0, 0
	}
	chromaStride := chromaWidthInput / 2
	return width, chromaStride, chromaStride
}

func validateEncoderPlaneGeometry(width int, height int, strideY int, strideCb int, strideCr int) error {
	_, chromaHeight, err := encoderI420ChromaDimensions(width, height)
	if err != nil {
		return err
	}
	if _, err := checkedMulInt(strideY, height); err != nil {
		return encoderInvalid("configured luma plane size overflows")
	}
	if _, err := checkedMulInt(strideCb, chromaHeight); err != nil {
		return encoderInvalid("configured chroma plane size overflows")
	}
	if _, err := checkedMulInt(strideCr, chromaHeight); err != nil {
		return encoderInvalid("configured chroma plane size overflows")
	}
	return nil
}

func appendEncoderAccessUnit(dst []byte, format EncoderOutputFormat, nals []encoderRawNAL) ([]byte, []EncoderNALUnit, error) {
	if len(nals) > maxEncoderRawNALListLen {
		return dst, nil, encoderInvalid("encoder NAL count overflows")
	}
	outputSize, err := encoderAccessUnitOutputSize(format, nals)
	if err != nil {
		return dst, nil, err
	}
	if _, err := checkedAddInt(len(dst), outputSize); err != nil {
		return dst, nil, encoderInvalid("encoder access-unit destination size overflows")
	}
	units := make([]EncoderNALUnit, 0, len(nals))
	for _, nal := range nals {
		if len(nal.raw) == 0 {
			return dst, nil, encoderInvalid("empty encoder NAL")
		}
		switch format {
		case EncoderOutputAVC:
			if uint64(len(nal.raw)) > uint64(^uint32(0)) {
				return dst, nil, encoderInvalid("encoder NAL is too large for AVC output")
			}
			n := len(nal.raw)
			dst = append(dst, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
			offset := len(dst)
			dst = append(dst, nal.raw...)
			units = append(units, EncoderNALUnit{
				Type:         nal.typ,
				Offset:       offset,
				Size:         len(nal.raw),
				KeyFrame:     nal.keyFrame,
				ParameterSet: nal.parameterSet,
			})
		case EncoderOutputAnnexB, EncoderOutputRTP:
			dst = append(dst, 0, 0, 0, 1)
			offset := len(dst)
			dst = append(dst, nal.raw...)
			units = append(units, EncoderNALUnit{
				Type:         nal.typ,
				Offset:       offset,
				Size:         len(nal.raw),
				KeyFrame:     nal.keyFrame,
				ParameterSet: nal.parameterSet,
			})
		default:
			return dst, nil, encoderInvalid("unknown encoder output format")
		}
	}
	return dst, units, nil
}

func encoderAccessUnitOutputSize(format EncoderOutputFormat, nals []encoderRawNAL) (int, error) {
	if len(nals) > maxEncoderRawNALListLen {
		return 0, encoderInvalid("encoder NAL count overflows")
	}
	var size int
	for _, nal := range nals {
		if len(nal.raw) == 0 {
			return 0, encoderInvalid("empty encoder NAL")
		}
		switch format {
		case EncoderOutputAVC:
			if uint64(len(nal.raw)) > uint64(^uint32(0)) {
				return 0, encoderInvalid("encoder NAL is too large for AVC output")
			}
			next, err := checkedAddInt(size, 4)
			if err != nil {
				return 0, encoderInvalid("encoder access-unit size overflows")
			}
			size, err = checkedAddInt(next, len(nal.raw))
			if err != nil {
				return 0, encoderInvalid("encoder access-unit size overflows")
			}
		case EncoderOutputAnnexB, EncoderOutputRTP:
			next, err := checkedAddInt(size, 4)
			if err != nil {
				return 0, encoderInvalid("encoder access-unit size overflows")
			}
			size, err = checkedAddInt(next, len(nal.raw))
			if err != nil {
				return 0, encoderInvalid("encoder access-unit size overflows")
			}
		default:
			return 0, encoderInvalid("unknown encoder output format")
		}
	}
	return size, nil
}

type encoderOutputBudget uint8

const (
	encoderOutputBudgetNone encoderOutputBudget = iota
	encoderOutputBudgetSlice
	encoderOutputBudgetFrame
)

func validateEncoderOutputBudgets(cfg EncoderConfig, nals []encoderRawNAL, outputSize int) error {
	_, err := encoderOutputBudgetMiss(cfg, nals, outputSize)
	return err
}

func encoderOutputBudgetMiss(cfg EncoderConfig, nals []encoderRawNAL, outputSize int) (encoderOutputBudget, error) {
	if cfg.SliceMaxBytes > 0 {
		for _, nal := range nals {
			if encoderRawNALIsVCL(nal) && len(nal.raw) > cfg.SliceMaxBytes {
				return encoderOutputBudgetSlice, encoderInvalid("encoded slice exceeds slice byte target")
			}
		}
	}
	if cfg.MaxFrameSize > 0 && outputSize > cfg.MaxFrameSize {
		return encoderOutputBudgetFrame, encoderInvalid("encoded access unit exceeds max frame size")
	}
	return encoderOutputBudgetNone, nil
}

func (e *Encoder) encoderBitrateBudgetMiss(outputSize int) bool {
	if e == nil || !encoderBitrateBudgetEnabled(e.cfg) {
		return false
	}
	e.ensureEncoderBitrateBudget()
	return outputSize > e.bitrateCreditBytes
}

func (e *Encoder) advanceEncoderBitrateBudget(outputSize int) {
	if e == nil || !encoderBitrateBudgetEnabled(e.cfg) {
		if e != nil {
			e.resetEncoderBitrateBudget()
		}
		return
	}
	e.ensureEncoderBitrateBudget()
	if outputSize >= e.bitrateCreditBytes {
		e.bitrateCreditBytes = 0
	} else {
		e.bitrateCreditBytes -= outputSize
	}
	e.bitrateCreditBytes += encoderBitrateFrameBudgetBytes(e.cfg)
	if capBytes := encoderVBVBufferBudgetBytes(e.cfg); capBytes > 0 && e.bitrateCreditBytes > capBytes {
		e.bitrateCreditBytes = capBytes
	}
}

func (e *Encoder) ensureEncoderBitrateBudget() {
	if e == nil || e.bitrateCreditInit {
		return
	}
	e.bitrateCreditBytes = encoderVBVBufferBudgetBytes(e.cfg)
	if e.bitrateCreditBytes == 0 {
		e.bitrateCreditBytes = encoderBitrateFrameBudgetBytes(e.cfg)
	}
	e.bitrateCreditInit = true
}

func (e *Encoder) resetEncoderBitrateBudget() {
	if e == nil {
		return
	}
	e.bitrateCreditBytes = 0
	e.bitrateCreditInit = false
}

func encoderBitrateBudgetEnabled(cfg EncoderConfig) bool {
	return cfg.FrameDrop == EncoderFrameDropToBitrate && cfg.RateControl != EncoderRateControlConstantQP
}

func encoderBitrateFrameBudgetBytes(cfg EncoderConfig) int {
	bytes, err := encoderBitrateFrameBudgetBytesChecked(cfg)
	if err != nil {
		return 0
	}
	return bytes
}

func encoderBitrateFrameBudgetBytesChecked(cfg EncoderConfig) (int, error) {
	if cfg.MaxBitrate <= 0 || cfg.FrameRateNum <= 0 || cfg.FrameRateDen <= 0 {
		return 0, nil
	}
	bitsNumerator, err := checkedMulUint64(uint64(cfg.MaxBitrate), uint64(cfg.FrameRateDen))
	if err != nil {
		return 0, encoderInvalid("bitrate frame budget overflows")
	}
	divisor := uint64(cfg.FrameRateNum)
	bitsPerFrame := bitsNumerator / divisor
	if bitsNumerator%divisor != 0 {
		bitsPerFrame++
	}
	bytesPerFrame := bitsPerFrame / 8
	if bitsPerFrame%8 != 0 {
		bytesPerFrame++
	}
	maxInt := uint64(int(^uint(0) >> 1))
	if bytesPerFrame > maxInt {
		return int(maxInt), nil
	}
	return int(bytesPerFrame), nil
}

func checkedMulUint64(a uint64, b uint64) (uint64, error) {
	if a != 0 && b > ^uint64(0)/a {
		return 0, ErrInvalidData
	}
	return a * b, nil
}

func encoderVBVBufferBudgetBytes(cfg EncoderConfig) int {
	if cfg.VBVBufferSize <= 0 {
		return 0
	}
	bytes := (uint64(cfg.VBVBufferSize) + 7) / 8
	maxInt := uint64(int(^uint(0) >> 1))
	if bytes > maxInt {
		return int(maxInt)
	}
	return int(bytes)
}

func encoderRawNALIsVCL(nal encoderRawNAL) bool {
	return nal.typ == uint8(h264.NALSlice) || nal.typ == uint8(h264.NALIDRSlice)
}

func packetizeEncoderRTPSingleNAL(nals []encoderRawNAL, maxPayloadSize int, timestamp uint32) ([]EncoderRTPPacket, error) {
	if maxPayloadSize < 1 {
		return nil, encoderInvalid("RTP max payload size must fit a NAL header")
	}
	payloadSize, err := encoderRawNALPayloadStorageSize(nals)
	if err != nil {
		return nil, err
	}
	headerSize, err := checkedMulInt(12, len(nals))
	if err != nil {
		return nil, encoderInvalid("RTP packet storage size overflows")
	}
	storageSize, err := checkedAddInt(payloadSize, headerSize)
	if err != nil {
		return nil, encoderInvalid("RTP packet storage size overflows")
	}
	packets := make([]EncoderRTPPacket, 0, len(nals))
	data := make([]byte, 0, storageSize)
	for _, nal := range nals {
		if len(nal.raw) == 0 {
			return nil, encoderInvalid("empty encoder NAL")
		}
		if len(nal.raw) > maxPayloadSize {
			return nil, encoderInvalid("encoder NAL exceeds RTP packetization-mode 0 payload size")
		}
		packetStart := len(data)
		data = appendEncoderRTPHeaderPadding(data)
		payloadStart := len(data)
		data = append(data, nal.raw...)
		packets = appendEncoderRTPPacketFromData(packets, data, packetStart, payloadStart, timestamp)
	}
	if len(packets) != 0 {
		packets[len(packets)-1].Marker = true
	}
	return packets, nil
}

func packetizeEncoderRTPMode1(nals []encoderRawNAL, maxPayloadSize int, timestamp uint32, stapa bool) ([]EncoderRTPPacket, error) {
	if maxPayloadSize < 3 {
		return nil, encoderInvalid("RTP max payload size must leave room for FU-A headers")
	}
	packetCap, payloadCap, err := encoderRTPMode1StoragePlan(nals, maxPayloadSize, stapa)
	if err != nil {
		return nil, err
	}
	headerSize, err := checkedMulInt(12, packetCap)
	if err != nil {
		return nil, encoderInvalid("RTP packet storage size overflows")
	}
	storageSize, err := checkedAddInt(payloadCap, headerSize)
	if err != nil {
		return nil, encoderInvalid("RTP packet storage size overflows")
	}
	packets := make([]EncoderRTPPacket, 0, packetCap)
	data := make([]byte, 0, storageSize)
	for i := 0; i < len(nals); {
		if stapa && nals[i].parameterSet {
			packetStart := len(data)
			data = appendEncoderRTPHeaderPadding(data)
			payloadStart := len(data)
			var count int
			data, count, err = appendEncoderSTAPA(data, nals[i:], maxPayloadSize)
			if err != nil {
				return nil, err
			}
			if count >= 2 {
				packets = appendEncoderRTPPacketFromData(packets, data, packetStart, payloadStart, timestamp)
				i += count
				continue
			}
			data = data[:packetStart]
		}
		nal := nals[i]
		if len(nal.raw) == 0 {
			return nil, encoderInvalid("empty encoder NAL")
		}
		if len(nal.raw) <= maxPayloadSize {
			packetStart := len(data)
			data = appendEncoderRTPHeaderPadding(data)
			payloadStart := len(data)
			data = append(data, nal.raw...)
			packets = appendEncoderRTPPacketFromData(packets, data, packetStart, payloadStart, timestamp)
			i++
			continue
		}
		header := nal.raw[0]
		payload := nal.raw[1:]
		maxFragment := maxPayloadSize - 2
		first := true
		for len(payload) != 0 {
			n := maxFragment
			if n > len(payload) {
				n = len(payload)
			}
			packetStart := len(data)
			data = appendEncoderRTPHeaderPadding(data)
			payloadStart := len(data)
			data = append(data, (header&0xe0)|28)
			fuHeader := header & 0x1f
			if first {
				fuHeader |= 0x80
			}
			if n == len(payload) {
				fuHeader |= 0x40
			}
			data = append(data, fuHeader)
			data = append(data, payload[:n]...)
			packets = appendEncoderRTPPacketFromData(packets, data, packetStart, payloadStart, timestamp)
			payload = payload[n:]
			first = false
		}
		i++
	}
	if len(packets) != 0 {
		packets[len(packets)-1].Marker = true
	}
	return packets, nil
}

func encoderRawNALPayloadStorageSize(nals []encoderRawNAL) (int, error) {
	if len(nals) > maxEncoderRawNALListLen {
		return 0, encoderInvalid("encoder NAL count overflows")
	}
	size := 0
	for _, nal := range nals {
		if len(nal.raw) == 0 {
			return 0, encoderInvalid("empty encoder NAL")
		}
		next, err := checkedAddInt(size, len(nal.raw))
		if err != nil {
			return 0, encoderInvalid("RTP payload storage size overflows")
		}
		size = next
	}
	return size, nil
}

func encoderRTPMode1StoragePlan(nals []encoderRawNAL, maxPayloadSize int, stapa bool) (int, int, error) {
	if maxPayloadSize < 3 {
		return 0, 0, encoderInvalid("RTP max payload size must leave room for FU-A headers")
	}
	if len(nals) > maxEncoderRawNALListLen {
		return 0, 0, encoderInvalid("encoder NAL count overflows")
	}
	packetCount := 0
	payloadSize := 0
	for i := 0; i < len(nals); {
		if stapa && nals[i].parameterSet {
			size, count, err := encoderSTAPASize(nals[i:], maxPayloadSize)
			if err != nil {
				return 0, 0, err
			}
			if count >= 2 {
				var addErr error
				packetCount, addErr = checkedAddInt(packetCount, 1)
				if addErr != nil {
					return 0, 0, encoderInvalid("RTP packet count overflows")
				}
				payloadSize, addErr = checkedAddInt(payloadSize, size)
				if addErr != nil {
					return 0, 0, encoderInvalid("RTP payload storage size overflows")
				}
				i += count
				continue
			}
		}
		nal := nals[i]
		if len(nal.raw) == 0 {
			return 0, 0, encoderInvalid("empty encoder NAL")
		}
		if len(nal.raw) <= maxPayloadSize {
			var addErr error
			packetCount, addErr = checkedAddInt(packetCount, 1)
			if addErr != nil {
				return 0, 0, encoderInvalid("RTP packet count overflows")
			}
			payloadSize, addErr = checkedAddInt(payloadSize, len(nal.raw))
			if addErr != nil {
				return 0, 0, encoderInvalid("RTP payload storage size overflows")
			}
			i++
			continue
		}
		fragmentPayload := len(nal.raw) - 1
		maxFragment := maxPayloadSize - 2
		fragmentCount := fragmentPayload / maxFragment
		if fragmentPayload%maxFragment != 0 {
			fragmentCount++
		}
		var addErr error
		packetCount, addErr = checkedAddInt(packetCount, fragmentCount)
		if addErr != nil {
			return 0, 0, encoderInvalid("RTP packet count overflows")
		}
		fuHeaderSize, addErr := checkedMulInt(2, fragmentCount)
		if addErr != nil {
			return 0, 0, encoderInvalid("RTP payload storage size overflows")
		}
		fragmentStorageSize, addErr := checkedAddInt(fragmentPayload, fuHeaderSize)
		if addErr != nil {
			return 0, 0, encoderInvalid("RTP payload storage size overflows")
		}
		payloadSize, addErr = checkedAddInt(payloadSize, fragmentStorageSize)
		if addErr != nil {
			return 0, 0, encoderInvalid("RTP payload storage size overflows")
		}
		i++
	}
	return packetCount, payloadSize, nil
}

func (e *Encoder) stampRTPPackets(packets []EncoderRTPPacket) {
	for i := range packets {
		packets[i].PayloadType = e.cfg.RTPPayloadType
		packets[i].SequenceNumber = e.rtpSequenceNumber
		packets[i].SSRC = e.cfg.RTPSSRC
		fillEncoderRTPPacketHeader(packets[i].Data, packets[i])
		e.rtpSequenceNumber++
	}
}

func appendEncoderRTPHeaderPadding(dst []byte) []byte {
	return append(dst, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
}

func appendEncoderRTPPacketFromData(packets []EncoderRTPPacket, data []byte, packetStart int, payloadStart int, timestamp uint32) []EncoderRTPPacket {
	return append(packets, EncoderRTPPacket{
		Data:      data[packetStart:len(data):len(data)],
		Payload:   data[payloadStart:len(data):len(data)],
		Timestamp: timestamp,
	})
}

func fillEncoderRTPPacketHeader(dst []byte, pkt EncoderRTPPacket) {
	markerPayloadType := pkt.PayloadType & 0x7f
	if pkt.Marker {
		markerPayloadType |= 0x80
	}
	dst[0] = 0x80
	dst[1] = markerPayloadType
	dst[2] = byte(pkt.SequenceNumber >> 8)
	dst[3] = byte(pkt.SequenceNumber)
	dst[4] = byte(pkt.Timestamp >> 24)
	dst[5] = byte(pkt.Timestamp >> 16)
	dst[6] = byte(pkt.Timestamp >> 8)
	dst[7] = byte(pkt.Timestamp)
	dst[8] = byte(pkt.SSRC >> 24)
	dst[9] = byte(pkt.SSRC >> 16)
	dst[10] = byte(pkt.SSRC >> 8)
	dst[11] = byte(pkt.SSRC)
}

func (e *Encoder) notifyRTPPacketCallback(packets []EncoderRTPPacket, frame EncoderFrame, rtpTime uint32, keyFrame bool, idr bool) {
	callback := e.rtpPacketCallback
	if callback == nil {
		return
	}
	for i, pkt := range packets {
		meta := encoderRTPPacketMetadataFromPayload(pkt.Payload)
		meta.PacketIndex = i
		meta.PacketCount = len(packets)
		meta.FramePTS = frame.PTS
		meta.FrameDTS = frame.PTS
		meta.RTPTime = rtpTime
		meta.KeyFrame = keyFrame
		meta.IDR = idr
		callback(cloneEncoderRTPPacket(pkt), meta)
	}
}

func cloneEncoderRTPPacket(pkt EncoderRTPPacket) EncoderRTPPacket {
	if len(pkt.Data) > maxInt/2 || len(pkt.Payload) > maxInt/2 {
		pkt.Data = nil
		pkt.Payload = nil
		return pkt
	}
	if len(pkt.Data) == 12+len(pkt.Payload) && bytes.Equal(pkt.Data[12:], pkt.Payload) {
		pkt.Data = cloneByteSlice(pkt.Data)
		if len(pkt.Data) < 12 {
			pkt.Payload = nil
			return pkt
		}
		pkt.Data = pkt.Data[:len(pkt.Data):len(pkt.Data)]
		pkt.Payload = pkt.Data[12:len(pkt.Data):len(pkt.Data)]
		return pkt
	}
	pkt.Data = cloneByteSlice(pkt.Data)
	pkt.Payload = cloneByteSlice(pkt.Payload)
	return pkt
}

func encoderRTPPacketMetadataFromPayload(payload []byte) EncoderRTPPacketMetadata {
	if len(payload) == 0 {
		return EncoderRTPPacketMetadata{}
	}
	typ := payload[0] & 0x1f
	switch typ {
	case 24:
		count, parameterSet := encoderRTPSTAPAMetadata(payload)
		return EncoderRTPPacketMetadata{
			PayloadFormat: EncoderRTPPayloadSTAPA,
			NALUnitType:   24,
			NALUnitCount:  count,
			ParameterSet:  parameterSet,
		}
	case 28:
		meta := EncoderRTPPacketMetadata{
			PayloadFormat: EncoderRTPPayloadFUA,
			NALUnitCount:  1,
		}
		if len(payload) >= 2 {
			meta.NALUnitType = payload[1] & 0x1f
			meta.StartOfNAL = payload[1]&0x80 != 0
			meta.EndOfNAL = payload[1]&0x40 != 0
			meta.ParameterSet = encoderRTPNALTypeIsParameterSet(meta.NALUnitType)
		}
		return meta
	default:
		return EncoderRTPPacketMetadata{
			PayloadFormat: EncoderRTPPayloadSingleNAL,
			NALUnitType:   typ,
			NALUnitCount:  1,
			StartOfNAL:    true,
			EndOfNAL:      true,
			ParameterSet:  encoderRTPNALTypeIsParameterSet(typ),
		}
	}
}

func encoderRTPSTAPAMetadata(payload []byte) (int, bool) {
	count := 0
	parameterSet := true
	for pos := 1; pos < len(payload); {
		if pos+2 > len(payload) {
			return count, false
		}
		size := int(payload[pos])<<8 | int(payload[pos+1])
		pos += 2
		if size <= 0 || pos+size > len(payload) {
			return count, false
		}
		if !encoderRTPNALTypeIsParameterSet(payload[pos] & 0x1f) {
			parameterSet = false
		}
		count++
		pos += size
	}
	return count, count != 0 && parameterSet
}

func encoderRTPNALTypeIsParameterSet(typ uint8) bool {
	return typ == uint8(h264.NALSPS) || typ == uint8(h264.NALPPS)
}

func encoderSTAPASize(nals []encoderRawNAL, maxPayloadSize int) (int, int, error) {
	if maxPayloadSize < 1 {
		return 0, 0, encoderInvalid("STAP-A payload size must fit an aggregation header")
	}
	size := 1
	count := 0
	for _, nal := range nals {
		if !nal.parameterSet {
			break
		}
		if len(nal.raw) == 0 {
			return 0, 0, encoderInvalid("empty encoder NAL")
		}
		if len(nal.raw) > 0xffff {
			return 0, 0, encoderInvalid("encoder NAL is too large for STAP-A")
		}
		need, err := checkedAddInt(2, len(nal.raw))
		if err != nil {
			return 0, 0, encoderInvalid("STAP-A payload size overflows")
		}
		nextSize, err := checkedAddInt(size, need)
		if err != nil || nextSize > maxPayloadSize {
			break
		}
		size = nextSize
		count++
	}
	return size, count, nil
}

func appendEncoderSTAPA(dst []byte, nals []encoderRawNAL, maxPayloadSize int) ([]byte, int, error) {
	start := len(dst)
	size, plannedCount, err := encoderSTAPASize(nals, maxPayloadSize)
	if err != nil {
		return dst, 0, err
	}
	if plannedCount < 2 {
		return dst, plannedCount, nil
	}
	if _, err := checkedAddInt(len(dst), size); err != nil {
		return dst, 0, encoderInvalid("STAP-A destination size overflows")
	}
	dst = append(dst, 24)
	var maxNRI byte
	count := 0
	for _, nal := range nals {
		if count == plannedCount {
			break
		}
		if !nal.parameterSet {
			break
		}
		if len(nal.raw) == 0 {
			return dst[:start], 0, encoderInvalid("empty encoder NAL")
		}
		if len(nal.raw) > 0xffff {
			return dst[:start], 0, encoderInvalid("encoder NAL is too large for STAP-A")
		}
		need, err := checkedAddInt(2, len(nal.raw))
		if err != nil {
			return dst[:start], 0, encoderInvalid("STAP-A payload size overflows")
		}
		currentSize := len(dst) - start
		nextSize, err := checkedAddInt(currentSize, need)
		if err != nil || nextSize > maxPayloadSize {
			break
		}
		if nri := nal.raw[0] & 0x60; nri > maxNRI {
			maxNRI = nri
		}
		dst = append(dst, byte(len(nal.raw)>>8), byte(len(nal.raw)))
		dst = append(dst, nal.raw...)
		count++
	}
	if count < 2 {
		return dst[:start], count, nil
	}
	dst[start] = maxNRI | 24
	return dst, count, nil
}

func encoderDeblockingFilterIDC(mode EncoderDeblockMode) uint32 {
	switch mode {
	case EncoderDeblockDisabled:
		return 1
	case EncoderDeblockSliceBoundary:
		return 2
	default:
		return 0
	}
}

func normalizeEncoderConfig(cfg EncoderConfig) (EncoderConfig, error) {
	return normalizeEncoderConfigWithExplicitQP(cfg, false, false, false)
}

func normalizeEncoderConfigWithExplicitQP(cfg EncoderConfig, explicitInitialQP, explicitMinQP, explicitMaxQP bool) (EncoderConfig, error) {
	if cfg.ExplicitQP {
		explicitInitialQP = true
		explicitMinQP = true
		explicitMaxQP = true
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return cfg, encoderInvalid("width and height must be positive")
	}
	if cfg.PixelFormat == 0 {
		cfg.PixelFormat = EncoderPixelFormatI420
	}
	if cfg.PixelFormat != EncoderPixelFormatI420 {
		return cfg, encoderUnsupported("only 8-bit I420 input is in the realtime encoder scope")
	}
	if cfg.Width%2 != 0 || cfg.Height%2 != 0 {
		return cfg, encoderInvalid("I420 width and height must be even")
	}
	if err := validateEncoderCrop(cfg.Crop, cfg.Width, cfg.Height); err != nil {
		return cfg, err
	}
	if cfg.StrideY == 0 {
		cfg.StrideY = cfg.Width
	}
	if cfg.StrideCb == 0 {
		cfg.StrideCb = cfg.Width / 2
	}
	if cfg.StrideCr == 0 {
		cfg.StrideCr = cfg.Width / 2
	}
	if cfg.StrideY < cfg.Width || cfg.StrideCb < cfg.Width/2 || cfg.StrideCr < cfg.Width/2 {
		return cfg, encoderInvalid("strides must cover the configured planes")
	}
	if err := validateEncoderPlaneGeometry(cfg.Width, cfg.Height, cfg.StrideY, cfg.StrideCb, cfg.StrideCr); err != nil {
		return cfg, err
	}
	if err := validateEncoderColorConfig(cfg.Color); err != nil {
		return cfg, err
	}
	if cfg.FrameRateNum <= 0 || cfg.FrameRateDen <= 0 {
		return cfg, encoderInvalid("frame rate numerator and denominator must be positive")
	}
	if cfg.TimeBaseNum == 0 {
		cfg.TimeBaseNum = 1
	}
	if cfg.TimeBaseDen == 0 {
		cfg.TimeBaseDen = 90000
	}
	if cfg.TimeBaseNum <= 0 || cfg.TimeBaseDen <= 0 {
		return cfg, encoderInvalid("time base numerator and denominator must be positive")
	}
	if cfg.Profile == 0 {
		cfg.Profile = EncoderProfileConstrainedBaseline
	}
	switch cfg.Profile {
	case EncoderProfileConstrainedBaseline, EncoderProfileBaseline:
	case EncoderProfileMain, EncoderProfileHigh:
		return cfg, encoderUnsupported("Main and High encoder profiles are outside the admitted encoder subset")
	default:
		return cfg, encoderInvalid("unknown encoder profile")
	}
	if cfg.LevelIDC == 0 {
		cfg.LevelIDC = 31
	}
	if cfg.EntropyMode == 0 {
		cfg.EntropyMode = EncoderEntropyCAVLC
	}
	if cfg.EntropyMode != EncoderEntropyCAVLC {
		return cfg, encoderUnsupported("CABAC is not admitted for the first realtime WebRTC encoder slice")
	}
	if cfg.DeblockMode == 0 {
		cfg.DeblockMode = EncoderDeblockEnabled
	}
	switch cfg.DeblockMode {
	case EncoderDeblockEnabled, EncoderDeblockDisabled, EncoderDeblockSliceBoundary:
	default:
		return cfg, encoderInvalid("unknown deblock mode")
	}
	if cfg.Transform8x8 {
		return cfg, encoderUnsupported("8x8 transform is outside the initial Baseline encoder scope")
	}
	if cfg.MaxReferenceFrames == 0 {
		cfg.MaxReferenceFrames = 1
	}
	if cfg.MaxReferenceFrames != 1 {
		return cfg, encoderUnsupported("the realtime encoder initially admits one reference frame")
	}
	if cfg.BFrames != 0 {
		return cfg, encoderUnsupported("B-frames are disabled for the realtime WebRTC encoder scope")
	}
	if cfg.SPSPPSMode == 0 {
		cfg.SPSPPSMode = EncoderSPSPPSInBandKeyframes
	}
	switch cfg.SPSPPSMode {
	case EncoderSPSPPSInBandKeyframes, EncoderSPSPPSOutOfBand, EncoderSPSPPSEveryIDR:
	default:
		return cfg, encoderInvalid("unknown SPS/PPS emission mode")
	}
	if cfg.RateControl == 0 {
		cfg.RateControl = EncoderRateControlCBR
	}
	switch cfg.RateControl {
	case EncoderRateControlCBR, EncoderRateControlConstantQP:
	case EncoderRateControlVBR:
		return cfg, encoderUnsupported("VBR rate control is outside the admitted encoder quality-decision subset")
	default:
		return cfg, encoderInvalid("unknown rate-control mode")
	}
	if cfg.TargetBitrate <= 0 {
		return cfg, encoderInvalid("target bitrate must be positive")
	}
	if cfg.MaxBitrate == 0 {
		cfg.MaxBitrate = cfg.TargetBitrate
	}
	if cfg.MaxBitrate < cfg.TargetBitrate {
		return cfg, encoderInvalid("max bitrate must be greater than or equal to target bitrate")
	}
	if cfg.VBVBufferSize < 0 || cfg.MaxFrameSize < 0 {
		return cfg, encoderInvalid("VBV buffer size and max frame size cannot be negative")
	}
	if cfg.InitialQP == 0 && !explicitInitialQP {
		cfg.InitialQP = 26
	}
	if cfg.MinQP == 0 && !explicitMinQP {
		cfg.MinQP = 10
	}
	if cfg.MaxQP == 0 && !explicitMaxQP {
		cfg.MaxQP = 42
	}
	if cfg.MinQP < 0 || cfg.MinQP > 51 || cfg.MaxQP < 0 || cfg.MaxQP > 51 || cfg.InitialQP < cfg.MinQP || cfg.InitialQP > cfg.MaxQP {
		return cfg, encoderInvalid("QP range must be within 0..51 and contain the initial QP")
	}
	if explicitInitialQP || explicitMinQP || explicitMaxQP {
		cfg.ExplicitQP = true
	}
	if cfg.Preset == 0 {
		cfg.Preset = EncoderPresetRealtime
	}
	switch cfg.Preset {
	case EncoderPresetRealtime:
	case EncoderPresetBalanced, EncoderPresetQuality:
		return cfg, encoderUnsupported("non-realtime presets are outside the admitted encoder mode-decision subset")
	default:
		return cfg, encoderInvalid("unknown encoder preset")
	}
	if cfg.FrameDrop == 0 {
		cfg.FrameDrop = EncoderFrameDropToBitrate
	}
	switch cfg.FrameDrop {
	case EncoderFrameDropDisabled, EncoderFrameDropLate, EncoderFrameDropToBitrate:
	default:
		return cfg, encoderInvalid("unknown frame drop mode")
	}
	if cfg.MaxEncodeTimeUS < 0 || cfg.SliceCount < 0 || cfg.SliceMaxBytes < 0 || cfg.Workers < 0 {
		return cfg, encoderInvalid("latency and worker controls cannot be negative")
	}
	if cfg.SliceCount == 0 {
		cfg.SliceCount = 1
	}
	macroblockCount, err := encoderMacroblockCountChecked(cfg.Width, cfg.Height)
	if err != nil {
		return cfg, err
	}
	if cfg.SliceCount > macroblockCount {
		return cfg, encoderInvalid("slice count cannot exceed coded macroblock count")
	}
	if cfg.Workers == 0 {
		cfg.Workers = 1
	}
	if cfg.Deterministic && cfg.Workers != 1 {
		return cfg, encoderInvalid("deterministic mode requires one worker")
	}
	if cfg.GOPSize <= 0 {
		cfg.GOPSize = 60
	}
	if cfg.IDRInterval <= 0 {
		cfg.IDRInterval = cfg.GOPSize
	}
	if cfg.IDRInterval > cfg.GOPSize {
		return cfg, encoderInvalid("IDR interval must be less than or equal to GOP size")
	}
	if cfg.IntraRefresh {
		return cfg, encoderUnsupported("intra refresh is outside the admitted encoder subset")
	}
	if cfg.OutputFormat == 0 {
		cfg.OutputFormat = EncoderOutputRTP
	}
	switch cfg.OutputFormat {
	case EncoderOutputAnnexB, EncoderOutputAVC, EncoderOutputRTP:
	default:
		return cfg, encoderInvalid("unknown encoder output format")
	}
	if err := validateEncoderRTPControlEnvelope(cfg); err != nil {
		return cfg, err
	}
	if cfg.OutputFormat == EncoderOutputRTP {
		if cfg.RTPMaxPayloadSize == 0 {
			cfg.RTPMaxPayloadSize = 1200
		}
		if err := validateEncoderRTPPacketPayloadBudget(cfg); err != nil {
			return cfg, err
		}
		if !cfg.DONDisabled {
			return cfg, encoderUnsupported("interleaved DON mode is not part of WebRTC RTP packetization")
		}
		if cfg.RTPPayloadType == 0 {
			cfg.RTPPayloadType = 96
		}
	}
	if cfg.RTPTimestampIncrement == 0 {
		increment, err := rtpTimestampIncrementChecked(cfg.TimeBaseDen, cfg.FrameRateNum, cfg.FrameRateDen)
		if err != nil {
			return cfg, err
		}
		cfg.RTPTimestampIncrement = increment
	}
	if encoderBitrateBudgetEnabled(cfg) {
		if _, err := encoderBitrateFrameBudgetBytesChecked(cfg); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

func validateEncoderRTPControlEnvelope(cfg EncoderConfig) error {
	if cfg.RTPMaxPayloadSize < 0 {
		return encoderInvalid("RTP max payload size cannot be negative")
	}
	switch cfg.RTPPacketizationMode {
	case EncoderRTPPacketizationSingleNAL, EncoderRTPPacketizationNonInterleaved:
	default:
		return encoderInvalid("unknown RTP packetization mode")
	}
	if cfg.STAPA && cfg.RTPPacketizationMode == EncoderRTPPacketizationSingleNAL {
		return encoderUnsupported("STAP-A aggregation requires RTP packetization-mode 1")
	}
	if cfg.RTPPayloadType > 127 {
		return encoderInvalid("RTP payload type must fit in seven bits")
	}
	if cfg.RTPMaxPayloadSize != 0 {
		return validateEncoderRTPPacketPayloadBudget(cfg)
	}
	return nil
}

func validateEncoderRTPPacketPayloadBudget(cfg EncoderConfig) error {
	switch cfg.RTPPacketizationMode {
	case EncoderRTPPacketizationSingleNAL:
		if cfg.RTPMaxPayloadSize < 2 {
			return encoderInvalid("RTP packetization-mode 0 max payload size must fit a NAL header and payload byte")
		}
	case EncoderRTPPacketizationNonInterleaved:
		if cfg.RTPMaxPayloadSize < 3 {
			return encoderInvalid("RTP max payload size must leave room for FU-A headers")
		}
	default:
		return encoderInvalid("unknown RTP packetization mode")
	}
	return nil
}

func validateEncoderColorConfig(color EncoderColorConfig) error {
	if color.SARNum < 0 || color.SARDen < 0 || color.SARNum > 0xffff || color.SARDen > 0xffff || (color.SARNum == 0) != (color.SARDen == 0) {
		return encoderInvalid("invalid SAR")
	}
	if color.VideoFormat < 0 || color.VideoFormat > 7 ||
		color.ColorPrimaries < 0 || color.ColorPrimaries > 255 ||
		color.ColorTransfer < 0 || color.ColorTransfer > 255 ||
		color.ColorMatrix < 0 || color.ColorMatrix > 255 {
		return encoderInvalid("invalid VUI color fields")
	}
	if color.ChromaSampleLocTypeTopField < 0 || color.ChromaSampleLocTypeTopField > 5 ||
		color.ChromaSampleLocTypeBottomField < 0 || color.ChromaSampleLocTypeBottomField > 5 {
		return encoderInvalid("invalid chroma sample location")
	}
	return nil
}

func validateEncoderCrop(crop EncoderCrop, width int, height int) error {
	if crop.Left < 0 || crop.Right < 0 || crop.Top < 0 || crop.Bottom < 0 {
		return encoderInvalid("crop offsets cannot be negative")
	}
	if crop.Left%2 != 0 || crop.Right%2 != 0 || crop.Top%2 != 0 || crop.Bottom%2 != 0 {
		return encoderInvalid("I420 crop offsets must be even")
	}
	horizontalCrop, err := checkedAddInt(crop.Left, crop.Right)
	if err != nil {
		return encoderInvalid("crop offsets are too large")
	}
	verticalCrop, err := checkedAddInt(crop.Top, crop.Bottom)
	if err != nil {
		return encoderInvalid("crop offsets are too large")
	}
	if horizontalCrop >= width || verticalCrop >= height {
		return encoderInvalid("crop offsets must leave a visible frame")
	}
	return nil
}

func rtpTimestampIncrement(clock, frameRateNum, frameRateDen int) uint32 {
	increment, err := rtpTimestampIncrementChecked(clock, frameRateNum, frameRateDen)
	if err != nil {
		return 0
	}
	return increment
}

func rtpTimestampIncrementChecked(clock, frameRateNum, frameRateDen int) (uint32, error) {
	if clock <= 0 || frameRateNum <= 0 || frameRateDen <= 0 {
		return 0, encoderInvalid("RTP clock and frame rate must be positive")
	}
	ticks, err := checkedMulInt(clock, frameRateDen)
	if err != nil {
		return 0, encoderInvalid("RTP timestamp increment overflows")
	}
	increment := ticks / frameRateNum
	if increment == 0 {
		return 0, encoderInvalid("RTP timestamp increment must be positive")
	}
	if uint64(increment) > uint64(^uint32(0)) {
		return 0, encoderInvalid("RTP timestamp increment must fit in 32 bits")
	}
	return uint32(increment), nil
}

func (e *Encoder) encoderRTPTime(frame EncoderFrame) uint32 {
	if frame.PTS != 0 || !e.rtpTimeInitialized {
		return uint32(frame.PTS)
	}
	return e.nextRTPTime
}

func (e *Encoder) advanceEncoderRTPTime(frame EncoderFrame, rtpTime uint32) {
	e.nextRTPTime = rtpTime + encoderFrameRTPDuration(e.cfg, frame)
	e.rtpTimeInitialized = true
}

func encoderFrameRTPDuration(cfg EncoderConfig, frame EncoderFrame) uint32 {
	if frame.Duration > 0 {
		return uint32(frame.Duration)
	}
	if cfg.RTPTimestampIncrement != 0 {
		return cfg.RTPTimestampIncrement
	}
	return rtpTimestampIncrement(cfg.TimeBaseDen, cfg.FrameRateNum, cfg.FrameRateDen)
}

func encoderProfileSyntax(profile EncoderProfile) (uint8, uint8, error) {
	switch profile {
	case EncoderProfileConstrainedBaseline:
		return 66, 0x03, nil
	case EncoderProfileBaseline:
		return 66, 0x01, nil
	default:
		return 0, 0, encoderUnsupported("profile is not admitted for parameter-set generation")
	}
}

func encoderInvalid(detail string) error {
	return fmt.Errorf("h264: encoder %s: %w", detail, ErrInvalidData)
}

func encoderUnsupported(detail string) error {
	return fmt.Errorf("h264: encoder %s: %w", detail, ErrUnsupported)
}
