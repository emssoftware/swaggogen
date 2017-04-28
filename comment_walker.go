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

func getRelevantComments(pkgPath string) ([]string, error) {

	bpkg, err := build.Import(pkgPath, srcPath, 0)
	if err != nil {
		return []string{}, nil
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, bpkg.Dir, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return nil, errors.Stack(err)
	}

	commentVisitor := &CommentVisitor{Fset: fset}
	for _, pkg := range pkgs {
		ast.Walk(commentVisitor, pkg)
	}

	return commentVisitor.Comments, nil
}

/*
We're using a map for the imports so we don't have to worry about duplicates.
*/
type CommentVisitor struct {
	Fset     *token.FileSet
	Comments []string
}

func (this *CommentVisitor) Visit(node ast.Node) (w ast.Visitor) {

	if this.Fset == nil {
		log.Println("fset is nil.")
		return nil
	}

	switch t := node.(type) {

	case *ast.CommentGroup:
		if this.Comments == nil {
			this.Comments = make([]string, 0)
		}

		this.Comments = append(this.Comments, t.Text())

		return nil

		//case nil:
		//default:
		//	fmt.Printf("unexpected type %T\n", t) // %T prints whatever type t has
	}

	return this
}

func extractOperationComments(comments []string) []string {
	return extractComments(comments, "@Router")
}

func extractApiComments(comments []string) []string {
	return extractComments(comments, "@APITitle")
}

func extractComments(comments []string, keyword string) []string {

	newComments := make([]string, 0)

	for _, comment := range comments {
		if strings.Contains(comment, keyword) {
			newComments = append(newComments, comment)
		}
	}

	return newComments
}
