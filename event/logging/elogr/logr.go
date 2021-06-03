// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package elogr is a logr implementation that uses events.
package elogr

import (
	"context"

	"github.com/go-logr/logr"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/keys"
	"golang.org/x/exp/event/severity"
)

type logger struct {
	exporter  *event.Exporter
	builder   event.Builder // never delivered, only cloned
	nameSep   string
	name      string
	verbosity int
}

var _ logr.Logger = (*logger)(nil)

func NewLogger(ctx context.Context, nameSep string) logr.Logger {
	return &logger{
		builder: event.To(ctx),
		nameSep: nameSep,
	}
}

// WithName adds a new element to the logger's name.
// Successive calls with WithName continue to append
// suffixes to the logger's name.  It's strongly recommended
// that name segments contain only letters, digits, and hyphens
// (see the package documentation for more information).
func (l *logger) WithName(name string) logr.Logger {
	l2 := *l
	if l.name == "" {
		l2.name = name
	} else {
		l2.name = l.name + l.nameSep + name
	}
	return &l2
}

// V returns an Logger value for a specific verbosity level, relative to
// this Logger.  In other words, V values are additive.  V higher verbosity
// level means a log message is less important.  It's illegal to pass a log
// level less than zero.
func (l *logger) V(level int) logr.Logger {
	l2 := *l
	l2.verbosity += level
	return &l2
}

// Enabled tests whether this Logger is enabled.  For example, commandline
// flags might be used to set the logging verbosity and disable some info
// logs.
func (l *logger) Enabled() bool {
	return true
}

// addLabels adds labels for keysAndValues to b.
func addLabels(b event.Builder, keysAndValues []interface{}) {
	for i := 0; i < len(keysAndValues); i += 2 {
		b.With(newLabel(keysAndValues[i], keysAndValues[i+1]))
	}
}

// Info logs a non-error message with the given key/value pairs as context.
//
// The msg argument should be used to add some constant description to
// the log line.  The key/value pairs can then be used to add additional
// variable information.  The key/value pairs should alternate string
// keys and arbitrary values.
func (l *logger) Info(msg string, keysAndValues ...interface{}) {
	l.log(msg, l.builder.Clone(), keysAndValues)
}

// Error logs an error, with the given message and key/value pairs as context.
// It functions similarly to calling Info with the "error" named value, but may
// have unique behavior, and should be preferred for logging errors (see the
// package documentations for more information).
//
// The msg field should be used to add context to any underlying error,
// while the err field should be used to attach the actual error that
// triggered this log line, if present.
func (l *logger) Error(err error, msg string, keysAndValues ...interface{}) {
	l.log(msg, l.builder.Clone().With(event.Error.Of(err)), keysAndValues)
}

func (l *logger) log(msg string, b event.Builder, keysAndValues []interface{}) {
	b.With(convertVerbosity(l.verbosity))
	b.With(event.Name.Of(l.name))
	addLabels(b, keysAndValues)
	b.Log(msg)
}

// WithValues adds some key-value pairs of context to a logger.
// See Info for documentation on how key/value pairs work.
func (l *logger) WithValues(keysAndValues ...interface{}) logr.Logger {
	l2 := *l
	l2.builder = l2.builder.Clone()
	addLabels(l2.builder, keysAndValues)
	return &l2
}

func newLabel(key, value interface{}) event.Label {
	return keys.Value(key.(string)).Of(value)
}

func convertVerbosity(v int) event.Label {
	return severity.Of(severity.Level(v))
}
