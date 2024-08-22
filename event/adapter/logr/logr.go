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

type logSink struct {
	ev        *event.Event // cloned, never delivered
	labels    []event.Label
	nameSep   string
	name      string
	verbosity int
}

func NewLogger(ctx context.Context, nameSep string) logr.Logger {
	return logr.New(&logSink{
		ev:      event.New(ctx, event.LogKind),
		nameSep: nameSep,
	})
}

func (*logSink) Init(logr.RuntimeInfo) {}

// WithName implements logr.LogSink.WithName.
func (l *logSink) WithName(name string) logr.LogSink {
	l2 := *l
	if l.name == "" {
		l2.name = name
	} else {
		l2.name = l.name + l.nameSep + name
	}
	return &l2
}

// Enabled tests whether this LogSink is enabled at the specified V-level.
// For example, commandline flags might be used to set the logging
// verbosity and disable some info logs.
func (l *logSink) Enabled(level int) bool {
	return true
}

// Info implements logr.LogSink.Info.
func (l *logSink) Info(level int, msg string, keysAndValues ...interface{}) {
	if l.ev == nil {
		return
	}
	ev := l.ev.Clone()
	ev.Labels = append(ev.Labels, convertVerbosity(level).Label())
	l.log(ev, msg, keysAndValues)
}

// Error implements logr.LogSink.Error.
func (l *logSink) Error(err error, msg string, keysAndValues ...interface{}) {
	if l.ev == nil {
		return
	}
	ev := l.ev.Clone()
	ev.Labels = append(ev.Labels, event.Value("error", err))
	l.log(ev, msg, keysAndValues)
}

func (l *logSink) log(ev *event.Event, msg string, keysAndValues []interface{}) {
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

// WithValues implements logr.LogSink.WithValues.
func (l *logSink) WithValues(keysAndValues ...interface{}) logr.LogSink {
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
