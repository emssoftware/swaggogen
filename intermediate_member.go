package main

import (
	"github.com/go-openapi/jsonreference"
	"github.com/go-openapi/spec"
	"github.com/jackmanlabs/errors"
	"log"
	"strconv"
	"strings"
)

type MemberIntermediate struct {
	PackageName   string // Necessary for canonical and swagger names.
	PackagePath   string
	Name          string // Name in Go struct.
	Type          string // Go type
	JsonName      string // JSON name.
	JsonOmitEmpty bool   // If the omitempty flag was given in the JSON.
	Description   string
	Validations   Validator
}

func (this *MemberIntermediate) IsRequired() bool {
	return this.Validations.IsRequired()
}

func (this *MemberIntermediate) DefinitionRef() string {
	return "#/definitions/" + this.SwaggerName()
}

func (this *MemberIntermediate) SwaggerName() string {

	goType := this.Type
	goType = strings.TrimPrefix(goType, "*")

	idx := strings.Index(goType, ".")
	if idx > -1 {
		goType = goType[idx+1:]
	}

	var name string

	switch *naming {
	case "full":
		name = this.CanonicalName()
	case "partial":
		name = this.PackageName + "." + goType
	case "simple":
		name = goType
	}

	return name

}

func (this *MemberIntermediate) CanonicalName() string {

	goType := this.Type
	goType = strings.TrimPrefix(goType, "*")

	idx := strings.Index(goType, ".")
	if idx > -1 {
		goType = goType[idx+1:]
	}

	name := this.PackagePath + "." + goType
	name = strings.Replace(name, "/", ".", -1)

	return name
}

func (this *MemberIntermediate) Schema() *spec.Schema {
	schema := new(spec.Schema)

	name := this.Name
	if this.JsonName != "" {
		name = this.JsonName
	}
	schema.Title = name
	schema.Description = this.Description

	if isPrimitive, t, f := IsPrimitive(this.Type); isPrimitive {
		schema.Typed(t, f)

		if t == "string" {
			if this.Validations.Min() >= 0 {
				schema.WithMinLength(int64(this.Validations.Min()))
			}

			if this.Validations.Max() >= 0 {
				schema.WithMaxLength(int64(this.Validations.Max()))
			}

			if this.Validations.Length() >= 0 {
				schema.WithMinLength(int64(this.Validations.Length()))
				schema.WithMaxLength(int64(this.Validations.Length()))
			}

			if s, ok := this.Validations.Equals(); ok {
				schema.WithPattern(s)
			}

			if this.Validations.GreaterThan() >= 0 {
				schema.WithMinLength(int64(this.Validations.GreaterThan() + 1))
			}

			if this.Validations.LessThan() >= 0 {
				schema.WithMaxLength(int64(this.Validations.LessThan() - 1))
			}
		} else if t == "number" || t == "integer" {
			if this.Validations.Min() >= 0 {
				schema.WithMinimum(this.Validations.Min(), false)
			}

			if this.Validations.Max() >= 0 {
				schema.WithMaximum(this.Validations.Max(), false)
			}

			if this.Validations.Length() >= 0 {
				schema.WithMinimum(this.Validations.Length(), false)
				schema.WithMaximum(this.Validations.Length(), false)
			}

			if s, ok := this.Validations.Equals(); ok {
				f, err := strconv.ParseFloat(s, 64)
				if err == nil {
					schema.WithMinimum(f, false)
					schema.WithMaximum(f, false)
				}
			}

			if this.Validations.GreaterThan() >= 0 {
				schema.WithMinimum(this.Validations.GreaterThan(), true)
			}

			if this.Validations.LessThan() >= 0 {
				schema.WithMaximum(this.Validations.LessThan(), true)
			}
		}

	} else {
		ref, err := jsonreference.New(this.DefinitionRef())
		if err != nil {
			log.Print(errors.Stack(err))
		}
		schema.Ref = spec.Ref{Ref: ref}
	}

	return schema
}

func (this *MemberIntermediate) DefineDefinitions(referringPackage string) error {

	if referringPackage == "" {
		return errors.New("Referencing Package Path is empty.")
	}

	var err error

	goType := this.Type
	if isPrimitive, _, _ := IsPrimitive(goType); isPrimitive {
		return nil
	}

	definition, ok := definitionStore.ExistsDefinition(referringPackage, goType)
	if !ok {
		definition, err = findDefinition(referringPackage, goType)
		if err != nil {
			return errors.Stack(err)
		} else if definition == nil {
			return errors.New("Failed to generate definition for type: " + goType)
		}

		definitionStore.Add(definition)
	}

	this.PackagePath = definition.PackagePath
	this.PackageName = definition.PackageName

	// This triggers the definition of all the members of the discovered type associated with the present member.
	definition.DefineDefinitions()

	return nil
}
