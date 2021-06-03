// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zerolog_test

import (
	"context"
	"io"
	"testing"

	"github.com/rs/zerolog"
	"golang.org/x/exp/event/adapter/eventtest"
	"golang.org/x/exp/event/bench"
)

var (
	zerologMsg = bench.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			zerolog.Ctx(ctx).Info().Int(bench.A.Name, a).Msg(bench.A.Msg)
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			zerolog.Ctx(ctx).Info().Str(bench.B.Name, b).Msg(bench.B.Msg)
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}

	zerologMsgf = bench.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			zerolog.Ctx(ctx).Info().Msgf(bench.A.Msgf, a)
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			zerolog.Ctx(ctx).Info().Msgf(bench.B.Msgf, b)
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}
)

func zerologPrint(w io.Writer) context.Context {
	zerolog.TimeFieldFormat = bench.TimeFormat
	zerolog.TimestampFunc = eventtest.ExporterOptions().Now
	logger := zerolog.New(zerolog.SyncWriter(w)).With().Timestamp().Logger()
	return logger.WithContext(context.Background())
}

func BenchmarkLogZerolog(b *testing.B) {
	bench.RunBenchmark(b, zerologPrint(io.Discard), zerologMsg)
}

func BenchmarkLogZerologf(b *testing.B) {
	bench.RunBenchmark(b, zerologPrint(io.Discard), zerologMsgf)
}

func TestLogZerologf(t *testing.T) {
	bench.TestBenchmark(t, zerologPrint, zerologMsgf, `
{"level":"info","time":"2020/03/05 14:27:48","message":"a where A=0"}
{"level":"info","time":"2020/03/05 14:27:49","message":"b where B=\"A value\""}
{"level":"info","time":"2020/03/05 14:27:50","message":"a where A=1"}
{"level":"info","time":"2020/03/05 14:27:51","message":"b where B=\"Some other value\""}
{"level":"info","time":"2020/03/05 14:27:52","message":"a where A=22"}
{"level":"info","time":"2020/03/05 14:27:53","message":"b where B=\"Some other value\""}
{"level":"info","time":"2020/03/05 14:27:54","message":"a where A=333"}
{"level":"info","time":"2020/03/05 14:27:55","message":"b where B=\"\""}
{"level":"info","time":"2020/03/05 14:27:56","message":"a where A=4444"}
{"level":"info","time":"2020/03/05 14:27:57","message":"b where B=\"prime count of values\""}
{"level":"info","time":"2020/03/05 14:27:58","message":"a where A=55555"}
{"level":"info","time":"2020/03/05 14:27:59","message":"b where B=\"V\""}
{"level":"info","time":"2020/03/05 14:28:00","message":"a where A=666666"}
{"level":"info","time":"2020/03/05 14:28:01","message":"b where B=\"A value\""}
{"level":"info","time":"2020/03/05 14:28:02","message":"a where A=7777777"}
{"level":"info","time":"2020/03/05 14:28:03","message":"b where B=\"A value\""}
`)
}
