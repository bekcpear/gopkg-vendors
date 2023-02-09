/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2017-2022 WireGuard LLC. All Rights Reserved.
 */

package device

import (
	"crypto/cipher"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tailscale/wireguard-go/replay"
)

/* Due to limitations in Go and /x/crypto there is currently
 * no way to ensure that key material is securely ereased in memory.
 *
 * Since this may harm the forward secrecy property,
 * we plan to resolve this issue; whenever Go allows us to do so.
 */

type Keypair struct {
	sendNonce    atomic.Uint64
	send         cipher.AEAD
	receive      cipher.AEAD
	replayFilter replay.Filter
	isInitiator  bool
	created      time.Time
	localIndex   uint32
	remoteIndex  uint32
}

type Keypairs struct {
	sync.Mutex
	current  *Keypair
	previous *Keypair
	next     *Keypair
}

func (kp *Keypairs) Current() *Keypair {
	kp.Lock()
	defer kp.Unlock()
	return kp.current
}

func (device *Device) DeleteKeypair(key *Keypair) {
	if key != nil {
		device.indexTable.Delete(key.localIndex)
	}
}