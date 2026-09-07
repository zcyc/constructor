package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	if err := os.WriteFile(filepath.Join(dir, "existing.go"), []byte("package test\n\ntype NewOther struct{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateGeneratedDeclarations(sourceFile, outputFile, `package test

func NewOther() *Box { return &Box{} }
`); err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("expected cross-kind declaration conflict, got %v", err)
	}
}

func TestValidateGeneratedDeclarationsAllowsBlankIdentifiers(t *testing.T) {
	dir := t.TempDir()
	sourceFile := filepath.Join(dir, "box.go")
	if err := os.WriteFile(sourceFile, []byte("package test\n\ntype Box struct{}\n\nvar _ = 1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "other.go"), []byte("package test\n\nvar _ = 2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateGeneratedDeclarations(sourceFile, filepath.Join(dir, "box_gen.go"), "package test\n\nfunc NewBox() *Box { return &Box{} }\n"); err != nil {
		t.Fatalf("repeated blank identifiers were rejected: %v", err)
	}
}

func TestValidateGeneratedDeclarationsAllowsMultipleInitFunctions(t *testing.T) {
	dir := t.TempDir()
	sourceFile := filepath.Join(dir, "box.go")
	if err := os.WriteFile(sourceFile, []byte("package test\n\ntype Box struct{}\n\nfunc init() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "other.go"), []byte("package test\n\nfunc init() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateGeneratedDeclarations(sourceFile, filepath.Join(dir, "box_gen.go"), "package test\n\nfunc NewBox() *Box { return &Box{} }\n"); err != nil {
		t.Fatalf("multiple init functions were rejected: %v", err)
	}
}

func TestValidateGeneratedDeclarationsHonorsGeneratedBuildConstraints(t *testing.T) {
	dir := t.TempDir()
	sourceFile := filepath.Join(dir, "box_linux.go")
	if err := os.WriteFile(sourceFile, []byte("package test\n\ntype Box struct{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "existing.go"), []byte("package test\n\ntype NewBox struct{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	generated := "//go:build linux\n\npackage test\n\nfunc NewBox() *Box { return &Box{} }\n"

	if err := validateGeneratedDeclarations(sourceFile, filepath.Join(dir, "box_gen.go"), generated); err != nil {
		t.Fatalf("excluded generated declarations were rejected: %v", err)
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
	restored, err = os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(restored), "Missing: true") {
		t.Fatalf("existing output was not restored after failed validation: %q", restored)
	}
}

func TestValidateGeneratedPackageCanRunConcurrently(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/test\n\ngo 1.24\n")
	write("box.go", "package test\n\ntype Box struct{}\n")

	var waitGroup sync.WaitGroup
	errors := make(chan error, 2)
	for range 2 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			errors <- validateGeneratedPackage(filepath.Join(dir, "box.go"), filepath.Join(dir, "box_gen.go"), "package test\n\nfunc NewBox() *Box { return &Box{} }\n")
		}()
	}
	waitGroup.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatalf("concurrent validation failed: %v", err)
		}
	}
}

func TestValidateGeneratedPackageWithExternalTestPackage(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/test\n\ngo 1.24\n")
	write("box.go", "package test\n\ntype Box struct{}\n")
	write("box_test.go", `package test_test

import (
	"testing"
	test "example.com/test"
)

func TestBox(t *testing.T) { _ = test.NewBox() }
`)

	if err := validateGeneratedPackage(filepath.Join(dir, "box.go"), filepath.Join(dir, "box_gen.go"), "package test\n\nfunc NewBox() *Box { return &Box{} }\n"); err != nil {
		t.Fatalf("external test package was rejected: %v", err)
	}
}

func TestValidateGeneratedPackageRemovesStaleCandidates(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/test\n\ngo 1.24\n")
	write("box.go", "package test\n\ntype Box struct{}\n")
	stale := filepath.Join(dir, "constructor-validate-stale.go")
	write("constructor-validate-stale.go", "package test\n\n// Code generated by constructor. DO NOT EDIT.\n\nfunc NewBox() *Box { return &Box{} }\n")

	if err := validateGeneratedPackage(filepath.Join(dir, "box.go"), filepath.Join(dir, "box_gen.go"), "package test\n\nfunc NewBox() *Box { return &Box{} }\n"); err != nil {
		t.Fatalf("stale candidate was not ignored: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale candidate still exists, stat error = %v", err)
	}
}

func TestValidateGeneratedPackageRemovesMalformedStaleCandidate(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/test\n\ngo 1.24\n")
	write("box.go", "package test\n\ntype Box struct{}\n")
	stale := filepath.Join(dir, "constructor-validate-stale.go")
	write("constructor-validate-stale.go", "package test\n\n// Code generated by constructor. DO NOT EDIT.\n\nfunc NewBox(\n")

	if err := validateGeneratedPackage(filepath.Join(dir, "box.go"), filepath.Join(dir, "box_gen.go"), "package test\n\nfunc NewBox() *Box { return &Box{} }\n"); err != nil {
		t.Fatalf("malformed stale candidate was not ignored: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("malformed stale candidate still exists, stat error = %v", err)
	}
}

func TestValidateGeneratedPackagePreservesLookalikeSource(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/test\n\ngo 1.24\n")
	write("box.go", "package test\n\ntype Box struct{}\n")
	userFile := filepath.Join(dir, "constructor-validate-user.go")
	userSource := "package test\n\nconst marker = \"// Code generated by constructor. DO NOT EDIT.\"\n\ntype User struct{}\n"
	write("constructor-validate-user.go", userSource)

	if err := validateGeneratedPackage(filepath.Join(dir, "box.go"), filepath.Join(dir, "box_gen.go"), "package test\n\nfunc NewBox() *Box { return &Box{} }\n"); err != nil {
		t.Fatalf("validation failed: %v", err)
	}
	content, err := os.ReadFile(userFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != userSource {
		t.Fatalf("lookalike source was changed: %q", content)
	}
}
