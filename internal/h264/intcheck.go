// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

const maxInt = int(^uint(0) >> 1)

func checkedAddInt(a int, b int) (int, error) {
	if b > 0 && a > maxInt-b {
		return 0, ErrInvalidData
	}
	if b < 0 && a < -maxInt-1-b {
		return 0, ErrInvalidData
	}
	return a + b, nil
}

func checkedMulInt(a int, b int) (int, error) {
	if a == 0 || b == 0 {
		return 0, nil
	}
	if a < 0 || b < 0 {
		return 0, ErrInvalidData
	}
	if a > maxInt/b {
		return 0, ErrInvalidData
	}
	return a * b, nil
}
