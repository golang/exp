// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gldriver

import (
	"image"
	"sync"

	"golang.org/x/exp/shiny/driver/internal/pump"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/gl"
)

var theScreen = &screenImpl{
	windows: make(map[uintptr]*windowImpl),
}

type screenImpl struct {
	mu      sync.Mutex
	windows map[uintptr]*windowImpl
	texture struct {
		program gl.Program
		pos     gl.Attrib
		mvp     gl.Uniform
		uvp     gl.Uniform
		inUV    gl.Attrib
		sample  gl.Uniform
		quadXY  gl.Buffer
		quadUV  gl.Buffer
	}
	fill struct {
		program gl.Program
		pos     gl.Attrib
		mvp     gl.Uniform
		color   gl.Uniform
		quadXY  gl.Buffer
	}
}

func (s *screenImpl) NewBuffer(size image.Point) (retBuf screen.Buffer, retErr error) {
	return &bufferImpl{
		rgba: image.NewRGBA(image.Rectangle{Max: size}),
		size: size,
	}, nil
}

func (s *screenImpl) NewTexture(size image.Point) (screen.Texture, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !gl.IsProgram(s.texture.program) {
		p, err := compileProgram(textureVertexSrc, textureFragmentSrc)
		if err != nil {
			return nil, err
		}
		s.texture.program = p
		s.texture.pos = gl.GetAttribLocation(p, "pos")
		s.texture.mvp = gl.GetUniformLocation(p, "mvp")
		s.texture.uvp = gl.GetUniformLocation(p, "uvp")
		s.texture.inUV = gl.GetAttribLocation(p, "inUV")
		s.texture.sample = gl.GetUniformLocation(p, "sample")
		s.texture.quadXY = gl.CreateBuffer()
		s.texture.quadUV = gl.CreateBuffer()

		gl.BindBuffer(gl.ARRAY_BUFFER, s.texture.quadXY)
		gl.BufferData(gl.ARRAY_BUFFER, quadXYCoords, gl.STATIC_DRAW)
		gl.BindBuffer(gl.ARRAY_BUFFER, s.texture.quadUV)
		gl.BufferData(gl.ARRAY_BUFFER, quadUVCoords, gl.STATIC_DRAW)
	}

	t := &textureImpl{
		id:   gl.CreateTexture(),
		size: size,
	}

	gl.BindTexture(gl.TEXTURE_2D, t.id)
	gl.TexImage2D(gl.TEXTURE_2D, 0, size.X, size.Y, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	return t, nil
}

func (s *screenImpl) NewWindow(opts *screen.NewWindowOptions) (screen.Window, error) {
	// TODO: look at opts.
	const width, height = 1024, 768

	// TODO: merge the newWindow, showWindow (and drawLoop?) functions.
	id := newWindow(width, height)
	w := &windowImpl{
		s:        s,
		id:       id,
		ctx:      showWindow(id),
		pump:     pump.Make(),
		publish:  make(chan struct{}, 1),
		draw:     make(chan struct{}),
		drawDone: make(chan struct{}),
	}

	s.mu.Lock()
	s.windows[id] = w
	s.mu.Unlock()

	go drawLoop(w)

	return w, nil
}
