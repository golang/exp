// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "fmt"

// A converter generates code to convert between a proto type and a veneer type.
type converter interface {
	// genFrom returns code to convert from proto to veneer.
	genFrom(string) string
	// genTo returns code to convert to proto from veneer.
	genTo(string) string
	// These return the function argument to Transform{Slice,MapValues}, or "" if we don't need it.
	genTransformFrom() string
	genTransformTo() string
}

// An identityConverter does no conversion.
type identityConverter struct{}

func (identityConverter) genFrom(arg string) string { return arg }
func (identityConverter) genTo(arg string) string   { return arg }
func (identityConverter) genTransformFrom() string  { return "" }
func (identityConverter) genTransformTo() string    { return "" }

// A derefConverter converts between T in the veneer and *T in the proto.
type derefConverter struct{}

func (derefConverter) genFrom(arg string) string { return fmt.Sprintf("support.DerefOrZero(%s)", arg) }
func (derefConverter) genTo(arg string) string   { return fmt.Sprintf("support.AddrOrNil(%s)", arg) }
func (derefConverter) genTransformFrom() string  { panic("can't handle deref slices") }
func (derefConverter) genTransformTo() string    { panic("can't handle deref slices") }

type enumConverter struct {
	protoName, veneerName string
}

func (c enumConverter) genFrom(arg string) string {
	return fmt.Sprintf("%s(%s)", c.veneerName, arg)
}

func (c enumConverter) genTransformFrom() string {
	return fmt.Sprintf("func(p pb.%s) %s { return %s }", c.protoName, c.veneerName, c.genFrom("p"))
}

func (c enumConverter) genTo(arg string) string {
	return fmt.Sprintf("pb.%s(%s)", c.protoName, arg)
}

func (c enumConverter) genTransformTo() string {
	return fmt.Sprintf("func(v %s) pb.%s { return %s }", c.veneerName, c.protoName, c.genTo("v"))
}

type protoConverter struct {
	veneerName string
}

func (c protoConverter) genFrom(arg string) string {
	return fmt.Sprintf("(%s{}).fromProto(%s)", c.veneerName, arg)
}

func (c protoConverter) genTransformFrom() string {
	return fmt.Sprintf("(%s{}).fromProto", c.veneerName)
}

func (c protoConverter) genTo(arg string) string {
	return fmt.Sprintf("%s.toProto()", arg)
}

func (c protoConverter) genTransformTo() string {
	return fmt.Sprintf("(*%s).toProto", c.veneerName)
}

type customConverter struct {
	toFunc, fromFunc string
}

func (c customConverter) genFrom(arg string) string {
	return fmt.Sprintf("%s(%s)", c.fromFunc, arg)
}

func (c customConverter) genTransformFrom() string { return c.fromFunc }

func (c customConverter) genTo(arg string) string {
	return fmt.Sprintf("%s(%s)", c.toFunc, arg)
}

func (c customConverter) genTransformTo() string { return c.toFunc }

type sliceConverter struct {
	eltConverter converter
}

func (c sliceConverter) genFrom(arg string) string {
	if fn := c.eltConverter.genTransformFrom(); fn != "" {
		return fmt.Sprintf("support.TransformSlice(%s, %s)", arg, fn)
	}
	return c.eltConverter.genFrom(arg)
}

func (c sliceConverter) genTo(arg string) string {
	if fn := c.eltConverter.genTransformTo(); fn != "" {
		return fmt.Sprintf("support.TransformSlice(%s, %s)", arg, fn)
	}
	return c.eltConverter.genTo(arg)
}

func (c sliceConverter) genTransformTo() string {
	panic("sliceConverter.genToSlice called")
}

func (c sliceConverter) genTransformFrom() string {
	panic("sliceConverter.genFromSlice called")
}

// Only the values are converted.
type mapConverter struct {
	valueConverter converter
}

func (c mapConverter) genFrom(arg string) string {
	if fn := c.valueConverter.genTransformFrom(); fn != "" {
		return fmt.Sprintf("support.TransformMapValues(%s, %s)", arg, fn)
	}
	return c.valueConverter.genFrom(arg)
}

func (c mapConverter) genTo(arg string) string {
	if fn := c.valueConverter.genTransformTo(); fn != "" {
		return fmt.Sprintf("support.TransformMapValues(%s, %s)", arg, fn)
	}
	return c.valueConverter.genTo(arg)
}

func (c mapConverter) genTransformTo() string {
	panic("mapConverter.genToSlice called")
}

func (c mapConverter) genTransformFrom() string {
	panic("mapConverter.genFromSlice called")
}
