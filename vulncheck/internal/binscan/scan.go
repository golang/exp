// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package binscan contains methods for parsing Go binary files for the purpose
// of extracting module dependency and symbol table information.
package binscan

// Code in this package is dervied from src/cmd/go/internal/version/version.go
// and cmd/go/internal/version/exe.go.

import (
	"debug/buildinfo"
	"debug/gosym"
	"errors"
	"fmt"
	"io"
	"net/url"
	"runtime/debug"
	"strings"

	"golang.org/x/tools/go/packages"
)

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
//
// TODO(#51412): detect inlined symbols too
func ExtractPackagesAndSymbols(bin io.ReaderAt) ([]*packages.Module, map[string][]string, error) {
	bi, err := buildinfo.Read(bin)
	if err != nil {
		return nil, nil, err
	}

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

	return debugModulesToPackagesModules(bi.Deps), packageSymbols, nil
}
