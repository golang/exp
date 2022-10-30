// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slog

import (
	"context"
	"log"
	"sync/atomic"
	"time"
)

var defaultLogger atomic.Value

func init() {
	defaultLogger.Store(Logger{
		handler: newDefaultHandler(log.Output),
	})
}

// Default returns the default Logger.
func Default() Logger { return defaultLogger.Load().(Logger) }

// SetDefault makes l the default Logger.
// After this call, output from the log package's default Logger
// (as with [log.Print], etc.) will be logged at InfoLevel using l's Handler.
func SetDefault(l Logger) {
	defaultLogger.Store(l)
	// If the default's handler is a defaultHandler, then don't use a handleWriter,
	// or we'll deadlock as they both try to acquire the log default mutex.
	// The defaultHandler will use whatever the log default writer is currently
	// set to, which is correct.
	// This can occur with SetDefault(Default()).
	// See TestSetDefault.
	if _, ok := l.Handler().(*defaultHandler); !ok {
		log.SetOutput(&handlerWriter{l.Handler(), log.Flags()})
		log.SetFlags(0) // we want just the log message, no time or location
	}
}

// handlerWriter is an io.Writer that calls a Handler.
// It is used to link the default log.Logger to the default slog.Logger.
type handlerWriter struct {
	h     Handler
	flags int
}

func (w *handlerWriter) Write(buf []byte) (int, error) {
	var depth int
	if w.flags&(log.Lshortfile|log.Llongfile) != 0 {
		depth = 2
	}
	// Remove final newline.
	origLen := len(buf) // Report that the entire buf was written.
	if len(buf) > 0 && buf[len(buf)-1] == '\n' {
		buf = buf[:len(buf)-1]
	}
	r := NewRecord(time.Now(), InfoLevel, string(buf), depth, nil)
	return origLen, w.h.Handle(r)
}

// A Logger records structured information about each call to its
// Log, Debug, Info, Warn, and Error methods.
// For each call, it creates a Record and passes it to a Handler.
//
// To create a new Logger, call [New] or a Logger method
// that begins "With".
type Logger struct {
	handler Handler // for structured logging
	ctx     context.Context
}

// Handler returns l's Handler.
func (l Logger) Handler() Handler { return l.handler }

// Context returns l's context.
func (l Logger) Context() context.Context { return l.ctx }

// With returns a new Logger that includes the given arguments, converted to
// Attrs as in [Logger.Log]. The Attrs will be added to each output from the
// Logger.
//
// The new Logger's handler is the result of calling WithAttrs on the receiver's
// handler.
func (l Logger) With(args ...any) Logger {
	var (
		attr  Attr
		attrs []Attr
	)
	for len(args) > 0 {
		attr, args = argsToAttr(args)
		attrs = append(attrs, attr)
	}
	l.handler = l.handler.WithAttrs(attrs)
	return l
}

// WithGroup returns a new Logger that starts a group. The keys of all
// attributes added to the Logger will be qualified by the given name.
func (l Logger) WithGroup(name string) Logger {
	l.handler = l.handler.WithGroup(name)
	return l
}

// WithContext returns a new Logger with the same handler
// as the receiver and the given context.
func (l Logger) WithContext(ctx context.Context) Logger {
	l.ctx = ctx
	return l
}

// New creates a new Logger with the given Handler.
func New(h Handler) Logger { return Logger{handler: h} }

// With calls Logger.With on the default logger.
func With(args ...any) Logger {
	return Default().With(args...)
}

// Enabled reports whether l emits log records at the given level.
func (l Logger) Enabled(level Level) bool {
	return l.Handler().Enabled(level)
}

// Log emits a log record with the current time and the given level and message.
// The Record's Attrs consist of the Logger's attributes followed by
// the Attrs specified by args.
//
// The attribute arguments are processed as follows:
//   - If an argument is an Attr, it is used as is.
//   - If an argument is a string and this is not the last argument,
//     the following argument is treated as the value and the two are combined
//     into an Attr.
//   - Otherwise, the argument is treated as a value with key "!BADKEY".
func (l Logger) Log(level Level, msg string, args ...any) {
	l.LogDepth(0, level, msg, args...)
}

// LogDepth is like [Logger.Log], but accepts a call depth to adjust the
// file and line number in the log record. 0 refers to the caller
// of LogDepth; 1 refers to the caller's caller; and so on.
func (l Logger) LogDepth(calldepth int, level Level, msg string, args ...any) {
	if !l.Enabled(level) {
		return
	}
	r := l.makeRecord(msg, level, calldepth)
	r.setAttrsFromArgs(args)
	_ = l.Handler().Handle(r)
}

func (l Logger) makeRecord(msg string, level Level, depth int) Record {
	return NewRecord(time.Now(), level, msg, depth+5, l.ctx)
}

// LogAttrs is a more efficient version of [Logger.Log] that accepts only Attrs.
func (l Logger) LogAttrs(level Level, msg string, attrs ...Attr) {
	l.LogAttrsDepth(0, level, msg, attrs...)
}

// LogAttrsDepth is like [Logger.LogAttrs], but accepts a call depth argument
// which it interprets like [Logger.LogDepth].
func (l Logger) LogAttrsDepth(calldepth int, level Level, msg string, attrs ...Attr) {
	if !l.Enabled(level) {
		return
	}
	r := l.makeRecord(msg, level, calldepth)
	r.AddAttrs(attrs...)
	_ = l.Handler().Handle(r)
}

// Debug logs at DebugLevel.
func (l Logger) Debug(msg string, args ...any) {
	l.LogDepth(0, DebugLevel, msg, args...)
}

// Info logs at InfoLevel.
func (l Logger) Info(msg string, args ...any) {
	l.LogDepth(0, InfoLevel, msg, args...)
}

// Warn logs at WarnLevel.
func (l Logger) Warn(msg string, args ...any) {
	l.LogDepth(0, WarnLevel, msg, args...)
}

// Error logs at ErrorLevel.
// If err is non-nil, Error appends Any("err", err)
// to the list of attributes.
func (l Logger) Error(msg string, err error, args ...any) {
	if err != nil {
		// TODO: avoid the copy.
		args = append(args[:len(args):len(args)], Any("err", err))
	}
	l.LogDepth(0, ErrorLevel, msg, args...)
}

// Debug calls Logger.Debug on the default logger.
func Debug(msg string, args ...any) {
	Default().LogDepth(0, DebugLevel, msg, args...)
}

// Info calls Logger.Info on the default logger.
func Info(msg string, args ...any) {
	Default().LogDepth(0, InfoLevel, msg, args...)
}

// Warn calls Logger.Warn on the default logger.
func Warn(msg string, args ...any) {
	Default().LogDepth(0, WarnLevel, msg, args...)
}

// Error calls Logger.Error on the default logger.
func Error(msg string, err error, args ...any) {
	if err != nil {
		// TODO: avoid the copy.
		args = append(args[:len(args):len(args)], Any("err", err))
	}
	Default().LogDepth(0, ErrorLevel, msg, args...)
}

// Log calls Logger.Log on the default logger.
func Log(level Level, msg string, args ...any) {
	Default().LogDepth(0, level, msg, args...)
}

// LogAttrs calls Logger.LogAttrs on the default logger.
func LogAttrs(level Level, msg string, attrs ...Attr) {
	Default().LogAttrsDepth(0, level, msg, attrs...)
}
