package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseStruct(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	content := `package test

type TestStruct struct {
	name string
	age  int
	email string
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Parse the struct
	info, err := ParseStruct(testFile, "TestStruct")
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}

	// Verify results
	if info.Name != "TestStruct" {
		t.Errorf("Expected name 'TestStruct', got '%s'", info.Name)
	}

	if info.PackageName != "test" {
		t.Errorf("Expected package 'test', got '%s'", info.PackageName)
	}

	if len(info.Fields) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(info.Fields))
	}

	// Check first field
	if info.Fields[0].Name != "name" {
		t.Errorf("Expected field name 'name', got '%s'", info.Fields[0].Name)
	}
	if info.Fields[0].Type != "string" {
		t.Errorf("Expected field type 'string', got '%s'", info.Fields[0].Type)
	}
}

func TestParseStructNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	content := `package test

type OtherStruct struct {
	name string
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Try to parse non-existent struct
	_, err := ParseStruct(testFile, "NonExistent")
	if err == nil {
		t.Error("Expected error when struct not found")
	}
}

func TestParseStructIgnoresLocalTypes(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package test

func define() {
	type Target struct { value string }
	_ = Target{}
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseStruct(testFile, "Target"); err == nil {
		t.Fatal("local types must not be selected as generation targets")
	}
}

func TestParseStructResolvesImportPackageName(t *testing.T) {
	tmpDir := t.TempDir()
	externalDir := filepath.Join(tmpDir, "external")
	if err := os.Mkdir(externalDir, 0755); err != nil {
		t.Fatal(err)
	}
	write := func(filename, content string) {
		t.Helper()
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(tmpDir, "go.mod"), "module example.com/test\n\ngo 1.24\n")
	write(filepath.Join(externalDir, "external.go"), "package actual\n\ntype Value struct{}\n")
	testFile := filepath.Join(tmpDir, "test.go")
	write(testFile, "package test\n\nimport \"example.com/test/external\"\n\ntype Box struct { value actual.Value }\n")

	info, err := ParseStruct(testFile, "Box")
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}
	if len(info.Imports) != 1 || info.Imports[0].Qualifier != "actual" {
		t.Fatalf("resolved imports = %#v, want qualifier actual", info.Imports)
	}

	code, err := NewGenerator(&GeneratorConfig{ConstructorTypes: []string{"allArgs"}}, info).Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if !strings.Contains(code, "actual.Value") || !strings.Contains(code, `"example.com/test/external"`) {
		t.Fatalf("generated code did not preserve the package name:\n%s", code)
	}
}

func TestParseStructTracksDotImportsUsedByGeneratedCode(t *testing.T) {
	tmpDir := t.TempDir()
	externalDir := filepath.Join(tmpDir, "external")
	if err := os.Mkdir(externalDir, 0755); err != nil {
		t.Fatal(err)
	}
	write := func(filename, content string) {
		t.Helper()
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(tmpDir, "go.mod"), "module example.com/test\n\ngo 1.24\n")
	write(filepath.Join(externalDir, "external.go"), "package actual\n\ntype Value struct{}\n")
	testFile := filepath.Join(tmpDir, "test.go")
	write(testFile, "package test\n\nimport . \"example.com/test/external\"\n\ntype Box struct { value Value }\n")

	info, err := ParseStruct(testFile, "Box")
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}
	if len(info.Imports) != 1 || !info.Imports[0].Used {
		t.Fatalf("dot import usage = %#v, want used", info.Imports)
	}
	code, err := NewGenerator(&GeneratorConfig{ConstructorTypes: []string{"allArgs"}}, info).Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if !strings.Contains(code, `import . "example.com/test/external"`) {
		t.Fatalf("generated code omitted used dot import:\n%s", code)
	}

	unusedFile := filepath.Join(tmpDir, "unused.go")
	write(unusedFile, "package test\n\nimport . \"example.com/test/external\"\n\nvar _ = Value{}\n\ntype Other struct { value int }\n")
	info, err = ParseStruct(unusedFile, "Other")
	if err != nil {
		t.Fatalf("ParseStruct for unused dot import failed: %v", err)
	}
	if len(info.Imports) != 1 || info.Imports[0].Used {
		t.Fatalf("unused dot import usage = %#v, want unused", info.Imports)
	}
	code, err = NewGenerator(&GeneratorConfig{ConstructorTypes: []string{"allArgs"}}, info).Generate()
	if err != nil {
		t.Fatalf("Generate for unused dot import failed: %v", err)
	}
	if strings.Contains(code, "example.com/test/external") {
		t.Fatalf("generated code retained unused dot import:\n%s", code)
	}
}

func TestParseStructIgnoresDotImportNamesThatAreNotReferences(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	content := "package test\n\nimport . \"fmt\"\n\nvar _ = Println\n\ntype Box struct { Println int }\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := ParseStruct(testFile, "Box")
	if err != nil {
		t.Fatal(err)
	}
	code, err := NewGenerator(&GeneratorConfig{ConstructorTypes: []string{"allArgs"}}, info).Generate()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(code, `"fmt"`) {
		t.Fatalf("generated code retained an unused dot import:\n%s", code)
	}
}

func TestParseFieldSkipTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected bool
	}{
		{"no tag", "", false},
		{"constructor skip", "`constructor:\"-\"`", true},
		{"legacy newc tag ignored", "`newc:\"-\"`", false},
		{"legacy gonstructor tag ignored", "`gonstructor:\"-\"`", false},
		{"other tag", "`json:\"name\"`", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip, _, _ := parseFieldSkipTags(tt.tag)
			if skip != tt.expected {
				t.Errorf("parseFieldSkipTags(%q) skip = %v, want %v", tt.tag, skip, tt.expected)
			}
		})
	}
}

func TestParseFieldSkipTags(t *testing.T) {
	tests := []struct {
		name         string
		tag          string
		expectSkip   bool
		expectGetter bool
		expectSetter bool
	}{
		{"no tag", "", false, false, false},
		{"constructor skip", "`constructor:\"-\"`", true, false, false},
		{"skip getter", "`constructor:\"getter:false\"`", false, true, false},
		{"skip setter", "`constructor:\"setter:false\"`", false, false, true},
		{"skip getter and setter", "`constructor:\"getter:false, setter:false\"`", false, true, true},
		{"other tag", "`json:\"name\"`", false, false, false},
		{"mixed tags", "`json:\"name\" constructor:\"getter:false\"`", false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip, skipGetter, skipSetter := parseFieldSkipTags(tt.tag)
			if skip != tt.expectSkip {
				t.Errorf("parseFieldSkipTags(%q) skip = %v, want %v", tt.tag, skip, tt.expectSkip)
			}
			if skipGetter != tt.expectGetter {
				t.Errorf("parseFieldSkipTags(%q) skipGetter = %v, want %v", tt.tag, skipGetter, tt.expectGetter)
			}
			if skipSetter != tt.expectSetter {
				t.Errorf("parseFieldSkipTags(%q) skipSetter = %v, want %v", tt.tag, skipSetter, tt.expectSetter)
			}
		})
	}
}

func TestParseConstructorFieldOptions(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	content := "package test\n\ntype Config struct {\n\tname string `constructor:\"required\"`\n\tport int `constructor:\"default=8080\"`\n\tvalues []int `constructor:\"default=[]int{1,2}\"`\n}\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := ParseStruct(testFile, "Config")
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}
	if !info.Fields[0].Required || info.Fields[0].Default != "" {
		t.Fatalf("required field parsed incorrectly: %+v", info.Fields[0])
	}
	if info.Fields[1].Default != "8080" || info.Fields[1].Required {
		t.Fatalf("default field parsed incorrectly: %+v", info.Fields[1])
	}
	if info.Fields[2].Default != "[]int{1,2}" {
		t.Fatalf("comma-containing default parsed incorrectly: %+v", info.Fields[2])
	}
}

func TestStructInfoGetFieldsForConstructor(t *testing.T) {
	info := &StructInfo{
		Name: "Test",
		Fields: []FieldInfo{
			{Name: "field1", Skip: false, SkipSetter: false},
			{Name: "field2", Skip: true, SkipSetter: false},
			{Name: "field3", Skip: false, SkipSetter: false},
			{Name: "field4", Skip: false, SkipSetter: true}, // Should be excluded
		},
	}

	fields := info.GetFieldsForConstructor()

	if len(fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(fields))
	}

	if fields[0].Name != "field1" || fields[1].Name != "field3" {
		t.Error("GetFieldsForConstructor returned wrong fields")
	}
}

func TestStructInfoGetFieldsForGetter(t *testing.T) {
	info := &StructInfo{
		Name: "Test",
		Fields: []FieldInfo{
			{Name: "field1", Skip: false, SkipGetter: false, Exported: false},
			{Name: "field2", Skip: true, SkipGetter: false, Exported: false}, // Skipped
			{Name: "field3", Skip: false, SkipGetter: true, Exported: false}, // Skip getter
			{Name: "Field4", Skip: false, SkipGetter: false, Exported: true}, // Exported, no getter
			{Name: "field5", Skip: false, SkipGetter: false, Exported: false},
		},
	}

	fields := info.GetFieldsForGetter()

	if len(fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(fields))
	}

	if fields[0].Name != "field1" || fields[1].Name != "field5" {
		t.Error("GetFieldsForGetter returned wrong fields")
	}
}

func TestExprToString(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	content := `package test

import "time"

type Base struct{}

type ComplexStruct struct {
	str        string
	ptr        *int
	slice      []string
	array      [5]int
	mapField   map[string]int
	channel    chan int
	timeField  time.Time
	iface      interface{}
	handler    func(string) error
	generic    Box[int]
	grouped    (map[string]int)
	*Base
	_          struct{}
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := ParseStruct(testFile, "ComplexStruct")
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}

	// Verify complex types are parsed correctly
	expectedTypes := map[string]string{
		"str":       "string",
		"ptr":       "*int",
		"slice":     "[]string",
		"array":     "[5]int",
		"mapField":  "map[string]int",
		"channel":   "chan int",
		"timeField": "time.Time",
		"iface":     "interface{}",
		"handler":   "func(string) error",
		"generic":   "Box[int]",
		"grouped":   "(map[string]int)",
		"Base":      "*Base",
	}

	if len(info.Fields) != len(expectedTypes) {
		t.Fatalf("expected %d usable fields, got %d", len(expectedTypes), len(info.Fields))
	}

	for _, field := range info.Fields {
		expected, ok := expectedTypes[field.Name]
		if !ok {
			continue
		}
		if field.Type != expected {
			t.Errorf("Field %s: expected type %s, got %s", field.Name, expected, field.Type)
		}
	}
}

func TestParseEmbeddedFieldName(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	content := `package test

type Base struct{}

type Container struct {
	*Base
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := ParseStruct(testFile, "Container")
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}

	if len(info.Fields) != 1 || info.Fields[0].Name != "Base" {
		t.Fatalf("embedded field = %#v, want field named Base", info.Fields)
	}
}

func TestParseGenericStruct(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	content := `package test

import "io"

type Box[T io.Reader, U comparable] struct {
	value T
	other U
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := ParseStruct(testFile, "Box")
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}

	if info.TypeParams != "[T io.Reader, U comparable]" {
		t.Fatalf("TypeParams = %q", info.TypeParams)
	}
	if info.TypeArgs != "[T, U]" {
		t.Fatalf("TypeArgs = %q", info.TypeArgs)
	}
}
