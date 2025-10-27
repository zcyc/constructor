package mixed

import "time"

//go:generate go run ../../. -type=Repository -constructorTypes=allArgs,builder,options -withGetter

// Repository represents a data repository
// This example demonstrates:
// 1. Multiple constructor patterns in one file
// 2. All three patterns: allArgs, builder, and options
// 3. Comprehensive field skipping examples
type Repository struct {
	// Fields included in all constructors with getters
	dsn         string
	maxConns    int
	idleTimeout time.Duration

	// Skip getter (will be in constructors but no getter)
	password string `constructor:"getter:false"`

	// Skip setter (will have getter but not in constructors)
	connCount int `constructor:"setter:false"`

	// Completely skip
	internal string `constructor:"-"`
}
