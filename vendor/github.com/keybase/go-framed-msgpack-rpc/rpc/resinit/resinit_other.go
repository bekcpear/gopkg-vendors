// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

//go:build !linux || android || noresinit
// +build !linux android noresinit

package resinit

func resInit() {
	// no-op
}
