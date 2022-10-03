// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slog_test

import (
	"net/http"
	"time"

	"golang.org/x/exp/slog"
)

func ExampleGroup() {
	var r *http.Request
	start := time.Now()
	// ...

	slog.Info("finished",
		slog.Group("req",
			slog.String("method", r.Method),
			slog.String("url", r.URL.String())),
		slog.Int("status", http.StatusOK),
		slog.Duration("duration", time.Since(start)))
}
