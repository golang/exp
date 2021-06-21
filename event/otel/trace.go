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

func NewTraceHandler(t trace.Tracer) *TraceHandler {
	return &TraceHandler{tracer: t}
}

type spanKey struct{}

func (t *TraceHandler) Event(ctx context.Context, ev *event.Event) context.Context {
	switch ev.Kind {
	case event.StartKind:
		name, opts := labelsToSpanOptions(ev.Labels)
		octx, span := t.tracer.Start(ctx, name, opts...)
		return context.WithValue(octx, spanKey{}, span)
	case event.EndKind:
		span, ok := ctx.Value(spanKey{}).(trace.Span)
		if !ok {
			panic("End called on context with no span")
		}
		span.End()
		return ctx
	default:
		return ctx
	}
}

func labelsToSpanOptions(ls []event.Label) (string, []trace.SpanOption) {
	var opts []trace.SpanOption
	var name string
	for _, l := range ls {
		switch l.Name {
		case "link":
			opts = append(opts, trace.WithLinks(l.Interface().(trace.Link)))
		case "newRoot":
			opts = append(opts, trace.WithNewRoot())
		case "spanKind":
			opts = append(opts, trace.WithSpanKind(l.Interface().(trace.SpanKind)))
		case "name":
			name = l.String()
		}
	}
	return name, opts
}
