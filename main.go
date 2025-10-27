package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const version = "1.0.0"

func main() {
	// Define flags
	var (
		typeName         = flag.String("type", "", "[mandatory] The struct type name to generate constructor for")
		constructorTypes = flag.String("constructorTypes", "allArgs", "[optional] Comma-separated list of constructor types: allArgs,builder,options")
		outputFile       = flag.String("output", "", "[optional] Output file path (default: <source_dir>/<type>_gen.go)")
		initFunc         = flag.String("init", "", "[optional] Name of initialization method to call after construction")
		returnValue      = flag.Bool("returnValue", false, "[optional] Return value instead of pointer")
		setterPrefix     = flag.String("setterPrefix", "", "[optional] Prefix for setter methods in builder pattern (e.g., 'With')")
		withGetter       = flag.Bool("withGetter", false, "[optional] Generate getter methods for private fields")
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

	// Find the source file containing the struct
	sourceFile, err := findSourceFile(*typeName)
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

	// Parse constructor types
	types := strings.Split(*constructorTypes, ",")
	for i, t := range types {
		types[i] = strings.TrimSpace(t)
	}

	// Validate constructor types
	for _, t := range types {
		if t != "allArgs" && t != "builder" && t != "options" {
			fmt.Fprintf(os.Stderr, "Error: invalid constructor type '%s'. Valid types: allArgs, builder, options\n", t)
			os.Exit(1)
		}
	}

	// Create generator config
	config := &GeneratorConfig{
		StructName:       *typeName,
		ConstructorTypes: types,
		OutputFile:       output,
		InitFunc:         *initFunc,
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

	// Write to file
	if err := os.WriteFile(output, []byte(code), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated constructor code in %s\n", output)
}

// findSourceFile searches for a Go file containing the struct definition
func findSourceFile(typeName string) (string, error) {
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
		if strings.HasSuffix(file, "_gen.go") {
			continue
		}

		// Try to parse the file
		_, err := ParseStruct(file, typeName)
		if err == nil {
			return file, nil
		}
	}

	return "", fmt.Errorf("could not find struct %s in current directory", typeName)
}
