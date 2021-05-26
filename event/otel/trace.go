// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package otel

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/event"
)

type TraceHandler struct {
	tracer trace.Tracer
}

var _ event.TraceHandler = (*TraceHandler)(nil)

func NewTraceHandler(t trace.Tracer) *TraceHandler {
	return &TraceHandler{tracer: t}
}

type spanKey struct{}

func (t *TraceHandler) Start(ctx context.Context, e *event.Event) context.Context {
	opts := labelsToSpanOptions(e.Labels)
	name, _ := event.Trace.Find(e)
	octx, span := t.tracer.Start(ctx, name, opts...)
	return context.WithValue(octx, spanKey{}, span)
}

func (t *TraceHandler) End(ctx context.Context, e *event.Event) {
	span, ok := ctx.Value(spanKey{}).(trace.Span)
	if !ok {
		panic("End called on context with no span")
	}
	span.End()
}

func labelsToSpanOptions(ls []event.Label) []trace.SpanOption {
	var opts []trace.SpanOption
	for _, l := range ls {
		switch l.Name {
		case "link":
			opts = append(opts, trace.WithLinks(l.Value.Interface().(trace.Link)))
		case "newRoot":
			opts = append(opts, trace.WithNewRoot())
		case "spanKind":
			opts = append(opts, trace.WithSpanKind(l.Value.Interface().(trace.SpanKind)))
		}
	}
	return opts
}
