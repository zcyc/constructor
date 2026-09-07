package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"
)

// Generator generates constructor code
type Generator struct {
	config *GeneratorConfig
	info   *StructInfo
}

// NewGenerator creates a new generator
func NewGenerator(config *GeneratorConfig, info *StructInfo) *Generator {
	return &Generator{
		config: config,
		info:   info,
	}
}

// Generate generates constructor code based on configuration
func (g *Generator) Generate() (string, error) {
	if g.config.InitReturnsError && g.config.InitFunc == "" {
		return "", fmt.Errorf("initReturnsError requires InitFunc")
	}
	if g.returnsError() {
		if _, exists := g.typeParameterNames()["error"]; exists {
			return "", fmt.Errorf("type parameter error cannot be used with constructors that return errors")
		}
	}
	if err := g.validateImportNameConflicts(); err != nil {
		return "", err
	}

	var buf bytes.Buffer

	if g.info.BuildConstraints != "" {
		buf.WriteString(g.info.BuildConstraints)
		buf.WriteString("\n\n")
	}

	// Write package declaration
	buf.WriteString(fmt.Sprintf("package %s\n\n", g.info.PackageName))
	if imports := g.importsForGeneratedTypes(); len(imports) > 0 {
		if len(imports) == 1 {
			if imports[0].Name == "" {
				fmt.Fprintf(&buf, "import %q\n\n", imports[0].Path)
			} else {
				fmt.Fprintf(&buf, "import %s %q\n\n", imports[0].Name, imports[0].Path)
			}
		} else {
			buf.WriteString("import (\n")
			for _, imp := range imports {
				if imp.Name == "" {
					fmt.Fprintf(&buf, "\t%q\n", imp.Path)
					continue
				}
				fmt.Fprintf(&buf, "\t%s %q\n", imp.Name, imp.Path)
			}
			buf.WriteString(")\n\n")
		}
	}
	buf.WriteString(generatedFileMarker + "\n\n")

	fields := g.info.GetFieldsForConstructor()

	// Generate constructors based on types
	for _, constructorType := range g.config.ConstructorTypes {
		switch constructorType {
		case "allArgs":
			code, err := g.generateAllArgsConstructor(fields)
			if err != nil {
				return "", err
			}
			buf.WriteString(code)
			buf.WriteString("\n\n")

		case "builder":
			code, err := g.generateBuilderConstructor(fields)
			if err != nil {
				return "", err
			}
			buf.WriteString(code)
			buf.WriteString("\n\n")

		case "options":
			code, err := g.generateOptionsConstructor(fields)
			if err != nil {
				return "", err
			}
			buf.WriteString(code)
			buf.WriteString("\n\n")

		default:
			return "", fmt.Errorf("unknown constructor type: %s", constructorType)
		}
	}

	// Generate getters if requested
	if g.config.WithGetter {
		getterFields := g.info.GetFieldsForGetter()
		code := g.generateGetters(getterFields)
		buf.WriteString(code)
		buf.WriteString("\n")
	}

	code, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("format generated code: %w", err)
	}
	return string(code), nil
}

func (g *Generator) importsForGeneratedTypes() []ImportInfo {
	fields := append([]FieldInfo{}, g.info.GetFieldsForConstructor()...)
	if g.config.WithGetter {
		fields = append(fields, g.info.GetFieldsForGetter()...)
	}

	imports := make([]ImportInfo, 0, len(g.info.Imports))
	addImport := func(imp ImportInfo) {
		for _, existing := range imports {
			if existing.Path == imp.Path {
				return
			}
		}
		imports = append(imports, imp)
	}
	for _, imp := range g.info.Imports {
		if imp.Name == "_" {
			continue
		}
		if (imp.Name == "." && imp.Used) || (imp.Name != "." && importQualifierUsed(fields, g.info.TypeParams, importQualifier(imp))) {
			addImport(imp)
		}
	}
	if g.hasRequiredFields() {
		addImport(g.importForPath("errors", "errors"))
		addImport(g.importForPath("reflect", "reflect"))
	}
	return imports
}

func (g *Generator) importForPath(importPath, defaultName string) ImportInfo {
	for _, imp := range g.info.Imports {
		if imp.Path == importPath && imp.Name != "_" {
			if imp.Name == "." && !imp.Used {
				imp.Name = defaultName
				imp.Qualifier = ""
			}
			return g.avoidImportNameConflict(imp, defaultName)
		}
	}
	return g.avoidImportNameConflict(ImportInfo{Name: defaultName, Path: importPath}, defaultName)
}

func (g *Generator) avoidImportNameConflict(imp ImportInfo, defaultName string) ImportInfo {
	if imp.Name == "." {
		return imp
	}
	if !g.importNameTaken(importQualifier(imp), imp.Path) {
		return imp
	}
	name := "constructor" + toUpperCamelCase(defaultName)
	for suffix := 1; ; suffix++ {
		candidate := name
		if suffix > 1 {
			candidate = fmt.Sprintf("constructor%s%d", toUpperCamelCase(defaultName), suffix)
		}
		if !g.importNameTaken(candidate, imp.Path) {
			imp.Name = candidate
			imp.Qualifier = ""
			return imp
		}
	}
}

func (g *Generator) importNameTaken(name, importPath string) bool {
	if _, exists := g.typeParameterNames()[name]; exists {
		return true
	}
	for _, imp := range g.info.Imports {
		if imp.Path != importPath && imp.Name != "." && imp.Name != "_" && importQualifier(imp) == name {
			return true
		}
	}
	return false
}

func (g *Generator) validateImportNameConflicts() error {
	fields := append([]FieldInfo{}, g.info.GetFieldsForConstructor()...)
	if g.config.WithGetter {
		fields = append(fields, g.info.GetFieldsForGetter()...)
	}
	for _, imp := range g.info.Imports {
		if imp.Name == "." || imp.Name == "_" {
			continue
		}
		qualifier := importQualifier(imp)
		if _, exists := g.typeParameterNames()[qualifier]; !exists {
			continue
		}
		if importQualifierUsed(fields, g.info.TypeParams, qualifier) {
			return fmt.Errorf("import qualifier %q conflicts with a type parameter", qualifier)
		}
	}
	if g.hasRequiredFields() {
		for _, helper := range []struct {
			path   string
			member string
		}{{"errors", "New"}, {"reflect", "ValueOf"}} {
			for _, imp := range g.info.Imports {
				if imp.Path == helper.path && imp.Name == "." && imp.Used {
					if _, exists := g.typeParameterNames()[helper.member]; exists {
						return fmt.Errorf("dot import %q conflicts with type parameter %q", helper.path, helper.member)
					}
				}
			}
		}
	}
	return nil
}

func (g *Generator) generatedSelector(importPath, defaultName, member string) string {
	imp := g.importForPath(importPath, defaultName)
	if imp.Name == "." {
		return member
	}
	return importQualifier(imp) + "." + member
}

func importQualifier(imp ImportInfo) string {
	if imp.Name != "" {
		return imp.Name
	}
	if imp.Qualifier != "" {
		return imp.Qualifier
	}

	qualifier := path.Base(imp.Path)
	if index := strings.LastIndex(qualifier, ".v"); index > 0 && allDigits(qualifier[index+2:]) {
		return qualifier[:index]
	}
	if strings.HasPrefix(qualifier, "v") && allDigits(qualifier[1:]) {
		return path.Base(path.Dir(imp.Path))
	}
	return qualifier
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func importQualifierUsed(fields []FieldInfo, typeParams, qualifier string) bool {
	if typeParamsQualifierUsed(typeParams, qualifier) {
		return true
	}

	for _, field := range fields {
		for _, value := range []string{field.Type, field.Default} {
			expr, err := parser.ParseExpr(value)
			if err != nil {
				continue
			}
			if qualifierUsedInNode(expr, qualifier) {
				return true
			}
		}
	}
	return false
}

func (g *Generator) hasRequiredFields() bool {
	for _, field := range g.info.Fields {
		if field.Required {
			return true
		}
	}
	return false
}

func (g *Generator) returnsError() bool {
	return g.config.InitReturnsError || g.hasRequiredFields()
}

func (g *Generator) requiredError(fieldName string) string {
	errorNew := g.generatedSelector("errors", "errors", "New")
	if g.config.ReturnValue {
		return fmt.Sprintf("return %s{}, %s(%q)", g.typeReference(), errorNew, "required field "+fieldName+" is zero")
	}
	return fmt.Sprintf("return nil, %s(%q)", errorNew, "required field "+fieldName+" is zero")
}

func (g *Generator) requiredValidation(receiver string) string {
	if !g.hasRequiredFields() {
		return ""
	}

	reflectValueOf := g.generatedSelector("reflect", "reflect", "ValueOf")
	var buf strings.Builder
	for _, field := range g.info.Fields {
		if !field.Required {
			continue
		}
		fmt.Fprintf(&buf, "\n\tif %s(&%s.%s).Elem().IsZero() {\n\t\t%s\n\t}", reflectValueOf, receiver, field.Name, g.requiredError(field.Name))
	}
	return buf.String()
}

func defaultAssignments(fields []FieldInfo) []string {
	assignments := make([]string, 0)
	for _, field := range fields {
		if field.Default != "" {
			assignments = append(assignments, fmt.Sprintf("%s: %s,", field.Name, field.Default))
		}
	}
	return assignments
}

func typeParamsQualifierUsed(typeParams, qualifier string) bool {
	if typeParams == "" {
		return false
	}

	file, err := parser.ParseFile(token.NewFileSet(), "", "package p\ntype _"+typeParams+" struct{}", 0)
	if err != nil {
		return false
	}
	return qualifierUsedInNode(file, qualifier)
}

func qualifierUsedInNode(node ast.Node, qualifier string) bool {
	used := false
	ast.Inspect(node, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := selector.X.(*ast.Ident)
		if ok && ident.Name == qualifier {
			used = true
			return false
		}
		return true
	})
	return used
}

// generateAllArgsConstructor generates a constructor with all fields as parameters
func (g *Generator) generateAllArgsConstructor(fields []FieldInfo) (string, error) {
	tmpl := `// New{{.StructName}} creates a new {{.StructName}}
func New{{.StructName}}{{.TypeParams}}({{.Params}}) {{.ReturnType}} {
	{{.VarDecl}}{{.TypeReference}}{
		{{.FieldAssignments}}
	}{{.InitCall}}{{if .ReturnValue}}
	return {{.ReturnValue}}{{end}}
}`

	paramFields := make([]FieldInfo, 0, len(fields))
	for _, field := range fields {
		if field.Default == "" {
			paramFields = append(paramFields, field)
		}
	}
	baseReturnType := "*" + g.typeReference()
	returnType := baseReturnType
	needsValue := g.config.InitFunc != "" || g.returnsError()
	valueName := g.localName("v")
	reservedNames := g.reservedNames()
	if needsValue {
		reservedNames[valueName] = struct{}{}
	}
	localNames := uniqueFieldNames(paramFields, func(field FieldInfo) string {
		return toLowerCamelCase(field.Name)
	}, reservedNames)

	params := make([]string, 0, len(paramFields))
	assignments := make([]string, 0, len(fields))
	for _, field := range fields {
		if field.Default != "" {
			assignments = append(assignments, fmt.Sprintf("%s: %s,", field.Name, field.Default))
			continue
		}
		paramName := localNames[field.Name]
		params = append(params, fmt.Sprintf("%s %s", paramName, field.Type))
		assignments = append(assignments, fmt.Sprintf("%s: %s,", field.Name, paramName))
	}

	varDecl := "return &"
	returnValue := ""
	if needsValue {
		if g.config.ReturnValue {
			varDecl = valueName + " := "
		} else {
			varDecl = valueName + " := &"
		}
		if g.returnsError() {
			returnValue = valueName + ", nil"
		} else {
			returnValue = valueName
		}
	}

	if g.config.ReturnValue {
		baseReturnType = g.typeReference()
		returnType = g.typeReference()
		if g.returnsError() {
			returnType = fmt.Sprintf("(%s, error)", baseReturnType)
		}
		if !needsValue {
			varDecl = "return "
			returnValue = ""
		}
	}
	if !g.config.ReturnValue && g.returnsError() {
		returnType = fmt.Sprintf("(%s, error)", baseReturnType)
	}

	// Handle validation and init function.
	initCall := ""
	if needsValue {
		initCall = g.requiredValidation(valueName)
	}
	if g.config.InitFunc != "" {
		if g.config.InitReturnsError {
			if g.config.ReturnValue {
				initCall += fmt.Sprintf("\n\tif err := %s.%s(); err != nil {\n\t\treturn %s{}, err\n\t}", valueName, g.config.InitFunc, g.typeReference())
			} else {
				initCall += fmt.Sprintf("\n\tif err := %s.%s(); err != nil {\n\t\treturn nil, err\n\t}", valueName, g.config.InitFunc)
			}
		} else {
			initCall += fmt.Sprintf("\n\t%s.%s()", valueName, g.config.InitFunc)
		}
	}

	data := map[string]string{
		"StructName":       g.info.Name,
		"TypeParams":       g.info.TypeParams,
		"TypeReference":    g.typeReference(),
		"Params":           strings.Join(params, ", "),
		"ReturnType":       returnType,
		"VarDecl":          varDecl,
		"FieldAssignments": strings.Join(assignments, "\n\t\t"),
		"InitCall":         initCall,
		"ReturnValue":      returnValue,
	}

	t, err := template.New("allArgs").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generateBuilderConstructor generates a builder pattern constructor
func (g *Generator) generateBuilderConstructor(fields []FieldInfo) (string, error) {
	var buf bytes.Buffer

	builderName := g.info.Name + "Builder"
	builderType := builderName + g.info.TypeArgs
	prefix := g.config.SetterPrefix
	if prefix == "" {
		prefix = "" // No prefix by default, methods named after fields
	}

	// Generate builder struct
	receiverName := g.localName("b")
	localNames := uniqueFieldNames(fields, func(field FieldInfo) string {
		return toLowerCamelCase(field.Name)
	}, g.reservedNames(receiverName))
	methodNames := uniqueFieldNames(fields, func(field FieldInfo) string {
		return prefix + toUpperCamelCase(field.Name)
	}, map[string]struct{}{"Build": {}})

	buf.WriteString(fmt.Sprintf("// %s is a builder for %s\n", builderName, g.info.Name))
	buf.WriteString(fmt.Sprintf("type %s%s struct {\n", builderName, g.info.TypeParams))
	for _, field := range fields {
		buf.WriteString(fmt.Sprintf("\t%s %s\n", localNames[field.Name], field.Type))
	}
	buf.WriteString("}\n\n")

	// Generate builder constructor
	buf.WriteString(fmt.Sprintf("// New%s creates a new %s\n", builderName, builderName))
	buf.WriteString(fmt.Sprintf("func New%s%s() *%s {\n", builderName, g.info.TypeParams, builderType))
	buf.WriteString(fmt.Sprintf("\treturn &%s{\n", builderType))
	for _, field := range fields {
		if field.Default != "" {
			buf.WriteString(fmt.Sprintf("\t\t%s: %s,\n", localNames[field.Name], field.Default))
		}
	}
	buf.WriteString("\t}\n")
	buf.WriteString("}\n\n")

	// Generate setter methods
	for _, field := range fields {
		methodName := methodNames[field.Name]
		paramName := localNames[field.Name]
		fieldName := localNames[field.Name]

		buf.WriteString(fmt.Sprintf("// %s sets the %s field\n", methodName, field.Name))
		buf.WriteString(fmt.Sprintf("func (%s *%s) %s(%s %s) *%s {\n",
			receiverName, builderType, methodName, paramName, field.Type, builderType))
		buf.WriteString(fmt.Sprintf("\t%s.%s = %s\n", receiverName, fieldName, paramName))
		buf.WriteString(fmt.Sprintf("\treturn %s\n", receiverName))
		buf.WriteString("}\n\n")
	}

	// Generate Build method
	returnType := "*" + g.typeReference()
	if g.config.ReturnValue {
		returnType = g.typeReference()
	}
	if g.returnsError() {
		returnType = fmt.Sprintf("(%s, error)", returnType)
	}

	buf.WriteString(fmt.Sprintf("// Build builds the %s\n", g.info.Name))
	valueName := g.localName("v")
	buf.WriteString(fmt.Sprintf("func (%s *%s) Build() %s {\n", receiverName, builderType, returnType))

	if g.config.ReturnValue {
		buf.WriteString(fmt.Sprintf("\t%s := %s{\n", valueName, g.typeReference()))
	} else {
		buf.WriteString(fmt.Sprintf("\t%s := &%s{\n", valueName, g.typeReference()))
	}

	for _, field := range fields {
		buf.WriteString(fmt.Sprintf("\t\t%s: %s.%s,\n", field.Name, receiverName, localNames[field.Name]))
	}
	buf.WriteString("\t}\n")
	if validation := g.requiredValidation(valueName); validation != "" {
		buf.WriteString(validation)
		buf.WriteByte('\n')
	}

	// Handle init function
	if g.config.InitFunc != "" {
		if g.config.InitReturnsError {
			buf.WriteString(fmt.Sprintf("\tif err := %s.%s(); err != nil {\n", valueName, g.config.InitFunc))
			if g.config.ReturnValue {
				buf.WriteString(fmt.Sprintf("\t\treturn %s{}, err\n", g.typeReference()))
			} else {
				buf.WriteString("\t\treturn nil, err\n")
			}
			buf.WriteString("\t}\n")
		} else {
			buf.WriteString(fmt.Sprintf("\t%s.%s()\n", valueName, g.config.InitFunc))
		}
	}

	if g.returnsError() {
		buf.WriteString(fmt.Sprintf("\treturn %s, nil\n", valueName))
	} else {
		buf.WriteString(fmt.Sprintf("\treturn %s\n", valueName))
	}
	buf.WriteString("}\n")

	return buf.String(), nil
}

// generateOptionsConstructor generates a functional options pattern constructor
func (g *Generator) generateOptionsConstructor(fields []FieldInfo) (string, error) {
	var buf bytes.Buffer

	returnType := "*" + g.typeReference()
	if g.config.ReturnValue {
		returnType = g.typeReference()
	}
	if g.returnsError() {
		returnType = fmt.Sprintf("(%s, error)", returnType)
	}

	// Generate option type
	optionType := g.info.Name + "Option"
	optionTypeReference := optionType + g.info.TypeArgs
	buf.WriteString(fmt.Sprintf("// %s is a functional option for configuring %s\n", optionType, g.info.Name))
	buf.WriteString(fmt.Sprintf("type %s%s func(*%s)\n\n", optionType, g.info.TypeParams, g.typeReference()))

	// Generate option functions
	receiverName := g.localName("s")
	localNames := uniqueFieldNames(fields, func(field FieldInfo) string {
		return toLowerCamelCase(field.Name)
	}, g.reservedNames(receiverName))
	optionNames := uniqueFieldNames(fields, func(field FieldInfo) string {
		return "With" + toUpperCamelCase(field.Name)
	})

	for _, field := range fields {
		optionName := optionNames[field.Name]
		paramName := localNames[field.Name]

		buf.WriteString(fmt.Sprintf("// %s sets the %s field\n", optionName, field.Name))
		buf.WriteString(fmt.Sprintf("func %s%s(%s %s) %s {\n", optionName, g.info.TypeParams, paramName, field.Type, optionTypeReference))
		buf.WriteString(fmt.Sprintf("\treturn func(%s *%s) {\n", receiverName, g.typeReference()))
		buf.WriteString(fmt.Sprintf("\t\t%s.%s = %s\n", receiverName, field.Name, paramName))
		buf.WriteString("\t}\n")
		buf.WriteString("}\n\n")
	}

	// Generate constructor with options
	buf.WriteString(fmt.Sprintf("// New%sWithOptions creates a new %s with functional options\n", g.info.Name, g.info.Name))
	optionListName := g.localName("opts")
	buf.WriteString(fmt.Sprintf("func New%sWithOptions%s(%s ...%s) %s {\n", g.info.Name, g.info.TypeParams, optionListName, optionTypeReference, returnType))

	valueName := g.localName("v")
	buf.WriteString(fmt.Sprintf("\t%s := &%s{\n", valueName, g.typeReference()))
	for _, assignment := range defaultAssignments(fields) {
		buf.WriteString("\t\t" + assignment + "\n")
	}
	buf.WriteString("\t}\n")

	optionValueName := g.localName("opt")
	buf.WriteString(fmt.Sprintf("\tfor _, %s := range %s {\n", optionValueName, optionListName))
	buf.WriteString(fmt.Sprintf("\t\t%s(%s)\n", optionValueName, valueName))
	buf.WriteString("\t}\n")

	if validation := g.requiredValidation(valueName); validation != "" {
		buf.WriteString(validation)
		buf.WriteByte('\n')
	}

	// Handle init function
	if g.config.InitFunc != "" {
		if g.config.InitReturnsError {
			buf.WriteString(fmt.Sprintf("\tif err := %s.%s(); err != nil {\n", valueName, g.config.InitFunc))
			if g.config.ReturnValue {
				buf.WriteString(fmt.Sprintf("\t\treturn %s{}, err\n", g.typeReference()))
			} else {
				buf.WriteString("\t\treturn nil, err\n")
			}
			buf.WriteString("\t}\n")
		} else {
			buf.WriteString(fmt.Sprintf("\t%s.%s()\n", valueName, g.config.InitFunc))
		}
	}

	if g.config.ReturnValue {
		if g.returnsError() {
			buf.WriteString(fmt.Sprintf("\treturn *%s, nil\n", valueName))
		} else {
			buf.WriteString(fmt.Sprintf("\treturn *%s\n", valueName))
		}
	} else {
		if g.returnsError() {
			buf.WriteString(fmt.Sprintf("\treturn %s, nil\n", valueName))
		} else {
			buf.WriteString(fmt.Sprintf("\treturn %s\n", valueName))
		}
	}
	buf.WriteString("}\n")

	return buf.String(), nil
}

// generateGetters generates getter methods for all fields
func (g *Generator) generateGetters(fields []FieldInfo) string {
	var buf bytes.Buffer
	getterNames := uniqueFieldNames(fields, func(field FieldInfo) string {
		return "Get" + toUpperCamelCase(field.Name)
	})
	receiverName := g.localName(firstRune(g.info.Name, unicode.ToLower))

	for _, field := range fields {
		if !field.Exported {
			getterName := getterNames[field.Name]

			buf.WriteString(fmt.Sprintf("// %s returns the %s field\n", getterName, field.Name))
			buf.WriteString(fmt.Sprintf("func (%s *%s) %s() %s {\n",
				receiverName, g.typeReference(), getterName, field.Type))
			buf.WriteString(fmt.Sprintf("\treturn %s.%s\n", receiverName, field.Name))
			buf.WriteString("}\n\n")
		}
	}

	return buf.String()
}

func (g *Generator) typeReference() string {
	return g.info.Name + g.info.TypeArgs
}

func (g *Generator) typeParameterNames() map[string]struct{} {
	names := make(map[string]struct{})
	for _, name := range strings.Split(strings.Trim(g.info.TypeArgs, "[]"), ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			names[name] = struct{}{}
		}
	}
	return names
}

func (g *Generator) localName(preferred string) string {
	name := preferred
	if token.IsKeyword(name) {
		name += "Value"
	}
	reserved := g.reservedNames()
	for suffix := 1; ; suffix++ {
		candidate := name
		if suffix > 1 {
			candidate = fmt.Sprintf("%s%d", name, suffix)
		}
		if _, exists := reserved[candidate]; !exists {
			return candidate
		}
	}
}

func (g *Generator) reservedNames(extra ...string) map[string]struct{} {
	reserved := g.typeParameterNames()
	for _, imp := range g.importsForGeneratedTypes() {
		if imp.Name == "." || imp.Name == "_" {
			continue
		}
		if qualifier := importQualifier(imp); qualifier != "" {
			reserved[qualifier] = struct{}{}
		}
	}
	for _, name := range extra {
		if name != "" {
			reserved[name] = struct{}{}
		}
	}
	return reserved
}

// toLowerCamelCase converts a string to lowerCamelCase
func toLowerCamelCase(s string) string {
	return changeFirstRune(s, unicode.ToLower)
}

// toUpperCamelCase converts a string to UpperCamelCase
func toUpperCamelCase(s string) string {
	return changeFirstRune(s, unicode.ToUpper)
}

func changeFirstRune(s string, change func(rune) rune) string {
	if s == "" {
		return s
	}

	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size == 1 {
		return s
	}
	return string(change(r)) + s[size:]
}

func firstRune(s string, change func(rune) rune) string {
	if s == "" {
		return s
	}
	r, _ := utf8.DecodeRuneInString(s)
	return string(change(r))
}

func uniqueFieldNames(fields []FieldInfo, base func(FieldInfo) string, reserved ...map[string]struct{}) map[string]string {
	names := make(map[string]string, len(fields))
	used := make(map[string]struct{}, len(fields))
	if len(reserved) > 0 {
		for name := range reserved[0] {
			used[name] = struct{}{}
		}
	}

	for _, field := range fields {
		name := base(field)
		if token.IsKeyword(name) {
			name += "Value"
		}
		for suffix := 1; ; suffix++ {
			candidate := name
			if suffix > 1 {
				candidate = fmt.Sprintf("%s%d", name, suffix)
			}
			if _, exists := used[candidate]; exists {
				continue
			}
			names[field.Name] = candidate
			used[candidate] = struct{}{}
			break
		}
	}

	return names
}
