package main

import "github.com/go-openapi/spec"

type SchemerDefiner interface {
	// This method should have the side effect of updating package information
	// on the receiver object.
	DefineDefinitions(referencingPackagePath string) error
	Schema() *spec.Schema
	IsRequired() bool
}
