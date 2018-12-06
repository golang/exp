// Command apidiff determines whether two versions of a package are compatible
package main

import (
	"flag"
	"fmt"
	"go/token"
	"go/types"
	"os"

	"golang.org/x/exp/apidiff"
	"golang.org/x/tools/go/gcexportdata"
	"golang.org/x/tools/go/packages"
)

var (
	exportDataOutfile = flag.String("w", "", "file for export data")
	incompatibleOnly  = flag.Bool("incompatible", false, "display only incompatible changes")
)

func main() {
	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "usage:\n")
		fmt.Fprintf(w, "apidiff OLD NEW\n")
		fmt.Fprintf(w, "   compares OLD and NEW package APIs\n")
		fmt.Fprintf(w, "   where OLD and NEW are either import paths or files of export data\n")
		fmt.Fprintf(w, "apidiff -w FILE IMPORT_PATH\n")
		fmt.Fprintf(w, "   writes export data of the package at IMPORT_PATH to FILE\n")
		flag.PrintDefaults()
	}

	flag.Parse()
	if *exportDataOutfile != "" {
		if len(flag.Args()) != 1 {
			flag.Usage()
		}
		pkg := mustLoadPackage(flag.Arg(0))
		if err := writeExportData(pkg, *exportDataOutfile); err != nil {
			die("writing export data: %v", err)
		}
	} else {
		if len(flag.Args()) != 2 {
			flag.Usage()
		}
		oldpkg := mustLoadOrRead(flag.Arg(0))
		newpkg := mustLoadOrRead(flag.Arg(1))

		report := apidiff.Changes(oldpkg, newpkg)
		var err error
		if *incompatibleOnly {
			err = report.TextIncompatible(os.Stdout)
		} else {
			err = report.Text(os.Stdout)
		}
		if err != nil {
			die("writing report: %v", err)
		}
	}
}

func mustLoadOrRead(importPathOrFile string) *types.Package {
	fileInfo, err := os.Stat(importPathOrFile)
	if err == nil && fileInfo.Mode().IsRegular() {
		pkg, err := readExportData(importPathOrFile)
		if err != nil {
			die("reading export data from %s: %v", importPathOrFile, err)
		}
		return pkg
	} else {
		return mustLoadPackage(importPathOrFile).Types
	}
}

func mustLoadPackage(importPath string) *packages.Package {
	pkg, err := loadPackage(importPath)
	if err != nil {
		die("loading %s: %v", importPath, err)
	}
	return pkg
}

func loadPackage(importPath string) (*packages.Package, error) {
	cfg := &packages.Config{Mode: packages.LoadTypes}
	pkgs, err := packages.Load(cfg, importPath)
	if err != nil {
		return nil, err
	}
	if len(pkgs[0].Errors) > 0 {
		return nil, pkgs[0].Errors[0]
	}
	return pkgs[0], nil
}

func readExportData(filename string) (*types.Package, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return gcexportdata.Read(f, token.NewFileSet(), map[string]*types.Package{}, filename)
}

func writeExportData(pkg *packages.Package, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	err1 := gcexportdata.Write(f, pkg.Fset, pkg.Types)
	err2 := f.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
