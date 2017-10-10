package main

import (
	"regexp"
	"strings"
)

func sContains(set []string, s string) bool {
	for _, s_ := range set {
		if s_ == s {
			return true
		}
	}

	return false
}

func shouldIgnore(path string) bool {
	for _, ignored := range ignoredPackages {
		if strings.Contains(path, ignored) {
			return true
		}
	}
	return false
}

// Returns true if primitive.
// If the type is primitive, the string parameters indicate the type and format per swagger spec.
// http://swagger.io/specification/ (Data Types)
func IsPrimitive(goType string) (bool, string, string) {

	// TODO: Figure out how complex types should be supported.

	goType = strings.Trim(goType, "*")

	switch goType {
	case "bool":
		return true, "boolean", ""
	case "byte":
		return true, "string", "byte"
	case "complex64":
		return true, "string", ""
	case "complex128":
		return true, "string", ""
	case "float32":
		return true, "number", "float"
	case "float64":
		return true, "number", "double"
	case "int":
		return true, "integer", ""
	case "int8":
		return true, "integer", ""
	case "int16":
		return true, "integer", ""
	case "int32":
		return true, "integer", "int32"
	case "int64":
		return true, "integer", "int64"
	case "rune":
		return true, "integer", ""
	case "string":
		return true, "string", ""
	case "uint":
		return true, "integer", ""
	case "uint8":
		return true, "integer", ""
	case "uint16":
		return true, "integer", ""
	case "uint32":
		return true, "integer", ""
	case "uint64":
		return true, "integer", ""
	case "uintptr":
		return true, "integer", ""
	case "[]byte":
		return true, "string", "binary"
	case "interface{}":
		return true, "object", ""

	case "time.Time":
		// This is a special case. While not strictly a primitive type, it's
		// something we can all agree is as simple as it needs to be.
		return true, "string", "date-time"
	}

	return false, "", ""
}

func IsMap(goType string) (bool, string, string) {

	rxMap := regexp.MustCompile(`map\[(.+)\]\**(.+)`)
	if rxMap.MatchString(goType) {
		matches := rxMap.FindStringSubmatch(goType)
		return true, matches[1], matches[2]
	}

	return false, "", ""
}

func IsSlice(goType string) (bool, string) {

	// This is a strange case. The Swagger spec doesn't recognize []byte as an
	// array, but as a binary string.
	if goType == "[]byte" {
		return false, ""
	}

	if strings.HasPrefix(goType, "[]") {
		return true, goType[2:]
	}

	return false, ""
}
