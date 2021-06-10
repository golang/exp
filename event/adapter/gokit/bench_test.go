// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gokit_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/go-kit/kit/log"
	"golang.org/x/exp/event/adapter/logfmt"
	"golang.org/x/exp/event/eventtest"
)

var (
	gokitLog = eventtest.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			gokitCtx(ctx).Log(eventtest.A.Name, a, "msg", eventtest.A.Msg)
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			gokitCtx(ctx).Log(eventtest.B.Name, b, "msg", eventtest.B.Msg)
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}
	gokitLogf = eventtest.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			gokitCtx(ctx).Log("msg", fmt.Sprintf(eventtest.A.Msgf, a))
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			gokitCtx(ctx).Log("msg", fmt.Sprintf(eventtest.B.Msgf, b))
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}
)

type gokitKey struct{}

func gokitCtx(ctx context.Context) log.Logger {
	return ctx.Value(gokitKey{}).(log.Logger)
}

func gokitPrint(w io.Writer) context.Context {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(w))
	now := eventtest.ExporterOptions().Now
	logger = log.With(logger, "time", log.TimestampFormat(now, logfmt.TimeFormat), "level", "info")
	return context.WithValue(context.Background(), gokitKey{}, logger)
}

func BenchmarkGokitLogDiscard(b *testing.B) {
	eventtest.RunBenchmark(b, gokitPrint(io.Discard), gokitLog)
}

func BenchmarkGokitLogfDiscard(b *testing.B) {
	eventtest.RunBenchmark(b, gokitPrint(io.Discard), gokitLogf)
}

func TestGokitLogfDiscard(t *testing.T) {
	eventtest.TestBenchmark(t, gokitPrint, gokitLogf, eventtest.LogfOutput)
}
func TestLogGokit(t *testing.T) {
	eventtest.TestBenchmark(t, gokitPrint, gokitLog, eventtest.LogfmtOutput)
}
