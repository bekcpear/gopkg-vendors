//go:build !linux && !openbsd && !freebsd

/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2017-2022 WireGuard LLC. All Rights Reserved.
 */

package conn

func (s *StdNetBind) SetMark(mark uint32) error {
	return nil
}