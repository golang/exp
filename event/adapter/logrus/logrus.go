// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

// Package logrus provides a logrus Formatter for events.
// To use for the global logger:
//   logrus.SetFormatter(elogrus.NewFormatter(exporter))
//   logrus.SetOutput(io.Discard)
// and for a Logger instance:
//   logger.SetFormatter(elogrus.NewFormatter(exporter))
//   logger.SetOutput(io.Discard)
//
// If you call elogging.SetExporter, then you can pass nil
// for the exporter above and it will use the global one.
package logrus

import (
	"context"

	"github.com/sirupsen/logrus"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/severity"
)

type formatter struct{}

func NewFormatter() logrus.Formatter {
	return &formatter{}
}

var _ logrus.Formatter = (*formatter)(nil)

// Format writes an entry to an event.Exporter. It always returns nil (see below).
// If e.Context is non-nil, Format gets the exporter from the context. Otherwise
// it uses the default exporter.
//
// Logrus first calls the Formatter to get a []byte, then writes that to the
// output. That doesn't work for events, so we subvert it by having the
// Formatter export the event (and thereby write it). That is why the logrus
// Output io.Writer should be set to io.Discard.
func (f *formatter) Format(e *logrus.Entry) ([]byte, error) {
	ctx := e.Context
	if ctx == nil {
		ctx = context.Background()
	}
	ev := event.New(ctx, event.LogKind)
	if ev == nil {
		return nil, nil
	}
	ev.At = e.Time
	ev.Labels = append(ev.Labels, convertLevel(e.Level).Label())
	for k, v := range e.Data {
		ev.Labels = append(ev.Labels, event.Value(k, v))
	}
	ev.Labels = append(ev.Labels, event.String("msg", e.Message))
	ev.Deliver()
	return nil, nil
}

func convertLevel(level logrus.Level) severity.Level {
	switch level {
	case logrus.PanicLevel:
		return severity.Fatal + 1
	case logrus.FatalLevel:
		return severity.Fatal
	case logrus.ErrorLevel:
		return severity.Error
	case logrus.WarnLevel:
		return severity.Warning
	case logrus.InfoLevel:
		return severity.Info
	case logrus.DebugLevel:
		return severity.Debug
	case logrus.TraceLevel:
		return severity.Trace
	default:
		return severity.Trace
	}
}
