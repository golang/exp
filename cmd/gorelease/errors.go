// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os/exec"
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/xerrors"
)

type usageError struct {
	err error
}

func usageErrorf(format string, args ...interface{}) error {
	return &usageError{err: fmt.Errorf(format, args...)}
}

const usageText = `usage: gorelease -base=version [-version=version]`

func (e *usageError) Error() string {
	msg := ""
	if !xerrors.Is(e.err, flag.ErrHelp) {
		msg = e.err.Error()
	}
	return usageText + "\n" + msg + "\nFor more information, run go doc golang.org/x/exp/cmd/gorelease"
}

type downloadError struct {
	m   module.Version
	err error
}

func (e *downloadError) Error() string {
	var msg string
	if xerr, ok := e.err.(*exec.ExitError); ok {
		msg = strings.TrimSpace(string(xerr.Stderr))
	} else {
		msg = e.err.Error()
	}
	sep := " "
	if strings.Contains(msg, "\n") {
		sep = "\n"
	}
	return fmt.Sprintf("error downloading module %s@%s:%s%s", e.m.Path, e.m.Version, sep, msg)
}
