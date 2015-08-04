// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package screen provides interfaces for portable two-dimensional graphics and
// input events.
//
// Screens are not created directly. Instead, driver packages provide access to
// the screen through a Main function that is designed to be called by the
// program's main function. The golang.org/x/exp/shiny/driver package provides
// the default driver for the system, such as the X11 driver for desktop Linux,
// but other drivers, such as the OpenGL driver, can be explicitly invoked by
// calling that driver's Main function. To use the default driver:
//
//	package main
//
//	import (
//		"golang.org/x/exp/shiny/driver"
//		"golang.org/x/exp/shiny/screen"
//	)
//
//	func main() {
//		driver.Main(func(s screen.Screen) {
//			w, err := s.NewWindow(nil)
//			if err != nil {
//				handleError(err)
//				return
//			}
//			for e := range w.Events() {
//				handleEvent(e)
//			}
//		})
//	}
package screen

import (
	"image"
	"image/draw"

	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/paint"
)

// TODO: specify image format (Alpha or Gray, not just RGBA) for NewBuffer
// and/or NewTexture?

// Screen creates Buffers, Textures and Windows.
type Screen interface {
	// NewBuffer returns a new Buffer for this screen.
	NewBuffer(size image.Point) (Buffer, error)

	// NewTexture returns a new Texture for this screen.
	NewTexture(size image.Point) (Texture, error)

	// NewWindow returns a new Window for this screen.
	NewWindow(opts *NewWindowOptions) (Window, error)
}

// Buffer is an in-memory pixel buffer. Its pixels can be modified by any Go
// code that takes an *image.RGBA, such as the standard library's image/draw
// package.
//
// To see a Buffer's contents on a screen, upload it to a Texture (and then
// draw the Texture on a Window) or upload it directly to a Window.
//
// When specifying a sub-Buffer via Upload, a Buffer's top-left pixel is always
// (0, 0) in its own coordinate space.
type Buffer interface {
	// Release releases the Buffer's resources, after all pending uploads and
	// draws resolve. The behavior of the Buffer after Release is undefined.
	Release()

	// Size returns the size of the Buffer's image.
	Size() image.Point

	// Bounds returns the bounds of the Buffer's image. It is equal to
	// image.Rectangle{Max: b.Size()}.
	Bounds() image.Rectangle

	// RGBA returns the pixel buffer as an *image.RGBA.
	//
	// Its contents should not be accessed while the Buffer is uploading.
	RGBA() *image.RGBA
}

// Texture is a pixel buffer, but not one that is directly accessible as a
// []byte. Conceptually, it could live on a GPU, in another process or even be
// across a network, instead of on a CPU in this process.
//
// Buffers can be uploaded to Textures, and Textures can be drawn on Windows.
//
// When specifying a sub-Texture via Draw, a Texture's top-left pixel is always
// (0, 0) in its own coordinate space.
type Texture interface {
	// Release releases the Texture's resources, after all pending uploads and
	// draws resolve. The behavior of the Texture after Release is undefined.
	Release()

	// Size returns the size of the Texture's image.
	Size() image.Point

	// Bounds returns the bounds of the Texture's image. It is equal to
	// image.Rectangle{Max: t.Size()}.
	Bounds() image.Rectangle

	Uploader

	// TODO: also implement Drawer? If so, merge the Uploader and Drawer
	// interfaces??
}

// Window is a top-level GUI window.
type Window interface {
	// Release closes the window and its event channel.
	Release()

	// Events returns the window's event channel, which carries key, mouse,
	// paint and other events.
	//
	// TODO: define and describe these events.
	Events() <-chan interface{}

	Sender

	Uploader

	Drawer

	// EndPaint flushes any pending Upload and Draw calls to the window.
	// If EndPaint is called with an old generation number, it is ignored.
	EndPaint(p paint.Event)
}

// NewWindowOptions are optional arguments to NewWindow.
type NewWindowOptions struct {
	// TODO: size, fullscreen, title, icon, cursorHidden?
}

// Sender is something you can send events to.
type Sender interface {
	// Send sends an event.
	Send(event interface{})
}

// Uploader is something you can upload a Buffer to.
type Uploader interface {
	// Upload uploads the sub-Buffer defined by src and sr to the destination
	// (the method receiver), such that sr.Min in src-space aligns with dp in
	// dst-space. If sender is not nil, an UploadedEvent will be sent to sender
	// after the upload is complete.
	//
	// It is valid to upload a Buffer while another upload of the same Buffer
	// is in progress, but a Buffer's image.RGBA pixel contents should not be
	// accessed while it is uploading. A Buffer is re-usable, in that its pixel
	// contents can be further modified, once all of its UploadedEvents have
	// been received.
	//
	// When uploading to a Window, there might not be any visible effect until
	// EndPaint is called.
	Upload(dp image.Point, src Buffer, sr image.Rectangle, sender Sender)
}

// UploadedEvent records that a Buffer was uploaded.
type UploadedEvent struct {
	Buffer   Buffer
	Uploader Uploader
}

// TODO: have a Downloader interface? Not every graphical app needs to be
// interactive or involve a window. You could use the GPU for hardware-
// accelerated image manipulation: upload a buffer, do some texture ops, then
// download the result.

// Drawer is something you can draw Textures on.
type Drawer interface {
	// Draw draws the sub-Texture defined by src and sr to the destination (the
	// method receiver). src2dst defines how to transform src coordinates to
	// dst coordinates. For example, if src2dst is the matrix
	//
	// m00 m01 m02
	// m10 m11 m12
	//
	// then the src-space point (sx, sy) maps to the dst-space point
	// (m00*sx + m01*sy + m02, m10*sx + m11*sy + m12).
	//
	// When drawing on a Window, there might not be any visible effect until
	// EndPaint is called.
	Draw(src2dst f64.Aff3, src Texture, sr image.Rectangle, op draw.Op, opts *DrawOptions)
}

// Copy copies the sub-Texture defined by src and sr to dst, such that sr.Min
// in src-space aligns with dp in dst-space.
//
// When drawing on a Window, there might not be any visible effect until
// EndPaint is called.
func Copy(dst Drawer, dp image.Point, src Texture, sr image.Rectangle, op draw.Op, opts *DrawOptions) {
	// TODO.
}

// Scale scales the sub-Texture defined by src and sr to dst, such that sr in
// src-space is mapped to dr in dst-space.
//
// When drawing on a Window, there might not be any visible effect until
// EndPaint is called.
func Scale(dst Drawer, dr image.Rectangle, src Texture, sr image.Rectangle, op draw.Op, opts *DrawOptions) {
	// TODO.
}

// These draw.Op constants are provided so that users of this package don't
// have to explicitly import "image/draw".
const (
	Over = draw.Over
	Src  = draw.Src
)

// DrawOptions are optional arguments to Draw.
type DrawOptions struct {
	// TODO: transparency in [0x0000, 0xffff]?
	// TODO: scaler (nearest neighbor vs linear)?
}
