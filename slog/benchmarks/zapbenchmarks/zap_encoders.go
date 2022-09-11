// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zapbenchmarks

import (
	"io"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// from zap/internal/ztest

// A syncer is a spy for the Sync portion of zapcore.WriteSyncer.
type syncer struct {
	err    error
	called bool
}

// SetError sets the error that the Sync method will return.
func (s *syncer) SetError(err error) {
	s.err = err
}

// Sync records that it was called, then returns the user-supplied error (if
// any).
func (s *syncer) Sync() error {
	s.called = true
	return s.err
}

// Called reports whether the Sync method was called.
func (s *syncer) Called() bool {
	return s.called
}

// A discarder sends all writes to ioutil.Discard.
type discarder struct{ syncer }

// Write implements io.Writer.
func (d *discarder) Write(b []byte) (int, error) {
	return io.Discard.Write(b)
}

// fastTextEncoder mimics slog/benchmarks.fastTextHandler.
type fastTextEncoder struct {
	buf *buffer.Buffer
	zapcore.ObjectEncoder
}

var bufferPool = buffer.NewPool()

var tePool = sync.Pool{
	New: func() any { return &fastTextEncoder{} },
}

func newFastTextEncoder() *fastTextEncoder {
	e := tePool.Get().(*fastTextEncoder)
	e.buf = bufferPool.Get()
	return e
}

func (c *fastTextEncoder) Clone() zapcore.Encoder {
	clone := newFastTextEncoder()
	if c.buf != nil {
		panic("buf should be nil")
		clone.buf.Write(c.buf.Bytes())
	}
	return clone
}

func (c *fastTextEncoder) EncodeEntry(e zapcore.Entry, fs []zap.Field) (*buffer.Buffer, error) {
	c2 := newFastTextEncoder()
	if !e.Time.IsZero() {
		c2.buf.AppendString("time=")
		c2.appendTime(e.Time)
		c2.buf.AppendByte(' ')
	}
	c2.buf.AppendString("level=")
	c2.buf.AppendInt(int64(e.Level))
	c2.buf.AppendByte(' ')
	c2.buf.AppendString("msg=")
	c2.buf.AppendString(e.Message)
	for _, f := range fs {
		c2.buf.AppendByte(' ')
		f.AddTo(c2)
	}
	c2.buf.AppendString("\n")
	buf := c2.buf
	tePool.Put(c2)
	return buf, nil
}

func (c *fastTextEncoder) AddString(key, value string) {
	c.buf.AppendString(key)
	c.buf.AppendByte('=')
	c.buf.AppendString(value)
}

func (c *fastTextEncoder) AddTime(key string, value time.Time) {
	c.buf.AppendString(key)
	c.buf.AppendByte('=')
	c.appendTime(value)
}

func (c *fastTextEncoder) AddDuration(key string, value time.Duration) {
	c.buf.AppendString(key)
	c.buf.AppendByte('=')
	c.buf.AppendInt(value.Nanoseconds())
}

func (c *fastTextEncoder) AddInt64(key string, value int64) {
	c.buf.AppendString(key)
	c.buf.AppendByte('=')
	c.buf.AppendInt(value)
}
func (c *fastTextEncoder) appendTime(t time.Time) {
	c.buf.AppendInt(t.Unix())
}

// asyncCore mimics slog/benchmarks.asyncHandler.
type asyncCore struct {
	ringBuffer [100]entryAndFields
	next       int
}

type entryAndFields struct {
	e zapcore.Entry
	f []zap.Field
}

func (*asyncCore) Enabled(zapcore.Level) bool      { return true }
func (c *asyncCore) With([]zap.Field) zapcore.Core { return c }
func (*asyncCore) Sync() error                     { return nil }

// Also needed to make this non-trivial.
func (c *asyncCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	return ce.AddCore(ent, c)
}

func (c *asyncCore) Write(e zapcore.Entry, f []zap.Field) error {
	c.ringBuffer[c.next] = entryAndFields{e, f}
	c.next = (c.next + 1) % len(c.ringBuffer)
	return nil
}
