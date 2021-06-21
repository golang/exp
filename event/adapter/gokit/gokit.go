// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gokit provides a go-kit logger for events.
package gokit

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/log"
	"golang.org/x/exp/event"
)

type logger struct {
}

// NewLogger returns a logger.
func NewLogger() log.Logger {
	return &logger{}
}

// Log writes a structured log message.
// If the first argument is a context.Context, it is used
// to find the exporter to which to deliver the message.
// Otherwise, the default exporter is used.
func (l *logger) Log(keyvals ...interface{}) error {
	ctx := context.Background()
	if len(keyvals) > 0 {
		if c, ok := keyvals[0].(context.Context); ok {
			ctx = c
			keyvals = keyvals[1:]
		}
	}
	ev := event.New(ctx, event.LogKind)
	if ev == nil {
		return nil
	}
	var msg string
	for i := 0; i < len(keyvals); i += 2 {
		key := keyvals[i].(string)
		value := keyvals[i+1]
		if key == "msg" || key == "message" {
			msg = fmt.Sprint(value)
		} else {
			ev.Labels = append(ev.Labels, event.Value(key, value))
		}
	}
	ev.Labels = append(ev.Labels, event.String("msg", msg))
	ev.Deliver()
	return nil
}
