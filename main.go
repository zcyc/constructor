package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"
)

var version = "1.0.0"

func main() {
	// Define flags
	var (
		typeName         = flag.String("type", "", "[mandatory] The struct type name to generate constructor for")
		inputFile        = flag.String("input", "", "[optional] Go source file containing the struct")
		constructorTypes = flag.String("constructor-types", "all-args", "[optional] Comma-separated constructor patterns: all-args,builder,options")
		outputFile       = flag.String("output", "", "[optional] Output file path (default: <source_dir>/<type>_gen.go)")
		initFunc         = flag.String("init", "", "[optional] Name of initialization method to call after construction")
		initReturnsError = flag.Bool("init-returns-error", false, "[optional] Return initialization errors from constructors")
		returnValue      = flag.Bool("return-value", false, "[optional] Return value instead of pointer")
		setterPrefix     = flag.String("setter-prefix", "", "[optional] Prefix for builder setter methods in builder pattern (e.g., 'With')")
		withGetter       = flag.Bool("getters", false, "[optional] Generate getter methods for private fields")
		dryRun           = flag.Bool("dry-run", false, "[optional] Validate generated code without writing it")
		stdoutOutput     = flag.Bool("stdout", false, "[optional] Write generated code to stdout instead of a file")
		validateOnly     = flag.Bool("validate-only", false, "[optional] Validate the existing output file without generating")
		showVersion      = flag.Bool("version", false, "[optional] Show version information")
	)

	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("constructor version %s\n", version)
		os.Exit(0)
	}

	// Validate required flags
	if *typeName == "" {
		fmt.Fprintf(os.Stderr, "Error: -type flag is mandatory\n\n")
		flag.Usage()
		os.Exit(1)
	}
	if *initReturnsError && *initFunc == "" {
		fmt.Fprintf(os.Stderr, "Error: -init-returns-error requires -init\n")
		os.Exit(1)
	}
	if *validateOnly && (*dryRun || *stdoutOutput) {
		fmt.Fprintf(os.Stderr, "Error: -validate-only cannot be combined with -dry-run or -stdout\n")
		os.Exit(1)
	}

	// Find the source file containing the struct
	sourceFile, err := findSourceFile(*typeName, *inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Parse the struct
	structInfo, err := ParseStruct(sourceFile, *typeName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing struct: %v\n", err)
		os.Exit(1)
	}

	// Determine output file
	output := *outputFile
	if output == "" {
		dir := filepath.Dir(sourceFile)
		output = filepath.Join(dir, strings.ToLower(*typeName)+"_gen.go")
	}
	if sameFile(sourceFile, output) {
		fmt.Fprintf(os.Stderr, "Error: output file must not overwrite the source file %s\n", sourceFile)
		os.Exit(1)
	}
	if *validateOnly {
		code, err := os.ReadFile(output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading output file: %v\n", err)
			os.Exit(1)
		}
		if err := validateGeneratedDeclarations(sourceFile, output, string(code)); err != nil {
			fmt.Fprintf(os.Stderr, "Error validating generated code: %v\n", err)
			os.Exit(1)
		}
		if err := validateGeneratedPackage(sourceFile, output, string(code)); err != nil {
			fmt.Fprintf(os.Stderr, "Error type-checking generated package: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Validation passed: %s\n", output)
		return
	}

	// Parse constructor types
	types := strings.Split(*constructorTypes, ",")
	for i, t := range types {
		types[i] = strings.TrimSpace(t)
		if types[i] == "all-args" {
			types[i] = "allArgs"
		}
	}

	// Validate constructor types
	seenTypes := make(map[string]struct{}, len(types))
	for _, t := range types {
		if t != "allArgs" && t != "builder" && t != "options" {
			fmt.Fprintf(os.Stderr, "Error: invalid constructor pattern '%s'. Valid patterns: all-args, builder, options\n", t)
			os.Exit(1)
		}
		if _, seen := seenTypes[t]; seen {
			fmt.Fprintf(os.Stderr, "Error: constructor pattern '%s' was specified more than once\n", t)
			os.Exit(1)
		}
		seenTypes[t] = struct{}{}
	}

	// Create generator config
	config := &GeneratorConfig{
		StructName:       *typeName,
		ConstructorTypes: types,
		OutputFile:       output,
		InitFunc:         *initFunc,
		InitReturnsError: *initReturnsError,
		ReturnValue:      *returnValue,
		SetterPrefix:     *setterPrefix,
		WithGetter:       *withGetter,
	}

	// Generate code
	generator := NewGenerator(config, structInfo)
	code, err := generator.Generate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating code: %v\n", err)
		os.Exit(1)
	}
	if err := validateGeneratedDeclarations(sourceFile, output, code); err != nil {
		fmt.Fprintf(os.Stderr, "Error validating generated code: %v\n", err)
		os.Exit(1)
	}
	if err := validateGeneratedPackage(sourceFile, output, code); err != nil {
		fmt.Fprintf(os.Stderr, "Error type-checking generated package: %v\n", err)
		os.Exit(1)
	}
	if *stdoutOutput {
		if _, err := os.Stdout.WriteString(code); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing generated code to stdout: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if *dryRun {
		fmt.Fprintf(os.Stderr, "Validation passed; no file written: %s\n", output)
		return
	}

	// Write to file
	if err := writeGeneratedFile(output, code); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated constructor code in %s\n", output)
}

func sameFile(first, second string) bool {
	first, firstErr := filepath.Abs(first)
	second, secondErr := filepath.Abs(second)
	if firstErr != nil || secondErr != nil {
		return false
	}
	if filepath.Clean(first) == filepath.Clean(second) {
		return true
	}

	firstInfo, firstErr := os.Stat(first)
	secondInfo, secondErr := os.Stat(second)
	return firstErr == nil && secondErr == nil && os.SameFile(firstInfo, secondInfo)
}

func writeGeneratedFile(filename, code string) error {
	dir := filepath.Dir(filename)
	mode := os.FileMode(0644)
	if info, err := os.Stat(filename); err == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect existing output: %w", err)
	}

	temporary, err := os.CreateTemp(dir, "."+filepath.Base(filename)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary output: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)

	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set temporary output permissions: %w", err)
	}
	if _, err := temporary.WriteString(code); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary output: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary output: %w", err)
	}
	if err := os.Rename(temporaryName, filename); err != nil {
		return fmt.Errorf("replace output: %w", err)
	}
	return nil
}

// findSourceFile searches for a Go file containing the struct definition
func findSourceFile(typeName, inputFile string) (string, error) {
	if inputFile != "" {
		return filepath.Clean(inputFile), nil
	}

	// First, check if GOFILE environment variable is set (set by go generate)
	if gofile := os.Getenv("GOFILE"); gofile != "" {
		return gofile, nil
	}

	// Otherwise, search current directory for .go files
	files, err := filepath.Glob("*.go")
	if err != nil {
		return "", fmt.Errorf("failed to list Go files: %w", err)
	}

	// Try to find the struct in each file
	for _, file := range files {
		// Skip generated files
		if strings.HasSuffix(file, "_gen.go") || strings.HasSuffix(file, "_test.go") {
			continue
		}
		matched, err := build.Default.MatchFile(".", file)
		if err != nil {
			return "", fmt.Errorf("failed to match build constraints for %s: %w", file, err)
		}
		if !matched {
			continue
		}

		// Try to parse the file
		_, parseErr := ParseStruct(file, typeName)
		if parseErr == nil {
			return file, nil
		}
	}

	return "", fmt.Errorf("could not find struct %s in current directory", typeName)
}
