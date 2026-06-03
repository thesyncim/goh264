// SPDX-License-Identifier: LGPL-2.1-or-later

package goh264

import (
	"errors"
	"testing"
)

const testsrc16High10CAVLCReferenceBoundaryAnnexBHex = `
00000001676e000aa6cb4f42000003000200000300051e244d400000000168ce0f2c8b00000165888432a218ab030217e0214c880004026b50618003c5e33a31a3a05f77ff81a40d900d01ca08b0cad5e6a86cce6e5bd76d22637cc4ca82fabbfc4fb291c84f22c08
ffec16cb43486e123fb6b86906110b49f07abd757181b3284d3d5ea03e4211314054fff7cb079d9f57068abfef83020004e618001518190013e6615663a0422fa93405b3a603213ec353f903b2b7f738004a82119748d200375c51558d10d09e28ee6d62e9fd846c959808d606feea03a914b82c4d2bbe8
00000001419a212f0ea1c3c1f40f1c01961d5ab4fe7ef7fc04c9189eb57c20003e01c6ba111220003d1090c1cae78951f2ab83e3cf9c7342b6f00071c14b8e9eab2957a103326c7f3f36d03f
`

func TestDecodeConfiguredAVCHigh10RetainsReferenceForResidualP(t *testing.T) {
	data := decodeHexFixture(t, testsrc16High10CAVLCReferenceBoundaryAnnexBHex)
	config, samples := annexBToAVCConfigAndSamples(t, data, 4)
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	fresh := NewDecoder()
	if _, err := fresh.ParseAVCDecoderConfigurationRecord(config); err != nil {
		t.Fatal(err)
	}
	if _, err := fresh.DecodeConfiguredAVC(samples[1]); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("P sample without retained reference err = %v, want ErrUnsupported", err)
	}

	dec := NewDecoder()
	if _, err := dec.ParseAVCDecoderConfigurationRecord(config); err != nil {
		t.Fatal(err)
	}
	first, err := dec.DecodeConfiguredAVC(samples[0])
	if err != nil {
		t.Fatalf("IDR sample decode: %v", err)
	}
	assertHigh10FrameMD5Strings(t, []*Frame{first}, []string{"fd302f00e365b8502c44005ea308c468"})

	second, err := dec.DecodeConfiguredAVC(samples[1])
	if err != nil {
		t.Fatalf("P sample with retained reference decode: %v", err)
	}
	assertHigh10FrameMD5Strings(t, []*Frame{second}, []string{"df16162e1c5420c45702aee7bb936b15"})
}
