// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package widget

import (
	"image"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

// Palette is a set of colors for a theme.
//
// The colors are expressed as *image.Uniform values so that they can be easily
// passed as the src argument to image/draw functions.
type Palette struct {
	// Light, Neutral and Dark are three color tones used to fill in widgets
	// such as buttons, menu bars and panels.
	Light   *image.Uniform
	Neutral *image.Uniform
	Dark    *image.Uniform

	// Accent is the color used to accentuate selections or suggestions.
	Accent *image.Uniform

	// Foreground is the color used for text, dividers and icons.
	Foreground *image.Uniform

	// Background is the color used behind large blocks of text. Short,
	// non-editable label text will typically be on the Neutral color.
	Background *image.Uniform
}

// FontFaceOptions allows asking for font face variants, such as style (e.g.
// italic) or weight (e.g. bold).
//
// TODO: include font.Hinting and font.Stretch typed fields?
//
// TODO: include font size? If so, directly as "12pt" or indirectly as an enum
// (Heading1, Heading2, Body, etc)?
type FontFaceOptions struct {
	Style  font.Style
	Weight font.Weight
}

// Theme is a set of colors and font faces.
type Theme interface {
	// Palette returns the color palette for this theme.
	Palette() Palette

	// AcquireFontFace returns a font.Face for this theme. ReleaseFontFace
	// should be called, with the same options, once a widget's measure, layout
	// or paint is done with the font.Face returned.
	//
	// Note that, in general, a font.Face is not safe for concurrent use by
	// multiple goroutines, as its methods may re-use implementation-specific
	// caches and mask image buffers.
	AcquireFontFace(FontFaceOptions) font.Face
	ReleaseFontFace(FontFaceOptions, font.Face)
}

var (
	// DefaultPalette is the default theme's palette.
	DefaultPalette = Palette{
		Light:      &image.Uniform{C: color.RGBA{0xf5, 0xf5, 0xf5, 0xff}}, // Material Design "Grey 100".
		Neutral:    &image.Uniform{C: color.RGBA{0xee, 0xee, 0xee, 0xff}}, // Material Design "Grey 200".
		Dark:       &image.Uniform{C: color.RGBA{0xe0, 0xe0, 0xe0, 0xff}}, // Material Design "Grey 300".
		Accent:     &image.Uniform{C: color.RGBA{0x21, 0x96, 0xf3, 0xff}}, // Material Design "Blue 500".
		Foreground: &image.Uniform{C: color.RGBA{0x00, 0x00, 0x00, 0xff}}, // Material Design "Black".
		Background: &image.Uniform{C: color.RGBA{0xff, 0xff, 0xff, 0xff}}, // Material Design "White".
	}

	// DefaultTheme is a theme using the default palette and a basic font face.
	DefaultTheme Theme = defaultTheme{}
)

// Note that a basicfont.Face is stateless and safe to use concurrently, so
// defaultTheme.ReleaseFontFace can be a no-op.

type defaultTheme struct{}

func (defaultTheme) Palette() Palette                           { return DefaultPalette }
func (defaultTheme) AcquireFontFace(FontFaceOptions) font.Face  { return basicfont.Face7x13 }
func (defaultTheme) ReleaseFontFace(FontFaceOptions, font.Face) {}
