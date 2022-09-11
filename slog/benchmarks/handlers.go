// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package benchmarks

// Handlers for benchmarking.

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"golang.org/x/exp/slog"
	"golang.org/x/exp/slog/internal/buffer"
)

// A fastTextHandler writes a Record to an io.Writer in a format similar to
// slog.TextHandler, but without quoting or locking. It has a few other
// performance-motivated shortcuts, like writing times as seconds since the
// epoch instead of strings.
//
// It is intended to represent a high-performance Handler that synchronously
// writes text (as opposed to binary).
type fastTextHandler struct {
	w io.Writer
}

func newFastTextHandler(w io.Writer) slog.Handler {
	return &fastTextHandler{w: w}
}

func (h *fastTextHandler) Enabled(slog.Level) bool { return true }

func (h *fastTextHandler) Handle(r slog.Record) error {
	buf := buffer.New()
	defer buf.Free()

	if !r.Time().IsZero() {
		buf.WriteString("time=")
		h.appendTime(buf, r.Time())
		buf.WriteByte(' ')
	}
	buf.WriteString("level=")
	*buf = strconv.AppendInt(*buf, int64(r.Level()), 10)
	buf.WriteByte(' ')
	buf.WriteString("msg=")
	buf.WriteString(r.Message())
	r.Attrs(func(a slog.Attr) {
		buf.WriteByte(' ')
		buf.WriteString(a.Key())
		buf.WriteByte('=')
		h.appendValue(buf, a)
	})
	buf.WriteByte('\n')
	_, err := h.w.Write(*buf)
	return err
}

func (h *fastTextHandler) appendValue(buf *buffer.Buffer, a slog.Attr) {
	switch a.Kind() {
	case slog.StringKind:
		buf.WriteString(a.String())
	case slog.Int64Kind:
		*buf = strconv.AppendInt(*buf, a.Int64(), 10)
	case slog.Uint64Kind:
		*buf = strconv.AppendUint(*buf, a.Uint64(), 10)
	case slog.Float64Kind:
		*buf = strconv.AppendFloat(*buf, a.Float64(), 'g', -1, 64)
	case slog.BoolKind:
		*buf = strconv.AppendBool(*buf, a.Bool())
	case slog.DurationKind:
		*buf = strconv.AppendInt(*buf, a.Duration().Nanoseconds(), 10)
	case slog.TimeKind:
		h.appendTime(buf, a.Time())
	case slog.AnyKind:
		v := a.Value()
		switch v := v.(type) {
		case error:
			buf.WriteString(v.Error())
		default:
			buf.WriteString(fmt.Sprint(v))
		}
	default:
		panic(fmt.Sprintf("bad kind: %s", a.Kind()))
	}
}

func (h *fastTextHandler) appendTime(buf *buffer.Buffer, t time.Time) {
	*buf = strconv.AppendInt(*buf, t.Unix(), 10)
}

func (h *fastTextHandler) With([]slog.Attr) slog.Handler {
	panic("textHandler: With unimplemented")
}

// An asyncHandler simulates a Handler that passes Records to a
// background goroutine for processing.
// Because sending to a channel can be expensive due to locking,
// we simulate a lock-free queue by adding the Record to a ring buffer.
// Omitting the locking makes this little more than a copy of the Record,
// but that is a worthwhile thing to measure because Records are on the large
// side.
type asyncHandler struct {
	ringBuffer [100]slog.Record
	next       int
}

func newAsyncHandler() *asyncHandler {
	return &asyncHandler{}
}

func (*asyncHandler) Enabled(slog.Level) bool { return true }

func (h *asyncHandler) Handle(r slog.Record) error {
	h.ringBuffer[h.next] = r.Clone()
	h.next = (h.next + 1) % len(h.ringBuffer)
	return nil
}

func (h *asyncHandler) With([]slog.Attr) slog.Handler {
	panic("asyncHandler: With unimplemented")
}
