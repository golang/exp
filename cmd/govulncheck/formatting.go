// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// wrap wraps s to fit in maxWidth by breaking it into lines at whitespace. If a
// single word is longer than maxWidth, it is retained as its own line.
func wrap(s string, maxWidth int) string {
	var b strings.Builder
	w := 0

	for _, f := range strings.Fields(s) {
		if w > 0 && w+len(f)+1 > maxWidth {
			b.WriteByte('\n')
			w = 0
		}
		if w != 0 {
			b.WriteByte(' ')
			w++
		}
		b.WriteString(f)
		w += len(f)
	}
	return b.String()
}

type table struct {
	headings []string
	lines    [][]string
}

func newTable(headings ...string) *table {
	return &table{headings: headings}
}

func (t *table) row(cells ...string) {
	// Split each cell into lines.
	// Track the max number of lines.
	var cls [][]string
	max := 0
	for _, c := range cells {
		ls := strings.Split(c, "\n")
		if len(ls) > max {
			max = len(ls)
		}
		cls = append(cls, ls)
	}
	// Add each line to the table.
	for i := 0; i < max; i++ {
		var line []string
		for _, cl := range cls {
			if i >= len(cl) {
				line = append(line, "")
			} else {
				line = append(line, cl[i])
			}
		}
		t.lines = append(t.lines, line)
	}
}

func (t *table) write(w io.Writer) (err error) {
	// Calculate column widths.
	widths := make([]int, len(t.headings))
	for i, h := range t.headings {
		widths[i] = len(h)
	}
	for _, l := range t.lines {
		for i, c := range l {
			if len(c) > widths[i] {
				widths[i] = len(c)
			}
		}
	}

	totalWidth := 0
	for _, w := range widths {
		totalWidth += w
	}
	// Account for a space between columns.
	totalWidth += len(widths) - 1
	dashes := strings.Repeat("-", totalWidth)

	writeLine := func(s string) {
		if err == nil {
			_, err = io.WriteString(w, s)
		}
		if err == nil {
			_, err = io.WriteString(w, "\n")
		}
	}

	writeCells := func(cells []string) {
		var buf bytes.Buffer
		for i, c := range cells {
			if i > 0 {
				buf.WriteByte(' ')
			}
			fmt.Fprintf(&buf, "%-*s", widths[i], c)
		}
		writeLine(strings.TrimRight(buf.String(), " "))
	}

	// Write headings.
	writeLine(dashes)
	writeCells(t.headings)
	writeLine(dashes)

	// Write body.
	for _, l := range t.lines {
		writeCells(l)
	}
	return err
}
