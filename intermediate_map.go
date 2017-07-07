package main

import (
	"github.com/go-openapi/spec"
	"github.com/jackmanlabs/errors"
	"strconv"
)

type MapIntermediate struct {
	Name          string // Name in Go struct.
	Type          string // Go type (how it was originally described)
	JsonName      string // JSON name.
	JsonOmitEmpty bool   // If the omitempty flag was given in the JSON.
	KeyType       *MemberIntermediate
	ValueType     *MemberIntermediate
	Description   string
	Validations   Validator
}

func (this *MapIntermediate) IsRequired() bool {
	return this.Validations.IsRequired()
}

func (this *MapIntermediate) Schema() *spec.Schema {

	schema := new(spec.Schema)

	name := this.Name
	if this.JsonName != "" {
		name = this.JsonName
	}
	schema.Title = name

	schema.Description = this.Description
	schema.Items = new(spec.SchemaOrArray)
	schema.Items.Schema = new(spec.Schema)

	schema.AdditionalProperties = new(spec.SchemaOrBool)
	schema.AdditionalProperties.Schema = new(spec.Schema)
	schema.AdditionalProperties.Schema.Items = new(spec.SchemaOrArray)

	// The map is always an object.
	schema.Typed("object", "")

	// The additional properties is always of type array.
	schema.AdditionalProperties.Schema.Typed("array", "")

	// The items on the additional properties is the type of the value type.
	schema.AdditionalProperties.Schema.Items.Schema = this.ValueType.Schema()

	// No one with whom I've spoken knows how maps work in Swagger.
	// Consequently, I'm hoping that the validations work just as well with maps
	// as they do with slices/arrays.

	if this.Validations.Min() >= 0 {
		schema.WithMinItems(int64(this.Validations.Min()))
	}

	if this.Validations.Max() >= 0 {
		schema.WithMaxItems(int64(this.Validations.Max()))
	}

	if this.Validations.Length() >= 0 {
		schema.WithMinItems(int64(this.Validations.Length()))
		schema.WithMaxItems(int64(this.Validations.Length()))
	}

	if s, ok := this.Validations.Equals(); ok {
		d, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			schema.WithMinItems(d)
			schema.WithMaxItems(d)
		}
	}

	if this.Validations.GreaterThan() >= 0 {
		schema.WithMinItems(int64(this.Validations.GreaterThan() + 1.0))
	}

	if this.Validations.LessThan() >= 0 {
		schema.WithMaxItems(int64(this.Validations.LessThan() - 1.0))
	}

	return schema
}

func (this *MapIntermediate) DefineDefinitions(referencingPackagePath string) error {

	err := this.ValueType.DefineDefinitions(referencingPackagePath)
	if err != nil {
		return errors.Stack(err)
	}

	err = this.KeyType.DefineDefinitions(referencingPackagePath)
	if err != nil {
		return errors.Stack(err)
	}

	return nil
}
