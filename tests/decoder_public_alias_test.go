// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	goh264 "github.com/thesyncim/goh264"
)

func TestMain(m *testing.M) {
	_, file, _, ok := runtime.Caller(0)
	if ok {
		if err := os.Chdir(filepath.Dir(filepath.Dir(file))); err != nil {
			panic(err)
		}
	}
	os.Exit(m.Run())
}

var (
	ErrInvalidData = goh264.ErrInvalidData
	ErrUnsupported = goh264.ErrUnsupported
	NewDecoder     = goh264.NewDecoder

	ParseAVCDecoderConfigurationRecord = goh264.ParseAVCDecoderConfigurationRecord
	ParseAVCC                          = goh264.ParseAVCC
	InspectAVCC                        = goh264.InspectAVCC
)

var _ = (*Decoder).ConfigureAVCDecoderConfigurationRecord
var _ = (*Decoder).ConfigureAVCC

type (
	Decoder                   = goh264.Decoder
	StreamInfo                = goh264.StreamInfo
	AVCDecoderConfiguration   = goh264.AVCDecoderConfiguration
	AVCConfig                 = goh264.AVCConfig
	PacketSideDataType        = goh264.PacketSideDataType
	PacketSideData            = goh264.PacketSideData
	Packet                    = goh264.Packet
	FrameSideData             = goh264.FrameSideData
	PictureTiming             = goh264.PictureTiming
	Timecode                  = goh264.Timecode
	RecoveryPoint             = goh264.RecoveryPoint
	BufferingPeriod           = goh264.BufferingPeriod
	GreenMetadata             = goh264.GreenMetadata
	ActiveFormat              = goh264.ActiveFormat
	FramePackingArrangement   = goh264.FramePackingArrangement
	Stereo3DType              = goh264.Stereo3DType
	Stereo3DView              = goh264.Stereo3DView
	Stereo3DPrimaryEye        = goh264.Stereo3DPrimaryEye
	Rational                  = goh264.Rational
	Stereo3D                  = goh264.Stereo3D
	SphericalProjection       = goh264.SphericalProjection
	SphericalMapping          = goh264.SphericalMapping
	ReferenceDisplaysInfo     = goh264.ReferenceDisplaysInfo
	ReferenceDisplay          = goh264.ReferenceDisplay
	DisplayOrientation        = goh264.DisplayOrientation
	AlternativeTransfer       = goh264.AlternativeTransfer
	AmbientViewingEnvironment = goh264.AmbientViewingEnvironment
	FilmGrainCharacteristics  = goh264.FilmGrainCharacteristics
	MasteringDisplay          = goh264.MasteringDisplay
	ContentLight              = goh264.ContentLight
	Frame                     = goh264.Frame
)

const (
	PacketSideDataNewExtradata              = goh264.PacketSideDataNewExtradata
	PacketSideDataDisplayMatrix             = goh264.PacketSideDataDisplayMatrix
	PacketSideDataStereo3D                  = goh264.PacketSideDataStereo3D
	PacketSideDataMasteringDisplayMetadata  = goh264.PacketSideDataMasteringDisplayMetadata
	PacketSideDataSpherical                 = goh264.PacketSideDataSpherical
	PacketSideDataContentLightLevel         = goh264.PacketSideDataContentLightLevel
	PacketSideDataA53ClosedCaptions         = goh264.PacketSideDataA53ClosedCaptions
	PacketSideDataActiveFormat              = goh264.PacketSideDataActiveFormat
	PacketSideDataICCProfile                = goh264.PacketSideDataICCProfile
	PacketSideDataS12MTimecode              = goh264.PacketSideDataS12MTimecode
	PacketSideDataDynamicHDR10Plus          = goh264.PacketSideDataDynamicHDR10Plus
	PacketSideDataAmbientViewingEnvironment = goh264.PacketSideDataAmbientViewingEnvironment
	PacketSideDataLCEVC                     = goh264.PacketSideDataLCEVC
	PacketSideData3DReferenceDisplays       = goh264.PacketSideData3DReferenceDisplays

	Stereo3DType2D                         = goh264.Stereo3DType2D
	Stereo3DTypeSideBySide                 = goh264.Stereo3DTypeSideBySide
	Stereo3DTypeTopBottom                  = goh264.Stereo3DTypeTopBottom
	Stereo3DTypeFrameSequence              = goh264.Stereo3DTypeFrameSequence
	Stereo3DTypeCheckerboard               = goh264.Stereo3DTypeCheckerboard
	Stereo3DTypeSideBySideQuincunx         = goh264.Stereo3DTypeSideBySideQuincunx
	Stereo3DTypeLines                      = goh264.Stereo3DTypeLines
	Stereo3DTypeColumns                    = goh264.Stereo3DTypeColumns
	Stereo3DTypeUnspecified                = goh264.Stereo3DTypeUnspecified
	Stereo3DViewPacked                     = goh264.Stereo3DViewPacked
	Stereo3DViewLeft                       = goh264.Stereo3DViewLeft
	Stereo3DViewRight                      = goh264.Stereo3DViewRight
	Stereo3DViewUnspecified                = goh264.Stereo3DViewUnspecified
	Stereo3DPrimaryEyeNone                 = goh264.Stereo3DPrimaryEyeNone
	Stereo3DPrimaryEyeLeft                 = goh264.Stereo3DPrimaryEyeLeft
	Stereo3DPrimaryEyeRight                = goh264.Stereo3DPrimaryEyeRight
	SphericalProjectionEquirectangular     = goh264.SphericalProjectionEquirectangular
	SphericalProjectionCubemap             = goh264.SphericalProjectionCubemap
	SphericalProjectionEquirectangularTile = goh264.SphericalProjectionEquirectangularTile
	SphericalProjectionHalfEquirectangular = goh264.SphericalProjectionHalfEquirectangular
	SphericalProjectionRectilinear         = goh264.SphericalProjectionRectilinear
	SphericalProjectionFisheye             = goh264.SphericalProjectionFisheye
	SphericalProjectionParametricImmersive = goh264.SphericalProjectionParametricImmersive
)
