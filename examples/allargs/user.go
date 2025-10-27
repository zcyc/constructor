package allargs

import "time"

//go:generate go run ../../. -type=User -constructorTypes=allArgs

// User represents a user in the system
// This example demonstrates:
// 1. Basic all-args constructor
// 2. Field skipping with constructor:"-"
// 3. Getter-only fields with constructor:"setter:false"
type User struct {
	id        int               // Will be included in constructor
	name      string            // Will be included in constructor
	email     string            // Will be included in constructor
	createdAt time.Time         // Will be included in constructor
	internal  string            `constructor:"-"`            // Completely skipped
	metadata  map[string]string `constructor:"setter:false"` // Only getter, not in constructor
}
