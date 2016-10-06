// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package icons

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/exp/shiny/iconvg"
)

func encodePNG(dstFilename string, src image.Image) error {
	f, err := os.Create(dstFilename)
	if err != nil {
		return err
	}
	encErr := png.Encode(f, src)
	closeErr := f.Close()
	if encErr != nil {
		return encErr
	}
	return closeErr
}

func TestManualInspection(t *testing.T) {
	// Set this to a non-empty string such as "/tmp/mdicons" to manually
	// inspect the icons.
	const tmpDir = ""

	if tmpDir == "" {
		t.Skip("no tmpDir specified")
	}
	t.Errorf("tmpDir %q is a non-empty string; do not commit code changes", tmpDir)

	dst := image.NewAlpha(image.Rect(0, 0, 256, 256))
	z := &iconvg.Rasterizer{}
	z.SetDstImage(dst, dst.Bounds(), draw.Src)
	for _, v := range list {
		if err := iconvg.Decode(z, v.data, nil); err != nil {
			t.Errorf("%q: %v", v.name, err)
			continue
		}
		filename := filepath.Join(tmpDir, v.name+".png")
		if err := encodePNG(filename, dst); err != nil {
			t.Error(err)
		}
		t.Logf("wrote %s", filename)
	}
}

func TestDecodeAll(t *testing.T) {
	for _, v := range list {
		if err := iconvg.Decode(nil, v.data, nil); err != nil {
			t.Errorf("%q: %v", v.name, err)
			continue
		}
	}
}
