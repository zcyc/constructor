package main

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// validateGeneratedPackage compiles the generated package and its tests without
// running them, catching unresolved imports, type errors, and stale test APIs.
func validateGeneratedPackage(sourceFile, outputFile, generated string) error {
	dir, err := filepath.Abs(filepath.Dir(sourceFile))
	if err != nil {
		return fmt.Errorf("resolve package directory: %w", err)
	}

	candidate, err := os.CreateTemp(dir, "constructor-validate-*.go")
	if err != nil {
		return fmt.Errorf("create validation file: %w", err)
	}
	candidatePath := candidate.Name()
	defer os.Remove(candidatePath)
	if _, err := candidate.WriteString(generated); err != nil {
		candidate.Close()
		return fmt.Errorf("write validation file: %w", err)
	}
	if err := candidate.Close(); err != nil {
		return fmt.Errorf("close validation file: %w", err)
	}

	outputPath, err := filepath.Abs(outputFile)
	if err != nil {
		return fmt.Errorf("resolve output file: %w", err)
	}
	sourceFiles, err := packageSourceFiles(dir, outputPath)
	if err != nil {
		return err
	}
	sourceFiles = append(sourceFiles, candidatePath)
	testFiles, err := packageTestFiles(dir)
	if err != nil {
		return err
	}

	testBinary, err := os.CreateTemp("", "constructor-test-*")
	if err != nil {
		return fmt.Errorf("create test binary: %w", err)
	}
	testBinaryPath := testBinary.Name()
	if err := testBinary.Close(); err != nil {
		_ = os.Remove(testBinaryPath)
		return fmt.Errorf("close test binary: %w", err)
	}
	defer os.Remove(testBinaryPath)

	args := []string{"test", "-c", "-o", testBinaryPath}
	args = append(args, sourceFiles...)
	args = append(args, testFiles...)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "go", args...)
	command.Dir = dir
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("module type check timed out: %w", ctx.Err())
		}
		if message := strings.TrimSpace(stderr.String()); message != "" {
			return fmt.Errorf("module type check failed: %s", message)
		}
		return fmt.Errorf("module type check failed: %w", err)
	}
	return nil
}

// validateGeneratedDeclarations prevents generated top-level declarations and
// methods from silently colliding with code already in the package.
func validateGeneratedDeclarations(sourceFile, outputFile, generated string) error {
	fset := token.NewFileSet()
	generatedFile, err := parser.ParseFile(fset, outputFile, []byte(generated), 0)
	if err != nil {
		return fmt.Errorf("parse generated file: %w", err)
	}

	decls := map[string]string{}
	if err := collectDeclarations(decls, generatedFile, outputFile); err != nil {
		return err
	}

	dir := filepath.Dir(sourceFile)
	files, err := packageSourceFiles(dir, outputFile)
	if err != nil {
		return err
	}
	for _, filename := range files {
		file, err := parser.ParseFile(fset, filename, nil, 0)
		if err != nil {
			return fmt.Errorf("parse package file %s: %w", filename, err)
		}
		if err := collectDeclarations(decls, file, filename); err != nil {
			return err
		}
	}
	return nil
}

func packageSourceFiles(dir, outputFile string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read package directory: %w", err)
	}

	outputFile, err = filepath.Abs(outputFile)
	if err != nil {
		return nil, fmt.Errorf("resolve output file: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		// Validation candidates are passed explicitly to go test and must not
		// be mistaken for package sources during a concurrent validation.
		if strings.HasPrefix(name, "constructor-validate-") {
			continue
		}

		filename := filepath.Join(dir, name)
		absolute, err := filepath.Abs(filename)
		if err != nil {
			return nil, fmt.Errorf("resolve package file %s: %w", filename, err)
		}
		if absolute == outputFile {
			continue
		}

		matched, err := build.Default.MatchFile(dir, name)
		if err != nil {
			return nil, fmt.Errorf("match build constraints for %s: %w", filename, err)
		}
		if matched {
			files = append(files, filename)
		}
	}

	sort.Strings(files)
	return files, nil
}

func packageTestFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read package directory: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || !strings.HasSuffix(name, "_test.go") {
			continue
		}

		filename := filepath.Join(dir, name)
		matched, err := build.Default.MatchFile(dir, name)
		if err != nil {
			return nil, fmt.Errorf("match build constraints for %s: %w", filename, err)
		}
		if matched {
			files = append(files, filename)
		}
	}

	sort.Strings(files)
	return files, nil
}

func collectDeclarations(decls map[string]string, file *ast.File, filename string) error {
	for _, declaration := range file.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			key := "func:" + declaration.Name.Name
			if declaration.Recv != nil {
				key = "method:" + receiverTypeName(declaration.Recv) + ":" + declaration.Name.Name
			}
			if err := rememberDeclaration(decls, key, filename); err != nil {
				return err
			}
		case *ast.GenDecl:
			for _, specification := range declaration.Specs {
				switch specification := specification.(type) {
				case *ast.TypeSpec:
					if err := rememberDeclaration(decls, "type:"+specification.Name.Name, filename); err != nil {
						return err
					}
				case *ast.ValueSpec:
					for _, name := range specification.Names {
						if err := rememberDeclaration(decls, declaration.Tok.String()+":"+name.Name, filename); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

func rememberDeclaration(decls map[string]string, key, filename string) error {
	if previous, exists := decls[key]; exists {
		return fmt.Errorf("declaration %q conflicts with %s (also declared in %s)", key, previous, filename)
	}
	decls[key] = filename
	return nil
}

func receiverTypeName(fields *ast.FieldList) string {
	if fields == nil || len(fields.List) == 0 {
		return ""
	}
	return embeddedFieldName(fields.List[0].Type)
}
