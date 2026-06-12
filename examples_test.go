// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"fmt"
	"log"

	"github.com/thesyncim/goh264"
)

func ExampleDecoder_DecodeFrames() {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.OutputFormat = goh264.EncoderOutputAnnexB
	cfg.RTPMaxPayloadSize = 0

	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		log.Fatal(err)
	}

	frame := cfg.I420Frame(
		make([]byte, cfg.StrideY*cfg.Height),
		make([]byte, cfg.StrideCb*(cfg.Height/2)),
		make([]byte, cfg.StrideCr*(cfg.Height/2)),
		0,
	)
	encoded, err := enc.Encode(frame)
	if err != nil {
		log.Fatal(err)
	}

	dec := goh264.NewDecoder()
	frames, err := dec.DecodeFrames(encoded.Data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(frames), frames[0].KeyFrame)
	// Output: 1 true
}

func ExampleDecoder_DecodeConfiguredAVCFrames() {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.OutputFormat = goh264.EncoderOutputAVC
	cfg.RTPMaxPayloadSize = 0

	headers, err := cfg.ParameterSets()
	if err != nil {
		log.Fatal(err)
	}
	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		log.Fatal(err)
	}

	frame := cfg.I420Frame(
		make([]byte, cfg.StrideY*cfg.Height),
		make([]byte, cfg.StrideCb*(cfg.Height/2)),
		make([]byte, cfg.StrideCr*(cfg.Height/2)),
		0,
	)
	encoded, err := enc.Encode(frame)
	if err != nil {
		log.Fatal(err)
	}

	dec := goh264.NewDecoder()
	avcc, err := dec.ParseAVCC(headers.AVCC())
	if err != nil {
		log.Fatal(err)
	}
	frames, err := dec.DecodeConfiguredAVCFrames(encoded.Data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(avcc.NALLengthSize, len(frames), frames[0].Width, frames[0].Height)
	// Output: 4 1 16 16
}

func ExampleEncoder_EncodeInto() {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.OutputFormat = goh264.EncoderOutputAnnexB
	cfg.RTPMaxPayloadSize = 0
	cfg.MaxEncodeTimeUS = 0

	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := enc.SetBitrate(700_000, 900_000); err != nil {
		log.Fatal(err)
	}
	if err := enc.SetQP(26, 10, 42); err != nil {
		log.Fatal(err)
	}

	frame := enc.I420Frame(
		make([]byte, cfg.StrideY*cfg.Height),
		make([]byte, cfg.StrideCb*(cfg.Height/2)),
		make([]byte, cfg.StrideCr*(cfg.Height/2)),
		0,
	)
	if err := enc.ValidateFrame(frame); err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, 0, 4096)
	encoded, err := enc.EncodeInto(buf, frame)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(encoded.NALUnits), encoded.KeyFrame)
	// Output: 3 true
}

func ExampleEncoder_SetRTPPacketCallback() {
	cfg := goh264.DefaultEncoderConfig(16, 16)
	cfg.FrameDrop = goh264.EncoderFrameDropDisabled
	cfg.MaxEncodeTimeUS = 0
	cfg.RTPMaxPayloadSize = 1200

	enc, err := goh264.NewEncoder(cfg)
	if err != nil {
		log.Fatal(err)
	}

	var first goh264.EncoderRTPPacketMetadata
	callbacks := 0
	enc.SetRTPPacketCallback(func(_ goh264.EncoderRTPPacket, meta goh264.EncoderRTPPacketMetadata) {
		if callbacks == 0 {
			first = meta
		}
		callbacks++
	})

	frame := enc.I420Frame(
		make([]byte, cfg.StrideY*cfg.Height),
		make([]byte, cfg.StrideCb*(cfg.Height/2)),
		make([]byte, cfg.StrideCr*(cfg.Height/2)),
		0,
	)
	encoded, err := enc.Encode(frame)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(encoded.IDR, len(encoded.RTPPackets), callbacks, first.PacketIndex, first.PacketCount, first.IDR)
	// Output: true 3 3 0 3 true
}
