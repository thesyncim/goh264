// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import "errors"

var (
	ErrInvalidData = errors.New("h264: invalid data")
	ErrUnsupported = errors.New("h264: unsupported bitstream feature")
)
