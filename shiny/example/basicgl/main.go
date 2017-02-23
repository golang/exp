// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build example
//
// This build tag means that "go install golang.org/x/exp/shiny/..." doesn't
// install this example program. Use "go run main.go" to run it or "go install
// -tags=example" to install it.

// Basicgl demonstrates the use of Shiny's glwidget.
package main

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"

	"golang.org/x/exp/shiny/driver/gldriver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/exp/shiny/unit"
	"golang.org/x/exp/shiny/widget"
	"golang.org/x/exp/shiny/widget/flex"
	"golang.org/x/exp/shiny/widget/glwidget"
	"golang.org/x/exp/shiny/widget/theme"
	"golang.org/x/image/colornames"
	"golang.org/x/mobile/gl"
)

func colorPatch(c color.Color, w, h unit.Value) *widget.Sizer {
	return widget.NewSizer(w, h, widget.NewUniform(theme.StaticColor(c), nil))
}

func main() {
	gldriver.Main(func(s screen.Screen) {
		t1, t2 := newTriangleGL(), newTriangleGL()
		defer t1.cleanup()
		defer t2.cleanup()

		body := widget.NewSheet(flex.NewFlex(
			colorPatch(colornames.Green, unit.Pixels(50), unit.Pixels(50)),
			widget.WithLayoutData(t1.w, flex.LayoutData{Grow: 1, Align: flex.AlignItemStretch}),
			colorPatch(colornames.Blue, unit.Pixels(50), unit.Pixels(50)),
			widget.WithLayoutData(t2.w, flex.LayoutData{MinSize: image.Point{80, 80}}),
			colorPatch(colornames.Green, unit.Pixels(50), unit.Pixels(50)),
		))

		if err := widget.RunWindow(s, body, &widget.RunWindowOptions{
			NewWindowOptions: screen.NewWindowOptions{
				Title: "BasicGL Shiny Example",
			},
		}); err != nil {
			log.Fatal(err)
		}
	})
}

func newTriangleGL() *triangleGL {
	t := new(triangleGL)
	t.w = glwidget.NewGL(t.draw)
	t.init()
	return t
}

type triangleGL struct {
	w *glwidget.GL

	program  gl.Program
	position gl.Attrib
	offset   gl.Uniform
	color    gl.Uniform
	buf      gl.Buffer

	green float32
}

func (t *triangleGL) init() {
	glctx := t.w.Ctx
	var err error
	t.program, err = createProgram(glctx, vertexShader, fragmentShader)
	if err != nil {
		log.Fatalf("error creating GL program: %v", err)
	}

	t.buf = glctx.CreateBuffer()
	glctx.BindBuffer(gl.ARRAY_BUFFER, t.buf)
	glctx.BufferData(gl.ARRAY_BUFFER, triangleData, gl.STATIC_DRAW)

	t.position = glctx.GetAttribLocation(t.program, "position")
	t.color = glctx.GetUniformLocation(t.program, "color")
	t.offset = glctx.GetUniformLocation(t.program, "offset")

	glctx.UseProgram(t.program)
	glctx.ClearColor(1, 0, 0, 1)
}

func (t *triangleGL) cleanup() {
	glctx := t.w.Ctx
	glctx.DeleteProgram(t.program)
	glctx.DeleteBuffer(t.buf)
}

func (t *triangleGL) draw(w *glwidget.GL) {
	glctx := t.w.Ctx

	glctx.Viewport(0, 0, w.Rect.Dx(), w.Rect.Dy())
	glctx.Clear(gl.COLOR_BUFFER_BIT)

	t.green += 0.01
	if t.green > 1 {
		t.green = 0
	}
	glctx.Uniform4f(t.color, 0, t.green, 0, 1)
	glctx.Uniform2f(t.offset, 0.2, 0.9)

	glctx.BindBuffer(gl.ARRAY_BUFFER, t.buf)
	glctx.EnableVertexAttribArray(t.position)
	glctx.VertexAttribPointer(t.position, coordsPerVertex, gl.FLOAT, false, 0, 0)
	glctx.DrawArrays(gl.TRIANGLES, 0, vertexCount)
	glctx.DisableVertexAttribArray(t.position)
	w.Publish()
}

// asBytes returns the byte representation of float32 values in the given byte
// order. byteOrder must be either binary.BigEndian or binary.LittleEndian.
func asBytes(byteOrder binary.ByteOrder, values ...float32) []byte {
	le := false
	switch byteOrder {
	case binary.BigEndian:
	case binary.LittleEndian:
		le = true
	default:
		panic(fmt.Sprintf("invalid byte order %v", byteOrder))
	}

	b := make([]byte, 4*len(values))
	for i, v := range values {
		u := math.Float32bits(v)
		if le {
			b[4*i+0] = byte(u >> 0)
			b[4*i+1] = byte(u >> 8)
			b[4*i+2] = byte(u >> 16)
			b[4*i+3] = byte(u >> 24)
		} else {
			b[4*i+0] = byte(u >> 24)
			b[4*i+1] = byte(u >> 16)
			b[4*i+2] = byte(u >> 8)
			b[4*i+3] = byte(u >> 0)
		}
	}
	return b
}

// createProgram creates, compiles, and links a gl.Program.
func createProgram(glctx gl.Context, vertexSrc, fragmentSrc string) (gl.Program, error) {
	program := glctx.CreateProgram()
	if program.Value == 0 {
		return gl.Program{}, fmt.Errorf("basicgl: no programs available")
	}

	vertexShader, err := loadShader(glctx, gl.VERTEX_SHADER, vertexSrc)
	if err != nil {
		return gl.Program{}, err
	}
	fragmentShader, err := loadShader(glctx, gl.FRAGMENT_SHADER, fragmentSrc)
	if err != nil {
		glctx.DeleteShader(vertexShader)
		return gl.Program{}, err
	}

	glctx.AttachShader(program, vertexShader)
	glctx.AttachShader(program, fragmentShader)
	glctx.LinkProgram(program)

	// Flag shaders for deletion when program is unlinked.
	glctx.DeleteShader(vertexShader)
	glctx.DeleteShader(fragmentShader)

	if glctx.GetProgrami(program, gl.LINK_STATUS) == 0 {
		defer glctx.DeleteProgram(program)
		return gl.Program{}, fmt.Errorf("basicgl: %s", glctx.GetProgramInfoLog(program))
	}
	return program, nil
}

func loadShader(glctx gl.Context, shaderType gl.Enum, src string) (gl.Shader, error) {
	shader := glctx.CreateShader(shaderType)
	if shader.Value == 0 {
		return gl.Shader{}, fmt.Errorf("basicgl: could not create shader (type %v)", shaderType)
	}
	glctx.ShaderSource(shader, src)
	glctx.CompileShader(shader)
	if glctx.GetShaderi(shader, gl.COMPILE_STATUS) == 0 {
		defer glctx.DeleteShader(shader)
		return gl.Shader{}, fmt.Errorf("basicgl: shader compile: %s", glctx.GetShaderInfoLog(shader))
	}
	return shader, nil
}

var triangleData = asBytes(binary.LittleEndian,
	0.0, 0.4, 0.0, // top left
	0.0, 0.0, 0.0, // bottom left
	0.4, 0.0, 0.0, // bottom right
)

const (
	coordsPerVertex = 3
	vertexCount     = 3
)

const vertexShader = `#version 100
uniform vec2 offset;

attribute vec4 position;
void main() {
	// offset comes in with x/y values between 0 and 1.
	// position bounds are -1 to 1.
	vec4 offset4 = vec4(2.0*offset.x-1.0, 1.0-2.0*offset.y, 0, 0);
	gl_Position = position + offset4;
}`

const fragmentShader = `#version 100
precision mediump float;
uniform vec4 color;
void main() {
	gl_FragColor = color;
}`
