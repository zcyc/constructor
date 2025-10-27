package main

// StructInfo represents parsed struct information
type StructInfo struct {
	Name        string      // Struct name, e.g., "User"
	Fields      []FieldInfo // List of fields
	PackageName string      // Package name
}

// FieldInfo represents a single field in a struct
type FieldInfo struct {
	Name       string // Field name, e.g., "userName"
	Type       string // Field type, e.g., "*string", "int"
	Tag        string // Field tag, e.g., `json:"user_name" constructor:"-"`
	Exported   bool   // Whether the field is exported (uppercase first letter)
	Skip       bool   // Whether to skip this field completely (from tag `constructor:"-"`)
	SkipGetter bool   // Whether to skip getter generation (from tag `constructor:"getter:false"`)
	SkipSetter bool   // Whether to skip setter/constructor parameter (from tag `constructor:"setter:false"`)
}

// GeneratorConfig holds configuration for code generation
type GeneratorConfig struct {
	StructName       string   // Target struct name
	ConstructorTypes []string // Types: "allArgs", "builder", "options"
	OutputFile       string   // Output file path
	InitFunc         string   // Initialization function name (optional)
	ReturnValue      bool     // Return value instead of pointer
	SetterPrefix     string   // Prefix for setter methods in builder (e.g., "With")
	WithGetter       bool     // Generate getter methods
}
