package main

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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
