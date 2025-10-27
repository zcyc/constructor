package builder

//go:generate go run ../../. -type=Database -constructorTypes=builder -withGetter

// Database represents a database configuration
// This example demonstrates:
// 1. Builder pattern with getters
// 2. Setter-only fields (no getter) with constructor:"getter:false"
// 3. Getter-only fields (no setter) with constructor:"setter:false"
type Database struct {
	host     string // Will have setter and getter
	port     int    // Will have setter and getter
	username string // Will have setter and getter
	password string `constructor:"getter:false"` // Will have setter but NO getter (security)
	poolSize int    `constructor:"setter:false"` // Will have getter but NO setter (internal management)
}
