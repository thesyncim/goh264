// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"errors"
	"fmt"
	"testing"
)

func TestDecodePPSRejectsFMOLikeFFmpeg(t *testing.T) {
	sps, err := DecodeSPS(buildSPSRBSP(t, 66, 30, 1, false, nil))
	if err != nil {
		t.Fatal(err)
	}
	var spsList [maxSPSCount]*SPS
	spsList[0] = sps

	for mapType := uint32(0); mapType <= 6; mapType++ {
		mapType := mapType
		t.Run(fmt.Sprintf("map-type-%d", mapType), func(t *testing.T) {
			_, err := DecodePPS(ppsWithSliceGroupsRBSP(mapType), &spsList)
			if !errors.Is(err, ErrUnsupported) {
				t.Fatalf("map_type=%d: err = %v, want ErrUnsupported", mapType, err)
			}
		})
	}
}

func ppsWithSliceGroupsRBSP(mapType uint32) []byte {
	var b spsBitBuilder
	b.writeUE(0)       // pic_parameter_set_id
	b.writeUE(0)       // seq_parameter_set_id
	b.writeBit(0)      // entropy_coding_mode_flag
	b.writeBit(0)      // bottom_field_pic_order_in_frame_present_flag
	b.writeUE(1)       // num_slice_groups_minus1 => FMO
	b.writeUE(mapType) // slice_group_map_type
	return b.rbsp()
}
