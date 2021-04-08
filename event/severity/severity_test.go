// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package severity_test

import (
	"context"
	"strings"
	"testing"

	"golang.org/x/exp/event"
	"golang.org/x/exp/event/eventtest"
	"golang.org/x/exp/event/severity"
)

func TestPrint(t *testing.T) {
	ctx := context.Background()
	for _, test := range []struct {
		name   string
		events func(context.Context)
		expect string
	}{{
		name:   "debug",
		events: func(ctx context.Context) { event.To(ctx).With(severity.Debug).Log("a message") },
		expect: `
2020/03/05 14:27:48 [log:1] a message
	severity=debug
`}, {
		name:   "info",
		events: func(ctx context.Context) { event.To(ctx).With(severity.Info).Log("a message") },
		expect: `
2020/03/05 14:27:48 [log:1] a message
	severity=info
`}} {
		h := &captureHandler{}
		h.printer = event.NewPrinter(&h.buf)
		e := event.NewExporter(h)
		e.Now = eventtest.TestNow()
		ctx := event.WithExporter(ctx, e)
		test.events(ctx)
		got := strings.TrimSpace(h.buf.String())
		expect := strings.TrimSpace(test.expect)
		if got != expect {
			t.Errorf("%s failed\ngot   : %q\nexpect: %q", test.name, got, expect)
		}
	}
}

type captureHandler struct {
	printer event.Printer
	buf     strings.Builder
}

func (e *captureHandler) Handle(ev *event.Event) {
	e.printer.Handle(ev)
}
