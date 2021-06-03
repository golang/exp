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
	"os"
	"testing"
	"time"

	"golang.org/x/exp/event"
	"golang.org/x/exp/event/adapter/logfmt"
)

// NewContext returns a context you should use for the active test.
func NewContext(ctx context.Context, tb testing.TB) context.Context {
	h := &testHandler{tb: tb}
	return event.WithExporter(ctx, event.NewExporter(h))
}

type testHandler struct {
	tb      testing.TB
	printer logfmt.Printer
}

func (h *testHandler) Log(ctx context.Context, ev *event.Event) {
	h.event(ctx, ev)
}

func (h *testHandler) Metric(ctx context.Context, ev *event.Event) {
	h.event(ctx, ev)
}

func (h *testHandler) Annotate(ctx context.Context, ev *event.Event) {
	h.event(ctx, ev)
}

func (h *testHandler) Start(ctx context.Context, ev *event.Event) context.Context {
	h.event(ctx, ev)
	return ctx
}

func (h *testHandler) End(ctx context.Context, ev *event.Event) {
	h.event(ctx, ev)
}

func (h *testHandler) event(ctx context.Context, ev *event.Event) {
	//TODO: choose between stdout and stderr based on the event
	//TODO: decide if we should be calling h.tb.Fail()
	h.printer.Event(os.Stdout, ev)
}

func ExporterOptions() event.ExporterOptions {
	nextTime, _ := time.Parse(time.RFC3339Nano, "2020-03-05T14:27:48Z")
	return event.ExporterOptions{
		Now: func() time.Time {
			thisTime := nextTime
			nextTime = nextTime.Add(time.Second)
			return thisTime
		},
	}
}
