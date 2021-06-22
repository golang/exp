// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !disable_events

package event

import (
	"reflect"
	"runtime"
	"sort"
	"strings"
)

const (
	// this is the maximum amount of helpers we scan past to find a non helper
	helperDepthLimit = 5
)

type sources struct {
	entries []caller
}

type Source struct {
	Space string
	Owner string
	Name  string
}

type caller struct {
	helper bool
	pc     uintptr
	source Source
}

var globalCallers chan *sources

// RegisterHelper records a function as being an event helper that should not
// be used when capturing the source infomation on events.
// v should be either a string or a function pointer.
// If v is a string it is of the form
//   Space.Owner.Name
// where Owner and Name cannot contain '/' and Name also cannot contain '.'
func RegisterHelper(v interface{}) {
	g := <-globalCallers
	defer func() { globalCallers <- g }()
	switch v := v.(type) {
	case string:
		g.entries = append(g.entries, caller{source: splitName(v), helper: true})
	default:
		g.helperFunction(v)
	}
}

func init() {
	g := &sources{}
	// make all entries in the event package helpers
	globalCallers = make(chan *sources, 1)
	globalCallers <- g
	RegisterHelper("golang.org/x/exp/event")
}

func newCallers() sources {
	g := <-globalCallers
	defer func() { globalCallers <- g }()
	c := sources{}
	c.entries = make([]caller, len(g.entries))
	copy(c.entries, g.entries)
	return c
}

func (c *sources) addCaller(entry caller) {
	i := sort.Search(len(c.entries), func(i int) bool {
		return c.entries[i].pc >= entry.pc
	})
	if i >= len(c.entries) {
		// add to end
		c.entries = append(c.entries, entry)
		return
	}
	if c.entries[i].pc == entry.pc {
		// already present
		return
	}
	//expand the array
	c.entries = append(c.entries, caller{})
	//make a space
	copy(c.entries[i+1:], c.entries[i:])
	// insert the entry
	c.entries[i] = entry
}

func (c *sources) getCaller(pc uintptr) (caller, bool) {
	i := sort.Search(len(c.entries), func(i int) bool {
		return c.entries[i].pc >= pc
	})
	if i == len(c.entries) || c.entries[i].pc != pc {
		return caller{}, false
	}
	return c.entries[i], true
}

func scanStack() Source {
	g := <-globalCallers
	defer func() { globalCallers <- g }()
	return g.scanStack()
}

func (c *sources) scanStack() Source {
	// first capture the caller stack
	var stack [helperDepthLimit]uintptr
	// we can skip the first three entries
	//   runtime.Callers
	//   event.(*sources).scanStack (this function)
	//   another function in this package (because scanStack is private)
	depth := runtime.Callers(3, stack[:]) // start at 2 to skip Callers and this function
	// do a cheap first pass to see if we have an entry for this stack
	for i := 0; i < depth; i++ {
		pc := stack[i]
		e, found := c.getCaller(pc)
		if found {
			if !e.helper {
				// exact non helper match match found, return it
				return e.source
			}
			// helper found, keep scanning
			continue
		}
		// stack entry not found, we need to fill one in
		f := runtime.FuncForPC(stack[i])
		if f == nil {
			// symtab lookup failed, pretend it does not exist
			continue
		}
		e = caller{
			source: splitName(f.Name()),
			pc:     pc,
		}
		e.helper = c.isHelper(e)
		c.addCaller(e)
		if !e.helper {
			// found a non helper entry, add it and return it
			return e.source
		}
	}
	// ran out of stack, was all helpers
	return Source{}
}

// we do helper matching by name, if the pc matched we would have already found
// that, but helper registration does not know the call stack pcs
func (c *sources) isHelper(entry caller) bool {
	// scan to see if it matches any of the helpers
	// we match by name in case of inlining
	for _, e := range c.entries {
		if !e.helper {
			// ignore all the non helper entries
			continue
		}
		if isMatch(entry.source.Space, e.source.Space) &&
			isMatch(entry.source.Owner, e.source.Owner) &&
			isMatch(entry.source.Name, e.source.Name) {
			return true
		}
	}
	return false
}

func isMatch(value, against string) bool {
	return len(against) == 0 || value == against
}

func (c *sources) helperFunction(v interface{}) {
	r := reflect.ValueOf(v)
	pc := r.Pointer()
	f := runtime.FuncForPC(pc)
	entry := caller{
		source: splitName(f.Name()),
		pc:     f.Entry(),
		helper: true,
	}
	c.addCaller(entry)
	if entry.pc != pc {
		entry.pc = pc
		c.addCaller(entry)
	}
}

func splitName(full string) Source {
	// Function is the fully-qualified function name. The name itself may
	// have dots (for a closure, for instance), but it can't have slashes.
	// So the package path ends at the first dot after the last slash.
	entry := Source{Space: full}
	slash := strings.LastIndexByte(full, '/')
	if slash < 0 {
		slash = 0
	}
	if dot := strings.IndexByte(full[slash:], '.'); dot >= 0 {
		entry.Space = full[:slash+dot]
		entry.Name = full[slash+dot+1:]
		if dot = strings.LastIndexByte(entry.Name, '.'); dot >= 0 {
			entry.Owner = entry.Name[:dot]
			entry.Name = entry.Name[dot+1:]
		}
	}
	return entry
}
