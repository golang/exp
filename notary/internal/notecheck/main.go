// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Notecheck checks a go.sum file against a notary.
//
// WARNING! This program is meant as a proof of concept demo and
// should not be used in production scripts.
// It does not set an exit status to report whether the
// checksums matched, and it does not filter the go.sum
// according to the $GONOVERIFY environment variable.
//
// Usage:
//
//	notecheck [-v] notary-key go.sum
//
// The -v flag enables verbose output.
//
package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/notary/internal/note"
	"golang.org/x/exp/notary/internal/tlog"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: notecheck [-u url] [-h H] [-v] notary-key go.sum...\n")
	os.Exit(2)
}

var height = flag.Int("h", 2, "tile height")
var vflag = flag.Bool("v", false, "enable verbose output")
var url = flag.String("u", "", "url to notary (overriding name)")

func main() {
	log.SetPrefix("notecheck: ")
	log.SetFlags(0)

	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 2 {
		usage()
	}

	vkey := flag.Arg(0)
	verifier, err := note.NewVerifier(vkey)
	if err != nil {
		log.Fatal(err)
	}
	if *url == "" {
		*url = "https://" + verifier.Name()
	}
	msg, err := httpGet(*url + "/latest")
	if err != nil {
		log.Fatal(err)
	}
	treeNote, err := note.Open(msg, note.NotaryList(verifier))
	if err != nil {
		log.Fatalf("reading note: %v\nnote:\n%s", err, msg)
	}
	tree, err := tlog.ParseTree([]byte(treeNote.Text))
	if err != nil {
		log.Fatal(err)
	}

	if *vflag {
		log.Printf("validating against %s @%d", verifier.Name(), tree.N)
	}

	verifierURL := *url
	tr := &tileReader{url: verifierURL + "/"}
	thr := tlog.TileHashReader(tree, tr)
	if _, err := tlog.TreeHash(tree.N, thr); err != nil {
		log.Fatal(err)
	}

	for _, arg := range flag.Args()[1:] {
		data, err := ioutil.ReadFile(arg)
		if err != nil {
			log.Fatal(err)
		}
		log.SetPrefix("notecheck: " + arg + ": ")
		checkGoSum(data, verifierURL, thr)
		log.SetPrefix("notecheck: ")
	}
}

func checkGoSum(data []byte, verifierURL string, thr tlog.HashReader) {
	lines := strings.SplitAfter(string(data), "\n")
	if lines[len(lines)-1] != "" {
		log.Printf("error: final line missing newline")
		return
	}
	lines = lines[:len(lines)-1]
	if len(lines)%2 != 0 {
		log.Printf("error: odd number of lines")
	}
	for i := 0; i+2 <= len(lines); i += 2 {
		f1 := strings.Fields(lines[i])
		f2 := strings.Fields(lines[i+1])
		if len(f1) != 3 || len(f2) != 3 || f1[0] != f2[0] || f1[1]+"/go.mod" != f2[1] {
			log.Printf("error: bad line pair:\n\t%s\t%s", lines[i], lines[i+1])
			continue
		}

		data, err := httpGet(verifierURL + "/lookup/" + f1[0] + "@" + f1[1])
		if err != nil {
			log.Printf("%s@%s: %v", f1[0], f1[1], err)
			continue
		}
		j := bytes.IndexByte(data, '\n')
		if j < 0 {
			log.Printf("%s@%s: short response from lookup", f1[0], f1[1])
			continue
		}
		id, err := strconv.ParseInt(strings.TrimSpace(string(data[:j])), 10, 64)
		if err != nil {
			log.Printf("%s@%s: unexpected response:\n%s", f1[0], f1[1], data)
			continue
		}
		ldata := data[j+1:]

		c := make(chan *tlog.Hash, 1)
		go func() {
			hashes, err := thr.ReadHashes([]int64{tlog.StoredHashIndex(0, id)})
			if err != nil {
				log.Printf("%s@%s: %v", f1[0], f1[1], err)
				c <- nil
				return
			}
			c <- &hashes[0]
		}()

		// The record lookup can be skipped in favor of using the /lookup response
		// but we fetch record and test that they match, to check the server.
		data, err = httpGet(verifierURL + "/record/" + fmt.Sprint(id))
		if err != nil {
			log.Printf("%s@%s: %v", f1[0], f1[1], err)
			continue
		}
		if !bytes.Equal(data, ldata) {
			log.Printf("%s@%s: different data from lookup and record:\n%s\n%s", hex.Dump(ldata), hex.Dump(data))
			continue
		}

		hash := tlog.RecordHash(data)
		hash1 := <-c
		if hash1 == nil {
			continue
		}
		if *hash1 != hash {
			log.Printf("%s@%s: inconsistent records on notary!", f1[0], f1[1])
			continue
		}
		if string(data) != lines[i]+lines[i+1] {
			log.Printf("%s@%s: invalid go.sum entries:\nhave:\n\t%s\t%swant:\n\t%s", f1[0], f1[1], lines[i], lines[i+1], strings.Replace(string(data), "\n", "\n\t", -1))
		}
	}
}

func init() {
	http.DefaultClient.Timeout = 10 * time.Second
}

func httpGet(url string) ([]byte, error) {
	start := time.Now()
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GET %v: %v", url, resp.Status)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if *vflag {
		fmt.Fprintf(os.Stderr, "%.3fs %s\n", time.Since(start).Seconds(), url)
	}
	return data, nil
}

type tileReader struct {
	url     string
	cache   map[tlog.Tile][]byte
	cacheMu sync.Mutex
}

func (r *tileReader) Height() int {
	return *height
}

func (r *tileReader) Reject(tile tlog.Tile) {
	log.Printf("tile rejected: %v", tile.Path())
}

// TODO(rsc): Move some variant of this to package tlog
// once we are more sure of the API.

func (r *tileReader) ReadTiles(tiles []tlog.Tile) ([][]byte, error) {
	var wg sync.WaitGroup
	out := make([][]byte, len(tiles))
	errs := make([]error, len(tiles))
	r.cacheMu.Lock()
	if r.cache == nil {
		r.cache = make(map[tlog.Tile][]byte)
	}
	for i, tile := range tiles {
		if data := r.cache[tile]; data != nil {
			out[i] = data
			continue
		}
		wg.Add(1)
		go func(i int, tile tlog.Tile) {
			defer wg.Done()
			data, err := httpGet(r.url + tile.Path())
			if err != nil && tile.W != 1<<uint(tile.H) {
				fullTile := tile
				fullTile.W = 1 << uint(tile.H)
				if fullData, err1 := httpGet(r.url + fullTile.Path()); err1 == nil {
					data = fullData[:tile.W*tlog.HashSize]
					err = nil
				}
			}
			if err != nil {
				errs[i] = err
				return
			}
			r.cacheMu.Lock()
			r.cache[tile] = data
			r.cacheMu.Unlock()
			out[i] = data
		}(i, tile)
	}
	r.cacheMu.Unlock()
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}
