package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strconv"
	"strings"
)

// ParseStruct parses a Go source file and extracts struct information
func ParseStruct(filename, structName string) (*StructInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	var structInfo *StructInfo

	ast.Inspect(node, func(n ast.Node) bool {
		// Look for type declarations
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		// Check if it's the struct we're looking for
		if typeSpec.Name.Name != structName {
			return true
		}

		// Check if it's a struct type
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		// Extract struct information
		structInfo = &StructInfo{
			Name:        structName,
			PackageName: node.Name.Name,
			Fields:      []FieldInfo{},
			Imports:     parseImports(node.Imports),
			TypeParams:  typeParamDecl(typeSpec.TypeParams),
			TypeArgs:    typeParamArgs(typeSpec.TypeParams),
		}

		// Parse each field
		for _, field := range structType.Fields.List {
			fieldType := exprToString(field.Type)

			// Get tag if exists
			var tag string
			if field.Tag != nil {
				tag = field.Tag.Value
			}

			// Parse field skip options
			skip, skipGetter, skipSetter := parseFieldSkipTags(tag)

			// Handle embedded fields (no name)
			if len(field.Names) == 0 {
				fieldName := embeddedFieldName(field.Type)
				if fieldName == "" {
					continue
				}
				structInfo.Fields = append(structInfo.Fields, FieldInfo{
					Name:       fieldName,
					Type:       fieldType,
					Tag:        tag,
					Exported:   ast.IsExported(fieldName),
					Skip:       skip,
					SkipGetter: skipGetter,
					SkipSetter: skipSetter,
				})
				continue
			}

			// Regular fields
			for _, name := range field.Names {
				if name.Name == "_" {
					continue
				}
				exported := ast.IsExported(name.Name)
				structInfo.Fields = append(structInfo.Fields, FieldInfo{
					Name:       name.Name,
					Type:       fieldType,
					Tag:        tag,
					Exported:   exported,
					Skip:       skip,
					SkipGetter: skipGetter,
					SkipSetter: skipSetter,
				})
			}
		}

		return false // Found the struct, stop searching
	})

	if structInfo == nil {
		return nil, fmt.Errorf("struct %s not found in file %s", structName, filename)
	}

	return structInfo, nil
}

// exprToString converts an ast.Expr to its string representation
func exprToString(expr ast.Expr) string {
	return nodeToString(expr)
}

func nodeToString(node ast.Node) string {
	if node == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, token.NewFileSet(), node); err != nil {
		return ""
	}
	return buf.String()
}

func typeParamArgs(params *ast.FieldList) string {
	if params == nil {
		return ""
	}

	names := make([]string, 0)
	for _, field := range params.List {
		for _, name := range field.Names {
			names = append(names, name.Name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	return "[" + strings.Join(names, ", ") + "]"
}

func typeParamDecl(params *ast.FieldList) string {
	if params == nil {
		return ""
	}

	parts := make([]string, 0, len(params.List))
	for _, field := range params.List {
		names := make([]string, 0, len(field.Names))
		for _, name := range field.Names {
			names = append(names, name.Name)
		}
		if len(names) == 0 {
			continue
		}
		parts = append(parts, strings.Join(names, ", ")+" "+nodeToString(field.Type))
	}
	if len(parts) == 0 {
		return ""
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func parseImports(specs []*ast.ImportSpec) []ImportInfo {
	imports := make([]ImportInfo, 0, len(specs))
	for _, spec := range specs {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}

		name := ""
		if spec.Name != nil {
			name = spec.Name.Name
		}
		imports = append(imports, ImportInfo{Name: name, Path: path})
	}
	return imports
}

func embeddedFieldName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.SelectorExpr:
		return expr.Sel.Name
	case *ast.StarExpr:
		return embeddedFieldName(expr.X)
	case *ast.IndexExpr:
		return embeddedFieldName(expr.X)
	case *ast.IndexListExpr:
		return embeddedFieldName(expr.X)
	default:
		return ""
	}
}

// parseFieldSkipTags parses field tags to determine skip behavior
// Returns: (skip, skipGetter, skipSetter)
// - skip: completely skip this field (constructor:"-" or newc:"-" or gonstructor:"-")
// - skipGetter: skip getter generation only (constructor:"getter:false")
// - skipSetter: skip setter/constructor parameter only (constructor:"setter:false")
func parseFieldSkipTags(tag string) (bool, bool, bool) {
	if tag == "" {
		return false, false, false
	}

	// Remove backticks
	tag = strings.Trim(tag, "`")

	skip := false
	skipGetter := false
	skipSetter := false

	// Split by spaces to get individual tags
	parts := strings.Fields(tag)
	for _, part := range parts {
		// Check for constructor tag with options
		if strings.HasPrefix(part, "constructor:") {
			tagValue := strings.TrimPrefix(part, "constructor:")
			tagValue = strings.Trim(tagValue, `"`)

			if tagValue == "-" {
				skip = true
			} else if strings.Contains(tagValue, "getter:false") {
				skipGetter = true
			} else if strings.Contains(tagValue, "setter:false") {
				skipSetter = true
			}
		}

		// Check for newc:"-" tag (backward compatibility)
		if strings.HasPrefix(part, "newc:") {
			tagValue := strings.TrimPrefix(part, "newc:")
			tagValue = strings.Trim(tagValue, `"`)
			if tagValue == "-" {
				skip = true
			}
		}

		// Also support gonstructor:"-" tag for compatibility
		if strings.HasPrefix(part, "gonstructor:") {
			tagValue := strings.TrimPrefix(part, "gonstructor:")
			tagValue = strings.Trim(tagValue, `"`)
			if tagValue == "-" {
				skip = true
			}
		}
	}

	return skip, skipGetter, skipSetter
}

// shouldSkipField checks if a field should be skipped based on its tag (backward compatibility)
func shouldSkipField(tag string) bool {
	skip, _, _ := parseFieldSkipTags(tag)
	return skip
}

// GetFieldsForConstructor returns fields that should be included in constructor
// Fields with Skip=true or SkipSetter=true are excluded
func (s *StructInfo) GetFieldsForConstructor() []FieldInfo {
	result := []FieldInfo{}
	for _, field := range s.Fields {
		if !field.Skip && !field.SkipSetter {
			result = append(result, field)
		}
	}
	return result
}

// GetFieldsForGetter returns fields that should have getters generated
// Fields with Skip=true or SkipGetter=true are excluded
func (s *StructInfo) GetFieldsForGetter() []FieldInfo {
	result := []FieldInfo{}
	for _, field := range s.Fields {
		if !field.Skip && !field.SkipGetter && !field.Exported {
			result = append(result, field)
		}
	}
	return result
}
