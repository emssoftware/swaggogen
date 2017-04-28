package main

import (
	"fmt"
	"github.com/jackmanlabs/bucket/jlog"
	"github.com/jackmanlabs/errors"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"log"
	"strings"
)

func findEnumValues(referringPackage, typeName string) ([]string, error) {

	pkgInfo := pkgInfos[referringPackage]
	typeName = strings.TrimPrefix(typeName, "*")

	importPaths := possibleImportPaths(pkgInfo, typeName)

	if len(importPaths) == 0 {
		log.Print("ERROR: Import paths not available for this enum type: ", typeName)
		jlog.Log(pkgInfo)
	}

	if len(importPaths) > 1 {
		log.Printf("WARNING: Multiple package candidates found for enum type (%s):", typeName)
		jlog.Log(importPaths)
	}

	if strings.Contains(typeName, ".") {
		index := strings.Index(typeName, ".") + 1
		typeName = typeName[index:]
	}

	for _, importPath := range importPaths {

		if importPath == "" {
			log.Print("Import path is blank!")
		}

		bpkg, err := build.Import(importPath, srcPath, 0)
		if err != nil {
			return nil, errors.Stack(err)
		}

		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, bpkg.Dir, nil, parser.AllErrors)
		if err != nil {
			return nil, errors.Stack(err)
		}

		for _, pkg := range pkgs {
			enumVisitor := &EnumVisitor{
				Fset:     fset,
				TypeName: typeName,
				Values:   make([]string, 0),
			}

			ast.Walk(enumVisitor, pkg)

			return enumVisitor.Values, nil
		}
	}

	return nil, nil
}

type EnumVisitor struct {
	Fset     *token.FileSet
	TypeName string
	Values   []string
}

func (this *EnumVisitor) Visit(node ast.Node) (w ast.Visitor) {

	if this.Fset == nil {
		log.Println("fset is nil.")
		return nil
	}

	switch t := node.(type) {

	case *ast.ValueSpec:
		valueType := resolveTypeExpression(t.Type)
		if valueType != this.TypeName {
			return nil
		}

		// Assume we have one name and one value.
		if len(t.Names) != 1 || len(t.Values) != 1 {
			log.Print("WARNING: A possible constant declaration was found, but has more than one name or value: " + valueType)
			return nil
		}

		valueValue := resolveValueExpression(t.Values[0])

		//ast.Fprint(os.Stderr, this.Fset, valueValue, nil)

		this.Values = append(this.Values, valueValue)

	case *ast.FuncDecl:
		// Ignore function declarations.
		return nil
	case *ast.ImportSpec:
		// Ignore import declarations.
		return nil
	case nil:
		//default:
		//	log.Printf("unexpected type %T\n", t) // %T prints whatever type t has
	}

	return this
}

func resolveValueExpression(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.BasicLit:
		return t.Value
	default:
		return fmt.Sprintf("Unknown<%T>", t)
	}
}
