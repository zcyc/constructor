package main

import (
	"os"
	"path/filepath"
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

func TestShouldSkipField(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected bool
	}{
		{"no tag", "", false},
		{"newc skip", "`newc:\"-\"`", true},
		{"gonstructor skip", "`gonstructor:\"-\"`", true},
		{"constructor skip", "`constructor:\"-\"`", true},
		{"other tag", "`json:\"name\"`", false},
		{"newc with value", "`newc:\"value\"`", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSkipField(tt.tag)
			if result != tt.expected {
				t.Errorf("shouldSkipField(%q) = %v, want %v", tt.tag, result, tt.expected)
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
		{"newc skip", "`newc:\"-\"`", true, false, false},
		{"gonstructor skip", "`gonstructor:\"-\"`", true, false, false},
		{"skip getter", "`constructor:\"getter:false\"`", false, true, false},
		{"skip setter", "`constructor:\"setter:false\"`", false, false, true},
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

type ComplexStruct struct {
	str        string
	ptr        *int
	slice      []string
	array      [5]int
	mapField   map[string]int
	channel    chan int
	timeField  time.Time
	iface      interface{}
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
