package options

//go:generate go run ../../. -type=Server -constructorTypes=options -withGetter

// Server represents a server configuration
// This example demonstrates:
// 1. Functional options pattern with getters
// 2. Option-only fields (no getter) with constructor:"getter:false"
// 3. Getter-only fields (no option) with constructor:"setter:false"
type Server struct {
	address    string // Will have With option and getter
	port       int    // Will have With option and getter
	tlsKey     string `constructor:"getter:false"` // Will have With option but NO getter (security)
	instanceID string `constructor:"setter:false"` // Will have getter but NO With option (auto-generated)
}
