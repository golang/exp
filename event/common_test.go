// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package event_test

import (
	"testing"

	"golang.org/x/exp/event"
	"golang.org/x/exp/event/eventtest"
	"golang.org/x/exp/event/keys"
)

func TestCommon(t *testing.T) {
	ctx, h := eventtest.NewCapture()
	m := event.NewCounter("m", "")

	const simple = "simple message"
	const trace = "a trace"

	event.Log(ctx, simple)
	checkMessage(t, h, "Log", simple)
	checkName(t, h, "Log", "")
	h.Reset()

	event.Logf(ctx, "logf %s message", "to")
	checkMessage(t, h, "Logf", "logf to message")
	checkName(t, h, "Logf", "")
	h.Reset()

	m.Record(ctx, 3)
	checkMessage(t, h, "Metric", "")
	checkName(t, h, "Metric", "")
	h.Reset()

	event.Annotate(ctx, keys.String("").Of(""))
	checkMessage(t, h, "Annotate", "")
	checkName(t, h, "Annotate", "")
	h.Reset()

	ctx = event.Start(ctx, trace)
	checkMessage(t, h, "Start", "")
	checkName(t, h, "Start", trace)
	h.Reset()

	event.End(ctx)
	checkMessage(t, h, "End", "")
	checkName(t, h, "End", "")
}

type finder interface {
	Find(*event.Event) (string, bool)
}

func checkMessage(t *testing.T, h *eventtest.CaptureHandler, method string, text string) {
	if len(h.Got) != 1 {
		t.Errorf("Got %d events, expected 1", len(h.Got))
		return
	}
	if h.Got[0].Message != text {
		t.Errorf("Expected event with Message %q from %s got %q", text, method, h.Got[0].Message)
	}
}

func checkName(t *testing.T, h *eventtest.CaptureHandler, method string, text string) {
	if len(h.Got) != 1 {
		t.Errorf("Got %d events, expected 1", len(h.Got))
		return
	}
	if h.Got[0].Name != text {
		t.Errorf("Expected event with Name %q from %s got %q", text, method, h.Got[0].Name)
	}
}
