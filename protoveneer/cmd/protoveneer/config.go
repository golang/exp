// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// config holds the configuration for a package.
type config struct {
	Package         string
	ProtoImportPath string `yaml:"protoImportPath"`
	// Import path for the support package needed by the generated code.
	SupportImportPath string `yaml:"supportImportPath"`

	// The types to process. Only these types and the types they depend
	// on will be output.
	// The key is the name of the proto type.
	Types map[string]*typeConfig
	// Omit the types in this list, even if they would normally be output.
	// Elements can be globs.
	OmitTypes []string `yaml:"omitTypes"`
	// Converter functions for types not in the proto package.
	// Each value should be "tofunc, fromfunc"
	Converters map[string]string
}

type typeConfig struct {
	// The name for the veneer type, if different.
	Name string
	// The prefix of the proto enum values. It will be removed.
	ProtoPrefix string `yaml:"protoPrefix"`
	// The prefix for the veneer enum values, if different from the type name.
	VeneerPrefix string `yaml:"veneerPrefix"`
	// Overrides for enum values.
	ValueNames map[string]string `yaml:"valueNames"`
	// Overrides for field types. Map key is proto field name.
	Fields map[string]fieldConfig
	// Custom conversion functions: "tofunc, fromfunc"
	ConvertToFrom string `yaml:"convertToFrom"`
	// Doc string for the type, omitting the initial type name.
	Doc string
	// Verb to place after type name in doc. Default: "is".
	// Ignored if Doc is non-empty.
	DocVerb string `yaml:"docVerb"`
}

type fieldConfig struct {
	Name string // veneer name
	Type string // veneer type
	// Omit from output.
	Omit bool
}

func (c *config) init() {
	for protoName, tc := range c.Types {
		if tc == nil {
			tc = &typeConfig{Name: protoName}
			c.Types[protoName] = tc
		}
		if tc.Name == "" {
			tc.Name = protoName
		}
		tc.init()
	}
}

func (tc *typeConfig) init() {
	if tc.VeneerPrefix == "" {
		tc.VeneerPrefix = tc.Name
	}
}

func readConfigFile(filename string) (*config, error) {
	if filename == "" {
		return nil, errors.New("missing config file")
	}
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	var c config
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("reading %s: %w", filename, err)
	}
	c.init()
	return &c, nil
}
