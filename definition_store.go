package main

import (
	"strings"
)

/*
	For all Go types, we want to always refer to them by some kind of canonical name.
	This is what we're going to use here:

		foo.bar.package.type

	This does NOT use the package name, only the package path and the type name.
	This is defined as a method, CanonicalName(), on MemberIntermediates and DefinitionIntermediates.
*/
type DefinitionStore map[string]*DefinitionIntermediate

func (this DefinitionStore) Add(intermediate *DefinitionIntermediate) {

	_, ok := this[intermediate.CanonicalName()]
	if ok {
		//log.Print("duplicate detected: " + intermediate.CanonicalName())
		//jlog.Log(this)
	}

	this[intermediate.CanonicalName()] = intermediate
}

func (this DefinitionStore) ExistsDefinition(referringPackage, typeName string) (*DefinitionIntermediate, bool) {

	pkgInfo := pkgInfos[referringPackage]
	importPaths := possibleImportPaths(pkgInfo, typeName)

	idx := strings.Index(typeName, ".")
	if idx != -1 {
		typeName = typeName[idx+1:]
	}

	for _, importPath := range importPaths {
		for _, def := range this {
			if def.PackagePath == importPath && def.Name == typeName {
				return def, true
			}
		}
	}

	return nil, false
}
