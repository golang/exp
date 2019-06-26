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
	graph, err := convert(in)
	if err != nil {
		t.Fatal(err)
	}

	want := `digraph gomodgraph {
	"test.com/A" -> "test.com/B@v1.2.3"
	"test.com/B" -> "test.com/C@v4.5.6"
}
`
	if string(graph) != want {
		t.Fatalf("\ngot: %s\nwant: %s", string(graph), want)
	}
}
