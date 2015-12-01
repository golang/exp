// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package x11driver

import (
	"image"
	"image/color"
	"image/draw"
	"log"
	"sync"
	"unsafe"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/render"
	"github.com/BurntSushi/xgb/shm"
	"github.com/BurntSushi/xgb/xproto"

	"golang.org/x/exp/shiny/driver/internal/swizzle"
	"golang.org/x/exp/shiny/screen"
)

type bufferImpl struct {
	s *screenImpl

	addr unsafe.Pointer
	buf  []byte
	rgba image.RGBA
	size image.Point
	xs   shm.Seg

	mu        sync.Mutex
	nUpload   uint32
	reusable  bool
	released  bool
	cleanedUp bool
}

func (b *bufferImpl) Size() image.Point       { return b.size }
func (b *bufferImpl) Bounds() image.Rectangle { return image.Rectangle{Max: b.size} }
func (b *bufferImpl) RGBA() *image.RGBA       { return &b.rgba }

func (b *bufferImpl) preUpload(reusable bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.released {
		panic("x11driver: Buffer.Upload called after Buffer.Release")
	}
	if b.nUpload == 0 {
		swizzle.BGRA(b.buf)
	}
	b.nUpload++
	b.reusable = b.reusable && reusable
}

func (b *bufferImpl) postUpload() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nUpload--
	if b.nUpload != 0 {
		return
	}

	if b.released {
		go b.cleanUp()
	} else if b.reusable {
		swizzle.BGRA(b.buf)
	}
}

func (b *bufferImpl) Release() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.released && b.nUpload == 0 {
		go b.cleanUp()
	}
	b.released = true
}

func (b *bufferImpl) cleanUp() {
	b.mu.Lock()
	if b.cleanedUp {
		b.mu.Unlock()
		panic("x11driver: Buffer clean-up occurred twice")
	}
	b.cleanedUp = true
	b.mu.Unlock()

	b.s.mu.Lock()
	delete(b.s.buffers, b.xs)
	b.s.mu.Unlock()

	shm.Detach(b.s.xc, b.xs)
	if err := shmClose(b.addr); err != nil {
		log.Printf("x11driver: shmClose: %v", err)
	}
}

func (b *bufferImpl) upload(u screen.Uploader, xd xproto.Drawable, xg xproto.Gcontext, depth uint8,
	dp image.Point, sr image.Rectangle) {

	b.preUpload(true)

	// TODO: adjust if dp is outside dst bounds, or sr is outside src bounds.
	dr := sr.Sub(sr.Min).Add(dp)

	b.s.mu.Lock()
	b.s.nPendingUploads++
	b.s.mu.Unlock()

	cookie := shm.PutImage(
		b.s.xc, xd, xg,
		uint16(b.size.X), uint16(b.size.Y), // TotalWidth, TotalHeight,
		uint16(sr.Min.X), uint16(sr.Min.Y), // SrcX, SrcY,
		uint16(dr.Dx()), uint16(dr.Dy()), // SrcWidth, SrcHeight,
		int16(dr.Min.X), int16(dr.Min.Y), // DstX, DstY,
		depth, xproto.ImageFormatZPixmap,
		1, b.xs, 0, // 1 means send a completion event, 0 means a zero offset.
	)

	completion := make(chan struct{})

	b.s.mu.Lock()
	b.s.uploads[cookie.Sequence] = completion
	b.s.nPendingUploads--
	b.s.handleCompletions()
	b.s.mu.Unlock()

	<-completion

	b.postUpload()
}

func fill(xc *xgb.Conn, xp render.Picture, dr image.Rectangle, src color.Color, op draw.Op) {
	r, g, b, a := src.RGBA()
	c := render.Color{
		Red:   uint16(r),
		Green: uint16(g),
		Blue:  uint16(b),
		Alpha: uint16(a),
	}
	x, y := dr.Min.X, dr.Min.Y
	if x < -0x8000 || 0x7fff < x || y < -0x8000 || 0x7fff < y {
		return
	}
	dx, dy := dr.Dx(), dr.Dy()
	if dx < 0 || 0xffff < dx || dy < 0 || 0xffff < dy {
		return
	}
	render.FillRectangles(xc, renderOp(op), xp, c, []xproto.Rectangle{{
		X:      int16(x),
		Y:      int16(y),
		Width:  uint16(dx),
		Height: uint16(dy),
	}})
}
