// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mmap

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestOpen(t *testing.T) {
	const filename = "mmap_test.go"
	r, err := Open(filename)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got := make([]byte, r.Len())
	if _, err := r.ReadAt(got, 0); err != nil && err != io.EOF {
		t.Fatalf("ReadAt: %v", err)
	}
	want, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("os.ReadFile: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d bytes, want %d", len(got), len(want))
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("\ngot  %q\nwant %q", string(got), string(want))
	}
}

func TestSeekRead(t *testing.T) {
	const filename = "mmap_test.go"
	r, err := Open(filename)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	buf := make([]byte, 1)
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("Seek: %v", err)
	}
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if n != 1 {
		t.Fatalf("Read: got %d bytes, want 1", n)
	}
	if buf[0] != '/' { // first comment slash
		t.Fatalf("Read: got %q, want '/'", buf[0])
	}
	if _, err := r.Seek(1, io.SeekCurrent); err != nil {
		t.Fatalf("Seek: %v", err)
	}
	n, err = r.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if n != 1 {
		t.Fatalf("Read: got %d bytes, want 1", n)
	}
	if buf[0] != ' ' { // space after comment
		t.Fatalf("Read: got %q, want ' '", buf[0])
	}
	if _, err := r.Seek(-1, io.SeekEnd); err != nil {
		t.Fatalf("Seek: %v", err)
	}
	n, err = r.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if n != 1 {
		t.Fatalf("Read: got %d bytes, want 1", n)
	}
	if buf[0] != '\n' { // last newline
		t.Fatalf("Read: got %q, want newline", buf[0])
	}
	if _, err := r.Seek(0, io.SeekEnd); err != nil {
		t.Fatalf("Seek: %v", err)
	}
	if _, err := r.Read(buf); err != io.EOF {
		t.Fatalf("Read: expected EOF, got %v", err)
	}
}

func TestWriterTo_idempotency(t *testing.T) {
	const filename = "mmap_test.go"
	r, err := Open(filename)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	buf := bytes.NewBuffer(make([]byte, 0, len(r.data)))
	// first run
	n, err := r.WriteTo(buf)
	if err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	if n != int64(len(r.data)) {
		t.Fatalf("WriteTo: got %d bytes, want %d", n, len(r.data))
	}
	if !bytes.Equal(buf.Bytes(), r.data) {
		t.Fatalf("WriteTo: got %q, want %q", buf.Bytes(), r.data)
	}
	// second run
	n, err = r.WriteTo(buf)
	if err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	if n != 0 {
		t.Fatalf("WriteTo: got %d bytes, want %d", n, 0)
	}
	if !bytes.Equal(buf.Bytes(), r.data) {
		t.Fatalf("WriteTo: got %q, want %q", buf.Bytes(), r.data)
	}
}

func BenchmarkMmapCopy(b *testing.B) {
	var f io.ReadSeeker

	// mmap some big-ish file; will only work on unix-like OSs.
	r, err := Open("/proc/self/exe")
	if err != nil {
		b.Fatalf("Open: %v", err)
	}

	// Sanity check: ensure we will run into the io.Copy optimization when using the NEW code above.
	var _ io.WriterTo = r

	// f = io.NewSectionReader(r, 0, int64(len(r.data))) // old
	f = r // new

	buf := bytes.NewBuffer(make([]byte, 0, len(r.data)))
	// "Hide" the ReaderFrom interface by wrapping the writer.
	// Otherwise we skew the results by optimizing the wrong side.
	writer := struct{ io.Writer }{buf}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = f.Seek(0, io.SeekStart)
		buf.Reset()

		n, _ := io.Copy(writer, f)
		b.SetBytes(n)
	}
}
