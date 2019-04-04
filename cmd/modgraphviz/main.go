// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The modgraphviz command translates the output for go mod graph into .dot
// notation, which can then be parsed by `dot` into visual graphs.
//
// Usage: GO111MODULE=on go mod graph | modgraphviz | dot -Tpng -o outfile.png
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func main() {
	flag.Usage = func() {
		log.Println("Usage: GO111MODULE=on go mod graph | modgraphviz | dot -Tpng -o outfile.png")
	}
	flag.Parse()

	var out bytes.Buffer

	if err := Run(os.Stdin, &out); err != nil {
		log.Fatal(err)
	}

	if _, err := out.WriteTo(os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func Run(in io.Reader, out io.Writer) error {
	if _, err := out.Write([]byte("digraph gomodgraph {\n")); err != nil {
		return err
	}

	r := bufio.NewScanner(in)
	for {
		if !r.Scan() {
			if r.Err() != nil {
				return r.Err()
			}
			break
		}

		parts := strings.Fields(r.Text())
		if len(parts) != 2 {
			continue
		}

		if _, err := fmt.Fprintf(out, "\t%q -> %q\n", parts[0], parts[1]); err != nil {
			return err
		}
	}

	if _, err := out.Write([]byte("}\n")); err != nil {
		return err
	}

	return nil
}
