// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"context"
	"fmt"
	"sync"
)

const (
	MetricKey      = "metric"
	MetricVal      = "metricValue"
	DurationMetric = interfaceKey("durationMetric")
)

type Kind int

const (
	unknownKind = Kind(iota)

	LogKind
	MetricKind
	StartKind
	EndKind

	dynamicKindStart
)

type (
	valueKey     string
	interfaceKey string
)

var (
	dynamicKindMu    sync.Mutex
	nextDynamicKind  = dynamicKindStart
	dynamicKindNames map[Kind]string
)

func NewKind(name string) Kind {
	dynamicKindMu.Lock()
	defer dynamicKindMu.Unlock()
	for _, n := range dynamicKindNames {
		if n == name {
			panic(fmt.Errorf("kind %s is already registered", name))
		}
	}
	k := nextDynamicKind
	nextDynamicKind++
	dynamicKindNames[k] = name
	return k
}

func (k Kind) String() string {
	switch k {
	case unknownKind:
		return "unknown"
	case LogKind:
		return "log"
	case MetricKind:
		return "metric"
	case StartKind:
		return "start"
	case EndKind:
		return "end"
	default:
		dynamicKindMu.Lock()
		defer dynamicKindMu.Unlock()
		name, ok := dynamicKindNames[k]
		if !ok {
			return fmt.Sprintf("?unknownKind:%d?", k)
		}
		return name
	}
}

func Log(ctx context.Context, msg string, labels ...Label) {
	ev := New(ctx, LogKind)
	if ev != nil {
		ev.Labels = append(ev.Labels, labels...)
		ev.Labels = append(ev.Labels, String("msg", msg))
		ev.Deliver()
	}
}

func Logf(ctx context.Context, msg string, args ...interface{}) {
	ev := New(ctx, LogKind)
	if ev != nil {
		ev.Labels = append(ev.Labels, String("msg", fmt.Sprintf(msg, args...)))
		ev.Deliver()
	}
}

func Error(ctx context.Context, msg string, err error, labels ...Label) {
	ev := New(ctx, LogKind)
	if ev != nil {
		ev.Labels = append(ev.Labels, labels...)
		ev.Labels = append(ev.Labels, String("msg", msg), Value("error", err))
		ev.Deliver()
	}
}

func Annotate(ctx context.Context, labels ...Label) {
	ev := New(ctx, 0)
	if ev != nil {
		ev.Labels = append(ev.Labels, labels...)
		ev.Deliver()
	}
}

func Start(ctx context.Context, name string, labels ...Label) context.Context {
	ev := New(ctx, StartKind)
	if ev != nil {
		ev.Labels = append(ev.Labels, String("name", name))
		ev.Labels = append(ev.Labels, labels...)
		ev.Trace()
		ctx = ev.Deliver()
	}
	return ctx
}

func End(ctx context.Context, labels ...Label) {
	ev := New(ctx, EndKind)
	if ev != nil {
		ev.Labels = append(ev.Labels, labels...)
		ev.prepare()
		// this was an end event, do we need to send a duration?
		if v, ok := DurationMetric.Find(ev); ok {
			//TODO: do we want the rest of the values from the end event?
			v.(*DurationDistribution).Record(ctx, ev.At.Sub(ev.target.startTime))
		}
		ev.Deliver()
	}
}

func (k interfaceKey) Of(v interface{}) Label {
	return Value(string(k), v)
}

func (k interfaceKey) Find(ev *Event) (interface{}, bool) {
	v, ok := lookupValue(string(k), ev.Labels)
	if !ok {
		return nil, false
	}
	return v.Interface(), true

}

func lookupValue(name string, labels []Label) (Label, bool) {
	for i := len(labels) - 1; i >= 0; i-- {
		if labels[i].Name == name {
			return labels[i], true
		}
	}
	return Label{}, false
}
