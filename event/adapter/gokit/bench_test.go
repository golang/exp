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
	"golang.org/x/exp/event/adapter/eventtest"
	"golang.org/x/exp/event/adapter/logfmt"
	"golang.org/x/exp/event/bench"
)

var (
	gokitLog = bench.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			gokitCtx(ctx).Log(bench.A.Name, a, "msg", bench.A.Msg)
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			gokitCtx(ctx).Log(bench.B.Name, b, "msg", bench.B.Msg)
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}
	gokitLogf = bench.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			gokitCtx(ctx).Log("msg", fmt.Sprintf(bench.A.Msgf, a))
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			gokitCtx(ctx).Log("msg", fmt.Sprintf(bench.B.Msgf, b))
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
	bench.RunBenchmark(b, gokitPrint(io.Discard), gokitLog)
}

func BenchmarkGokitLogfDiscard(b *testing.B) {
	bench.RunBenchmark(b, gokitPrint(io.Discard), gokitLogf)
}

func TestGokitLogfDiscard(t *testing.T) {
	bench.TestBenchmark(t, gokitPrint, gokitLogf, bench.LogfOutput)
}
func TestLogGokit(t *testing.T) {
	bench.TestBenchmark(t, gokitPrint, gokitLog, bench.LogfmtOutput)
}
