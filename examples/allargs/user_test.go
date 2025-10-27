package allargs

import (
	"testing"
	"time"
)

func TestNewUser(t *testing.T) {
	now := time.Now()
	user := NewUser(1, "Alice", "alice@example.com", now)

	if user == nil {
		t.Fatal("NewUser returned nil")
	}

	if user.id != 1 {
		t.Errorf("Expected id 1, got %d", user.id)
	}

	if user.name != "Alice" {
		t.Errorf("Expected name Alice, got %s", user.name)
	}

	if user.email != "alice@example.com" {
		t.Errorf("Expected email alice@example.com, got %s", user.email)
	}

	if user.createdAt != now {
		t.Errorf("Expected createdAt %v, got %v", now, user.createdAt)
	}
}

func TestUserFieldSkipping(t *testing.T) {
	// Verify that internal field is not in constructor
	// This is verified by compilation - if internal was a parameter, this test wouldn't compile
	now := time.Now()
	user := NewUser(1, "Bob", "bob@example.com", now)

	// We can still set internal field directly
	user.internal = "test-internal"
	if user.internal != "test-internal" {
		t.Error("Internal field should be settable directly")
	}

	// Verify metadata field (setter:false) is not in constructor but exists
	user.metadata = map[string]string{"key": "value"}
	if user.metadata["key"] != "value" {
		t.Error("Metadata field should be settable directly")
	}
}
