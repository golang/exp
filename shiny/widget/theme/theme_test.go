// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package theme

import (
	"testing"

	"golang.org/x/exp/shiny/unit"
	"golang.org/x/image/math/fixed"
)

func approxEqual(x, y, tolerance float64) bool {
	delta := x - y
	return -tolerance < delta && delta < +tolerance
}

func TestThemeIsAUnitConverter(t *testing.T) {
	// 1.5 inches (at the default 72 DPI) should be 108 pixels.
	c := unit.Converter(Default)
	got := c.Pixels(unit.Inches(1.5))
	want := fixed.I(108)
	if got != want {
		t.Errorf("1 inch in pixels: got %v, want %v", got, want)
	}

	// 3 em (at inconsolata.Regular8x16's 16 pixel em-height) should be 48
	// pixels, regardless of the DPI. That font face is based on a bitmap font,
	// not a vector font, so its height does not depend on the DPI. 48 pixels
	// is 48 points at 72 DPI, and 48 pixels is 21.6 points at 160 DPI.
	for _, dpi := range []float64{72, 160} {
		c := unit.Converter(&Theme{
			DPI: dpi,
		})
		got := c.Convert(unit.Ems(3), unit.Pt)
		want := unit.Points(3 * 16 * unit.PointsPerInch / dpi)
		if got.U != want.U || !approxEqual(got.F, want.F, 1e-10) {
			t.Errorf("dpi=%v: 3 em in points: got %v, want %v", dpi, got, want)
		}
	}
}
