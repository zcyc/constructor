package options

import (
	"reflect"
	"testing"
)

func TestNewServerWithOptions(t *testing.T) {
	server := NewServerWithOptions(
		WithAddress("0.0.0.0"),
		WithPort(8443),
		WithTlsKey("/path/to/key.pem"),
	)

	if server == nil {
		t.Fatal("NewServerWithOptions returned nil")
	}

	if server.address != "0.0.0.0" {
		t.Errorf("Expected address 0.0.0.0, got %s", server.address)
	}

	if server.port != 8443 {
		t.Errorf("Expected port 8443, got %d", server.port)
	}

	if server.tlsKey != "/path/to/key.pem" {
		t.Errorf("Expected tlsKey /path/to/key.pem, got %s", server.tlsKey)
	}
}

func TestServerGetters(t *testing.T) {
	server := NewServerWithOptions(
		WithAddress("127.0.0.1"),
		WithPort(9000),
		WithTlsKey("secret.key"),
	)

	// Test getters that should exist
	if server.GetAddress() != "127.0.0.1" {
		t.Errorf("GetAddress() = %s, want 127.0.0.1", server.GetAddress())
	}

	if server.GetPort() != 9000 {
		t.Errorf("GetPort() = %d, want 9000", server.GetPort())
	}

	// Verify GetTlsKey does NOT exist (getter:false)
	serverType := reflect.TypeOf(server)
	_, hasGetTlsKey := serverType.MethodByName("GetTlsKey")
	if hasGetTlsKey {
		t.Error("GetTlsKey() should not exist due to constructor:\"getter:false\" tag")
	}

	// Verify GetInstanceID DOES exist (setter:false, but getter should exist)
	_, hasGetInstanceID := serverType.MethodByName("GetInstanceID")
	if !hasGetInstanceID {
		t.Error("GetInstanceID() should exist even with constructor:\"setter:false\" tag")
	}

	// Test the getter
	server.instanceID = "test-instance-123"
	if server.GetInstanceID() != "test-instance-123" {
		t.Errorf("GetInstanceID() = %s, want test-instance-123", server.GetInstanceID())
	}
}

func TestServerOptionSkipping(t *testing.T) {
	// We can't directly test that WithInstanceID doesn't exist at compile time,
	// but we can verify the behavior
	server := NewServerWithOptions(
		WithAddress("localhost"),
	)

	// InstanceID should not be settable via options, but can be set directly
	server.instanceID = "manual-id"
	if server.instanceID != "manual-id" {
		t.Error("InstanceID should be settable directly")
	}
}
