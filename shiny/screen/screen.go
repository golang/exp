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
	"image/color"
	"image/draw"

	"golang.org/x/image/math/f64"
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

// TODO: rename Buffer to Image, to be less confusing with a Window's back and
// front buffers.

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

// Window is a top-level, double-buffered GUI window.
type Window interface {
	// Release closes the window and its event channel.
	Release()

	// Events returns the window's event channel, which carries key, mouse,
	// paint and other events.
	//
	// TODO: define and describe these events.
	Events() <-chan interface{}

	// Send sends an event to the window.
	Send(event interface{})

	Uploader

	Drawer

	// Publish flushes any pending Upload and Draw calls to the window, and
	// swaps the back buffer to the front.
	Publish() PublishResult
}

// PublishResult is the result of an Window.Publish call.
type PublishResult struct {
	// BackBufferPreserved is whether the contents of the back buffer was
	// preserved. If false, the contents are undefined.
	BackBufferPreserved bool
}

// NewWindowOptions are optional arguments to NewWindow.
type NewWindowOptions struct {
	// Width and Height specify the dimensions of the new window. If Width
	// or Height are zero, a driver-dependent default will be used for each
	// zero value dimension.
	Width, Height int

	// TODO: fullscreen, title, icon, cursorHidden?
}

// Uploader is something you can upload a Buffer to.
type Uploader interface {
	// Upload uploads the sub-Buffer defined by src and sr to the destination
	// (the method receiver), such that sr.Min in src-space aligns with dp in
	// dst-space. The destination's contents are overwritten; the draw operator
	// is implicitly draw.Src.
	//
	// It is valid to upload a Buffer while another upload of the same Buffer
	// is in progress, but a Buffer's image.RGBA pixel contents should not be
	// accessed while it is uploading. A Buffer is re-usable, in that its pixel
	// contents can be further modified, once all outstanding calls to Upload
	// have returned.
	//
	// When uploading to a Window, there will not be any visible effect until
	// Publish is called.
	Upload(dp image.Point, src Buffer, sr image.Rectangle)

	// Fill fills that part of the destination (the method receiver) defined by
	// dr with the given color.
	//
	// When filling a Window, there will not be any visible effect until
	// Publish is called.
	Fill(dr image.Rectangle, src color.Color, op draw.Op)
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
	// When drawing on a Window, there will not be any visible effect until
	// Publish is called.
	Draw(src2dst f64.Aff3, src Texture, sr image.Rectangle, op draw.Op, opts *DrawOptions)
}

// Copy copies the sub-Texture defined by src and sr to dst, such that sr.Min
// in src-space aligns with dp in dst-space.
//
// When drawing on a Window, there will not be any visible effect until Publish
// is called.
func Copy(dst Drawer, dp image.Point, src Texture, sr image.Rectangle, op draw.Op, opts *DrawOptions) {
	dst.Draw(f64.Aff3{
		1, 0, float64(dp.X - sr.Min.X),
		0, 1, float64(dp.Y - sr.Min.Y),
	}, src, sr, op, opts)
}

// Scale scales the sub-Texture defined by src and sr to dst, such that sr in
// src-space is mapped to dr in dst-space.
//
// When drawing on a Window, there will not be any visible effect until Publish
// is called.
func Scale(dst Drawer, dr image.Rectangle, src Texture, sr image.Rectangle, op draw.Op, opts *DrawOptions) {
	rx := float64(dr.Dx()) / float64(sr.Dx())
	ry := float64(dr.Dy()) / float64(sr.Dy())
	dst.Draw(f64.Aff3{
		rx, 0, float64(dr.Min.X) - rx*float64(sr.Min.X),
		0, ry, float64(dr.Min.Y) - ry*float64(sr.Min.Y),
	}, src, sr, op, opts)
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
