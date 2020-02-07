// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

const comment = "This is a txtar archive.\n"

const testdata = `This is a txtar archive.
-- one.txt --
one
-- dir/two.txt --
two
-- $SPECIAL_LOCATION/three.txt --
three
`

func TestMain(m *testing.M) {
	code := m.Run()
	txtarBin.once.Do(func() {})
	if txtarBin.name != "" {
		os.Remove(txtarBin.name)
	}
	os.Exit(code)
}

func TestRoundTrip(t *testing.T) {
	os.Setenv("SPECIAL_LOCATION", "special")
	defer os.Unsetenv("SPECIAL_LOCATION")

	// Expand the testdata archive into a temporary directory.
	parentDir, err := ioutil.TempDir("", "txtar")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(parentDir)
	dir := filepath.Join(parentDir, "dir")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if out := txtar(t, dir, testdata, "--extract"); out != comment {
		t.Fatalf("txtar --extract: stdout:\n%s\nwant:\n%s", out, comment)
	}

	// Now, re-archive its contents explicitly and ensure that the result matches
	// the original.
	args := []string{"one.txt", "dir", "$SPECIAL_LOCATION"}
	if out := txtar(t, dir, comment, args...); out != testdata {
		t.Fatalf("txtar %s: archive:\n%s\n\nwant:\n%s", strings.Join(args, " "), out, testdata)
	}
}

// txtar runs the txtar command in the given directory with the given input and
// arguments.
func txtar(t *testing.T, dir, input string, args ...string) string {
	t.Helper()
	cmd := exec.Command(txtarName(t), args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(input)
	stderr := new(strings.Builder)
	cmd.Stderr = stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s: %v\n%s", strings.Join(cmd.Args, " "), err, stderr)
	}
	if stderr.String() != "" {
		t.Logf("OK: %s\n%s", strings.Join(cmd.Args, " "), stderr)
	}
	return string(out)
}

var txtarBin struct {
	once sync.Once
	name string
	err  error
}

// txtarName returns the name of the txtar executable, building it if needed.
func txtarName(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("cannot build txtar binary: %v", err)
	}

	txtarBin.once.Do(func() {
		exe, err := ioutil.TempFile("", "txtar-*.exe")
		if err != nil {
			txtarBin.err = err
			return
		}
		exe.Close()
		txtarBin.name = exe.Name()

		cmd := exec.Command("go", "build", "-o", txtarBin.name, ".")
		out, err := cmd.CombinedOutput()
		if err != nil {
			txtarBin.err = fmt.Errorf("%s: %v\n%s", strings.Join(cmd.Args, " "), err, out)
		}
	})

	if txtarBin.err != nil {
		if runtime.GOOS == "android" {
			t.Skipf("skipping test after failing to build txtar binary: go_android_exec may have failed to copy needed dependencies (see https://golang.org/issue/37088)")
		}
		t.Fatal(txtarBin.err)
	}
	return txtarBin.name
}
