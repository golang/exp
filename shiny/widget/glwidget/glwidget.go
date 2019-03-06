// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !android

// Package glwidget provides a widget containing a GL ES framebuffer.
package glwidget

import (
	"fmt"
	"image"
	"image/draw"

	"golang.org/x/exp/shiny/driver/gldriver"
	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/mobile/gl"
)

// GL is a widget that maintains an OpenGL ES context.
//
// The Draw function is responsible for configuring the GL viewport
// and for publishing the result to the widget by calling the Publish
// method when the frame is complete. A typical draw function:
//
//	func(w *glwidget.GL) {
//		w.Ctx.Viewport(0, 0, w.Rect.Dx(), w.Rect.Dy())
//		w.Ctx.ClearColor(0, 0, 0, 1)
//		w.Ctx.Clear(gl.COLOR_BUFFER_BIT)
//		// ... draw the frame
//		w.Publish()
//	}
//
// The GL context is separate from the one used by the gldriver to
// render the window, and is only used by the glwidget package during
// initialization and for the duration of the Publish call. This means
// a glwidget user is free to use Ctx as a background GL context
// concurrently with the primary UI drawing done by the gldriver.
type GL struct {
	node.LeafEmbed

	Ctx gl.Context

	draw        func(*GL)
	framebuffer gl.Framebuffer
	tex         gl.Texture
	dst         *image.RGBA
	origin      image.Point
}

// NewGL creates a GL widget with a Draw function called when painted.
func NewGL(drawFunc func(*GL)) *GL {
	// TODO: use the size of the monitor as a bound for texture size.
	const maxWidth, maxHeight = 4096, 3072

	glctx, err := gldriver.NewContext()
	if err != nil {
		panic(fmt.Sprintf("glwidget: %v", err)) // TODO: return error?
	}
	w := &GL{
		Ctx:  glctx,
		draw: drawFunc,
	}
	w.tex = w.Ctx.CreateTexture()
	w.Ctx.BindTexture(gl.TEXTURE_2D, w.tex)

	w.Ctx.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, maxWidth, maxHeight, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	w.Ctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	w.Ctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	w.Ctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	w.Ctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	w.framebuffer = w.Ctx.CreateFramebuffer()
	w.Ctx.BindFramebuffer(gl.FRAMEBUFFER, w.framebuffer)
	w.Ctx.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, w.tex, 0)

	// TODO: delete the framebuffer, texture, and gl.Context.
	// TODO: explicit or finalizer cleanup?

	w.Wrapper = w

	return w
}

func (w *GL) PaintBase(ctx *node.PaintBaseContext, origin image.Point) error {
	w.Marks.UnmarkNeedsPaintBase()
	if w.Rect.Empty() {
		return nil
	}
	w.dst = ctx.Dst
	w.origin = origin
	w.draw(w)
	w.dst = nil
	return nil
}

// Publish renders the default framebuffer of Ctx onto the area of the
// window occupied by the widget.
func (w *GL) Publish() {
	if w.dst == nil {
		panic("glwidget: no destination, Publish called outside of Draw")
	}
	// TODO: draw the widget texture directly into the window framebuffer.
	m := image.NewRGBA(image.Rect(0, 0, w.Rect.Dx(), w.Rect.Dy()))
	w.Ctx.PixelStorei(gl.PACK_ALIGNMENT, 1)
	w.Ctx.ReadPixels(m.Pix, 0, 0, w.Rect.Dx(), w.Rect.Dy(), gl.RGBA, gl.UNSIGNED_BYTE)
	draw.Draw(w.dst, w.Rect.Add(w.origin), m, image.Point{}, draw.Over)
}
