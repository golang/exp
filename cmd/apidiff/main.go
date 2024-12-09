// Command apidiff determines whether two versions of a package are compatible
package main

import (
	"archive/zip"
	"bufio"
	"errors"
	"flag"
	"fmt"
	"go/token"
	"go/types"
	"io"
	"os"
	"strings"

	"golang.org/x/exp/apidiff"
	"golang.org/x/tools/go/gcexportdata"
	"golang.org/x/tools/go/packages"
)

var (
	exportDataOutfile = flag.String("w", "", "file for export data")
	incompatibleOnly  = flag.Bool("incompatible", false, "display only incompatible changes")
	allowInternal     = flag.Bool("allow-internal", false, "allow apidiff to compare internal packages")
	moduleMode        = flag.Bool("m", false, "compare modules instead of packages")
)

func main() {
	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "usage:\n")
		fmt.Fprintf(w, "apidiff OLD NEW\n")
		fmt.Fprintf(w, "   compares OLD and NEW package APIs\n")
		fmt.Fprintf(w, "   where OLD and NEW are either import paths or files of export data\n")
		fmt.Fprintf(w, "apidiff -m OLD NEW\n")
		fmt.Fprintf(w, "   compares OLD and NEW module APIs\n")
		fmt.Fprintf(w, "   where OLD and NEW are module paths\n")
		fmt.Fprintf(w, "apidiff -w FILE IMPORT_PATH\n")
		fmt.Fprintf(w, "   writes export data of the package at IMPORT_PATH to FILE\n")
		fmt.Fprintf(w, "   NOTE: In a GOPATH-less environment, this option consults the\n")
		fmt.Fprintf(w, "   module cache by default, unless used in the directory that\n")
		fmt.Fprintf(w, "   contains the go.mod module definition that IMPORT_PATH belongs\n")
		fmt.Fprintf(w, "   to. In most cases users want the latter behavior, so be sure\n")
		fmt.Fprintf(w, "   to cd to the exact directory which contains the module\n")
		fmt.Fprintf(w, "   definition of IMPORT_PATH.\n")
		fmt.Fprintf(w, "apidiff -m -w FILE MODULE_PATH\n")
		fmt.Fprintf(w, "   writes export data of the module at MODULE_PATH to FILE\n")
		fmt.Fprintf(w, "   Same NOTE for packages applies to modules.\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	fset := token.NewFileSet()

	if *exportDataOutfile != "" {
		if len(flag.Args()) != 1 {
			flag.Usage()
			os.Exit(2)
		}
		if err := loadAndWrite(fset, flag.Arg(0)); err != nil {
			die("writing export data: %v", err)
		}
		os.Exit(0)
	}

	if len(flag.Args()) != 2 {
		flag.Usage()
		os.Exit(2)
	}

	var report apidiff.Report
	if *moduleMode {
		oldmod := mustLoadOrReadModule(fset, flag.Arg(0))
		newmod := mustLoadOrReadModule(fset, flag.Arg(1))

		report = apidiff.ModuleChanges(oldmod, newmod)
	} else {
		oldpkg := mustLoadOrReadPackage(fset, flag.Arg(0))
		newpkg := mustLoadOrReadPackage(fset, flag.Arg(1))
		if !*allowInternal {
			if isInternalPackage(oldpkg.Path(), "") && isInternalPackage(newpkg.Path(), "") {
				fmt.Fprintf(os.Stderr, "Ignoring internal package %s\n", oldpkg.Path())
				os.Exit(0)
			}
		}
		report = apidiff.Changes(oldpkg, newpkg)
	}

	var err error
	if *incompatibleOnly {
		err = report.TextIncompatible(os.Stdout, false)
	} else {
		err = report.Text(os.Stdout)
	}
	if err != nil {
		die("writing report: %v", err)
	}
}

func loadAndWrite(fset *token.FileSet, path string) error {
	if *moduleMode {
		module := mustLoadModule(fset, path)
		return writeModuleExportData(fset, module, *exportDataOutfile)
	}

	// Loading and writing data for only a single package.
	pkg := mustLoadPackage(fset, path)
	return writePackageExportData(pkg, *exportDataOutfile)
}

func mustLoadOrReadPackage(fset *token.FileSet, importPathOrFile string) *types.Package {
	fileInfo, err := os.Stat(importPathOrFile)
	if err == nil && fileInfo.Mode().IsRegular() {
		pkg, err := readPackageExportData(fset, importPathOrFile)
		if err != nil {
			die("reading export data from %s: %v", importPathOrFile, err)
		}
		return pkg
	} else {
		return mustLoadPackage(fset, importPathOrFile).Types
	}
}

func mustLoadPackage(fset *token.FileSet, importPath string) *packages.Package {
	pkg, err := loadPackage(fset, importPath)
	if err != nil {
		die("loading %s: %v", importPath, err)
	}
	return pkg
}

func loadPackage(fset *token.FileSet, importPath string) (*packages.Package, error) {
	cfg := &packages.Config{
		Fset: fset,
		Mode: packages.NeedName | packages.NeedTypes,
	}
	pkgs, err := packages.Load(cfg, importPath)
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("found no packages for import %s", importPath)
	}
	if len(pkgs[0].Errors) > 0 {
		// TODO: use errors.Join once Go 1.21 is released.
		return nil, pkgs[0].Errors[0]
	}
	return pkgs[0], nil
}

func mustLoadOrReadModule(fset *token.FileSet, modulePathOrFile string) *apidiff.Module {
	var module *apidiff.Module
	fileInfo, err := os.Stat(modulePathOrFile)
	if err == nil && fileInfo.Mode().IsRegular() {
		module, err = readModuleExportData(fset, modulePathOrFile)
		if err != nil {
			die("reading export data from %s: %v", modulePathOrFile, err)
		}
	} else {
		module = mustLoadModule(fset, modulePathOrFile)
	}

	filterInternal(module, *allowInternal)

	return module
}

func mustLoadModule(fset *token.FileSet, modulepath string) *apidiff.Module {
	module, err := loadModule(fset, modulepath)
	if err != nil {
		die("loading %s: %v", modulepath, err)
	}
	return module
}

func loadModule(fset *token.FileSet, modulepath string) (*apidiff.Module, error) {
	cfg := &packages.Config{
		Fset: fset,
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedModule,
	}
	loaded, err := packages.Load(cfg, fmt.Sprintf("%s/...", modulepath))
	if err != nil {
		return nil, err
	}
	if len(loaded) == 0 {
		return nil, fmt.Errorf("found no packages for module %s", modulepath)
	}
	var tpkgs []*types.Package
	for _, p := range loaded {
		if len(p.Errors) > 0 {
			// TODO: use errors.Join once Go 1.21 is released.
			return nil, p.Errors[0]
		}
		tpkgs = append(tpkgs, p.Types)
	}

	return &apidiff.Module{Path: loaded[0].Module.Path, Packages: tpkgs}, nil
}

func readModuleExportData(fset *token.FileSet, filename string) (*apidiff.Module, error) {
	f, err := zip.OpenReader(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	imports := make(map[string]*types.Package)

	var modPath string
	var pkgs []*types.Package
	for _, entry := range f.File {
		if err := func() error {
			r, err := entry.Open()
			if err != nil {
				return err
			}
			defer r.Close()
			if entry.Name == "module" {
				data, err := io.ReadAll(r)
				if err != nil {
					return err
				}
				modPath = string(data)
			} else {
				pkg, err := gcexportdata.Read(r, fset, imports, strings.TrimSuffix(entry.Name, ".x"))
				if err != nil {
					return err
				}
				if imports[entry.Name] != nil {
					panic("not in topological order")
				}
				imports[entry.Name] = pkg
				pkgs = append(pkgs, pkg)
			}
			return nil
		}(); err != nil {
			return nil, err
		}
	}
	return &apidiff.Module{Path: modPath, Packages: pkgs}, nil
}

func writeModuleExportData(fset *token.FileSet, module *apidiff.Module, filename string) (err error) {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	fmt.Fprintln(f, module.Path)

	// Write types for each package into a zip archive.

	// First write the module path.
	w := zip.NewWriter(f)
	entry, err := w.Create("module")
	if err != nil {
		return err
	}
	if _, err := io.WriteString(entry, module.Path); err != nil {
		return err
	}

	// Then emit packages, dependencies first.
	seen := map[*types.Package]bool{types.Unsafe: true}
	var emit func(pkg *types.Package) error
	emit = func(pkg *types.Package) error {
		if pkg.Name() == "main" {
			return nil // nonimportable
		}
		if !seen[pkg] {
			seen[pkg] = true
			for _, dep := range pkg.Imports() {
				emit(dep)
			}
			entry, err := w.Create(pkg.Path() + ".x")
			if err != nil {
				return err
			}
			if err := gcexportdata.Write(entry, fset, pkg); err != nil {
				return err
			}
		}
		return nil
	}
	for _, pkg := range module.Packages {
		if err := emit(pkg); err != nil {
			return err
		}
	}

	return w.Close()
}

func readPackageExportData(fset *token.FileSet, filename string) (*types.Package, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	pkgPath, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	pkgPath = pkgPath[:len(pkgPath)-1] // remove delimiter
	imports := make(map[string]*types.Package)
	return gcexportdata.Read(r, fset, imports, pkgPath)
}

func writePackageExportData(pkg *packages.Package, filename string) (err error) {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	// Include the package path in the file. The exportdata format does
	// not record the path of the package being written.
	if _, err := fmt.Fprintln(f, pkg.PkgPath); err != nil {
		return err
	}
	return gcexportdata.Write(f, pkg.Fset, pkg.Types)
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func filterInternal(m *apidiff.Module, allow bool) {
	if allow {
		return
	}

	var nonInternal []*types.Package
	for _, p := range m.Packages {
		if !isInternalPackage(p.Path(), m.Path) {
			nonInternal = append(nonInternal, p)
		} else {
			fmt.Fprintf(os.Stderr, "Ignoring internal package %s\n", p.Path())
		}
	}
	m.Packages = nonInternal
}

func isInternalPackage(pkgPath, modulePath string) bool {
	pkgPath = strings.TrimPrefix(pkgPath, modulePath)
	switch {
	case strings.HasSuffix(pkgPath, "/internal"):
		return true
	case strings.Contains(pkgPath, "/internal/"):
		return true
	case pkgPath == "internal":
		return true
	case strings.HasPrefix(pkgPath, "internal/"):
		return true
	}
	return false
}
