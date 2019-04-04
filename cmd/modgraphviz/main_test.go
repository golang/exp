// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"testing"
)

func TestRun(t *testing.T) {
	in := bytes.NewBuffer([]byte(`
test.com/A test.com/B@v1.2.3
test.com/B test.com/C@v4.5.6
`))
	out := bytes.Buffer{}

	if err := Run(in, &out); err != nil {
		t.Fatal(err)
	}

	want := `digraph gomodgraph {
	"test.com/A" -> "test.com/B@v1.2.3"
	"test.com/B" -> "test.com/C@v4.5.6"
}
`
	if out.String() != want {
		t.Fatalf("\ngot: %s\nwant: %s", out.String(), want)
	}
}
