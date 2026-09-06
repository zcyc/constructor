package main

import (
	"go/ast"
	"go/importer"
	goparser "go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestGenerateAllArgsConstructor(t *testing.T) {
	info := &StructInfo{
		Name:        "TestStruct",
		PackageName: "test",
		Fields: []FieldInfo{
			{Name: "name", Type: "string", Skip: false},
			{Name: "age", Type: "int", Skip: false},
		},
	}

	config := &GeneratorConfig{
		StructName:       "TestStruct",
		ConstructorTypes: []string{"allArgs"},
		ReturnValue:      false,
	}

	gen := NewGenerator(config, info)
	code, err := gen.Generate()

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check generated code contains expected elements
	if !strings.Contains(code, "func NewTestStruct") {
		t.Error("Generated code should contain NewTestStruct function")
	}

	if !strings.Contains(code, "name string") {
		t.Error("Generated code should contain name parameter")
	}

	if !strings.Contains(code, "age int") {
		t.Error("Generated code should contain age parameter")
	}

	if !strings.Contains(code, "*TestStruct") {
		t.Error("Generated code should return pointer by default")
	}
}

func TestGenerateBuilderConstructor(t *testing.T) {
	info := &StructInfo{
		Name:        "TestStruct",
		PackageName: "test",
		Fields: []FieldInfo{
			{Name: "name", Type: "string", Skip: false},
			{Name: "age", Type: "int", Skip: false},
		},
	}

	config := &GeneratorConfig{
		StructName:       "TestStruct",
		ConstructorTypes: []string{"builder"},
		SetterPrefix:     "With",
	}

	gen := NewGenerator(config, info)
	code, err := gen.Generate()

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check builder components
	if !strings.Contains(code, "type TestStructBuilder struct") {
		t.Error("Generated code should contain builder struct")
	}

	if !strings.Contains(code, "func NewTestStructBuilder()") {
		t.Error("Generated code should contain builder constructor")
	}

	if !strings.Contains(code, "func (b *TestStructBuilder) WithName") {
		t.Error("Generated code should contain WithName setter")
	}

	if !strings.Contains(code, "func (b *TestStructBuilder) Build()") {
		t.Error("Generated code should contain Build method")
	}
}

func TestGenerateOptionsConstructor(t *testing.T) {
	info := &StructInfo{
		Name:        "TestStruct",
		PackageName: "test",
		Fields: []FieldInfo{
			{Name: "name", Type: "string", Skip: false},
		},
	}

	config := &GeneratorConfig{
		StructName:       "TestStruct",
		ConstructorTypes: []string{"options"},
	}

	gen := NewGenerator(config, info)
	code, err := gen.Generate()

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check options pattern components
	if !strings.Contains(code, "type TestStructOption func(*TestStruct)") {
		t.Error("Generated code should contain option type")
	}

	if !strings.Contains(code, "func WithName") {
		t.Error("Generated code should contain WithName option")
	}

	if !strings.Contains(code, "func NewTestStructWithOptions") {
		t.Error("Generated code should contain constructor with options")
	}
}

func TestGenerateWithInitFunc(t *testing.T) {
	info := &StructInfo{
		Name:        "TestStruct",
		PackageName: "test",
		Fields: []FieldInfo{
			{Name: "name", Type: "string", Skip: false},
		},
	}

	config := &GeneratorConfig{
		StructName:       "TestStruct",
		ConstructorTypes: []string{"allArgs"},
		InitFunc:         "initialize",
	}

	gen := NewGenerator(config, info)
	code, err := gen.Generate()

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check init function is called
	if !strings.Contains(code, "v.initialize()") {
		t.Error("Generated code should call initialize()")
	}
}

func TestGeneratedInitErrorCodeTypeChecks(t *testing.T) {
	for _, returnValue := range []bool{false, true} {
		info := &StructInfo{
			Name:        "TestStruct",
			PackageName: "test",
			Fields:      []FieldInfo{{Name: "name", Type: "string"}},
		}
		code, err := NewGenerator(&GeneratorConfig{
			ConstructorTypes: []string{"allArgs", "builder", "options"},
			InitFunc:         "initialize",
			InitReturnsError: true,
			ReturnValue:      returnValue,
		}, info).Generate()
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		fset := token.NewFileSet()
		source, err := goparser.ParseFile(fset, "source.go", []byte(`package test

import "errors"

type TestStruct struct { name string }

func (s *TestStruct) initialize() error { return errors.New("invalid") }
`), 0)
		if err != nil {
			t.Fatal(err)
		}
		generated, err := goparser.ParseFile(fset, "generated.go", []byte(code), 0)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := (&types.Config{Importer: importer.Default()}).Check("test", fset, []*ast.File{source, generated}, nil); err != nil {
			t.Fatalf("generated init error code does not type-check (returnValue=%t): %v", returnValue, err)
		}
	}
}

func TestGenerateWithReturnValue(t *testing.T) {
	info := &StructInfo{
		Name:        "TestStruct",
		PackageName: "test",
		Fields: []FieldInfo{
			{Name: "name", Type: "string", Skip: false},
		},
	}

	config := &GeneratorConfig{
		StructName:       "TestStruct",
		ConstructorTypes: []string{"allArgs"},
		ReturnValue:      true,
	}

	gen := NewGenerator(config, info)
	code, err := gen.Generate()

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check returns value not pointer
	if !strings.Contains(code, ") TestStruct {") {
		t.Error("Generated code should return TestStruct value, not pointer")
	}
}

func TestGenerateWithGetters(t *testing.T) {
	info := &StructInfo{
		Name:        "TestStruct",
		PackageName: "test",
		Fields: []FieldInfo{
			{Name: "name", Type: "string", Exported: false, Skip: false},
			{Name: "PublicField", Type: "int", Exported: true, Skip: false},
		},
	}

	config := &GeneratorConfig{
		StructName:       "TestStruct",
		ConstructorTypes: []string{"allArgs"},
		WithGetter:       true,
	}

	gen := NewGenerator(config, info)
	code, err := gen.Generate()

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check getter for private field
	if !strings.Contains(code, "func (t *TestStruct) GetName()") {
		t.Error("Generated code should contain GetName getter")
	}

	// Public field should not have getter
	if strings.Contains(code, "GetPublicField") {
		t.Error("Generated code should not contain getter for public field")
	}
}

func TestToLowerCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Name", "name"},
		{"HTTPClient", "hTTPClient"},
		{"Éclair", "éclair"},
		{"", ""},
	}

	for _, tt := range tests {
		result := toLowerCamelCase(tt.input)
		if result != tt.expected {
			t.Errorf("toLowerCamelCase(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestToUpperCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"name", "Name"},
		{"httpClient", "HttpClient"},
		{"éclair", "Éclair"},
		{"", ""},
	}

	for _, tt := range tests {
		result := toUpperCamelCase(tt.input)
		if result != tt.expected {
			t.Errorf("toUpperCamelCase(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGenerateAvoidsNameCollisions(t *testing.T) {
	info := &StructInfo{
		Name:        "TestStruct",
		PackageName: "test",
		Fields: []FieldInfo{
			{Name: "Name", Type: "string"},
			{Name: "name", Type: "string"},
		},
	}
	config := &GeneratorConfig{
		ConstructorTypes: []string{"allArgs", "builder", "options"},
		WithGetter:       true,
	}

	code, err := NewGenerator(config, info).Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	for _, want := range []string{
		"func NewTestStruct(name string, name2 string)",
		"func (b *TestStructBuilder) Name2(name2 string)",
		"func WithName2(name2 string) TestStructOption",
		"func (t *TestStruct) GetName2() string",
	} {
		if !strings.Contains(code, want) {
			t.Errorf("generated code missing %q", want)
		}
	}
}

func TestGenerateIncludesOnlyUsedImports(t *testing.T) {
	info := &StructInfo{
		Name:        "TestStruct",
		PackageName: "test",
		Imports: []ImportInfo{
			{Path: "time"},
			{Path: "fmt"},
			{Path: "gopkg.in/yaml.v3"},
		},
		Fields: []FieldInfo{
			{Name: "timeout", Type: "time.Duration"},
			{Name: "document", Type: "yaml.Node"},
		},
	}

	code, err := NewGenerator(&GeneratorConfig{
		ConstructorTypes: []string{"allArgs"},
	}, info).Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(code, `"time"`) {
		t.Error("generated code should import time")
	}
	if !strings.Contains(code, `"gopkg.in/yaml.v3"`) {
		t.Error("generated code should import versioned yaml package")
	}
	if strings.Contains(code, `"fmt"`) {
		t.Error("generated code should not import unused fmt")
	}
}

func TestGenerateGenericStruct(t *testing.T) {
	info := &StructInfo{
		Name:        "Box",
		PackageName: "test",
		Imports:     []ImportInfo{{Path: "io"}},
		TypeParams:  "[T io.Reader, U comparable]",
		TypeArgs:    "[T, U]",
		Fields: []FieldInfo{
			{Name: "value", Type: "T"},
			{Name: "other", Type: "U"},
		},
	}

	code, err := NewGenerator(&GeneratorConfig{
		ConstructorTypes: []string{"allArgs", "builder", "options"},
		WithGetter:       true,
	}, info).Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if !strings.Contains(code, `"io"`) {
		t.Error("generated generic code should import io for its type constraint")
	}

	for _, want := range []string{
		"func NewBox[T io.Reader, U comparable](value T, other U) *Box[T, U]",
		"type BoxBuilder[T io.Reader, U comparable] struct",
		"func NewBoxBuilder[T io.Reader, U comparable]() *BoxBuilder[T, U]",
		"func (b *BoxBuilder[T, U]) Build() *Box[T, U]",
		"type BoxOption[T io.Reader, U comparable] func(*Box[T, U])",
		"func WithValue[T io.Reader, U comparable](value T) BoxOption[T, U]",
		"func NewBoxWithOptions[T io.Reader, U comparable](opts ...BoxOption[T, U]) *Box[T, U]",
		"func (b *Box[T, U]) GetValue() T",
	} {
		if !strings.Contains(code, want) {
			t.Errorf("generated generic code missing %q", want)
		}
	}
}

func TestGeneratedGenericCodeTypeChecks(t *testing.T) {
	info := &StructInfo{
		Name:        "Box",
		PackageName: "test",
		Imports:     []ImportInfo{{Path: "io"}},
		TypeParams:  "[T io.Reader, U comparable]",
		TypeArgs:    "[T, U]",
		Fields: []FieldInfo{
			{Name: "value", Type: "T"},
			{Name: "other", Type: "U"},
		},
	}
	code, err := NewGenerator(&GeneratorConfig{
		ConstructorTypes: []string{"allArgs", "builder", "options"},
		WithGetter:       true,
	}, info).Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	fset := token.NewFileSet()
	source, err := goparser.ParseFile(fset, "box.go", []byte(`package test

import "io"

type Box[T io.Reader, U comparable] struct {
	value T
	other U
}

func use(r io.Reader) {
	_ = NewBox(r, 1)
	_ = NewBoxBuilder[io.Reader, int]().Value(r).Other(1).Build()
	_ = NewBoxWithOptions[io.Reader, int](WithValue[io.Reader, int](r), WithOther[io.Reader, int](1))
}
`), 0)
	if err != nil {
		t.Fatal(err)
	}
	generated, err := goparser.ParseFile(fset, "box_gen.go", []byte(code), 0)
	if err != nil {
		t.Fatal(err)
	}

	_, err = (&types.Config{Importer: importer.Default()}).Check("test", fset, []*ast.File{source, generated}, nil)
	if err != nil {
		t.Fatalf("generated generic code does not type-check: %v", err)
	}
}

func TestSkipFieldsInGeneration(t *testing.T) {
	info := &StructInfo{
		Name:        "TestStruct",
		PackageName: "test",
		Fields: []FieldInfo{
			{Name: "name", Type: "string", Skip: false},
			{Name: "internal", Type: "string", Skip: true},
		},
	}

	config := &GeneratorConfig{
		StructName:       "TestStruct",
		ConstructorTypes: []string{"allArgs"},
	}

	gen := NewGenerator(config, info)
	code, err := gen.Generate()

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check skipped field is not in constructor
	if strings.Contains(code, "internal string") {
		t.Error("Generated code should not contain skipped field 'internal'")
	}
}
