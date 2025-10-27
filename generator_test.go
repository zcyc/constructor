package main

import (
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
		{"", ""},
	}

	for _, tt := range tests {
		result := toUpperCamelCase(tt.input)
		if result != tt.expected {
			t.Errorf("toUpperCamelCase(%q) = %q, want %q", tt.input, result, tt.expected)
		}
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
