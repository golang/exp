// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

const (
	Message        = stringKey("msg")
	Name           = stringKey("name")
	Trace          = traceKey("trace")
	End            = tagKey("end")
	MetricKey      = interfaceKey("metric")
	MetricVal      = valueKey("metricValue")
	DurationMetric = interfaceKey("durationMetric")
	Error          = errorKey("error")
)

type (
	stringKey    string
	traceKey     string
	tagKey       string
	valueKey     string
	interfaceKey string
	errorKey     string
)

// Of creates a new message Label.
func (k stringKey) Of(msg string) Label {
	return Label{Name: string(k), Value: StringOf(msg)}
}

func (k stringKey) Matches(ev *Event) bool {
	_, found := k.Find(ev)
	return found
}

func (k stringKey) Find(ev *Event) (string, bool) {
	for i := len(ev.Labels) - 1; i >= 0; i-- {
		if ev.Labels[i].Name == string(k) {
			return ev.Labels[i].Value.String(), true
		}
	}
	return "", false
}

// Of creates a new start Label.
func (k traceKey) Of(id uint64) Label {
	return Label{Name: string(k), Value: Uint64Of(id)}
}

func (k traceKey) Matches(ev *Event) bool {
	_, found := k.Find(ev)
	return found
}

func (k traceKey) Find(ev *Event) (uint64, bool) {
	if v, ok := lookupValue(string(k), ev.Labels); ok {
		return v.Uint64(), true
	}
	return 0, false
}

// Value creates a new tag Label.
func (k tagKey) Value() Label {
	return Label{Name: string(k)}
}

func (k tagKey) Matches(ev *Event) bool {
	_, ok := lookupValue(string(k), ev.Labels)
	return ok
}

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

// Of creates a new error Label.
func (k errorKey) Of(err error) Label {
	return Label{Name: string(k), Value: ValueOf(err)}
}

func (k errorKey) Matches(ev *Event) bool {
	_, found := k.Find(ev)
	return found
}

func (k errorKey) Find(ev *Event) (error, bool) {
	for i := len(ev.Labels) - 1; i >= 0; i-- {
		if ev.Labels[i].Name == string(k) {
			return ev.Labels[i].Value.Interface().(error), true
		}
	}
	return nil, false
}
