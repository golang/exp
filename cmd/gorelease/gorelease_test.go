// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"
)

var workDir string

var (
	testwork     = flag.Bool("testwork", false, "preserve work directory")
	updateGolden = flag.Bool("u", false, "update expected text in test files instead of failing")
)

func TestMain(m *testing.M) {
	status := 1
	defer func() {
		if !*testwork && workDir != "" {
			os.RemoveAll(workDir)
		}
		os.Exit(status)
	}()

	flag.Parse()

	proxyDir, proxyURL, err := buildProxyDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	os.Setenv("GOPROXY", proxyURL)
	if *testwork {
		fmt.Fprintf(os.Stderr, "test proxy dir: %s\ntest proxy URL: %s\n", proxyDir, proxyURL)
	} else {
		defer os.RemoveAll(proxyDir)
	}

	cacheDir, err := ioutil.TempDir("", "gorelease_test-gocache")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	os.Setenv("GOPATH", cacheDir)
	if *testwork {
		fmt.Fprintf(os.Stderr, "test cache dir: %s\n", cacheDir)
	} else {
		defer func() {
			if err := exec.Command("go", "clean", "-modcache").Run(); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
			if err := os.RemoveAll(cacheDir); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}()
	}

	os.Setenv("GO111MODULE", "on")
	os.Setenv("GOSUMDB", "off")

	status = m.Run()
}

// test describes an individual test case, written as a .test file in the
// testdata directory.
//
// Each test is a txtar archive (see golang.org/x/tools/txtar). The comment
// section (before the first file) contains a sequence of key=value pairs
// (one per line) that configure the test.
//
// Most tests include a file named "want". The output of gorelease is compared
// against this file. If the -u flag is set, this file is replaced with the
// actual output of gorelease, and the test is written back to disk. This is
// useful for updating tests after cosmetic changes.
type test struct {
	txtar.Archive

	// testPath is the name of the .test file describing the test.
	testPath string

	// modPath (set with mod=...) is the path of the module being tested. Used
	// to retrieve files from the test proxy.
	modPath string

	// version (set with version=...) is the name of a version to check out
	// from the test proxy into the working directory. Some tests use this
	// instead of specifying files they need in the txtar archive.
	version string

	// baseVersion (set with base=...) is the value of the -base flag to pass
	// to gorelease.
	baseVersion string

	// releaseVersion (set with version=...) is the value of the -version flag
	// to pass to gorelease.
	releaseVersion string

	// dir (set with dir=...) is the directory where gorelease should be invoked.
	// If unset, gorelease is invoked in the directory where the txtar archive
	// is unpacked. This is useful for invoking gorelease in a subdirectory.
	dir string

	// wantError (set with error=...) is true if the test expects a hard error
	// (returned by runRelease).
	wantError bool

	// wantSuccess (set with success=...) is true if the test expects a report
	// to be returned without errors or diagnostics. True by default.
	wantSuccess bool

	// skip (set with skip=...) is non-empty if the test should be skipped.
	skip string

	// want is set to the contents of the file named "want" in the txtar archive.
	want []byte
}

// readTest reads and parses a .test file with the given name.
func readTest(testPath string) (*test, error) {
	arc, err := txtar.ParseFile(testPath)
	if err != nil {
		return nil, err
	}
	t := &test{
		Archive:     *arc,
		testPath:    testPath,
		wantSuccess: true,
	}

	for n, line := range bytes.Split(t.Comment, []byte("\n")) {
		lineNum := n + 1
		if i := bytes.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var key, value string
		if i := bytes.IndexByte(line, '='); i < 0 {
			return nil, fmt.Errorf("%s:%d: no '=' found", testPath, lineNum)
		} else {
			key = strings.TrimSpace(string(line[:i]))
			value = strings.TrimSpace(string(line[i+1:]))
		}
		switch key {
		case "mod":
			t.modPath = value
		case "version":
			t.version = value
		case "base":
			t.baseVersion = value
		case "release":
			t.releaseVersion = value
		case "dir":
			t.dir = value
		case "skip":
			t.skip = value
		case "success":
			t.wantSuccess, err = strconv.ParseBool(value)
			if err != nil {
				return nil, fmt.Errorf("%s:%d: %v", testPath, lineNum, err)
			}
		case "error":
			t.wantError, err = strconv.ParseBool(value)
			if err != nil {
				return nil, fmt.Errorf("%s:%d: %v", testPath, lineNum, err)
			}
		default:
			return nil, fmt.Errorf("%s:%d: unknown key: %q", testPath, lineNum, key)
		}
	}
	if t.modPath == "" && (t.version != "" || (t.baseVersion != "" && t.baseVersion != "none")) {
		return nil, fmt.Errorf("%s: version or base was set but mod was not set", testPath)
	}

	haveFiles := false
	for _, f := range t.Files {
		if f.Name == "want" {
			t.want = bytes.TrimSpace(f.Data)
			continue
		}
		haveFiles = true
	}

	if haveFiles && t.version != "" {
		return nil, fmt.Errorf("%s: version is set but files are present", testPath)
	}

	return t, nil
}

// updateTest replaces the contents of the file named "want" within a test's
// txtar archive, then formats and writes the test file.
func updateTest(t *test, want []byte) error {
	var wantFile *txtar.File
	for i := range t.Files {
		if t.Files[i].Name == "want" {
			wantFile = &t.Files[i]
			break
		}
	}
	if wantFile == nil {
		t.Files = append(t.Files, txtar.File{Name: "want"})
		wantFile = &t.Files[len(t.Files)-1]
	}

	wantFile.Data = want
	data := txtar.Format(&t.Archive)
	return ioutil.WriteFile(t.testPath, data, 0666)
}

func TestRelease(t *testing.T) {
	testPaths, err := filepath.Glob(filepath.FromSlash("testdata/*/*.test"))
	if err != nil {
		t.Fatal(err)
	}
	if len(testPaths) == 0 {
		t.Fatal("no .test files found in testdata directory")
	}

	for _, testPath := range testPaths {
		testPath := testPath
		testName := strings.TrimSuffix(strings.TrimPrefix(filepath.ToSlash(testPath), "testdata/"), ".test")
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			test, err := readTest(testPath)
			if err != nil {
				t.Fatal(err)
			}

			if test.skip != "" {
				t.Skip(test.skip)
			}

			// Extract the files in the release version. They may be part of the
			// test archive or in testdata/mod.
			testDir, err := ioutil.TempDir("", "")
			if err != nil {
				t.Fatal(err)
			}
			if *testwork {
				fmt.Fprintf(os.Stderr, "test dir: %s\n", testDir)
			} else {
				defer os.RemoveAll(testDir)
			}

			var arc *txtar.Archive
			if test.version != "" {
				arcBase := fmt.Sprintf("%s_%s.txt", strings.ReplaceAll(test.modPath, "/", "_"), test.version)
				arcPath := filepath.Join("testdata/mod", arcBase)
				var err error
				arc, err = txtar.ParseFile(arcPath)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				arc = &test.Archive
			}
			if err := extractTxtar(testDir, arc); err != nil {
				t.Fatal(err)
			}

			// Generate the report and compare it against the expected text.
			var args []string
			if test.baseVersion != "" {
				args = append(args, "-base="+test.baseVersion)
			}
			if test.releaseVersion != "" {
				args = append(args, "-version="+test.releaseVersion)
			}
			buf := &bytes.Buffer{}
			releaseDir := filepath.Join(testDir, test.dir)
			success, err := runRelease(buf, releaseDir, args)
			if err != nil {
				if !test.wantError {
					t.Fatalf("unexpected error: %v", err)
				}
				if errMsg := []byte(err.Error()); !bytes.Equal(errMsg, bytes.TrimSpace(test.want)) {
					if *updateGolden {
						if err := updateTest(test, errMsg); err != nil {
							t.Fatal(err)
						}
					} else {
						t.Fatalf("got error: %s; want error: %s", errMsg, test.want)
					}
				}
				return
			}
			if test.wantError {
				t.Fatalf("got success; want error %s", test.want)
			}

			got := bytes.TrimSpace(buf.Bytes())
			if filepath.Separator != '/' {
				got = bytes.ReplaceAll(got, []byte{filepath.Separator}, []byte{'/'})
			}
			if !bytes.Equal(got, test.want) {
				if *updateGolden {
					if err := updateTest(test, got); err != nil {
						t.Fatal(err)
					}
				} else {
					t.Fatalf("got:\n%s\n\nwant:\n%s", got, test.want)
				}
			}
			if success != test.wantSuccess {
				t.Fatalf("got success: %v; want success %v", success, test.wantSuccess)
			}
		})
	}
}
