// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

const Message = stringKey("msg")
const Trace = stringKey("name")

type stringKey string

// Of creates a new message Label.
func (k stringKey) Of(msg string) Label {
	return Label{Name: string(k), Value: StringOf(msg)}
}

func (k stringKey) Find(ev *Event) (string, bool) {
	for i := len(ev.Labels) - 1; i >= 0; i-- {
		if ev.Labels[i].Name == string(k) {
			return ev.Labels[i].Value.String(), true
		}
	}
	return "", false
}
