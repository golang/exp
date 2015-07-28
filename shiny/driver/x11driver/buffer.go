// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package x11driver

import (
	"image"
	"log"
	"sync"
	"unsafe"

	"github.com/BurntSushi/xgb/shm"
)

type bufferImpl struct {
	s *screenImpl

	addr unsafe.Pointer
	buf  []byte
	rgba image.RGBA
	size image.Point
	xs   shm.Seg

	mu       sync.Mutex
	released bool
}

func (b *bufferImpl) Release() {
	if b.release() {
		b.cleanUp()
	}
}

// release returns whether the caller should clean up.
//
// TODO: don't clean up while uploading.
func (b *bufferImpl) release() (ret bool) {
	b.mu.Lock()
	ret, b.released = !b.released, true
	b.mu.Unlock()
	return ret
}

func (b *bufferImpl) cleanUp() {
	shm.Detach(b.s.xc, b.xs)
	if err := shmClose(b.addr); err != nil {
		log.Printf("x11driver: shmClose: %v", err)
	}
}

func (b *bufferImpl) Size() image.Point {
	return b.size
}

func (b *bufferImpl) RGBA() *image.RGBA {
	return &b.rgba
}
