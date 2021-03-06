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
					definition.Enums = values
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

		controlsDoc := parseOpenApiControls(t.Doc.Text())
		controlsComment := parseOpenApiControls(t.Comment.Text())

		controls := OpenApiControls{
			Ignore:     controlsDoc.Ignore || controlsComment.Ignore,
			Deprecated: controlsDoc.Deprecated || controlsComment.Deprecated,
		}

		if controls.Ignore {
			return nil
		}

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

		var (
			jsonName      string
			jsonOmitEmpty bool
			validations   ValidationMap
		)

		if t.Tag != nil {
			jsonName, jsonOmitEmpty = parseJsonInfo(t.Tag.Value)
			if jsonName == "-" {
				return nil
			}

			validations = parseValidateTag(t.Tag.Value)
		} else {
			validations = make(ValidationMap)
		}

		var desc string = parseMemberDescription(t.Doc.Text())
		if desc == "" {
			desc = parseMemberDescription(t.Comment.Text())
		}

		goType := resolveTypeExpression(t.Type)

		var member SchemerDefiner

		if isMap, k, v := IsMap(goType); isMap {
			keyType := &MemberIntermediate{
				Type:        k,
				Name:        name,
				Validations: validations,
			}

			valueType := &MemberIntermediate{
				Type:        v,
				Name:        name,
				Validations: validations,
			}

			member = &MapIntermediate{
				Name:          name,
				Type:          goType,
				JsonName:      jsonName,
				JsonOmitEmpty: jsonOmitEmpty,
				ValueType:     valueType,
				KeyType:       keyType,
				Description:   desc,
				Validations:   validations,
				Deprecated:    controls.Deprecated,
			}

		} else if isSlice, v := IsSlice(goType); isSlice {
			valueType := &MemberIntermediate{
				Type:        v,
				Name:        name,
				Validations: validations,
			}

			member = &SliceIntermediate{
				Name:          name,
				Type:          goType,
				JsonName:      jsonName,
				JsonOmitEmpty: jsonOmitEmpty,
				ValueType:     valueType,
				Description:   desc,
				Validations:   validations,
				Deprecated:    controls.Deprecated,
			}
		} else {
			member = &MemberIntermediate{
				Type:          goType,
				Name:          name,
				JsonName:      jsonName,
				JsonOmitEmpty: jsonOmitEmpty,
				Description:   desc,
				Validations:   validations,
				Deprecated:    controls.Deprecated,
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

	rxDesc := regexp.MustCompile(`@(?i:desc)\s+"?([^"]+)"?`)

	if !rxDesc.MatchString(s) {
		return ""
	}

	matches := rxDesc.FindStringSubmatch(s)

	return matches[1]
}

func parseValidateTag(s string) map[string]string {
	rxValidate := regexp.MustCompile(`validate:"([^"]+)"`)

	validations := make(map[string]string)

	if !rxValidate.MatchString(s) {
		return validations
	}

	matches := rxValidate.FindStringSubmatch(s)

	expressions := strings.Split(matches[1], ",")
	for _, expression := range expressions {

		parts := strings.Split(expression, "=")
		k := parts[0]
		v := ""

		if len(parts) > 1 {
			v = parts[1]
		}

		validations[k] = v

	}

	return validations

}

type OpenApiControls struct {
	Ignore     bool
	Deprecated bool
}

func parseOpenApiControls(s string) OpenApiControls {

	var controls OpenApiControls

	rxIgnore := regexp.MustCompile(`@(?i:ignore)`)
	rxDeprecated := regexp.MustCompile(`@(?i:deprecated)`)

	if rxIgnore.MatchString(s) {
		controls.Ignore = true
	}

	if rxDeprecated.MatchString(s) {
		controls.Deprecated = true
	}

	return controls
}
