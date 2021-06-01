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
	"strings"
	"testing"
	"time"

	"golang.org/x/exp/event"
	"golang.org/x/exp/event/adapter/logfmt"
)

// NewContext returns a context you should use for the active test.
func NewContext(ctx context.Context, tb testing.TB) context.Context {
	h := &testHandler{tb: tb}
	h.p = logfmt.NewHandler(&h.buf)
	return event.WithExporter(ctx, event.NewExporter(h))
}

type testHandler struct {
	tb  testing.TB
	buf strings.Builder
	p   *logfmt.Handler
}

func (h *testHandler) Log(ctx context.Context, ev *event.Event) {
	h.p.Log(ctx, ev)
	h.deliver()
}

func (h *testHandler) Metric(ctx context.Context, ev *event.Event) {
	h.p.Metric(ctx, ev)
	h.deliver()
}

func (h *testHandler) Annotate(ctx context.Context, ev *event.Event) {
	h.p.Annotate(ctx, ev)
	h.deliver()
}

func (h *testHandler) Start(ctx context.Context, ev *event.Event) context.Context {
	ctx = h.p.Start(ctx, ev)
	h.deliver()
	return ctx
}

func (h *testHandler) End(ctx context.Context, ev *event.Event) {
	h.p.End(ctx, ev)
	h.deliver()
}

func (h *testHandler) deliver() {
	if h.buf.Len() == 0 {
		return
	}
	h.tb.Log(h.buf.String())
	h.buf.Reset()
}

func TestNow() func() time.Time {
	nextTime, _ := time.Parse(time.RFC3339Nano, "2020-03-05T14:27:48Z")
	return func() time.Time {
		thisTime := nextTime
		nextTime = nextTime.Add(time.Second)
		return thisTime
	}
}
