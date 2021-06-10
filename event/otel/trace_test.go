// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package otel_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/otel"
)

func TestTrace(t *testing.T) {
	// Verify that otel and event traces work well together.
	// This test uses a single, fixed span tree (see makeTraceSpec).
	// Each test case varies which of the individual spans are
	// created directly from an otel tracer, and which are created
	// using the event package.

	want := "root (f (g h) p (q r))"

	for i, tfunc := range []func(int) bool{
		func(int) bool { return true },
		func(int) bool { return false },
		func(i int) bool { return i%2 == 0 },
		func(i int) bool { return i%2 == 1 },
		func(i int) bool { return i%3 == 0 },
		func(i int) bool { return i%3 == 1 },
	} {
		ctx, tr, shutdown := setupOtel()
		// There are 7 spans, so we create a 7-element slice.
		// tfunc determines, for each index, whether it holds
		// an otel tracer or nil.
		tracers := make([]trace.Tracer, 7)
		for i := 0; i < len(tracers); i++ {
			if tfunc(i) {
				tracers[i] = tr
			}
		}
		s := makeTraceSpec(tracers)
		s.apply(ctx)
		got := shutdown()
		if got != want {
			t.Errorf("#%d: got %v, want %v", i, got, want)
		}
	}
}

func makeTraceSpec(tracers []trace.Tracer) *traceSpec {
	return &traceSpec{
		name:   "root",
		tracer: tracers[0],
		children: []*traceSpec{
			{
				name:   "f",
				tracer: tracers[1],
				children: []*traceSpec{
					{name: "g", tracer: tracers[2]},
					{name: "h", tracer: tracers[3]},
				},
			},
			{
				name:   "p",
				tracer: tracers[4],
				children: []*traceSpec{
					{name: "q", tracer: tracers[5]},
					{name: "r", tracer: tracers[6]},
				},
			},
		},
	}
}

type traceSpec struct {
	name     string
	tracer   trace.Tracer // nil for event
	children []*traceSpec
}

// apply builds spans for the traceSpec and all its children,
// If the traceSpec has a non-nil tracer, it is used to create the span.
// Otherwise, event.Trace.Start is used.
func (s *traceSpec) apply(ctx context.Context) {
	if s.tracer != nil {
		var span trace.Span
		ctx, span = s.tracer.Start(ctx, s.name)
		defer span.End()
	} else {
		ctx = event.Start(ctx, s.name)
		defer event.End(ctx)
	}
	for _, c := range s.children {
		c.apply(ctx)
	}
}

func setupOtel() (context.Context, trace.Tracer, func() string) {
	ctx := context.Background()
	e := newTestExporter()
	bsp := sdktrace.NewSimpleSpanProcessor(e)
	stp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(bsp))
	tracer := stp.Tracer("")

	ee := event.NewExporter(otel.NewTraceHandler(tracer), nil)
	ctx = event.WithExporter(ctx, ee)
	return ctx, tracer, func() string { stp.Shutdown(ctx); return e.got }
}

// testExporter is an otel exporter for traces
type testExporter struct {
	m   map[trace.SpanID][]*sdktrace.SpanSnapshot // key is parent SpanID
	got string
}

var _ sdktrace.SpanExporter = (*testExporter)(nil)

func newTestExporter() *testExporter {
	return &testExporter{m: map[trace.SpanID][]*sdktrace.SpanSnapshot{}}
}

func (e *testExporter) ExportSpans(ctx context.Context, ss []*sdktrace.SpanSnapshot) error {
	for _, s := range ss {
		sid := s.Parent.SpanID()
		e.m[sid] = append(e.m[sid], s)
	}
	return nil
}

func (e *testExporter) Shutdown(ctx context.Context) error {
	root := e.m[trace.SpanID{}][0]
	var buf bytes.Buffer
	e.print(&buf, root)
	e.got = buf.String()
	return nil
}

func (e *testExporter) print(w io.Writer, ss *sdktrace.SpanSnapshot) {
	fmt.Fprintf(w, "%s", ss.Name)
	children := e.m[ss.SpanContext.SpanID()]
	if len(children) > 0 {
		fmt.Fprint(w, " (")
		for i, ss := range children {
			if i != 0 {
				fmt.Fprint(w, " ")
			}
			e.print(w, ss)
		}
		fmt.Fprint(w, ")")
	}
}
