package builder

import "time"

//go:generate go run ../../. -type=Service -constructorTypes=builder -setterPrefix=With -init=initialize

// Service represents a service configuration
// This example demonstrates:
// 1. Builder pattern with "With" prefix for setters
// 2. Init function support
// 3. Field skipping in builder
type Service struct {
	name       string
	host       string
	port       int
	timeout    time.Duration
	maxRetries int
	internal   string `constructor:"-"` // Completely skipped - no setter method
}

// initialize is called after the service is built
func (s *Service) initialize() {
	if s.timeout == 0 {
		s.timeout = 30 * time.Second
	}
	if s.maxRetries == 0 {
		s.maxRetries = 3
	}
}
