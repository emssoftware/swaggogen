package main

import (
	"github.com/go-openapi/spec"
	"log"
	"strings"
)

func swaggerizeApi(intermediate ApiIntermediate) *spec.Swagger {

	var info *spec.Info = &spec.Info{
		// This is ugly, but apparently you can't do direct assignment on embedded members.
		InfoProps: spec.InfoProps{
			Description: intermediate.ApiDescription,
			Title:       intermediate.ApiTitle,
			Version:     intermediate.ApiVersion,
		},
	}

	var swagger *spec.Swagger = &spec.Swagger{
		// This is ugly, but apparently you can't do direct assignment on embedded members.
		SwaggerProps: spec.SwaggerProps{
			BasePath: intermediate.BasePath,
			Info:     info,
			Swagger:  "2.0",
		},
	}

	//for _, subApi := range intermediate.SubApis{
	//	swagger.Paths.Paths[subApi.Path] = spec.PathItem{}
	//}

	return swagger
}

/*
This method attempts to fit the operation intermediate type onto the swagger spec.
See the following for reference:
	https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md
	https://godoc.org/github.com/go-openapi/spec
*/
func swaggerizeOperations(intermediates []OperationIntermediate) map[string]spec.PathItem {
	pathItems := make(map[string]spec.PathItem)

	for _, operationIntermediate := range intermediates {

		pathItem, ok := pathItems[operationIntermediate.Path]
		if !ok {
			pathItem = spec.PathItem{}
		}

		operationObject := &spec.Operation{
			OperationProps: spec.OperationProps{
				Summary:     operationIntermediate.Title,
				Description: operationIntermediate.Description,
				Consumes:    operationIntermediate.Accepts,
				Produces:    operationIntermediate.Accepts,
				Tags:        []string{operationIntermediate.Tag},
			},
		}

		for _, responseIntermediate := range operationIntermediate.Responses {
			response := new(spec.Response)
			response.Description = responseIntermediate.Description
			response.Schema = responseIntermediate.Schema()
			operationObject.RespondsWith(responseIntermediate.StatusCode, response)
		}

		for _, parameterIntermediate := range operationIntermediate.Parameters {
			parameter := new(spec.Parameter)
			parameter.Name = parameterIntermediate.Type.JsonName
			parameter.In = parameterIntermediate.In
			parameter.Required = parameterIntermediate.Required
			parameter.Description = parameterIntermediate.Description

			if parameterIntermediate.In == "body" {
				parameter.Schema = parameterIntermediate.Schema()
			} else {
				isPrimitive, t, _ := IsPrimitive(parameterIntermediate.Type.Type)
				parameter.Type = t
				if !isPrimitive {
					log.Print("WARNING: It appears there is non-primitive response parameter someplace other than the request body:" + parameterIntermediate.Type.CanonicalName())
				}
			}

			operationObject.AddParam(parameter)
		}

		switch strings.ToLower(operationIntermediate.Method) {
		case "put":
			pathItem.Put = operationObject
		case "get":
			pathItem.Get = operationObject
		case "post":
			pathItem.Post = operationObject
		case "delete":
			pathItem.Delete = operationObject
		case "options":
			pathItem.Options = operationObject
		case "head":
			pathItem.Head = operationObject
		case "patch":
			pathItem.Patch = operationObject
		}

		pathItems[operationIntermediate.Path] = pathItem
	}

	return pathItems
}

func swaggerizeDefinitions() map[string]spec.Schema {

	schemas := make(map[string]spec.Schema)

	for _, definition := range definitionStore {
		swaggerName := definition.CanonicalName()
		schemas[swaggerName] = definition.Schema()
	}

	return schemas
}
