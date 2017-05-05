package main

import (
	"github.com/jackmanlabs/errors"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"log"
	"strings"
)

type PackageInfo struct {
	ImportPath  string
	PackageName string
	Imports     map[string][]string // map[importPath]aliases
}

func getPackageInfoRecursive(pkgPath string) (map[string]PackageInfo, error) {

	pkgInfos := make(map[string]PackageInfo) // map[pkgInfoPath]PackageInfo

	// The map key is the imported package path.
	// The map value indicates if the package has already been scanned.
	allImports := make(map[string]bool)
	allImports[pkgPath] = false

	// This is building a lot of functionality into the loop.
	// Let's pray it's not too clever.
	for currentImportPath := getUnscannedImport(allImports); currentImportPath != ""; currentImportPath = getUnscannedImport(allImports) {
		pkgName, pkgImportPaths, err := getPackageInfo(currentImportPath)
		if err != nil {
			return nil, errors.Stack(err)
		} else if pkgName == "" {
			allImports[currentImportPath] = true
			continue
		} else if shouldIgnore(currentImportPath) {
			log.Print("Detected ignored package: " + currentImportPath)
			allImports[currentImportPath] = true
			continue
		}

		// For each import extracted, add it to the master list as necessary.
		for newImportPath := range pkgImportPaths {
			if _, ok := allImports[newImportPath]; !ok {
				allImports[newImportPath] = false
			}
		}

		pkgInfo := PackageInfo{
			ImportPath:  currentImportPath,
			PackageName: pkgName,
			Imports:     pkgImportPaths,
		}

		pkgInfos[currentImportPath] = pkgInfo
		allImports[currentImportPath] = true
	}

	// We need to make sure that the import paths without pkgInfo have the default alias (package name).
	for pkgInfoPath, pkgInfo := range pkgInfos {
		for importPath, aliases := range pkgInfo.Imports {
			pkgName := pkgInfos[importPath].PackageName
			if !sContains(aliases, pkgName) {
				// This probably doesn't need to be so verbose, but no harm done.
				pkgInfos[pkgInfoPath].Imports[importPath] = append(aliases, pkgName)
			}
		}
	}

	return pkgInfos, nil
}

func getUnscannedImport(imports map[string]bool) string {
	for mprt, scanned := range imports {
		if !scanned {
			return mprt
		}
	}

	return ""
}

/*
Returns the package name, the list of imports (import paths), and error.
This function returns a slice to force the consumer to avoid reusing the map
within ImportVisitor.
*/
func getPackageInfo(pkgPath string) (string, map[string][]string, error) {

	bpkg, err := build.Import(pkgPath, srcPath, 0)
	if err != nil {
		//logPackageNotFound(pkgPath)
		return "", nil, nil
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, bpkg.Dir, nil, parser.AllErrors)
	if err != nil {
		return "", nil, errors.Stack(err)
	}

	// Some packages irrelevant have "main" packages and "_test" packages.
	// We need to prioritize packages that don't have these names.
	var pkgToScan *ast.Package
	for _, pkg := range pkgs {

		// We absolutely don't care about test packages.
		if strings.HasSuffix(pkg.Name, "_test") {
			continue
		}

		// We'll take any package if we don't already have one.
		if pkgToScan == nil {
			pkgToScan = pkg
			continue
		}

		// If we already have a package, we prefer the package that isn't a "main" package.
		if pkgToScan.Name == "main" {
			pkgToScan = pkg
		}
	}

	if pkgToScan == nil {
		return "", nil, errors.New("Did not find a usable package in package path: " + pkgPath)
	}

	importVisitor := &ImportVisitor{Fset: fset}
	if pkgToScan != nil {
		ast.Walk(importVisitor, pkgToScan)
	}

	return pkgToScan.Name, importVisitor.Imports, nil
}

/*
We could have made the ImportVisitor much more complicated by picking out the
data that we know we're going to need later on. Instead of doing that, however,
I'm deliberately choosing to make the ImportVisitor do a single job for the sake
of clarity. I realize this may result in some small measure of code duplication.
*/
type ImportVisitor struct {
	Fset    *token.FileSet      // for debugging
	Imports map[string][]string // map[pkgPath]aliases
}

func (this *ImportVisitor) Visit(node ast.Node) (w ast.Visitor) {

	if this.Fset == nil {
		log.Println("fset is nil.")
		return nil
	}

	switch t := node.(type) {

	case *ast.ImportSpec:
		if this.Imports == nil {
			this.Imports = make(map[string][]string)
		}

		alias := t.Name.String()

		// unused imports don't interest us.
		if alias == "_" {
			return nil
		}

		importPath := t.Path.Value
		importPath = strings.Trim(importPath, "\"")
		_, ok := this.Imports[importPath]
		if !ok {
			this.Imports[importPath] = make([]string, 0)
		}

		if alias != "\u003cnil\u003e" {
			if !sContains(this.Imports[importPath], alias) {
				this.Imports[importPath] = append(this.Imports[importPath], alias)
			}
		}

		return nil
	}

	return this
}

var missingPackages = make(map[string]bool)

func logPackageNotFound(pkgPath string) {
	if _, ok := missingPackages[pkgPath]; !ok {
		log.Print("WARNING: Could not find package: ", pkgPath)
		missingPackages[pkgPath] = false
	}
}
