package main

import (
	"bufio"
	"bytes"
	"github.com/go-openapi/spec"
	"regexp"
	"strconv"
	"strings"
)

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
	Type        SchemerDefiner
}

func (this *ResponseIntermediate) Schema() *spec.Schema {

	schema := this.Type.Schema()
	schema.Title = ""

	return schema
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
		rxAccept      *regexp.Regexp = regexp.MustCompile(`@Accept\s+(.+)`)
		rxDescription *regexp.Regexp = regexp.MustCompile(`@Description\s+(.+)`)
		rxParameter   *regexp.Regexp = regexp.MustCompile(`@Param\s+([\w-]+)\s+(\w+)\s+([\w\.]+)\s+(\w+)\s+\"(.+)\"`)
		rxResponse    *regexp.Regexp = regexp.MustCompile(`@(Success|Failure)\s+(\d+)\s+\{([\w]+)\}\s+([\w\.]+)\s+\"(.+)\"`)
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

			raw := rxAccept.FindStringSubmatch(line)[1]
			accepts := strings.Split(raw, ",")
			for _, accept := range accepts {
				accept = strings.TrimSpace(accept)
				accept = strings.ToLower(accept)

				if accept == "" {
					continue
				} else if accept == "json" {
					accept = "application/json"
				} else if accept == "xml" {
					accept = "application/xml"
				}

				operationIntermediate.Accepts = append(operationIntermediate.Accepts, accept)
			}

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

			goType := matches[4]
			goTypeMeta := matches[3]
			if strings.ToLower(goTypeMeta) == "array" && !strings.HasPrefix(goType, "[]") {
				goType = "[]" + goType
			}

			var responseType SchemerDefiner

			if isSlice, v := IsSlice(goType); isSlice {
				valueType := &MemberIntermediate{
					Type:        v,
					Validations: make(ValidationMap),
				}

				responseType = &SliceIntermediate{
					Type:        goType,
					ValueType:   valueType,
					Validations: make(ValidationMap),
				}
			} else {
				responseType = &MemberIntermediate{
					Type:        goType,
					Validations: make(ValidationMap),
				}
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
