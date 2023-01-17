// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slog_test

import (
	"net/http"
	"os"
	"time"

	"golang.org/x/exp/slog"
)

func ExampleGroup() {
	r, _ := http.NewRequest("GET", "localhost", nil)
	// ...

	// Remove the time attribute to make Output deterministic.
	removeTime := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey && len(groups) == 0 {
			a.Key = ""
		}
		return a
	}
	logger := slog.New(slog.HandlerOptions{ReplaceAttr: removeTime}.NewTextHandler(os.Stdout))
	slog.SetDefault(logger)

	slog.Info("finished",
		slog.Group("req",
			slog.String("method", r.Method),
			slog.String("url", r.URL.String())),
		slog.Int("status", http.StatusOK),
		slog.Duration("duration", time.Second))

	// Output:
	// level=INFO msg=finished req.method=GET req.url=localhost status=200 duration=1s
}
