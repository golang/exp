// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package logr is a logr implementation that uses events.
package logr

import (
	"context"

	"github.com/go-logr/logr"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/severity"
)

type logger struct {
	ev        *event.Event // cloned, never delivered
	labels    []event.Label
	nameSep   string
	name      string
	verbosity int
}

var _ logr.Logger = (*logger)(nil)

func NewLogger(ctx context.Context, nameSep string) logr.Logger {
	return &logger{
		ev:      event.New(ctx, event.LogKind),
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

// Info logs a non-error message with the given key/value pairs as context.
//
// The msg argument should be used to add some constant description to
// the log line.  The key/value pairs can then be used to add additional
// variable information.  The key/value pairs should alternate string
// keys and arbitrary values.
func (l *logger) Info(msg string, keysAndValues ...interface{}) {
	if l.ev == nil {
		return
	}
	l.log(l.ev.Clone(), msg, keysAndValues)
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
	if l.ev == nil {
		return
	}
	ev := l.ev.Clone()
	ev.Labels = append(ev.Labels, event.Value("error", err))
	l.log(ev, msg, keysAndValues)
}

func (l *logger) log(ev *event.Event, msg string, keysAndValues []interface{}) {
	ev.Labels = append(ev.Labels, convertVerbosity(l.verbosity).Label())
	ev.Labels = append(ev.Labels, l.labels...)
	for i := 0; i < len(keysAndValues); i += 2 {
		ev.Labels = append(ev.Labels, newLabel(keysAndValues[i], keysAndValues[i+1]))
	}
	ev.Labels = append(ev.Labels,
		event.String("name", l.name),
		event.String("msg", msg),
	)
	ev.Deliver()
}

// WithValues adds some key-value pairs of context to a logger.
// See Info for documentation on how key/value pairs work.
func (l *logger) WithValues(keysAndValues ...interface{}) logr.Logger {
	l2 := *l
	if len(keysAndValues) > 0 {
		l2.labels = make([]event.Label, len(l.labels), len(l.labels)+(len(keysAndValues)/2))
		copy(l2.labels, l.labels)
		for i := 0; i < len(keysAndValues); i += 2 {
			l2.labels = append(l2.labels, newLabel(keysAndValues[i], keysAndValues[i+1]))
		}
	}
	return &l2
}

func newLabel(key, value interface{}) event.Label {
	return event.Value(key.(string), value)
}

func convertVerbosity(v int) severity.Level {
	//TODO: this needs to be more complicated, v decreases with increasing severity
	return severity.Level(v)
}
