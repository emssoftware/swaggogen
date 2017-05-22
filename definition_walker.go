package main

import (
	"fmt"
	"github.com/jackmanlabs/bucket/jlog"
	"github.com/jackmanlabs/errors"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"log"
	"regexp"
	"strings"
	"unicode"
)

func findDefinition(referringPackage, typeName string) (*DefinitionIntermediate, error) {

	pkgInfo := pkgInfos[referringPackage]
	typeName = strings.TrimPrefix(typeName, "*")
	importPaths := possibleImportPaths(pkgInfo, typeName)

	if len(importPaths) == 0 {
		log.Print("ERROR: Import paths not available for this type: ", typeName)
		jlog.Log(pkgInfo)
	}

	if len(importPaths) > 1 {
		log.Printf("WARNING: Multiple package candidates found for type (%s):", typeName)
		jlog.Log(importPaths)
	}

	if strings.Contains(typeName, ".") {
		index := strings.Index(typeName, ".") + 1
		typeName = typeName[index:]
	}

	for _, importPath := range importPaths {

		if importPath == "" {
			log.Print("Import path is blank!")
		}

		bpkg, err := build.Import(importPath, srcPath, 0)
		if err != nil {
			return nil, errors.Stack(err)
		}

		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, bpkg.Dir, nil, parser.AllErrors|parser.ParseComments)
		if err != nil {
			return nil, errors.Stack(err)
		}

		for _, pkg := range pkgs {
			definitionVisitor := &DefinitionVisitor{
				Fset:     fset,
				TypeName: typeName,
			}

			ast.Walk(definitionVisitor, pkg)

			if definitionVisitor.Definition != nil {
				definition := definitionVisitor.Definition
				definition.PackageName = pkg.Name
				definition.PackagePath = importPath

				// If this definition is an enum (underlying type is primitive), then we assume it's an enum type that needs enum values.
				if isPrimitive, _, _ := IsPrimitive(definition.UnderlyingType); isPrimitive {
					values, err := findEnumValues(definition.PackagePath, definition.Name)
					if err != nil {
						return nil, errors.Stack(err)
					}
					definition.Enumerations = values
				}

				return definition, nil
			}
		}
	}

	return nil, nil
}

type DefinitionVisitor struct {
	Fset       *token.FileSet
	TypeName   string
	Definition *DefinitionIntermediate
}

func (this *DefinitionVisitor) Visit(node ast.Node) (w ast.Visitor) {
	/*
		Name          string
		PackageName   string
		PackagePath   string
		Members       []MemberIntermediate
		IsStruct      bool
		EmbeddedTypes []string
	*/

	if this.Fset == nil {
		log.Println("fset is nil.")
		return nil
	}

	switch t := node.(type) {

	case *ast.TypeSpec:
		if t.Name.String() == this.TypeName {
			this.Definition = &DefinitionIntermediate{
				Name:           t.Name.String(),
				Comment:        t.Comment.Text(),
				Documentation:  t.Doc.Text(),
				UnderlyingType: resolveTypeExpression(t.Type),
				Members:        make(map[string]SchemerDefiner),
			}
		} else {
			return nil
		}
	case *ast.Field:

		if this.Definition == nil {
			// Ignore fields that don't belong to our definition.
			return nil
		}

		if len(t.Names) > 1 {
			// On our test code base, this was never printed.
			log.Print("WARNING: Multiple names discovered.")
		}

		// Handle embedded structs.
		if len(t.Names) == 0 {
			//ast.Fprint(os.Stdout, this.Fset, t, nil)

			if this.Definition.EmbeddedTypes == nil {
				this.Definition.EmbeddedTypes = make([]string, 0)
			}
			embedded := resolveTypeExpression(t.Type)
			this.Definition.EmbeddedTypes = append(this.Definition.EmbeddedTypes, embedded)
			return nil
		}

		name := t.Names[0].String()

		// Ignore fields that are not exported.
		name_ := []rune(name)
		if unicode.IsLower(name_[0]) {
			return nil
		}

		jsonName, jsonOmitEmpty := parseJsonInfo(t.Tag.Value)
		if jsonName == "-" {
			return nil
		}

		isRequired := parseValidateTag(t.Tag.Value)

		var desc string = parseMemberDescription(t.Doc.Text())
		if desc == "" {
			desc = parseMemberDescription(t.Comment.Text())
		}

		goType := resolveTypeExpression(t.Type)

		var member SchemerDefiner

		if isMap, k, v := IsMap(goType); isMap {
			keyType := &MemberIntermediate{
				Type: k,
				Name: name,
			}

			valueType := &MemberIntermediate{
				Type: v,
				Name: name,
			}

			member = &MapIntermediate{
				Name:          name,
				Type:          goType,
				JsonName:      jsonName,
				JsonOmitEmpty: jsonOmitEmpty,
				ValueType:     valueType,
				KeyType:       keyType,
				Description:   desc,
				Required:      isRequired,
			}

		} else if isSlice, v := IsSlice(goType); isSlice {
			valueType := &MemberIntermediate{
				Type: v,
				Name: name,
			}

			member = &SliceIntermediate{
				Name:          name,
				Type:          goType,
				JsonName:      jsonName,
				JsonOmitEmpty: jsonOmitEmpty,
				ValueType:     valueType,
				Description:   desc,
				Required:      isRequired,
			}
		} else {
			member = &MemberIntermediate{
				Type:          goType,
				Name:          name,
				JsonName:      jsonName,
				JsonOmitEmpty: jsonOmitEmpty,
				Description:   desc,
				Required:      isRequired,
			}
		}

		this.Definition.Members[name] = member

		return nil

	case *ast.FuncDecl:
		// Ignore function declarations.
		return nil
	case *ast.ImportSpec:
		// Ignore import declarations.
		return nil
	case nil:
	default:
		//log.Printf("unexpected type %T\n", t) // %T prints whatever type t has
	}

	return this
}

func resolveTypeExpression(expr ast.Expr) string {

	switch t := expr.(type) {
	case *ast.StarExpr:
		return "*" + resolveTypeExpression(t.X)
	case *ast.ArrayType:
		return "[]" + resolveTypeExpression(t.Elt)
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return resolveTypeExpression(t.X) + "." + t.Sel.Name
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", resolveTypeExpression(t.Key), resolveTypeExpression(t.Value))
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct"
	default:
		return fmt.Sprintf("Unknown<%T>", t)
	}

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

	if strings.HasPrefix(goType, "[]") {
		return true, goType[2:]
	}

	return false, ""
}

func parseJsonInfo(s string) (string, bool) {
	rxJson := regexp.MustCompile(`json:"([^"]+)"`)

	if !rxJson.MatchString(s) {
		return "", false
	}

	matches := rxJson.FindStringSubmatch(s)
	words := strings.Split(matches[1], ",")

	name := words[0]
	if len(words) == 1 {
		return name, false
	}

	for _, word := range words[1:] {
		if word == "omitempty" {
			return name, true
		}
	}

	return name, false
}

func parseMemberDescription(s string) string {

	if s == "" {
		return ""
	}

	rxDesc := regexp.MustCompile(`@(?i:desc)\s+"([^"]+)"`)

	if !rxDesc.MatchString(s) {
		return ""
	}

	matches := rxDesc.FindStringSubmatch(s)

	return matches[1]
}

func parseValidateTag(s string) (isRequired bool) {
	rxValidate := regexp.MustCompile(`validate:"([^"]+)"`)

	if !rxValidate.MatchString(s) {
		return false
	}

	matches := rxValidate.FindStringSubmatch(s)

	words := strings.Split(matches[1], ",")
	for _, word := range words {
		if word == "required" {
			return true
		}
	}

	return false

}
