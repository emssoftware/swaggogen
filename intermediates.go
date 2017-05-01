package main

import (
	"bufio"
	"bytes"
	"github.com/go-openapi/jsonreference"
	"github.com/go-openapi/spec"
	"github.com/jackmanlabs/errors"
	"log"
	"regexp"
	"strconv"
	"strings"
)

type SchemerDefiner interface {
	// This method should have the side effect of updating package information on the receiver object.
	DefineDefinitions(referencingPackagePath string) error
	Schema() *spec.Schema
}

type ApiIntermediate struct {
	ApiVersion     string
	ApiTitle       string
	ApiDescription string
	BasePath       string
	SubApis        []SubApiIntermediate
}

type SubApiIntermediate struct {
	Name string
	Path string
}

// This is an intermediate representation of a path and/or operation as parsed
// in the comments. A collection of these can be combined and transformed to
// create the swagger hierarchy.
type OperationIntermediate struct {
	Title       string
	Description string
	Accepts     []string
	Parameters  []ParameterIntermediate
	Responses   []*ResponseIntermediate
	Path        string
	Method      string
	PackagePath string // Where this operation was found.
	Tag         string
}

type ParameterIntermediate struct {
	In          string
	Required    bool
	Description string
	Type        *MemberIntermediate
}

func (this *ParameterIntermediate) Schema() *spec.Schema {
	return this.Type.Schema()
}

type ResponseIntermediate struct {
	Success     bool
	StatusCode  int
	Description string
	Type        *MemberIntermediate
}

func (this *ResponseIntermediate) Schema() *spec.Schema {

	schema := this.Type.Schema()
	schema.Title = ""

	return schema
}

type DefinitionIntermediate struct {
	Comment        string
	Documentation  string
	EmbeddedTypes  []string
	Members        map[string]SchemerDefiner // map[name]schemer
	Name           string
	PackageName    string   // The actual package name of this type.
	PackagePath    string   // The actual package path of this type.
	UnderlyingType string   // This isn't used right now. In our test codebase, non-struct types were never used.
	Enumerations   []string // If the underlying type is a primitive type, it's assumed it's an enum type, these being the values.

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
	name := this.PackageName + "." + this.Name
	return name
}

func (this *DefinitionIntermediate) Schema() spec.Schema {

	var schema spec.Schema
	schema.Title = this.SwaggerName()

	if isPrimitive, t, f := IsPrimitive(this.UnderlyingType); isPrimitive {
		schema.Typed(t, f)
		schema.Enum = make([]interface{}, 0)

		for _, enum := range this.Enumerations {
			if strings.HasPrefix(enum, "\"") {
				schema.Enum = append(schema.Enum, strings.Trim(enum, "\""))
			} else {
				f, err := strconv.ParseFloat(enum, 64)
				if err != nil {
					log.Print(errors.Stack(err))
				}
				schema.Enum = append(schema.Enum, f)
			}
		}
	} else {
		schema.Typed("object", "")

		properties := make(map[string]spec.Schema)
		for _, member := range this.Members {
			property := member.Schema()
			properties[property.Title] = *property
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

type MemberIntermediate struct {
	PackageName   string // Necessary for canonical and swagger names.
	PackagePath   string
	Name          string // Name in Go struct.
	Type          string // Go type
	JsonName      string // JSON name.
	JsonOmitEmpty bool   // If the omitempty flag was given in the JSON.
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

	name := this.PackageName + "." + goType

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

	if isPrimitive, t, f := IsPrimitive(this.Type); isPrimitive {
		schema.Typed(t, f)
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

type SliceIntermediate struct {
	Name          string // Name in Go struct.
	Type          string // Go type
	JsonName      string // JSON name.
	JsonOmitEmpty bool   // If the omitempty flag was given in the JSON.
	ValueType     *MemberIntermediate
}

func (this *SliceIntermediate) Schema() *spec.Schema {

	schema := new(spec.Schema)
	schema.Title = this.JsonName
	schema.Items = new(spec.SchemaOrArray)

	schema.Typed("array", "")

	schema.Items.Schema = this.ValueType.Schema()

	return schema
}

func (this *SliceIntermediate) DefineDefinitions(referencingPackagePath string) error {

	err := this.ValueType.DefineDefinitions(referencingPackagePath)
	if err != nil {
		return errors.Stack(err)
	}

	return nil
}

type MapIntermediate struct {
	Name          string // Name in Go struct.
	Type          string // Go type (how it was originally described)
	JsonName      string // JSON name.
	JsonOmitEmpty bool   // If the omitempty flag was given in the JSON.
	KeyType       *MemberIntermediate
	ValueType     *MemberIntermediate
}

func (this *MapIntermediate) Schema() *spec.Schema {

	schema := new(spec.Schema)
	schema.Title = this.JsonName
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

func intermediatateApi(commentBlocks []string) ApiIntermediate {

	// @APIVersion 1.0.0
	// @APITitle REST API
	// @APIDescription EMS Rest API
	// @BasePath /api/v1
	// @SubApi HealthCheck [/health]

	var (
		// At the time of writing, IntelliJ erroneously warns on unnecessary
		// escape sequences. Do not trust IntelliJ.
		rxApiVersion     *regexp.Regexp = regexp.MustCompile(`@APIVersion\s+([\d\.]+)`)
		rxApiTitle       *regexp.Regexp = regexp.MustCompile(`@APITitle\s+(.+)`)
		rxApiDescription *regexp.Regexp = regexp.MustCompile(`@APIDescription\s+(.+)`)
		rxBasePath       *regexp.Regexp = regexp.MustCompile(`@BasePath\s+([/a-zA-Z0-9-]+)`)
		rxSubApi         *regexp.Regexp = regexp.MustCompile(`@SubApi\s+([0-9a-zA-Z]+)\s+\[([/a-zA-Z0-9-]+)\]`)
	)

	var apiIntermediate ApiIntermediate = ApiIntermediate{
		SubApis: make([]SubApiIntermediate, 0),
	}

	for _, commentBlock := range commentBlocks {

		b := bytes.NewBufferString(commentBlock)
		scanner := bufio.NewScanner(b)
		for scanner.Scan() {
			line := scanner.Text()

			switch {

			case rxApiDescription.MatchString(line):
				apiIntermediate.ApiDescription = rxApiDescription.FindStringSubmatch(line)[1]
			case rxApiTitle.MatchString(line):
				apiIntermediate.ApiTitle = rxApiTitle.FindStringSubmatch(line)[1]
			case rxApiVersion.MatchString(line):
				apiIntermediate.ApiVersion = rxApiVersion.FindStringSubmatch(line)[1]
			case rxBasePath.MatchString(line):
				apiIntermediate.BasePath = rxBasePath.FindStringSubmatch(line)[1]

			case rxSubApi.MatchString(line):
				matches := rxSubApi.FindStringSubmatch(line)
				subApi := SubApiIntermediate{
					Name: matches[1],
					Path: matches[2],
				}
				apiIntermediate.SubApis = append(apiIntermediate.SubApis, subApi)
			}
		}
	}

	return apiIntermediate
}

func intermediatateOperation(commentBlock string) OperationIntermediate {

	// @Title Get TimeZone
	// @Description Return a TimeZone, given its id
	// @Accept  json
	//
	// @Param   x-ems-consumer	header	string  true	"Defines the consumer of the API. MobileApp, etc."
	// @Param   x-ems-api-token	header	string	true	"Auth token, from /authenticate request"
	// @Param   id				path	int     true	"TimeZone ID"
	// @Param   timestamp      	query   string	true    "dateTime in timeZone local time, for which to get timezone info with offsets adjusted for DST, RFC3339"
	//
	// @Success 200 {object} model.TimeZoneModel "Success"
	// @Failure 400 {object} apicommon.ErrorResponse "Bad Request"
	// @Failure 401 {object} apicommon.ErrorResponse "Invalid or missing consumer credentials"
	// @Router /timezones/{id} [get]

	var (
		// At the time of writing, IntelliJ erroneously warns on unnecessary
		// escape sequences. Do not trust IntelliJ.
		rxAccept      *regexp.Regexp = regexp.MustCompile(`@Accept\s+([\w/]+)`)
		rxDescription *regexp.Regexp = regexp.MustCompile(`@Description\s+(.+)`)
		rxParameter   *regexp.Regexp = regexp.MustCompile(`@Param\s+([\w-]+)\s+(\w+)\s+([\w\.]+)\s+(\w+)\s+\"(.+)\"`)
		rxResponse    *regexp.Regexp = regexp.MustCompile(`@(Success|Failure)\s+(\d+)\s+([{}\w]+)\s([\w\.]+)\s+\"(.+)\"`)
		rxRouter      *regexp.Regexp = regexp.MustCompile(`@Router\s+([/\w\d-{}]+)\s+\[(\w+)\]`)
		rxTitle       *regexp.Regexp = regexp.MustCompile(`@Title\s+(.+)`)
	)

	var operationIntermediate OperationIntermediate = OperationIntermediate{
		Accepts:    make([]string, 0),
		Parameters: make([]ParameterIntermediate, 0),
		Responses:  make([]*ResponseIntermediate, 0),
	}

	b := bytes.NewBufferString(commentBlock)
	scanner := bufio.NewScanner(b)
	for scanner.Scan() {
		line := scanner.Text()

		switch {

		case rxAccept.MatchString(line):
			operationIntermediate.Accepts = append(operationIntermediate.Accepts, rxAccept.FindStringSubmatch(line)[1])
		case rxDescription.MatchString(line):
			operationIntermediate.Description = rxDescription.FindStringSubmatch(line)[1]
		case rxParameter.MatchString(line):

			matches := rxParameter.FindStringSubmatch(line)

			parameterType := &MemberIntermediate{
				Type:     matches[3],
				JsonName: matches[1],
			}

			parameterIntermediate := ParameterIntermediate{
				In:          matches[2],
				Type:        parameterType,
				Required:    strings.ToLower(matches[4]) == "true",
				Description: matches[5],
			}

			operationIntermediate.Parameters = append(operationIntermediate.Parameters, parameterIntermediate)

		case rxResponse.MatchString(line):
			matches := rxResponse.FindStringSubmatch(line)
			statusCode, _ := strconv.Atoi(matches[2])

			responseType := &MemberIntermediate{
				Type:     matches[4],
				JsonName: matches[3],
			}

			responseIntermediate := &ResponseIntermediate{
				Success:     strings.ToLower(matches[1]) == "success",
				StatusCode:  statusCode,
				Type:        responseType,
				Description: matches[5],
			}

			operationIntermediate.Responses = append(operationIntermediate.Responses, responseIntermediate)

		case rxRouter.MatchString(line):
			matches := rxRouter.FindStringSubmatch(line)
			operationIntermediate.Path = matches[1]
			operationIntermediate.Method = matches[2]

		case rxTitle.MatchString(line):
			operationIntermediate.Title = rxTitle.FindStringSubmatch(line)[1]

		default:
			//log.Print(line)
		}
	}

	return operationIntermediate
}

func mergeDefinitions(dst, src *DefinitionIntermediate) {
	for srcName, srcMember := range src.Members {
		_, exists := dst.Members[srcName]
		if !exists {
			dst.Members[srcName] = srcMember
		}
	}
}

func tagOperations(apiIntermediate ApiIntermediate, operationIntermediates []OperationIntermediate) []OperationIntermediate {
	newOperationIntermediates := make([]OperationIntermediate, 0)

	for _, operationIntermediate := range operationIntermediates {
		for _, subApi := range apiIntermediate.SubApis {
			if strings.HasPrefix(operationIntermediate.Path, subApi.Path) {
				operationIntermediate.Tag = subApi.Name
				break
			}
		}
		newOperationIntermediates = append(newOperationIntermediates, operationIntermediate)
	}

	return newOperationIntermediates
}
