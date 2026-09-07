package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/importer"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const goCommandTimeout = 10 * time.Second

var errStructNotFound = errors.New("struct not found")

func hasGeneratedFileMarker(filename string, content []byte) bool {
	file, err := parser.ParseFile(token.NewFileSet(), filename, content, parser.ParseComments)
	if err != nil {
		return false
	}
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if strings.TrimSpace(comment.Text) == generatedFileMarker {
				return true
			}
		}
	}
	return false
}

// ParseStruct parses a Go source file and extracts struct information
func ParseStruct(filename, structName string) (*StructInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	var structInfo *StructInfo
	var parseErr error
	var structType *ast.StructType
	var typeParams *ast.FieldList

	// Only inspect top-level type declarations. A local type inside a function
	// cannot be referenced by the generated top-level declarations.
	for _, declaration := range node.Decls {
		genDecl, ok := declaration.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, specification := range genDecl.Specs {
			typeSpec, ok := specification.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != structName {
				continue
			}
			structType, ok = typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			typeParams = typeSpec.TypeParams

			structInfo = &StructInfo{
				Name:             structName,
				PackageName:      node.Name.Name,
				Fields:           []FieldInfo{},
				Imports:          parseImports(node.Imports, filepath.Dir(filename)),
				BuildConstraints: buildConstraints(node, filename),
				TypeParams:       typeParamDecl(typeSpec.TypeParams),
				TypeArgs:         typeParamArgs(typeSpec.TypeParams),
			}

			for _, field := range structType.Fields.List {
				fieldType := exprToString(field.Type)

				var tag string
				if field.Tag != nil {
					tag = field.Tag.Value
				}

				skip, skipGetter, skipSetter := parseFieldSkipTags(tag)
				defaultValue, required, err := parseConstructorFieldOptions(tag)
				if err != nil {
					parseErr = fmt.Errorf("invalid constructor options for field %s: %w", fieldNameForError(field), err)
					break
				}
				if (skip || skipSetter) && (defaultValue != "" || required) {
					parseErr = fmt.Errorf("field %s cannot combine constructor default/required with skip or setter:false", fieldNameForError(field))
					break
				}
				if defaultValue != "" {
					if _, err := parser.ParseExpr(defaultValue); err != nil {
						parseErr = fmt.Errorf("invalid default for field %s: %w", fieldNameForError(field), err)
						break
					}
				}

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
						Default:    defaultValue,
						Required:   required,
					})
					continue
				}

				for _, name := range field.Names {
					if name.Name == "_" {
						continue
					}
					structInfo.Fields = append(structInfo.Fields, FieldInfo{
						Name:       name.Name,
						Type:       fieldType,
						Tag:        tag,
						Exported:   ast.IsExported(name.Name),
						Skip:       skip,
						SkipGetter: skipGetter,
						SkipSetter: skipSetter,
						Default:    defaultValue,
						Required:   required,
					})
				}
			}
			break
		}
		if structInfo != nil {
			break
		}
	}

	if structInfo == nil {
		return nil, fmt.Errorf("%w: struct %s not found in file %s", errStructNotFound, structName, filename)
	}
	if parseErr != nil {
		return nil, parseErr
	}
	markDotImportsUsed(structInfo.Imports, node, structType, typeParams, structInfo.Fields, fset, filepath.Dir(filename))

	return structInfo, nil
}

func buildConstraints(file *ast.File, filename string) string {
	var modern, legacy constraint.Expr
	for _, group := range file.Comments {
		if group.Pos() >= file.Package {
			break
		}
		for _, comment := range group.List {
			line := strings.TrimSpace(comment.Text)
			parsed, err := constraint.Parse(line)
			if err != nil {
				continue
			}
			if constraint.IsGoBuild(line) {
				modern = andConstraint(modern, parsed)
			} else if constraint.IsPlusBuild(line) {
				legacy = andConstraint(legacy, parsed)
			}
		}
	}

	expression := modern
	if expression == nil {
		expression = legacy
	}
	if filenameExpression := filenameBuildConstraint(filename); filenameExpression != nil {
		expression = andConstraint(expression, filenameExpression)
	}
	if expression == nil {
		return ""
	}
	return "//go:build " + expression.String()
}

func andConstraint(left, right constraint.Expr) constraint.Expr {
	if left == nil {
		return right
	}
	return &constraint.AndExpr{X: left, Y: right}
}

func filenameBuildConstraint(filename string) constraint.Expr {
	name, _, _ := strings.Cut(filepath.Base(filename), ".")
	underscore := strings.IndexByte(name, '_')
	if underscore < 0 {
		return nil
	}
	name = strings.TrimSuffix(name[underscore+1:], "_test")
	parts := strings.Split(name, "_")
	if len(parts) < 2 {
		if len(parts) == 1 && (isKnownGOOS(parts[0]) || isKnownGOARCH(parts[0])) {
			return &constraint.TagExpr{Tag: parts[0]}
		}
		return nil
	}

	last := parts[len(parts)-1]
	if isKnownGOOS(last) {
		return &constraint.TagExpr{Tag: last}
	}
	if isKnownGOARCH(last) {
		if previous := parts[len(parts)-2]; isKnownGOOS(previous) {
			return &constraint.AndExpr{
				X: &constraint.TagExpr{Tag: previous},
				Y: &constraint.TagExpr{Tag: last},
			}
		}
		return &constraint.TagExpr{Tag: last}
	}
	return nil
}

// ponytail: keep the Go 1.24 filename suffix list local; update it when Go adds platforms.
func isKnownGOOS(value string) bool {
	switch value {
	case "aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios", "js", "linux", "nacl", "netbsd", "openbsd", "plan9", "solaris", "wasip1", "windows", "zos":
		return true
	default:
		return false
	}
}

func isKnownGOARCH(value string) bool {
	switch value {
	case "386", "amd64", "amd64p32", "arm", "arm64", "arm64be", "armbe", "loong64", "mips", "mips64", "mips64le", "mips64p32", "mips64p32le", "mipsle", "ppc", "ppc64", "ppc64le", "riscv", "riscv64", "s390", "s390x", "sparc", "sparc64", "wasm":
		return true
	default:
		return false
	}
}

func fieldNameForError(field *ast.Field) string {
	if len(field.Names) > 0 {
		return field.Names[0].Name
	}
	return embeddedFieldName(field.Type)
}

func parseConstructorFieldOptions(tag string) (string, bool, error) {
	if tag == "" {
		return "", false, nil
	}

	value := reflect.StructTag(strings.Trim(tag, "`")).Get("constructor")
	if value == "" {
		return "", false, nil
	}

	var defaultValue string
	var required bool
	if index := strings.Index(value, "default="); index >= 0 {
		for _, option := range strings.Split(strings.TrimSuffix(strings.TrimSpace(value[:index]), ","), ",") {
			option = strings.TrimSpace(option)
			if option == "" {
				continue
			}
			if option == "-" && strings.TrimSpace(value) != "-" {
				return "", false, fmt.Errorf("skip option cannot be combined with other options")
			}
			if !isConstructorOption(option) {
				return "", false, fmt.Errorf("unknown option %q", option)
			}
			if option == "required" {
				required = true
			}
		}
		defaultValue = strings.TrimSpace(value[index+len("default="):])
		if defaultValue == "" {
			return "", false, fmt.Errorf("default expression is empty")
		}
	} else {
		for _, option := range strings.Split(value, ",") {
			option = strings.TrimSpace(option)
			if option == "" {
				continue
			}
			if option == "-" && strings.TrimSpace(value) != "-" {
				return "", false, fmt.Errorf("skip option cannot be combined with other options")
			}
			if !isConstructorOption(option) {
				return "", false, fmt.Errorf("unknown option %q", option)
			}
			if option == "required" {
				required = true
			}
		}
	}
	if required && defaultValue != "" {
		return "", false, fmt.Errorf("required cannot be combined with default")
	}
	return defaultValue, required, nil
}

func isConstructorOption(option string) bool {
	switch option {
	case "-", "getter:false", "setter:false", "required":
		return true
	default:
		return false
	}
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

func parseImports(specs []*ast.ImportSpec, dir string) []ImportInfo {
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
		imp := ImportInfo{Name: name, Path: path}
		if name == "" {
			imp.Qualifier = resolveImportQualifier(dir, path)
		}
		imports = append(imports, imp)
	}
	return imports
}

func resolveImportQualifier(dir, importPath string) string {
	metadata, err := listGoPackage(dir, importPath, false)
	if err != nil {
		return ""
	}
	return metadata.Name
}

func markDotImportsUsed(imports []ImportInfo, file *ast.File, target *ast.StructType, typeParams *ast.FieldList, fields []FieldInfo, fset *token.FileSet, dir string) {
	if target == nil {
		return
	}

	dotImportPaths := make(map[string]struct{})
	for _, imp := range imports {
		if imp.Name == "." {
			dotImportPaths[imp.Path] = struct{}{}
		}
	}
	if len(dotImportPaths) == 0 {
		return
	}

	info := &types.Info{Uses: make(map[*ast.Ident]types.Object)}
	checkFile := *file
	checkFile.Decls = append([]ast.Decl(nil), file.Decls...)
	defaultExpressions := make([]ast.Expr, 0)
	for _, field := range fields {
		if field.Default == "" {
			continue
		}
		expression, err := parser.ParseExpr(field.Default)
		if err != nil {
			continue
		}
		defaultExpressions = append(defaultExpressions, expression)
		checkFile.Decls = append(checkFile.Decls, &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{&ast.ValueSpec{
				Names:  []*ast.Ident{ast.NewIdent("_")},
				Values: []ast.Expr{expression},
			}},
		})
	}

	_, _ = (&types.Config{
		Importer: moduleImporter(fset, dir),
		Error:    func(error) {},
	}).Check(file.Name.Name, fset, []*ast.File{&checkFile}, info)
	for i := range imports {
		if imports[i].Name != "." {
			continue
		}
		exported, err := exportedNamesForPackage(dir, imports[i].Path)
		if err != nil {
			continue
		}
		imports[i].Used = dotImportReferencesPackage(info, target, typeParams, defaultExpressions, imports[i].Path, exported)
	}
}

func exportedNamesForPackage(dir, importPath string) (map[string]struct{}, error) {
	// ponytail: use a source-level fallback only when module export data is
	// unavailable; the typed importer handles valid module-aware packages.
	metadata, err := listGoPackage(dir, importPath, false)
	if err != nil {
		return nil, err
	}
	exported := make(map[string]struct{})
	files := append(metadata.GoFiles, metadata.CgoFiles...)
	for _, filename := range files {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(metadata.Dir, filename), nil, 0)
		if err != nil {
			return nil, err
		}
		for _, declaration := range file.Decls {
			switch declaration := declaration.(type) {
			case *ast.FuncDecl:
				if declaration.Recv == nil && ast.IsExported(declaration.Name.Name) {
					exported[declaration.Name.Name] = struct{}{}
				}
			case *ast.GenDecl:
				for _, specification := range declaration.Specs {
					switch specification := specification.(type) {
					case *ast.TypeSpec:
						if ast.IsExported(specification.Name.Name) {
							exported[specification.Name.Name] = struct{}{}
						}
					case *ast.ValueSpec:
						for _, name := range specification.Names {
							if ast.IsExported(name.Name) {
								exported[name.Name] = struct{}{}
							}
						}
					}
				}
			}
		}
	}
	return exported, nil
}

type listedPackage struct {
	Name     string
	Dir      string
	Export   string
	GoFiles  []string
	CgoFiles []string
}

func listGoPackage(dir, importPath string, export bool) (listedPackage, error) {
	args := []string{"list", "-json"}
	if export {
		args = append(args, "-export")
	}
	args = append(args, importPath)
	output, err := runGoCommand(dir, args...)
	if err != nil {
		return listedPackage{}, err
	}

	var metadata listedPackage
	if err := json.Unmarshal(output, &metadata); err != nil {
		return listedPackage{}, err
	}
	return metadata, nil
}

func runGoCommand(dir string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), goCommandTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, "go", args...)
	command.Dir = dir
	output, err := command.Output()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("go command timed out: %w", ctx.Err())
	}
	return output, err
}

func moduleImporter(fset *token.FileSet, dir string) types.Importer {
	return importer.ForCompiler(fset, "gc", func(importPath string) (io.ReadCloser, error) {
		metadata, err := listGoPackage(dir, importPath, true)
		if err != nil {
			return nil, err
		}
		if metadata.Export == "" {
			return nil, fmt.Errorf("package %q has no export data", importPath)
		}
		return os.Open(metadata.Export)
	})
}

func dotImportReferencesPackage(info *types.Info, target *ast.StructType, typeParams *ast.FieldList, defaults []ast.Expr, importPath string, exported map[string]struct{}) bool {
	usesPackage := func(node ast.Node) bool {
		used := false
		ast.Inspect(node, func(node ast.Node) bool {
			ident, ok := node.(*ast.Ident)
			if !ok {
				return true
			}
			object := info.Uses[ident]
			returnValue := false
			if object != nil {
				returnValue = object.Pkg() != nil && object.Pkg().Path() == importPath
			} else {
				_, returnValue = exported[ident.Name]
			}
			if returnValue {
				used = true
			}
			return !used
		})
		return used
	}

	if target != nil {
		for _, field := range target.Fields.List {
			if usesPackage(field.Type) {
				return true
			}
		}
	}
	if typeParams != nil {
		for _, field := range typeParams.List {
			if usesPackage(field.Type) {
				return true
			}
		}
	}
	for _, expression := range defaults {
		if usesPackage(expression) {
			return true
		}
	}
	return false
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
// - skip: completely skip this field (constructor:"-")
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

	tagValue := strings.TrimSpace(reflect.StructTag(tag).Get("constructor"))
	if tagValue == "-" {
		skip = true
	} else {
		for _, option := range strings.Split(tagValue, ",") {
			switch strings.TrimSpace(option) {
			case "getter:false":
				skipGetter = true
			case "setter:false":
				skipSetter = true
			}
		}
	}

	return skip, skipGetter, skipSetter
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
