// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package event_test

import (
	"context"
	"testing"

	"golang.org/x/exp/event"
)

func TestCommon(t *testing.T) {
	h := &catchHandler{}
	ctx := event.WithExporter(context.Background(), event.NewExporter(h, nil))
	m := event.NewCounter("m")

	const simple = "simple message"
	const trace = "a trace"

	event.To(ctx).Log(simple)
	checkFind(t, h, "Log", event.Message, true, simple)
	checkFind(t, h, "Log", event.Name, false, "")

	event.To(ctx).Metric(m.Record(3))
	checkFind(t, h, "Metric", event.Message, false, "")
	checkFind(t, h, "Metric", event.Name, false, "")

	event.To(ctx).Annotate()
	checkFind(t, h, "Annotate", event.Message, false, "")
	checkFind(t, h, "Annotate", event.Name, false, "")

	_, end := event.To(ctx).Start(trace)
	checkFind(t, h, "Start", event.Message, false, "")
	checkFind(t, h, "Start", event.Name, true, trace)

	end()
	checkFind(t, h, "End", event.Message, false, "")
	checkFind(t, h, "End", event.Name, false, "")
}

type finder interface {
	Find(*event.Event) (string, bool)
}

func checkFind(t *testing.T, h *catchHandler, method string, key finder, match bool, text string) {
	m, ok := key.Find(&h.ev)
	if ok && !match {
		t.Errorf("%s produced an event with a %v", method, key)
	}
	if !ok && match {
		t.Errorf("%s did not produce an event with a %v", method, key)
	}
	if m != text {
		t.Errorf("Expected event with %v %q from %s got %q", key, text, method, m)
	}
}

type catchHandler struct {
	ev event.Event
}

func (h *catchHandler) Log(ctx context.Context, ev *event.Event) {
	h.event(ctx, ev)
}

func (h *catchHandler) Metric(ctx context.Context, ev *event.Event) {
	h.event(ctx, ev)
}

func (h *catchHandler) Annotate(ctx context.Context, ev *event.Event) {
	h.event(ctx, ev)
}

func (h *catchHandler) Start(ctx context.Context, ev *event.Event) context.Context {
	h.event(ctx, ev)
	return ctx
}

func (h *catchHandler) End(ctx context.Context, ev *event.Event) {
	h.event(ctx, ev)
}

func (h *catchHandler) event(ctx context.Context, ev *event.Event) {
	h.ev = *ev
	h.ev.Labels = make([]event.Label, len(ev.Labels))
	copy(h.ev.Labels, ev.Labels)
}
