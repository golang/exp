// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imageutil

import (
	"image"
	"testing"
)

func area(r image.Rectangle) int {
	dx, dy := r.Dx(), r.Dy()
	if dx <= 0 || dy <= 0 {
		return 0
	}
	return dx * dy
}

func TestBorder(t *testing.T) {
	r := image.Rect(100, 200, 400, 300)

	insets := []int{
		-100,
		-1,
		+0,
		+1,
		+20,
		+49,
		+50,
		+51,
		+149,
		+150,
		+151,
	}

	for _, inset := range insets {
		border := Border(r, inset)

		outer, inner := r, r.Inset(inset)
		if inset < 0 {
			outer, inner = inner, outer
		}

		got := 0
		for _, b := range border {
			got += area(b)
		}
		want := area(outer) - area(inner)
		if got != want {
			t.Errorf("inset=%d: total area: got %d, want %d", inset, got, want)
		}

		for i, bi := range border {
			for j, bj := range border {
				if i <= j {
					continue
				}
				if !bi.Intersect(bj).Empty() {
					t.Errorf("inset=%d: %v and %v overlap", inset, bi, bj)
				}
			}
		}

		for _, b := range border {
			if got := outer.Intersect(b); got != b {
				t.Errorf("inset=%d: outer intersection: got %v, want %v", inset, got, b)
			}
			if got := inner.Intersect(b); !got.Empty() {
				t.Errorf("inset=%d: inner intersection: got %v, want empty", inset, got)
			}
		}
	}
}
