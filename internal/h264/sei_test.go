// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"bytes"
	"math/bits"
	"testing"
)

func TestH264SEIContextResetInitializesX264BuildLikeFFmpeg(t *testing.T) {
	var ctx H264SEIContext
	ctx.Common.Unregistered.X264Build = 165
	ctx.Reset()
	if ctx.Common.Unregistered.X264Build != -1 {
		t.Fatalf("x264 build = %d, want -1", ctx.Common.Unregistered.X264Build)
	}
}

func TestDecodeSEIMessages(t *testing.T) {
	sps := &SPS{
		SPSID:                        0,
		NALHRDParametersPresentFlag:  1,
		CPBCount:                     2,
		InitialCPBRemovalDelayLength: 5,
		CPBRemovalDelayLength:        5,
		DPBOutputDelayLength:         4,
		PicStructPresentFlag:         1,
		TimeOffsetLength:             3,
	}
	var spsList [maxSPSCount]*SPS
	spsList[0] = sps

	ctx, err := DecodeSEI(buildSEIRBSP(
		seiTestMessage{typ: seiTypeBufferingPeriod, payload: seiBufferingPeriodPayload()},
		seiTestMessage{typ: seiTypePicTiming, payload: seiPictureTimingPayload()},
		seiTestMessage{typ: seiTypeRecoveryPoint, payload: seiRecoveryPointPayload()},
		seiTestMessage{typ: seiTypeGreenMetadata, payload: []byte{0, 2, 0x01, 0x23, 1, 2, 3, 4}},
		seiTestMessage{typ: seiTypeUserDataRegisteredITUTT35, payload: seiRegisteredAFDPayload(0x0f)},
		seiTestMessage{typ: seiTypeUserDataRegisteredITUTT35, payload: seiRegisteredA53Payload([]byte{0x04, 0x05, 0x06, 0x07, 0x08, 0x09})},
		seiTestMessage{typ: seiTypeUserDataRegisteredITUTT35, payload: seiRegisteredLCEVCPayload([]byte{0x7e, 0x00, 0x00, 0x03, 0x01})},
		seiTestMessage{typ: seiTypeUserDataUnregistered, payload: seiUnregisteredPayload()},
		seiTestMessage{typ: seiTypeDisplayOrientation, payload: seiDisplayOrientationPayload()},
		seiTestMessage{typ: seiTypeFramePackingArrangement, payload: seiFramePackingPayload()},
		seiTestMessage{typ: seiTypeAlternativeTransfer, payload: []byte{16}},
		seiTestMessage{typ: seiTypeAmbientViewingEnvironment, payload: seiAmbientViewingPayload()},
		seiTestMessage{typ: seiTypeFilmGrainCharacteristics, payload: seiFilmGrainPayload()},
		seiTestMessage{typ: seiTypeMasteringDisplayColourVolume, payload: seiMasteringDisplayPayload()},
		seiTestMessage{typ: seiTypeContentLightLevelInfo, payload: []byte{0x03, 0xe8, 0x00, 0xfa}},
	), &spsList)
	if err != nil {
		t.Fatal(err)
	}

	if ctx.BufferingPeriod.Present != 1 || ctx.BufferingPeriod.InitialCPBRemovalDelay[0] != 17 || ctx.BufferingPeriod.InitialCPBRemovalDelay[1] != 9 {
		t.Fatalf("buffering period = present %d delays %v", ctx.BufferingPeriod.Present, ctx.BufferingPeriod.InitialCPBRemovalDelay[:2])
	}
	if ctx.PictureTiming.Present != 1 || ctx.PictureTiming.PayloadSize != len(seiPictureTimingPayload()) {
		t.Fatalf("picture timing = present %d payload %d", ctx.PictureTiming.Present, ctx.PictureTiming.PayloadSize)
	}
	if err := ctx.PictureTiming.Process(sps); err != nil {
		t.Fatalf("process picture timing: %v", err)
	}
	if ctx.PictureTiming.CPBRemovalDelay != 21 || ctx.PictureTiming.DPBOutputDelay != 7 ||
		ctx.PictureTiming.PicStruct != h264SEIPicStructTopBottom || ctx.PictureTiming.CTType != 1<<2 ||
		ctx.PictureTiming.TimecodeCount != 1 {
		t.Fatalf("picture timing processed = cpb %d dpb %d pic %d ct %#x count %d",
			ctx.PictureTiming.CPBRemovalDelay, ctx.PictureTiming.DPBOutputDelay,
			ctx.PictureTiming.PicStruct, ctx.PictureTiming.CTType, ctx.PictureTiming.TimecodeCount)
	}
	tc := ctx.PictureTiming.Timecode[0]
	if tc.Full != 1 || tc.DropFrame != 1 || tc.Frame != 12 || tc.Seconds != 34 || tc.Minutes != 56 || tc.Hours != 7 {
		t.Fatalf("timecode = %+v", tc)
	}

	if ctx.RecoveryPoint.RecoveryFrameCount != 4 {
		t.Fatalf("recovery frame count = %d", ctx.RecoveryPoint.RecoveryFrameCount)
	}
	if ctx.GreenMetadata.Present != 1 || ctx.GreenMetadata.GreenMetadataType != 0 || ctx.GreenMetadata.PeriodType != 2 || ctx.GreenMetadata.NumSeconds != 0x0123 ||
		ctx.GreenMetadata.PercentNonZeroMacroblocks != 1 || ctx.GreenMetadata.PercentIntraCodedMacroblocks != 2 ||
		ctx.GreenMetadata.PercentSixTapFiltering != 3 || ctx.GreenMetadata.PercentAlphaPointDeblockingInstance != 4 {
		t.Fatalf("green metadata = %+v", ctx.GreenMetadata)
	}
	if ctx.Common.AFD.Present != 1 || ctx.Common.AFD.ActiveFormatDescription != 0x0f {
		t.Fatalf("afd = %+v", ctx.Common.AFD)
	}
	if got, want := ctx.Common.A53Caption.Data, []byte{0x04, 0x05, 0x06, 0x07, 0x08, 0x09}; !bytes.Equal(got, want) {
		t.Fatalf("a53 caption = %x, want %x", got, want)
	}
	if got, want := ctx.Common.LCEVC.Data, []byte{0x7e, 0x00, 0x00, 0x03, 0x01}; !bytes.Equal(got, want) {
		t.Fatalf("lcevc = %x, want %x", got, want)
	}
	if len(ctx.Common.Unregistered.Data) != 1 || ctx.Common.Unregistered.X264Build != 165 {
		t.Fatalf("unregistered = count %d x264 %d", len(ctx.Common.Unregistered.Data), ctx.Common.Unregistered.X264Build)
	}
	if ctx.Common.DisplayOrientation.Present != 1 || ctx.Common.DisplayOrientation.HFlip != 1 ||
		ctx.Common.DisplayOrientation.VFlip != 0 || ctx.Common.DisplayOrientation.AnticlockwiseRotation != 0x4000 {
		t.Fatalf("display orientation = %+v", ctx.Common.DisplayOrientation)
	}
	if ctx.Common.FramePacking.Present != 1 || ctx.Common.FramePacking.ArrangementID != 2 ||
		ctx.Common.FramePacking.ArrangementType != 3 || ctx.Common.FramePacking.ContentInterpretationType != 2 ||
		ctx.Common.FramePacking.CurrentFrameIsFrame0Flag != 1 || ctx.Common.FramePacking.ArrangementRepetitionPeriod != 5 {
		t.Fatalf("frame packing = %+v", ctx.Common.FramePacking)
	}
	if ctx.Common.AlternativeTransfer.Present != 1 || ctx.Common.AlternativeTransfer.PreferredTransferCharacteristics != 16 {
		t.Fatalf("alternative transfer = %+v", ctx.Common.AlternativeTransfer)
	}
	if ctx.Common.AmbientViewing.Present != 1 || ctx.Common.AmbientViewing.AmbientIlluminance != 12345 ||
		ctx.Common.AmbientViewing.AmbientLightX != 25000 || ctx.Common.AmbientViewing.AmbientLightY != 16667 {
		t.Fatalf("ambient viewing = %+v", ctx.Common.AmbientViewing)
	}
	fg := ctx.Common.FilmGrain
	if fg.Present != 1 || fg.ModelID != 1 || fg.SeparateColourDescriptionPresentFlag != 1 ||
		fg.BitDepthLuma != 10 || fg.BitDepthChroma != 8 || fg.FullRange != 1 ||
		fg.ColorPrimaries != 9 || fg.TransferCharacteristics != 16 || fg.MatrixCoeffs != 9 ||
		fg.BlendingModeID != 1 || fg.Log2ScaleFactor != 7 || fg.RepetitionPeriod != 4 {
		t.Fatalf("film grain header = %+v", fg)
	}
	if fg.CompModelPresentFlag != [3]int32{1, 1, 0} ||
		fg.NumIntensityIntervals != [3]uint16{1, 2, 0} ||
		fg.NumModelValues != [3]uint8{2, 1, 0} {
		t.Fatalf("film grain component counts = present %+v intervals %+v values %+v",
			fg.CompModelPresentFlag, fg.NumIntensityIntervals, fg.NumModelValues)
	}
	if fg.IntensityIntervalLowerBound[0][0] != 10 || fg.IntensityIntervalUpperBound[0][0] != 20 ||
		fg.CompModelValue[0][0][0] != 3 || fg.CompModelValue[0][0][1] != -2 ||
		fg.IntensityIntervalLowerBound[1][1] != 41 || fg.IntensityIntervalUpperBound[1][1] != 60 ||
		fg.CompModelValue[1][1][0] != 5 {
		t.Fatalf("film grain component data = %+v %+v %+v", fg.IntensityIntervalLowerBound, fg.IntensityIntervalUpperBound, fg.CompModelValue)
	}
	if ctx.Common.MasteringDisplay.Present != 2 ||
		ctx.Common.MasteringDisplay.DisplayPrimaries != [3][2]uint16{{10000, 20000}, {15000, 25000}, {30000, 35000}} ||
		ctx.Common.MasteringDisplay.WhitePoint != [2]uint16{15635, 16450} ||
		ctx.Common.MasteringDisplay.MaxLuminance != 10000000 ||
		ctx.Common.MasteringDisplay.MinLuminance != 100 {
		t.Fatalf("mastering display = %+v", ctx.Common.MasteringDisplay)
	}
	if ctx.Common.ContentLight.Present != 2 || ctx.Common.ContentLight.MaxContentLightLevel != 1000 ||
		ctx.Common.ContentLight.MaxPicAverageLightLevel != 250 {
		t.Fatalf("content light = %+v", ctx.Common.ContentLight)
	}
}

func TestDecodeSEIExtendedMessageHeader(t *testing.T) {
	payload := make([]byte, 260)
	for i := range payload {
		payload[i] = uint8(i)
	}
	ctx, err := DecodeSEI(buildSEIRBSP(
		seiTestMessage{typ: 300, payload: payload},
		seiTestMessage{typ: seiTypeAlternativeTransfer, payload: []byte{18}},
	), nil)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Common.AlternativeTransfer.Present != 1 || ctx.Common.AlternativeTransfer.PreferredTransferCharacteristics != 18 {
		t.Fatalf("alternative transfer after extended header = %+v", ctx.Common.AlternativeTransfer)
	}
}

func TestDecodeSEIRejectsTruncatedPayload(t *testing.T) {
	if _, err := DecodeSEI([]byte{seiTypeRecoveryPoint, 5, 0x80}, nil); err != ErrInvalidData {
		t.Fatalf("err = %v, want ErrInvalidData", err)
	}
}

func TestParseX264BuildMatchesFFmpegQuirk(t *testing.T) {
	if build, ok := parseX264Build([]byte("x264 - core 165 r3095")); !ok || build != 165 {
		t.Fatalf("normal x264 build = %d/%v, want 165/true", build, ok)
	}
	if build, ok := parseX264Build([]byte("x264 - core 0000")); ok || build != 0 {
		t.Fatalf("0000 x264 build = %d/%v, want 0/false", build, ok)
	}
	if build, ok := parseX264Build([]byte("x264 - core 00001")); !ok || build != 67 {
		t.Fatalf("00001 x264 build = %d/%v, want 67/true", build, ok)
	}
}

func TestDecodeUnregisteredUserDataRejectsOverflowedPayloadSize(t *testing.T) {
	var sei H2645SEIUnregistered
	if err := sei.decodeUnregisteredUserData(fakeRBSPBytesLen(maxInt)); err != ErrInvalidData {
		t.Fatalf("decodeUnregisteredUserData overflow error = %v, want ErrInvalidData", err)
	}
	if len(sei.Data) != 0 {
		t.Fatalf("overflowed payload appended %d entries, want 0", len(sei.Data))
	}
}

func TestDecodeSEIBufferingPeriodMissingSPSIsNonFatalMasterError(t *testing.T) {
	var spsList [maxSPSCount]*SPS
	if _, err := DecodeSEI(buildSEIRBSP(seiTestMessage{typ: seiTypeBufferingPeriod, payload: seiBufferingPeriodPayload()}), &spsList); err != errParamSetNotFound {
		t.Fatalf("err = %v, want errParamSetNotFound", err)
	}
}

func TestDecodeSEIRegisteredA53CaptionsMergeAcrossMessages(t *testing.T) {
	ctx, err := DecodeSEI(buildSEIRBSP(
		seiTestMessage{typ: seiTypeUserDataRegisteredITUTT35, payload: seiRegisteredA53Payload([]byte{0x01, 0x02, 0x03})},
		seiTestMessage{typ: seiTypeUserDataRegisteredITUTT35, payload: seiRegisteredA53Payload([]byte{0x04, 0x05, 0x06})},
	), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ctx.Common.A53Caption.Data, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}; !bytes.Equal(got, want) {
		t.Fatalf("a53 caption = %x, want %x", got, want)
	}
}

func TestDecodeSEIRegisteredA53RejectsTruncatedCCData(t *testing.T) {
	payload := seiRegisteredA53Payload([]byte{0x01, 0x02, 0x03})
	payload = payload[:len(payload)-1]
	if _, err := DecodeSEI(buildSEIRBSP(seiTestMessage{typ: seiTypeUserDataRegisteredITUTT35, payload: payload}), nil); err != ErrInvalidData {
		t.Fatalf("err = %v, want ErrInvalidData", err)
	}
}

func TestDecodeSEIRegisteredLCEVCReplacesPriorPayload(t *testing.T) {
	ctx, err := DecodeSEI(buildSEIRBSP(
		seiTestMessage{typ: seiTypeUserDataRegisteredITUTT35, payload: seiRegisteredLCEVCPayload([]byte{0x01, 0x02})},
		seiTestMessage{typ: seiTypeUserDataRegisteredITUTT35, payload: seiRegisteredLCEVCPayload([]byte{0x7e, 0x00, 0x00, 0x03, 0x01})},
	), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ctx.Common.LCEVC.Data, []byte{0x7e, 0x00, 0x00, 0x03, 0x01}; !bytes.Equal(got, want) {
		t.Fatalf("lcevc = %x, want %x", got, want)
	}
}

func TestDecodeSEIRegisteredLCEVCRejectsMissingPayload(t *testing.T) {
	payload := []byte{ituTT35CountryCodeUK, 0x00, 0x50, 0x01}
	if _, err := DecodeSEI(buildSEIRBSP(seiTestMessage{typ: seiTypeUserDataRegisteredITUTT35, payload: payload}), nil); err != ErrInvalidData {
		t.Fatalf("err = %v, want ErrInvalidData", err)
	}
}

func TestDecodeSEIAmbientViewingRejectsInvalidValues(t *testing.T) {
	for _, payload := range [][]byte{
		{0, 0, 0, 0, 0x61, 0xa8, 0x41, 0x1b},
		{0, 0, 0x30, 0x39, 0xc3, 0x51, 0x41, 0x1b},
		{0, 0, 0x30, 0x39, 0x61, 0xa8, 0xc3, 0x51},
		{0, 0, 0x30, 0x39, 0x61, 0xa8, 0x41},
	} {
		if _, err := DecodeSEI(buildSEIRBSP(seiTestMessage{typ: seiTypeAmbientViewingEnvironment, payload: payload}), nil); err != ErrInvalidData {
			t.Fatalf("payload %x err = %v, want ErrInvalidData", payload, err)
		}
	}
}

func TestDecodeSEIFilmGrainRejectsTooManyModelValues(t *testing.T) {
	if _, err := DecodeSEI(buildSEIRBSP(seiTestMessage{typ: seiTypeFilmGrainCharacteristics, payload: seiFilmGrainTooManyValuesPayload()}), nil); err != ErrInvalidData {
		t.Fatalf("err = %v, want ErrInvalidData", err)
	}
}

func TestDecodeSimpleNALUnitsParsesLeadingSEI(t *testing.T) {
	var spsList [maxSPSCount]*SPS
	var ppsList [maxPPSCount]*PPS
	var dpb simpleFrameDPB
	var sei H264SEIContext
	dpb.reset()
	sei.Reset()

	_, err := decodeSimpleNALUnitsWithState([]NALUnit{{
		Type: NALSEI,
		RBSP: buildSEIRBSP(seiTestMessage{typ: seiTypeUserDataUnregistered, payload: seiUnregisteredPayload()}),
	}}, &spsList, &ppsList, &dpb, &sei, DecodedFrameSideData{}, false)
	if err != ErrInvalidData {
		t.Fatalf("err = %v, want ErrInvalidData for packet without slices", err)
	}
	if sei.Common.Unregistered.X264Build != 165 {
		t.Fatalf("simple decoder SEI x264 build = %d", sei.Common.Unregistered.X264Build)
	}
}

func TestDecodedFrameSideDataFromSEICopiesUserData(t *testing.T) {
	ctx, err := DecodeSEI(buildSEIRBSP(
		seiTestMessage{typ: seiTypeUserDataUnregistered, payload: seiUnregisteredPayload()},
		seiTestMessage{typ: seiTypeContentLightLevelInfo, payload: []byte{0x03, 0xe8, 0x00, 0xfa}},
	), nil)
	if err != nil {
		t.Fatal(err)
	}
	side := decodedFrameSideDataFromSEI(ctx)
	if side.X264Build != 165 || len(side.UserDataUnregistered) != 1 {
		t.Fatalf("side unregistered = build %d count %d", side.X264Build, len(side.UserDataUnregistered))
	}
	ctx.Common.A53Caption.Data = []uint8{1, 2, 3}
	side = decodedFrameSideDataFromSEI(ctx)
	ctx.Common.A53Caption.Data[0] ^= 0xff
	if got, want := side.A53ClosedCaptions, []uint8{1, 2, 3}; !bytes.Equal(got, want) {
		t.Fatalf("side a53 = %x, want %x", got, want)
	}
	ctx.Common.Unregistered.Data[0][0] ^= 0xff
	if side.UserDataUnregistered[0][0] == ctx.Common.Unregistered.Data[0][0] {
		t.Fatal("side data aliases SEI context user data")
	}
	if side.ContentLight.Present != 2 || side.ContentLight.MaxContentLightLevel != 1000 ||
		side.ContentLight.MaxPicAverageLightLevel != 250 {
		t.Fatalf("content light side data = %+v", side.ContentLight)
	}
	ctx.Common.LCEVC.Data = []uint8{0x7e, 0x00, 0x00, 0x03, 0x01}
	side = decodedFrameSideDataFromSEI(ctx)
	ctx.Common.LCEVC.Data[0] ^= 0xff
	if got, want := side.LCEVC, []uint8{0x7e, 0x00, 0x00, 0x03, 0x01}; !bytes.Equal(got, want) {
		t.Fatalf("side lcevc = %x, want %x", got, want)
	}
}

type seiTestMessage struct {
	typ     int
	payload []byte
}

func buildSEIRBSP(messages ...seiTestMessage) []byte {
	var out []byte
	for _, msg := range messages {
		out = appendSEIHeaderValue(out, msg.typ)
		out = appendSEIHeaderValue(out, len(msg.payload))
		out = append(out, msg.payload...)
	}
	return append(out, 0x80)
}

func appendSEIHeaderValue(out []byte, value int) []byte {
	for value >= 255 {
		out = append(out, 255)
		value -= 255
	}
	return append(out, uint8(value))
}

func seiBufferingPeriodPayload() []byte {
	var b seiBitBuilder
	b.writeUE(0)
	b.writeBits(17, 5)
	b.writeBits(3, 5)
	b.writeBits(9, 5)
	b.writeBits(2, 5)
	return b.bytes()
}

func seiPictureTimingPayload() []byte {
	var b seiBitBuilder
	b.writeBits(21, 5)
	b.writeBits(7, 4)
	b.writeBits(h264SEIPicStructTopBottom, 4)
	b.writeBit(1)
	b.writeBits(2, 2)
	b.writeBit(0)
	b.writeBits(3, 5)
	b.writeBit(1)
	b.writeBit(0)
	b.writeBit(1)
	b.writeBits(12, 8)
	b.writeBits(34, 6)
	b.writeBits(56, 6)
	b.writeBits(7, 5)
	b.writeBits(5, 3)
	b.writeBit(0)
	return b.bytes()
}

func seiRecoveryPointPayload() []byte {
	var b seiBitBuilder
	b.writeUE(4)
	b.writeBit(1)
	b.writeBit(0)
	b.writeBits(2, 2)
	return b.bytes()
}

func seiRegisteredA53Payload(cc []byte) []byte {
	if len(cc)%3 != 0 {
		panic("A53 test payload must contain whole three-byte CC entries")
	}
	out := []byte{ituTT35CountryCodeUS, 0x00, 0x31, 'G', 'A', '9', '4', a53UserDataTypeCodeCaption}
	out = append(out, 0x40|uint8(len(cc)/3), 0xff)
	out = append(out, cc...)
	out = append(out, 0xff)
	return out
}

func seiRegisteredAFDPayload(description uint8) []byte {
	return []byte{ituTT35CountryCodeUS, 0x00, 0x31, 'D', 'T', 'G', '1', 0x40, description}
}

func seiRegisteredLCEVCPayload(data []byte) []byte {
	out := []byte{ituTT35CountryCodeUK, 0x00, 0x50, 0x01}
	return append(out, data...)
}

func seiDisplayOrientationPayload() []byte {
	var b seiBitBuilder
	b.writeBit(0)
	b.writeBit(1)
	b.writeBit(0)
	b.writeBits(0x4000, 16)
	return b.bytes()
}

func seiFramePackingPayload() []byte {
	var b seiBitBuilder
	b.writeUE(2)
	b.writeBit(0)
	b.writeBits(3, 7)
	b.writeBit(0)
	b.writeBits(2, 6)
	b.writeBits(0, 3)
	b.writeBit(1)
	b.writeBits(0, 2)
	b.writeBits(0x1234, 16)
	b.writeBits(0, 8)
	b.writeUE(5)
	b.writeBit(0)
	return b.bytes()
}

func seiAmbientViewingPayload() []byte {
	return []byte{0x00, 0x00, 0x30, 0x39, 0x61, 0xa8, 0x41, 0x1b}
}

func seiMasteringDisplayPayload() []byte {
	return []byte{
		0x27, 0x10, 0x4e, 0x20,
		0x3a, 0x98, 0x61, 0xa8,
		0x75, 0x30, 0x88, 0xb8,
		0x3d, 0x13, 0x40, 0x42,
		0x00, 0x98, 0x96, 0x80,
		0x00, 0x00, 0x00, 0x64,
	}
}

func seiFilmGrainPayload() []byte {
	var b seiBitBuilder
	b.writeBit(0)
	b.writeBits(1, 2)
	b.writeBit(1)
	b.writeBits(2, 3)
	b.writeBits(0, 3)
	b.writeBit(1)
	b.writeBits(9, 8)
	b.writeBits(16, 8)
	b.writeBits(9, 8)
	b.writeBits(1, 2)
	b.writeBits(7, 4)
	b.writeBit(1)
	b.writeBit(1)
	b.writeBit(0)
	b.writeBits(0, 8)
	b.writeBits(1, 3)
	b.writeBits(10, 8)
	b.writeBits(20, 8)
	b.writeSE(3)
	b.writeSE(-2)
	b.writeBits(1, 8)
	b.writeBits(0, 3)
	b.writeBits(30, 8)
	b.writeBits(40, 8)
	b.writeSE(-1)
	b.writeBits(41, 8)
	b.writeBits(60, 8)
	b.writeSE(5)
	b.writeUE(4)
	return b.bytes()
}

func seiFilmGrainTooManyValuesPayload() []byte {
	var b seiBitBuilder
	b.writeBit(0)
	b.writeBits(0, 2)
	b.writeBit(0)
	b.writeBits(0, 2)
	b.writeBits(0, 4)
	b.writeBit(1)
	b.writeBit(0)
	b.writeBit(0)
	b.writeBits(0, 8)
	b.writeBits(6, 3)
	return b.bytes()
}

func seiUnregisteredPayload() []byte {
	payload := []byte{
		0x00, 0x01, 0x02, 0x03,
		0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b,
		0x0c, 0x0d, 0x0e, 0x0f,
	}
	return append(payload, []byte("x264 - core 165 r3095")...)
}

type seiBitBuilder struct {
	bits []byte
}

func (b *seiBitBuilder) writeBit(v uint32) {
	if v&1 != 0 {
		b.bits = append(b.bits, 1)
	} else {
		b.bits = append(b.bits, 0)
	}
}

func (b *seiBitBuilder) writeBits(v uint32, n uint32) {
	for i := int(n) - 1; i >= 0; i-- {
		b.writeBit(v >> uint(i))
	}
}

func (b *seiBitBuilder) writeUE(v uint32) {
	codeNum := v + 1
	width := bits.Len32(codeNum)
	for i := 0; i < width-1; i++ {
		b.writeBit(0)
	}
	b.writeBits(codeNum, uint32(width))
}

func (b *seiBitBuilder) writeSE(v int32) {
	var ue uint32
	if v <= 0 {
		ue = uint32(-v) * 2
	} else {
		ue = uint32(v)*2 - 1
	}
	b.writeUE(ue)
}

func (b *seiBitBuilder) bytes() []byte {
	out := make([]byte, (len(b.bits)+7)/8)
	for i, bit := range b.bits {
		if bit != 0 {
			out[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	return out
}
