// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

const (
	MetricKey      = interfaceKey("metric")
	MetricVal      = valueKey("metricValue")
	DurationMetric = interfaceKey("durationMetric")
)

type Kind int

const (
	unknownKind = Kind(iota)

	LogKind
	MetricKind
	TraceKind
)

type (
	valueKey     string
	interfaceKey string
)

func (k valueKey) Of(v Value) Label {
	return Label{Name: string(k), Value: v}
}

func (k valueKey) Matches(ev *Event) bool {
	_, found := k.Find(ev)
	return found
}

func (k valueKey) Find(ev *Event) (Value, bool) {
	return lookupValue(string(k), ev.Labels)
}

func (k interfaceKey) Of(v interface{}) Label {
	return Label{Name: string(k), Value: ValueOf(v)}
}

func (k interfaceKey) Matches(ev *Event) bool {
	_, found := k.Find(ev)
	return found
}

func (k interfaceKey) Find(ev *Event) (interface{}, bool) {
	v, ok := lookupValue(string(k), ev.Labels)
	if !ok {
		return nil, false
	}
	return v.Interface(), true

}

func lookupValue(name string, labels []Label) (Value, bool) {
	for i := len(labels) - 1; i >= 0; i-- {
		if labels[i].Name == name {
			return labels[i].Value, true
		}
	}
	return Value{}, false
}
