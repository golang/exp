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
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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

var height = flag.Int("h", 8, "tile height")
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

	// TODO(rsc): Load initial db.latest, db.latestNote from on-disk cache.
	db := &GoSumDB{
		url:       *url,
		verifiers: note.VerifierList(verifier),
	}
	db.httpClient.Timeout = 1 * time.Minute
	db.tileReader.db = db
	db.tileReader.url = db.url + "/"

	for _, arg := range flag.Args()[1:] {
		data, err := ioutil.ReadFile(arg)
		if err != nil {
			log.Fatal(err)
		}
		log.SetPrefix("notecheck: " + arg + ": ")
		checkGoSum(db, data)
		log.SetPrefix("notecheck: ")
	}
}

func checkGoSum(db *GoSumDB, data []byte) {
	lines := strings.Split(string(data), "\n")
	if lines[len(lines)-1] != "" {
		log.Printf("error: final line missing newline")
		return
	}
	// TODO(rsc): This assumes that the /go.mod and the whole-tree hashes
	// always appear together in a go.sum.
	// Sometimes the /go.mod can appear alone.
	// The code needs to be updated to handle that case.
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

		dbLines, err := db.Lookup(f1[0], f1[1])
		if err != nil {
			log.Printf("%s@%s: %v", f1[0], f1[1], err)
			continue
		}

		if strings.Join(lines[i:i+2], "\n") != strings.Join(dbLines, "\n") {
			log.Printf("%s@%s: invalid go.sum entries:\ngo.sum:\n\t%s\nsum.golang.org:\n\t%s", f1[0], f1[1], strings.Join(lines[i:i+2], "\n\t"), strings.Join(dbLines, "\n\t"))
		}
	}
}

// A GoSumDB is a client for a go.sum database.
type GoSumDB struct {
	url        string         // root url of database, without trailing slash
	verifiers  note.Verifiers // accepted verifiers for signed trees
	tileReader tileReader     // tlog.TileReader implementation
	httpCache  parCache
	httpClient http.Client

	// latest accepted tree head
	mu         sync.Mutex
	latest     tlog.Tree
	latestNote []byte // signed note
}

// parCache is a minimal simulation of cmd/go's par.Cache.
// When this code moves into cmd/go, it should use the real par.Cache
type parCache struct {
}

func (c *parCache) Do(key interface{}, f func() interface{}) interface{} {
	return f()
}

// Lookup returns the go.sum lines for the given module path and version.
func (db *GoSumDB) Lookup(path, vers string) ([]string, error) {
	// TODO(rsc): !-encode the path.
	data, err := db.httpGet(db.url + "/lookup/" + path + "@" + vers)
	if err != nil {
		return nil, err
	}

	id, text, treeMsg, err := tlog.ParseRecord(data)
	if err != nil {
		return nil, fmt.Errorf("%s@%s: %v", path, vers, err)
	}
	if err := db.updateLatest(treeMsg); err != nil {
		return nil, fmt.Errorf("%s@%s: %v", path, vers, err)
	}
	if err := db.checkRecord(id, text); err != nil {
		return nil, fmt.Errorf("%s@%s: %v", path, vers, err)
	}

	prefix := path + " " + vers + " "
	prefixGoMod := path + " " + vers + "/go.mod "
	var hashes []string
	for _, line := range strings.Split(string(text), "\n") {
		if strings.HasPrefix(line, prefix) || strings.HasPrefix(line, prefixGoMod) {
			hashes = append(hashes, line)
		}
	}
	return hashes, nil
}

// updateLatest updates db's idea of the latest tree head
// to incorporate the signed tree head in msg.
// If msg is before the current latest tree head,
// updateLatest still checks that it fits into the known timeline.
// updateLatest returns an error for non-malicious problems.
// If it detects a fork in the tree history, it prints a detailed
// message and calls log.Fatal.
func (db *GoSumDB) updateLatest(msg []byte) error {
	if len(msg) == 0 {
		return nil
	}
	note, err := note.Open(msg, db.verifiers)
	if err != nil {
		return fmt.Errorf("reading tree note: %v\nnote:\n%s", err, msg)
	}
	tree, err := tlog.ParseTree([]byte(note.Text))
	if err != nil {
		return fmt.Errorf("reading tree: %v\ntree:\n%s", err, note.Text)
	}

Update:
	for {
		db.mu.Lock()
		latest := db.latest
		latestNote := db.latestNote
		db.mu.Unlock()

		switch {
		case tree.N <= latest.N:
			return db.checkTrees(tree, msg, latest, latestNote)

		case tree.N > latest.N:
			if err := db.checkTrees(latest, latestNote, tree, msg); err != nil {
				return err
			}
			db.mu.Lock()
			if db.latest != latest {
				if db.latest.N > latest.N {
					db.mu.Unlock()
					continue Update
				}
				log.Fatalf("go.sum database changed underfoot:\n\t%v ->\n\t%v", latest, db.latest)
			}
			db.latest = tree
			db.latestNote = msg
			db.mu.Unlock()
			return nil
		}
	}
}

// checkTrees checks that older (from olderNote) is contained in newer (from newerNote).
// If an error occurs, such as malformed data or a network problem, checkTrees returns that error.
// If on the other hand checkTrees finds evidence of misbehavior, it prepares a detailed
// message and calls log.Fatal.
func (db *GoSumDB) checkTrees(older tlog.Tree, olderNote []byte, newer tlog.Tree, newerNote []byte) error {
	thr := tlog.TileHashReader(newer, &db.tileReader)
	h, err := tlog.TreeHash(older.N, thr)
	if err != nil {
		return fmt.Errorf("checking tree#%d against tree#%d: %v", older.N, newer.N, err)
	}
	if h == older.Hash {
		return nil
	}

	// Detected a fork in the tree timeline.
	// Start by reporting the inconsistent signed tree notes.
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "SECURITY ERROR\n")
	fmt.Fprintf(&buf, "go.sum database server misbehavior detected!\n\n")
	indent := func(b []byte) []byte {
		return bytes.Replace(b, []byte("\n"), []byte("\n\t"), -1)
	}
	fmt.Fprintf(&buf, "old database:\n\t%v\n", indent(olderNote))
	fmt.Fprintf(&buf, "new database:\n\t%v\n", indent(newerNote))

	// The notes alone are not enough to prove the inconsistency.
	// We also need to show that the newer note's tree hash for older.N
	// does not match older.Hash. The consumer of this report could
	// of course consult the server to try to verify the inconsistency,
	// but we are holding all the bits we need to prove it right now,
	// so we might as well print them and make the report not depend
	// on the continued availability of the misbehaving server.
	// Preparing this data only reuses the tiled hashes needed for
	// tlog.TreeHash(older.N, thr) above, so assuming thr is caching tiles,
	// there are no new access to the server here, and these operations cannot fail.
	fmt.Fprintf(&buf, "proof of misbehavior:\n\t%v", h)
	if p, err := tlog.ProveTree(newer.N, older.N, thr); err != nil {
		fmt.Fprintf(&buf, "\tinternal error: %v\n", err)
	} else if err := tlog.CheckTree(p, newer.N, newer.Hash, older.N, h); err != nil {
		fmt.Fprintf(&buf, "\tinternal error: generated inconsistent proof\n")
	} else {
		for _, h := range p {
			fmt.Fprintf(&buf, "\n\t%v", h)
		}
	}
	log.Fatalf("%v", buf.String())
	panic("not reached")
}

// checkRecord checks that record #id's hash matches data.
func (db *GoSumDB) checkRecord(id int64, data []byte) error {
	db.mu.Lock()
	tree := db.latest
	db.mu.Unlock()

	if id >= tree.N {
		return fmt.Errorf("cannot validate record %d in tree of size %d", id, tree.N)
	}
	hashes, err := tlog.TileHashReader(tree, &db.tileReader).ReadHashes([]int64{tlog.StoredHashIndex(0, id)})
	if err != nil {
		return err
	}
	if hashes[0] == tlog.RecordHash(data) {
		return nil
	}
	return fmt.Errorf("cannot authenticate record data in server response")
}

type tileReader struct {
	url     string
	cache   map[tlog.Tile][]byte
	cacheMu sync.Mutex
	db      *GoSumDB
}

func (r *tileReader) Height() int {
	return *height
}

func (r *tileReader) SaveTiles(tiles []tlog.Tile, data [][]byte) {
	// TODO(rsc): On-disk cache in GOPATH.
}

func (r *tileReader) ReadTiles(tiles []tlog.Tile) ([][]byte, error) {
	// TODO(rsc): Look in on-disk cache in GOPATH.

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
			data, err := r.db.httpGet(r.url + tile.Path())
			if err != nil && tile.W != 1<<uint(tile.H) {
				fullTile := tile
				fullTile.W = 1 << uint(tile.H)
				if fullData, err1 := r.db.httpGet(r.url + fullTile.Path()); err1 == nil {
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

func (db *GoSumDB) httpGet(url string) ([]byte, error) {
	type cached struct {
		data []byte
		err  error
	}

	c := db.httpCache.Do(url, func() interface{} {
		start := time.Now()
		resp, err := db.httpClient.Get(url)
		if err != nil {
			return cached{nil, err}
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return cached{nil, fmt.Errorf("GET %v: %v", url, resp.Status)}
		}
		data, err := ioutil.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return cached{nil, err}
		}
		if *vflag {
			fmt.Fprintf(os.Stderr, "%.3fs %s\n", time.Since(start).Seconds(), url)
		}
		return cached{data, nil}
	}).(cached)

	return c.data, c.err
}
