// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slog

func concat(l1, l2 []Attr) []Attr {
	return concat3(l1, l2, nil)
}

func concat3(l1, l2, l3 []Attr) []Attr {
	l := make([]Attr, len(l1)+len(l2)+len(l3))
	copy(l, l1)
	copy(l[len(l1):], l2)
	copy(l[len(l1)+len(l2):], l3)
	return l
}

// Cheap integer to fixed-width decimal ASCII. Give a negative width to avoid zero-padding.
// Copied from log/log.go.
func itoa(buf *[]byte, i int, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}
