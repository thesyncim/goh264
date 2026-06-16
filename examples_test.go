// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264_test

import (
	"log"
	"os"

	"github.com/thesyncim/goh264"
)

func ExampleDecoder_DecodeFrames() {
	data, err := os.ReadFile("testdata/h264/high10_inter_cavlc_idrp.h264")
	if err != nil {
		log.Fatal(err)
	}

	dec := goh264.NewDecoder()
	frames, err := dec.DecodeFrames(data)
	if err != nil {
		log.Fatal(err)
	}
	for _, frame := range frames {
		_, err := frame.AppendRawYUVBytesLE(nil)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func ExampleInspectAnnexBHeaders() {
	data, err := os.ReadFile("testdata/h264/high10_inter_cavlc_idrp.h264")
	if err != nil {
		log.Fatal(err)
	}

	if _, err := goh264.InspectAnnexBHeaders(data); err != nil {
		log.Fatal(err)
	}
}
