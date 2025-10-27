package builder

import (
	"reflect"
	"testing"
	"time"
)

func TestServiceBuilder(t *testing.T) {
	service := NewServiceBuilder().
		WithName("API Service").
		WithHost("localhost").
		WithPort(8080).
		WithTimeout(60 * time.Second).
		WithMaxRetries(5).
		Build()

	if service == nil {
		t.Fatal("Build returned nil")
	}

	if service.name != "API Service" {
		t.Errorf("Expected name 'API Service', got %s", service.name)
	}

	if service.host != "localhost" {
		t.Errorf("Expected host localhost, got %s", service.host)
	}

	if service.port != 8080 {
		t.Errorf("Expected port 8080, got %d", service.port)
	}

	if service.timeout != 60*time.Second {
		t.Errorf("Expected timeout 60s, got %v", service.timeout)
	}

	if service.maxRetries != 5 {
		t.Errorf("Expected maxRetries 5, got %d", service.maxRetries)
	}
}

func TestServiceInitFunction(t *testing.T) {
	// Build service without setting timeout and maxRetries
	service := NewServiceBuilder().
		WithName("Test Service").
		WithHost("example.com").
		WithPort(443).
		Build()

	// Verify that initialize() was called and set defaults
	if service.timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", service.timeout)
	}

	if service.maxRetries != 3 {
		t.Errorf("Expected default maxRetries 3, got %d", service.maxRetries)
	}
}

func TestServiceFieldSkipping(t *testing.T) {
	// Verify that WithInternal method does NOT exist (field has constructor:"-")
	builderType := reflect.TypeOf(&ServiceBuilder{})
	_, hasWithInternal := builderType.MethodByName("WithInternal")
	if hasWithInternal {
		t.Error("WithInternal() should not exist due to constructor:\"-\" tag")
	}
}
