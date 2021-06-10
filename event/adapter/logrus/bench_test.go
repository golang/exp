// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logrus_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/exp/event/eventtest"
)

var (
	logrusLog = eventtest.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			logrusCtx(ctx).WithField(eventtest.A.Name, a).Info(eventtest.A.Msg)
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			logrusCtx(ctx).WithField(eventtest.B.Name, b).Info(eventtest.B.Msg)
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}

	logrusLogf = eventtest.Hooks{
		AStart: func(ctx context.Context, a int) context.Context {
			logrusCtx(ctx).Infof(eventtest.A.Msgf, a)
			return ctx
		},
		AEnd: func(ctx context.Context) {},
		BStart: func(ctx context.Context, b string) context.Context {
			logrusCtx(ctx).Infof(eventtest.B.Msgf, b)
			return ctx
		},
		BEnd: func(ctx context.Context) {},
	}
)

type logrusKey struct{}
type logrusTimeFormatter struct {
	now     func() time.Time
	wrapped logrus.Formatter
}

func (f *logrusTimeFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	entry.Time = f.now()
	return f.wrapped.Format(entry)
}

func logrusCtx(ctx context.Context) *logrus.Logger {
	return ctx.Value(logrusKey{}).(*logrus.Logger)
}

func logrusPrint(w io.Writer) context.Context {
	logger := &logrus.Logger{
		Out:   w,
		Level: logrus.InfoLevel,
		Formatter: &logrusTimeFormatter{
			now: eventtest.ExporterOptions().Now,
			wrapped: &logrus.TextFormatter{
				FullTimestamp:   true,
				TimestampFormat: eventtest.TimeFormat,
				DisableSorting:  true,
				DisableColors:   true,
			},
		},
	}
	return context.WithValue(context.Background(), logrusKey{}, logger)
}

func BenchmarkLogrusLogDiscard(b *testing.B) {
	eventtest.RunBenchmark(b, logrusPrint(io.Discard), logrusLog)
}

func BenchmarkLogrusLogfDiscard(b *testing.B) {
	eventtest.RunBenchmark(b, logrusPrint(io.Discard), logrusLogf)
}

func TestLogrusf(t *testing.T) {
	eventtest.TestBenchmark(t, logrusPrint, logrusLogf, eventtest.LogfOutput)
}
