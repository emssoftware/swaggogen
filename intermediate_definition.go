package main

import (
	"github.com/go-openapi/spec"
	"github.com/jackmanlabs/errors"
	"strconv"
	"strings"
)

type DefinitionIntermediate struct {
	Comment        string
	Documentation  string
	EmbeddedTypes  []string
	Members        map[string]SchemerDefiner // map[name]schemer
	Name           string
	PackageName    string   // The actual package name of this type.
	PackagePath    string   // The actual package path of this type.
	UnderlyingType string   // This isn't used right now. In our test codebase, non-struct types were never used.
	Enums          []string // If the underlying type is a primitive type, it's assumed it's an enum type, these being the values.

	// While it may not strictly be equivalent from a language specification
	// perspective, we're going to call a non-struct type with an underlying
	// type equivalent to a struct type with a single embedded type.
	//
	// By doing this, we'll only need to write a single algorithm to flatten the
	// types for Swagger.
}

func (this *DefinitionIntermediate) CanonicalName() string {
	name := this.PackagePath + "." + this.Name
	name = strings.Replace(name, "/", ".", -1)
	return name
}

func (this *DefinitionIntermediate) SwaggerName() string {

	var name string

	switch *naming {
	case "full":
		name = this.CanonicalName()
	case "partial":
		name = this.PackageName + "." + this.Name
	case "simple":
		name = this.Name
	}

	return name
}

func (this *DefinitionIntermediate) Schema() spec.Schema {

	var schema spec.Schema
	schema.Title = this.SwaggerName()

	if isPrimitive, t, f := IsPrimitive(this.UnderlyingType); isPrimitive {
		schema.Typed(t, f)
		schema.Enum = make([]interface{}, 0)

		for _, enum := range this.Enums {
			if strings.HasPrefix(enum, "\"") {
				schema.Enum = append(schema.Enum, strings.Trim(enum, "\""))
			} else {
				// store numerical enums as numbers, otherwise strings.
				if f, err := strconv.ParseFloat(enum, 64); err == nil {
					schema.Enum = append(schema.Enum, f)
				} else {
					enum = strings.Trim(enum, "\"")
					schema.Enum = append(schema.Enum, enum)
				}
			}
		}
	} else {
		schema.Typed("object", "")
		schema.Required = make([]string, 0)

		properties := make(map[string]spec.Schema)
		for _, member := range this.Members {
			property := member.Schema()
			properties[property.Title] = *property

			if member.IsRequired() {
				schema.Required = append(schema.Required, property.Title)
			}

		}

		schema.Properties = properties
	}

	return schema
}

func (this *DefinitionIntermediate) DefineDefinitions() error {

	var err error
	goType := this.Name
	if isPrimitive, _, _ := IsPrimitive(goType); isPrimitive {
		// This was never hit with our test code base.
		return nil
	}

	for _, embeddedType := range this.EmbeddedTypes {
		definition, ok := definitionStore.ExistsDefinition(this.PackagePath, embeddedType)
		if !ok {
			definition, err = findDefinition(this.PackagePath, embeddedType)
			if err != nil {
				return errors.Stack(err)
			} else if definition == nil {
				return errors.Newf("Failed to find definition for embedded member: %s:%s", goType, embeddedType)
			}

			definitionStore.Add(definition)
		}

		mergeDefinitions(this, definition)
	}

	for _, member := range this.Members {
		err := member.DefineDefinitions(this.PackagePath)
		if err != nil {
			return errors.Stack(err)
		}
	}

	return nil
}
