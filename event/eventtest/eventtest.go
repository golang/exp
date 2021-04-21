// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package eventtest supports logging events to a test.
// You can use NewContext to create a context that knows how to deliver
// telemetry events back to the test.
// You must use this context or a derived one anywhere you want telemetry to be
// correctly routed back to the test it was constructed with.
package eventtest

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"golang.org/x/exp/event"
)

// NewContext returns a context you should use for the active test.
func NewContext(ctx context.Context, tb testing.TB) context.Context {
	h := &testHandler{tb: tb}
	return event.WithExporter(ctx, event.NewExporter(h))
}

type testHandler struct {
	tb  testing.TB
	buf strings.Builder
}

func (w *testHandler) Handle(ev *event.Event) {
	// build our log message in buffer
	w.buf.Reset()
	fmt.Fprint(&w.buf, ev)
	// log to the testing.TB
	msg := w.buf.String()
	if len(msg) > 0 {
		w.tb.Log(msg)
	}
}

func TestNow() func() time.Time {
	nextTime, _ := time.Parse(time.RFC3339Nano, "2020-03-05T14:27:48Z")
	return func() time.Time {
		thisTime := nextTime
		nextTime = nextTime.Add(time.Second)
		return thisTime
	}
}
