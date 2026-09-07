package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateGeneratedDeclarations(t *testing.T) {
	dir := t.TempDir()
	sourceFile := filepath.Join(dir, "box.go")
	outputFile := filepath.Join(dir, "box_gen.go")
	source := `package test

type Box struct{}
`
	if err := os.WriteFile(sourceFile, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	if err := validateGeneratedDeclarations(sourceFile, outputFile, `package test

func NewBox() *Box { return &Box{} }
`); err != nil {
		t.Fatalf("valid declarations rejected: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "existing.go"), []byte(`package test

func NewBox() *Box { return &Box{} }
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateGeneratedDeclarations(sourceFile, outputFile, `package test

func NewBox() *Box { return &Box{} }
`); err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("expected declaration conflict, got %v", err)
	}
}

func TestValidateGeneratedPackage(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/test\n\ngo 1.24\n")
	write("box.go", `package test

type Box struct { Name string }
`)
	write("box_test.go", `package test

import "testing"

func TestGeneratedConstructor(t *testing.T) {
	_ = NewBox()
}
`)

	sourceFile := filepath.Join(dir, "box.go")
	outputFile := filepath.Join(dir, "box_gen.go")
	if err := validateGeneratedPackage(sourceFile, outputFile, `package test

func NewBox() *Box { return &Box{} }
`); err != nil {
		t.Fatalf("valid package and tests rejected: %v", err)
	}

	if err := validateGeneratedPackage(sourceFile, outputFile, `package test

func NewBox() *Box { return &Box{Missing: true} }
`); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected package type error, got %v", err)
	}

	write("box_gen.go", `package test

func NewBox() *Box { return &Box{Missing: true} }
`)
	if err := validateGeneratedPackage(sourceFile, outputFile, `package test

func NewBox() *Box { return &Box{} }
`); err != nil {
		t.Fatalf("existing output was not replaced by candidate: %v", err)
	}
	restored, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(restored), "Missing: true") {
		t.Fatalf("existing output was not restored after validation: %q", restored)
	}

	if err := validateGeneratedPackage(sourceFile, outputFile, `package test

func NewBox() (Box, error) { return Box{}, nil }
`); err == nil || !strings.Contains(err.Error(), "assignment mismatch") {
		t.Fatalf("test API mismatch was not detected: %v", err)
	}
}
