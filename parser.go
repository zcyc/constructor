package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
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
				structInfo.Fields = append(structInfo.Fields, FieldInfo{
					Name:       fieldType, // Use type as name for embedded fields
					Type:       fieldType,
					Tag:        tag,
					Exported:   true, // Embedded fields are always exported
					Skip:       skip,
					SkipGetter: skipGetter,
					SkipSetter: skipSetter,
				})
				continue
			}

			// Regular fields
			for _, name := range field.Names {
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
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + exprToString(t.Elt)
		}
		return "[" + exprToString(t.Len) + "]" + exprToString(t.Elt)
	case *ast.MapType:
		return "map[" + exprToString(t.Key) + "]" + exprToString(t.Value)
	case *ast.ChanType:
		switch t.Dir {
		case ast.SEND:
			return "chan<- " + exprToString(t.Value)
		case ast.RECV:
			return "<-chan " + exprToString(t.Value)
		default:
			return "chan " + exprToString(t.Value)
		}
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.InterfaceType:
		if len(t.Methods.List) == 0 {
			return "interface{}"
		}
		return "interface{...}"
	case *ast.FuncType:
		return "func(...)"
	case *ast.StructType:
		return "struct{...}"
	case *ast.Ellipsis:
		return "..." + exprToString(t.Elt)
	case *ast.BasicLit:
		return t.Value
	default:
		// Fallback for unknown types
		return fmt.Sprintf("%v", reflect.TypeOf(expr).Elem().Name())
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
