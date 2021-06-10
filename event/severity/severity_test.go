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
	"golang.org/x/exp/event/adapter/logfmt"
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
		events: func(ctx context.Context) { severity.Debug.Log(ctx, "a message") },
		expect: `time="2020/03/05 14:27:48" level=debug msg="a message"`,
	}, {
		name:   "info",
		events: func(ctx context.Context) { severity.Info.Log(ctx, "a message") },
		expect: `time="2020/03/05 14:27:48" level=info msg="a message"`},
	} {
		buf := &strings.Builder{}
		ctx := event.WithExporter(ctx, event.NewExporter(logfmt.NewHandler(buf), eventtest.ExporterOptions()))
		test.events(ctx)
		got := strings.TrimSpace(buf.String())
		expect := strings.TrimSpace(test.expect)
		if got != expect {
			t.Errorf("%s failed\ngot   : %q\nexpect: %q", test.name, got, expect)
		}
	}
}
