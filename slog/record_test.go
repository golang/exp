// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slog

import (
	"strings"
	"testing"
	"time"

	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog/internal/buffer"
)

func TestRecordAttrs(t *testing.T) {
	as := []Attr{Int("k1", 1), String("k2", "foo"), Int("k3", 3),
		Int64("k4", -1), Float64("f", 3.1), Uint64("u", 999)}
	r := newRecordWithAttrs(as)
	if g, w := r.NumAttrs(), len(as); g != w {
		t.Errorf("NumAttrs: got %d, want %d", g, w)
	}
	if got := r.Attrs(); !attrsEqual(got, as) {
		t.Errorf("got %v, want %v", got, as)
	}
}

func TestRecordSourceLine(t *testing.T) {
	// Zero call depth => empty file/line
	for _, test := range []struct {
		depth            int
		wantFile         string
		wantLinePositive bool
	}{
		{0, "", false},
		{-16, "", false},
		{1, "record.go", true},
	} {
		r := MakeRecord(time.Time{}, 0, "", test.depth)
		gotFile, gotLine := r.SourceLine()
		if i := strings.LastIndexByte(gotFile, '/'); i >= 0 {
			gotFile = gotFile[i+1:]
		}
		if gotFile != test.wantFile || (gotLine > 0) != test.wantLinePositive {
			t.Errorf("depth %d: got (%q, %d), want (%q, %t)",
				test.depth, gotFile, gotLine, test.wantFile, test.wantLinePositive)
		}
	}
}

func TestAliasing(t *testing.T) {
	intAttrs := func(from, to int) []Attr {
		var as []Attr
		for i := from; i < to; i++ {
			as = append(as, Int("k", i))
		}
		return as
	}

	check := func(r *Record, want []Attr) {
		t.Helper()
		got := r.Attrs()
		if !attrsEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	r1 := MakeRecord(time.Time{}, 0, "", 0)
	for i := 0; i < nAttrsInline+3; i++ {
		r1.AddAttr(Int("k", i))
	}
	check(&r1, intAttrs(0, nAttrsInline+3))
	r2 := r1
	check(&r2, intAttrs(0, nAttrsInline+3))
	// if cap(r1.attrs2) <= len(r1.attrs2) {
	// 	t.Fatal("cap not greater than len")
	// }
	r1.AddAttr(Int("k", nAttrsInline+3))
	r2.AddAttr(Int("k", -1))
	check(&r1, intAttrs(0, nAttrsInline+4))
	check(&r2, append(intAttrs(0, nAttrsInline+3), Int("k", -1)))
}

func newRecordWithAttrs(as []Attr) Record {
	r := MakeRecord(time.Now(), InfoLevel, "", 0)
	for _, a := range as {
		r.AddAttr(a)
	}
	return r
}

func attrsEqual(as1, as2 []Attr) bool {
	return slices.EqualFunc(as1, as2, Attr.Equal)
}

// Currently, pc(2) takes over 400ns, which is too expensive
// to call it for every log message.
func BenchmarkPC(b *testing.B) {
	b.ReportAllocs()
	var x uintptr
	for i := 0; i < b.N; i++ {
		x = pc(3)
	}
	_ = x
}

func BenchmarkSourceLine(b *testing.B) {
	r := MakeRecord(time.Now(), InfoLevel, "", 5)
	b.Run("alone", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			file, line := r.SourceLine()
			_ = file
			_ = line
		}
	})
	b.Run("stringifying", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			file, line := r.SourceLine()
			buf := buffer.New()
			buf.WriteString(file)
			buf.WriteByte(':')
			itoa((*[]byte)(buf), line, -1)
			s := string(*buf)
			buf.Free()
			_ = s
		}
	})
}

func BenchmarkRecord(b *testing.B) {
	const nAttrs = nAttrsInline * 10
	var a Attr

	for i := 0; i < b.N; i++ {
		r := MakeRecord(time.Time{}, InfoLevel, "", 0)
		for j := 0; j < nAttrs; j++ {
			r.AddAttr(Int("k", j))
		}
		for j := 0; j < nAttrs; j++ {
			a = r.Attr(j)
		}
	}
	_ = a
}
