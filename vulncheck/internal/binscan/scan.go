// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package binscan contains methods for parsing Go binary files for the purpose
// of extracting module dependency and symbol table information.
package binscan

// Code in this package is dervied from src/cmd/go/internal/version/version.go
// and cmd/go/internal/version/exe.go.

import (
	"bytes"
	"debug/gosym"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/url"
	"runtime/debug"
	"strings"

	"golang.org/x/tools/go/packages"
)

// buildInfoMagic, findVers, and readString are copied from
// cmd/go/internal/version

// The build info blob left by the linker is identified by
// a 16-byte header, consisting of buildInfoMagic (14 bytes),
// the binary's pointer size (1 byte),
// and whether the binary is big endian (1 byte).
var buildInfoMagic = []byte("\xff Go buildinf:")

// findVers finds and returns the Go version and module version information
// in the executable x.
func findVers(x exe) string {
	// Read the first 64kB of text to find the build info blob.
	text := x.DataStart()
	data, err := x.ReadData(text, 64*1024)
	if err != nil {
		return ""
	}
	for ; !bytes.HasPrefix(data, buildInfoMagic); data = data[32:] {
		if len(data) < 32 {
			return ""
		}
	}

	// Decode the blob.
	ptrSize := int(data[14])
	bigEndian := data[15] != 0
	var bo binary.ByteOrder
	if bigEndian {
		bo = binary.BigEndian
	} else {
		bo = binary.LittleEndian
	}
	var readPtr func([]byte) uint64
	if ptrSize == 4 {
		readPtr = func(b []byte) uint64 { return uint64(bo.Uint32(b)) }
	} else {
		readPtr = bo.Uint64
	}
	vers := readString(x, ptrSize, readPtr, readPtr(data[16:]))
	if vers == "" {
		return ""
	}
	mod := readString(x, ptrSize, readPtr, readPtr(data[16+ptrSize:]))
	if len(mod) >= 33 && mod[len(mod)-17] == '\n' {
		// Strip module framing.
		mod = mod[16 : len(mod)-16]
	} else {
		mod = ""
	}
	return mod
}

// readString returns the string at address addr in the executable x.
func readString(x exe, ptrSize int, readPtr func([]byte) uint64, addr uint64) string {
	hdr, err := x.ReadData(addr, uint64(2*ptrSize))
	if err != nil || len(hdr) < 2*ptrSize {
		return ""
	}
	dataAddr := readPtr(hdr)
	dataLen := readPtr(hdr[ptrSize:])
	data, err := x.ReadData(dataAddr, dataLen)
	if err != nil || uint64(len(data)) < dataLen {
		return ""
	}
	return string(data)
}

// readBuildInfo is copied from runtime/debug
func readBuildInfo(data string) (*debug.BuildInfo, bool) {
	if len(data) == 0 {
		return nil, false
	}

	const (
		pathLine = "path\t"
		modLine  = "mod\t"
		depLine  = "dep\t"
		repLine  = "=>\t"
	)

	readEntryFirstLine := func(elem []string) (debug.Module, bool) {
		if len(elem) != 2 && len(elem) != 3 {
			return debug.Module{}, false
		}
		sum := ""
		if len(elem) == 3 {
			sum = elem[2]
		}
		return debug.Module{
			Path:    elem[0],
			Version: elem[1],
			Sum:     sum,
		}, true
	}

	var (
		info = &debug.BuildInfo{}
		last *debug.Module
		line string
		ok   bool
	)
	// Reverse of cmd/go/internal/modload.PackageBuildInfo
	for len(data) > 0 {
		i := strings.IndexByte(data, '\n')
		if i < 0 {
			break
		}
		line, data = data[:i], data[i+1:]
		switch {
		case strings.HasPrefix(line, pathLine):
			elem := line[len(pathLine):]
			info.Path = elem
		case strings.HasPrefix(line, modLine):
			elem := strings.Split(line[len(modLine):], "\t")
			last = &info.Main
			*last, ok = readEntryFirstLine(elem)
			if !ok {
				return nil, false
			}
		case strings.HasPrefix(line, depLine):
			elem := strings.Split(line[len(depLine):], "\t")
			last = new(debug.Module)
			info.Deps = append(info.Deps, last)
			*last, ok = readEntryFirstLine(elem)
			if !ok {
				return nil, false
			}
		case strings.HasPrefix(line, repLine):
			elem := strings.Split(line[len(repLine):], "\t")
			if len(elem) != 3 {
				return nil, false
			}
			if last == nil {
				return nil, false
			}
			last.Replace = &debug.Module{
				Path:    elem[0],
				Version: elem[1],
				Sum:     elem[2],
			}
			last = nil
		}
	}
	return info, true
}

func debugModulesToPackagesModules(debugModules []*debug.Module) []*packages.Module {
	packagesModules := make([]*packages.Module, len(debugModules))
	for i, mod := range debugModules {
		packagesModules[i] = &packages.Module{
			Path:    mod.Path,
			Version: mod.Version,
		}
		if mod.Replace != nil {
			packagesModules[i].Replace = &packages.Module{
				Path:    mod.Replace.Path,
				Version: mod.Replace.Version,
			}
		}
	}
	return packagesModules
}

// ExtractPackagesAndSymbols extracts the symbols, packages, and their associated module versions
// from a Go binary. Stripped binaries are not supported.
func ExtractPackagesAndSymbols(bin io.ReaderAt) ([]*packages.Module, map[string][]string, error) {
	x, err := openExe(bin)
	if err != nil {
		return nil, nil, err
	}

	pclntab, textOffset := x.PCLNTab()
	if pclntab == nil {
		// TODO(roland): if we have build information, but not PCLN table, we should be able to
		// fall back to much higher granularity vulnerability checking.
		return nil, nil, errors.New("unable to load the PCLN table")
	}
	lineTab := gosym.NewLineTable(pclntab, textOffset)
	if lineTab == nil {
		return nil, nil, errors.New("invalid line table")
	}
	tab, err := gosym.NewTable(nil, lineTab)
	if err != nil {
		return nil, nil, err
	}

	packageSymbols := map[string][]string{}
	for _, f := range tab.Funcs {
		if f.Func == nil {
			continue
		}
		symName := f.Func.BaseName()
		if r := f.Func.ReceiverName(); r != "" {
			if strings.HasPrefix(r, "(*") {
				r = strings.Trim(r, "(*)")
			}
			symName = fmt.Sprintf("%s.%s", r, symName)
		}

		pkgName := f.Func.PackageName()
		if pkgName == "" {
			continue
		}
		pkgName, err := url.PathUnescape(pkgName)
		if err != nil {
			return nil, nil, err
		}

		packageSymbols[pkgName] = append(packageSymbols[pkgName], symName)
	}

	bi, ok := readBuildInfo(findVers(x))
	if !ok {
		return nil, nil, err
	}

	return debugModulesToPackagesModules(bi.Deps), packageSymbols, nil
}
