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
