// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import goh264 "github.com/thesyncim/goh264"

var (
	DefaultEncoderConfig = goh264.DefaultEncoderConfig
	NewEncoder           = goh264.NewEncoder
)

type (
	EncoderPixelFormat          = goh264.EncoderPixelFormat
	EncoderProfile              = goh264.EncoderProfile
	EncoderEntropyMode          = goh264.EncoderEntropyMode
	EncoderDeblockMode          = goh264.EncoderDeblockMode
	EncoderSPSPPSMode           = goh264.EncoderSPSPPSMode
	EncoderRateControlMode      = goh264.EncoderRateControlMode
	EncoderPreset               = goh264.EncoderPreset
	EncoderFrameDropMode        = goh264.EncoderFrameDropMode
	EncoderOutputFormat         = goh264.EncoderOutputFormat
	EncoderRTPPacketizationMode = goh264.EncoderRTPPacketizationMode
	EncoderCrop                 = goh264.EncoderCrop
	EncoderColorConfig          = goh264.EncoderColorConfig
	EncoderConfig               = goh264.EncoderConfig
	EncoderFrame                = goh264.EncoderFrame
	EncoderNALUnit              = goh264.EncoderNALUnit
	EncoderRTPPacket            = goh264.EncoderRTPPacket
	EncoderRTPPayloadFormat     = goh264.EncoderRTPPayloadFormat
	EncoderRTPPacketMetadata    = goh264.EncoderRTPPacketMetadata
	EncoderRTPPacketCallback    = goh264.EncoderRTPPacketCallback
	EncodedFrame                = goh264.EncodedFrame
	EncoderParameterSets        = goh264.EncoderParameterSets
	EncoderSEI                  = goh264.EncoderSEI
	EncoderReconfigure          = goh264.EncoderReconfigure
	EncoderLimits               = goh264.EncoderLimits
	Encoder                     = goh264.Encoder
)

const (
	EncoderPixelFormatI420 = goh264.EncoderPixelFormatI420

	EncoderProfileBaseline            = goh264.EncoderProfileBaseline
	EncoderProfileConstrainedBaseline = goh264.EncoderProfileConstrainedBaseline
	EncoderProfileMain                = goh264.EncoderProfileMain
	EncoderProfileHigh                = goh264.EncoderProfileHigh

	EncoderEntropyCAVLC = goh264.EncoderEntropyCAVLC
	EncoderEntropyCABAC = goh264.EncoderEntropyCABAC

	EncoderDeblockDisabled      = goh264.EncoderDeblockDisabled
	EncoderDeblockEnabled       = goh264.EncoderDeblockEnabled
	EncoderDeblockSliceBoundary = goh264.EncoderDeblockSliceBoundary

	EncoderSPSPPSInBandKeyframes = goh264.EncoderSPSPPSInBandKeyframes
	EncoderSPSPPSOutOfBand       = goh264.EncoderSPSPPSOutOfBand
	EncoderSPSPPSEveryIDR        = goh264.EncoderSPSPPSEveryIDR

	EncoderRateControlCBR        = goh264.EncoderRateControlCBR
	EncoderRateControlVBR        = goh264.EncoderRateControlVBR
	EncoderRateControlConstantQP = goh264.EncoderRateControlConstantQP

	EncoderPresetRealtime = goh264.EncoderPresetRealtime
	EncoderPresetBalanced = goh264.EncoderPresetBalanced
	EncoderPresetQuality  = goh264.EncoderPresetQuality

	EncoderFrameDropDisabled  = goh264.EncoderFrameDropDisabled
	EncoderFrameDropLate      = goh264.EncoderFrameDropLate
	EncoderFrameDropToBitrate = goh264.EncoderFrameDropToBitrate

	EncoderOutputAnnexB = goh264.EncoderOutputAnnexB
	EncoderOutputAVC    = goh264.EncoderOutputAVC
	EncoderOutputRTP    = goh264.EncoderOutputRTP

	EncoderRTPPacketizationSingleNAL      = goh264.EncoderRTPPacketizationSingleNAL
	EncoderRTPPacketizationNonInterleaved = goh264.EncoderRTPPacketizationNonInterleaved

	EncoderRTPPayloadSingleNAL = goh264.EncoderRTPPayloadSingleNAL
	EncoderRTPPayloadSTAPA     = goh264.EncoderRTPPayloadSTAPA
	EncoderRTPPayloadFUA       = goh264.EncoderRTPPayloadFUA
)

var _ = (EncoderConfig).Normalize
var _ = (EncoderConfig).Validate
var _ = (EncoderConfig).ValidateFrame
var _ = (EncoderConfig).I420Frame
var _ = (EncoderConfig).ParameterSets
var _ = (EncoderConfig).RecoveryPointSEIMessage

var _ = (EncoderFrame).Clone
var _ = (EncoderParameterSets).AVCC
var _ = (EncoderParameterSets).AppendSPS
var _ = (EncoderParameterSets).AppendPPS
var _ = (EncoderParameterSets).AppendAnnexB
var _ = (EncoderParameterSets).AppendAVCC
var _ = (EncoderParameterSets).Clone
var _ = (EncoderSEI).AppendNAL
var _ = (EncoderSEI).AppendAnnexB
var _ = (EncoderSEI).AppendAVC
var _ = (EncoderSEI).Clone
var _ = (EncoderRTPPacket).PacketData
var _ = (EncoderRTPPacket).AppendPacketData
var _ = (EncoderRTPPacket).PayloadData
var _ = (EncoderRTPPacket).AppendPayloadData
var _ = (EncoderRTPPacket).Clone
var _ = (EncodedFrame).NALData
var _ = (EncodedFrame).AppendNALData
var _ = (EncodedFrame).AccessUnitData
var _ = (EncodedFrame).AppendAccessUnitData
var _ = (EncodedFrame).RTPPacketData
var _ = (EncodedFrame).AppendRTPPacketData
var _ = (EncodedFrame).RTPPayloadData
var _ = (EncodedFrame).AppendRTPPayloadData
var _ = (EncodedFrame).Clone

var _ = (*Encoder).Config
var _ = (*Encoder).ParameterSets
var _ = (*Encoder).RecoveryPointSEI
var _ = (*Encoder).ValidateFrame
var _ = (*Encoder).I420Frame
var _ = (*Encoder).Encode
var _ = (*Encoder).SetLimits
var _ = (*Encoder).EncodeInto
var _ = (*Encoder).Reset
