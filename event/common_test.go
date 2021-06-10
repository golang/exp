// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package event_test

import (
	"testing"

	"golang.org/x/exp/event"
	"golang.org/x/exp/event/adapter/eventtest"
)

func TestCommon(t *testing.T) {
	ctx, h := eventtest.NewCapture()
	m := event.NewCounter("m", "")

	const simple = "simple message"
	const trace = "a trace"

	event.To(ctx).Log(simple)
	checkFind(t, h, "Log", event.Message, true, simple)
	checkFind(t, h, "Log", event.Name, false, "")
	h.Reset()

	event.To(ctx).Metric(m.Record(3))
	checkFind(t, h, "Metric", event.Message, false, "")
	checkFind(t, h, "Metric", event.Name, false, "")
	h.Reset()

	event.To(ctx).Annotate()
	checkFind(t, h, "Annotate", event.Message, false, "")
	checkFind(t, h, "Annotate", event.Name, false, "")
	h.Reset()

	_, end := event.To(ctx).Start(trace)
	checkFind(t, h, "Start", event.Message, false, "")
	checkFind(t, h, "Start", event.Name, true, trace)
	h.Reset()

	end()
	checkFind(t, h, "End", event.Message, false, "")
	checkFind(t, h, "End", event.Name, false, "")
}

type finder interface {
	Find(*event.Event) (string, bool)
}

func checkFind(t *testing.T, h *eventtest.CaptureHandler, method string, key finder, match bool, text string) {
	if len(h.Got) != 1 {
		t.Errorf("Got %d events, expected 1", len(h.Got))
		return
	}
	m, ok := key.Find(&h.Got[0])
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
