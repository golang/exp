// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iconvg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpgradeToFileFormatVersion1(t *testing.T) {
	for _, tc := range testdataTestCases {
		original, err := os.ReadFile(filepath.FromSlash(tc.filename) + ".ivg")
		if err != nil {
			t.Errorf("%s: ReadFile: %v", tc.filename, err)
			continue
		}

		upgraded, err := UpgradeToFileFormatVersion1(original, nil)
		if err != nil {
			t.Errorf("%s: Upgrade: %v", tc.filename, err)
			continue
		}

		// For most of the testdataTestCases, we just check (above) that
		// calling UpgradeToFileFormatVersion1 returns a nil error. As a
		// further basic consistency check, we hard-code the expected results
		// for upgrading the "action-info.lores" icon.
		//
		// These 36 bytes (and its disassembly via the cmd/iconvg-disassemble
		// tool) is also a file in the test/data directory of the
		// github.com/google/iconvg repository (the repository that is
		// generally responsible for "File Format Version 1").
		if tc.filename == "testdata/action-info.lores" {
			const want = "" +
				"\x8A\x49\x56\x47\x03\x0B\x11\x51\x51\xB1\xB1\x35\x81\x59\x33\x59" +
				"\x81\x81\xA9\x35\x85\x95\x34\x7D\x95\x7D\x7D\x35\x85\x75\x34\x7D" +
				"\x75\x7D\x6D\x88"
			if got := string(upgraded); got != want {
				t.Errorf("%s: Upgrade: got:\n% 02x\nwant:\n% 02x", tc.filename, got, want)
				continue
			}
		}
	}
}
