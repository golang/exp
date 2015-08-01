// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gldriver

import (
	"encoding/binary"
	"image"

	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/gl"
)

type textureImpl struct {
	id   gl.Texture
	size image.Point
}

func (t *textureImpl) Size() image.Point       { return t.size }
func (t *textureImpl) Bounds() image.Rectangle { return image.Rectangle{Max: t.size} }

func (t *textureImpl) Release() {
	gl.DeleteTexture(t.id)
	t.id = gl.Texture{}
}

func (t *textureImpl) Upload(dp image.Point, src screen.Buffer, sr image.Rectangle, sender screen.Sender) {
	// TODO: adjust if dp is outside dst bounds, or sr is outside src bounds.
	gl.BindTexture(gl.TEXTURE_2D, t.id)
	m := src.RGBA().SubImage(sr).(*image.RGBA)
	b := m.Bounds()
	// TODO check m bounds smaller than t.size
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, b.Dx(), b.Dy(), gl.RGBA, gl.UNSIGNED_BYTE, m.Pix)
	// TODO: send a screen.UploadedEvent.
}

var quadXYCoords = f32Bytes(binary.LittleEndian,
	-1, +1, // top left
	+1, +1, // top right
	-1, -1, // bottom left
	+1, -1, // bottom right
)

var quadUVCoords = f32Bytes(binary.LittleEndian,
	0, 0, // top left
	1, 0, // top right
	0, 1, // bottom left
	1, 1, // bottom right
)

const vertexShaderSrc = `#version 100
uniform mat3 mvp;
uniform mat3 uvp;
attribute vec3 pos;
attribute vec2 inUV;
varying vec2 uv;
void main() {
	vec3 p = pos;
	p.z = 1.0;
	gl_Position = vec4(mvp * p, 1);
	uv = (uvp * vec3(inUV, 1)).xy;
}
`

const fragmentShaderSrc = `#version 100
precision mediump float;
varying vec2 uv;
uniform sampler2D sample;
void main() {
	gl_FragColor = texture2D(sample, uv);
}
`
