// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package severity_test

import (
	"context"
	"strings"
	"testing"

	"golang.org/x/exp/event"
	"golang.org/x/exp/event/adapter/eventtest"
	"golang.org/x/exp/event/adapter/logfmt"
	"golang.org/x/exp/event/severity"
)

func TestPrint(t *testing.T) {
	//TODO: print the textual form of severity
	ctx := context.Background()
	for _, test := range []struct {
		name   string
		events func(context.Context)
		expect string
	}{{
		name:   "debug",
		events: func(ctx context.Context) { event.To(ctx).With(severity.Debug).Log("a message") },
		expect: `time=2020-03-05T14:27:48 id=1 kind=log msg="a message" level=debug`,
	}, {
		name:   "info",
		events: func(ctx context.Context) { event.To(ctx).With(severity.Info).Log("a message") },
		expect: `time=2020-03-05T14:27:48 id=1 kind=log msg="a message" level=info`},
	} {
		buf := &strings.Builder{}
		h := logfmt.Printer(buf)
		e := event.NewExporter(h)
		e.Now = eventtest.TestNow()
		ctx := event.WithExporter(ctx, e)
		test.events(ctx)
		got := strings.TrimSpace(buf.String())
		expect := strings.TrimSpace(test.expect)
		if got != expect {
			t.Errorf("%s failed\ngot   : %q\nexpect: %q", test.name, got, expect)
		}
	}
}
