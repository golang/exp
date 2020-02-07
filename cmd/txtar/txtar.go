// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The txtar command writes or extracts a text-based file archive in the format
// provided by the golang.org/x/tools/txtar package.
//
// The default behavior is to read a comment from stdin and write the archive
// file containing the recursive contents of the named files and directories,
// including hidden files, to stdout. Any non-flag arguments to the command name
// the files and/or directories to include, with the contents of directories
// included recursively. An empty argument list is equivalent to ".".
//
// The --extract (or -x) flag instructs txtar to instead read the archive file
// from stdin and extract all of its files to corresponding locations relative
// to the current, writing the archive's comment to stdout.
//
// Shell variables in paths are expanded (using os.Expand) if the corresponding
// variable is set in the process environment. When writing an archive, the
// variables (before expansion) are preserved in the archived paths.
//
// Example usage:
//
// 	txtar *.go <README >testdata/example.txt
//
// 	txtar --extract <playground_example.txt >main.go
//
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/tools/txtar"
)

var (
	extractFlag = flag.Bool("extract", false, "if true, extract files from the archive instead of writing to it")
)

func init() {
	flag.BoolVar(extractFlag, "x", *extractFlag, "short alias for --extract")
}

func main() {
	flag.Parse()

	var err error
	if *extractFlag {
		if len(flag.Args()) > 0 {
			fmt.Fprintln(os.Stderr, "Usage: txtar --extract <archive.txt")
			os.Exit(2)
		}
		err = extract()
	} else {
		paths := flag.Args()
		if len(paths) == 0 {
			paths = []string{"."}
		}
		err = archive(paths)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func extract() (err error) {
	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	ar := txtar.Parse(b)
	for _, f := range ar.Files {
		fileName := filepath.FromSlash(path.Clean(expand(f.Name)))
		if err := os.MkdirAll(filepath.Dir(fileName), 0777); err != nil {
			return err
		}
		if err := ioutil.WriteFile(fileName, f.Data, 0666); err != nil {
			return err
		}
	}

	if len(ar.Comment) > 0 {
		os.Stdout.Write(ar.Comment)
	}
	return nil
}

func archive(paths []string) (err error) {
	txtarHeader := regexp.MustCompile(`(?m)^-- .* --$`)

	ar := new(txtar.Archive)
	for _, p := range paths {
		root := filepath.Clean(expand(p))
		prefix := root + string(filepath.Separator)
		err := filepath.Walk(root, func(fileName string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}

			suffix := ""
			if fileName != root {
				suffix = strings.TrimPrefix(fileName, prefix)
			}
			name := filepath.ToSlash(filepath.Join(p, suffix))

			data, err := ioutil.ReadFile(fileName)
			if err != nil {
				return err
			}
			if txtarHeader.Match(data) {
				return fmt.Errorf("cannot archive %s: file contains a txtar header", name)
			}

			ar.Files = append(ar.Files, txtar.File{Name: name, Data: data})
			return nil
		})
		if err != nil {
			return err
		}
	}

	// After we have read all of the source files, read the comment from stdin.
	//
	// Wait until the read has been blocked for a while before prompting the user
	// to enter it: if they are piping the comment in from some other file, the
	// read should complete very quickly and there is no need for a prompt.
	// (200ms is typically long enough to read a reasonable comment from the local
	// machine, but short enough that humans don't notice it.)
	//
	// Don't prompt until we have successfully read the other files:
	// if we encountered an error, we don't need to ask for a comment.
	timer := time.AfterFunc(200*time.Millisecond, func() {
		fmt.Fprintln(os.Stderr, "Enter comment:")
	})
	comment, err := ioutil.ReadAll(os.Stdin)
	timer.Stop()
	if err != nil {
		return fmt.Errorf("reading comment from %s: %v", os.Stdin.Name(), err)
	}
	ar.Comment = bytes.TrimSpace(comment)

	_, err = os.Stdout.Write(txtar.Format(ar))
	return err
}

// expand is like os.ExpandEnv, but preserves unescaped variables (instead
// of escaping them to the empty string) if the variable is not set.
func expand(p string) string {
	return os.Expand(p, func(key string) string {
		v, ok := os.LookupEnv(key)
		if !ok {
			return "$" + key
		}
		return v
	})
}
