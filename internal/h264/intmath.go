// SPDX-License-Identifier: LGPL-2.1-or-later
//
// Source-shaped ports of the small libavutil/intmath.h helpers used by the
// H.264 decoder path.

package h264

import "math/bits"

func avLog2(v uint32) int {
	return bits.Len32(v|1) - 1
}
