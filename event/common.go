// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

const Message = stringKey("msg")
const Name = stringKey("name")
const Trace = traceKey("trace")
const End = tagKey("end")
const Metric = tagKey("metric")

type stringKey string
type traceKey string
type tagKey string

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
	for i := len(ev.Labels) - 1; i >= 0; i-- {
		if ev.Labels[i].Name == string(k) {
			return ev.Labels[i].Value.Uint64(), true
		}
	}
	return 0, false
}

// Value creates a new tag Label.
func (k tagKey) Value() Label {
	return Label{Name: string(k)}
}

func (k tagKey) Matches(ev *Event) bool {
	for i := len(ev.Labels) - 1; i >= 0; i-- {
		if ev.Labels[i].Name == string(k) {
			return true
		}
	}
	return false
}
