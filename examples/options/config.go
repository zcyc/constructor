package options

import "time"

//go:generate go run ../../. -type=AppConfig -constructorTypes=options -returnValue

// AppConfig represents application configuration
// This example demonstrates:
// 1. Functional options pattern
// 2. Return value instead of pointer
// 3. Field skipping with options pattern
type AppConfig struct {
	appName    string
	version    string
	debug      bool
	timeout    time.Duration
	maxWorkers int
	internal   string `constructor:"-"` // Completely skipped - no With option
	cacheDir   string // Will have With option
}
