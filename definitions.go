package main

import (
	"github.com/jackmanlabs/errors"
	"strings"
)

func deriveDefinitionsFromOperations(operationIntermediates []OperationIntermediate) error {
	for _, operationIntermediate := range operationIntermediates {
		for _, responseIntermediate := range operationIntermediate.Responses {
			err := responseIntermediate.Type.DefineDefinitions(operationIntermediate.PackagePath)
			if err != nil {
				return errors.Stack(err)
			}
		}
		for _, parameterIntermediate := range operationIntermediate.Parameters {
			err := parameterIntermediate.Type.DefineDefinitions(operationIntermediate.PackagePath)
			if err != nil {
				return errors.Stack(err)
			}
		}
	}

	return nil
}

// What packages could have possibly contained this type?
func possibleImportPaths(pkgInfo PackageInfo, goType string) []string {

	if !strings.Contains(goType, ".") {
		return []string{pkgInfo.ImportPath}
	}

	chunks := strings.Split(goType, ".")

	alias := chunks[0]

	importPaths := make([]string, 0)

	for importPath, aliases := range pkgInfo.Imports {
		for _, alias_ := range aliases {
			if alias_ == alias {
				// I'm pretty sure that there should never be duplicate importPaths here.
				// Otherwise, check for duplicates.
				importPaths = append(importPaths, importPath)
			}
		}
	}

	return importPaths
}
