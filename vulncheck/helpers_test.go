// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vulncheck

import (
	"fmt"

	"golang.org/x/vulndb/osv"
)

type mockClient struct {
	ret map[string][]*osv.Entry
}

func (mc *mockClient) GetByModule(a string) ([]*osv.Entry, error) {
	return mc.ret[a], nil
}

func (mc *mockClient) GetByID(a string) (*osv.Entry, error) {
	return nil, nil
}

func moduleVulnerabilitiesToString(mv moduleVulnerabilities) string {
	var s string
	for _, m := range mv {
		s += fmt.Sprintf("mod: %v\n", m.mod)
		for _, v := range m.vulns {
			s += fmt.Sprintf("\t%v\n", v)
		}
	}
	return s
}

func vulnsToString(vulns []*osv.Entry) string {
	var s string
	for _, v := range vulns {
		s += fmt.Sprintf("\t%v\n", v)
	}
	return s
}
