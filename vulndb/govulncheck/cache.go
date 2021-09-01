// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/vulndb/osv"
)

// fsCache is file-system cache implementing osv.Cache
type fsCache struct {
	rootDir string
}

// use cfg.GOMODCACHE available in cmd/go/internal?
var defaultCacheRoot = filepath.Join(build.Default.GOPATH, "/pkg/mod/cache/download/vulndb")

func defaultCache() *fsCache {
	return &fsCache{rootDir: defaultCacheRoot}
}

type cachedIndex struct {
	Retrieved time.Time
	Index     osv.DBIndex
}

func (c *fsCache) ReadIndex(dbName string) (osv.DBIndex, time.Time, error) {
	b, err := ioutil.ReadFile(filepath.Join(c.rootDir, dbName, "index.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, time.Time{}, nil
		}
		return nil, time.Time{}, err
	}
	var index cachedIndex
	if err := json.Unmarshal(b, &index); err != nil {
		return nil, time.Time{}, err
	}
	return index.Index, index.Retrieved, nil
}

func (c *fsCache) WriteIndex(dbName string, index osv.DBIndex, retrieved time.Time) error {
	path := filepath.Join(c.rootDir, dbName)
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	j, err := json.Marshal(cachedIndex{
		Index:     index,
		Retrieved: retrieved,
	})
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(path, "index.json"), j, 0666); err != nil {
		return err
	}
	return nil
}

func (c *fsCache) ReadEntries(dbName string, p string) ([]*osv.Entry, error) {
	b, err := ioutil.ReadFile(filepath.Join(c.rootDir, dbName, p, "vulns.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []*osv.Entry
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (c *fsCache) WriteEntries(dbName string, p string, entries []*osv.Entry) error {
	path := filepath.Join(c.rootDir, dbName, p)
	if err := os.MkdirAll(path, 0777); err != nil {
		return err
	}
	j, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(path, "vulns.json"), j, 0666); err != nil {
		return err
	}
	return nil
}
