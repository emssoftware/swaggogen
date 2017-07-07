package main

import (
	"github.com/go-openapi/spec"
	"github.com/jackmanlabs/errors"
	"strconv"
)

type SliceIntermediate struct {
	Name          string // Name in Go struct.
	Type          string // Go type
	JsonName      string // JSON name.
	JsonOmitEmpty bool   // If the omitempty flag was given in the JSON.
	ValueType     *MemberIntermediate
	Description   string
	Validations   Validator
}

func (this *SliceIntermediate) IsRequired() bool {
	return this.Validations.IsRequired()
}

func (this *SliceIntermediate) Schema() *spec.Schema {

	schema := new(spec.Schema)

	name := this.Name
	if this.JsonName != "" {
		name = this.JsonName
	}
	schema.Title = name

	schema.Description = this.Description
	schema.Items = new(spec.SchemaOrArray)

	schema.Typed("array", "")

	schema.Items.Schema = this.ValueType.Schema()

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
		schema.WithMinItems(int64(this.Validations.GreaterThan() + 1))
	}

	if this.Validations.LessThan() >= 0 {
		schema.WithMaxItems(int64(this.Validations.LessThan() - 1))
	}

	return schema
}

func (this *SliceIntermediate) DefineDefinitions(referencingPackagePath string) error {

	err := this.ValueType.DefineDefinitions(referencingPackagePath)
	if err != nil {
		return errors.Stack(err)
	}

	return nil
}
