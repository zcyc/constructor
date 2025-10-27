package builder

import (
	"reflect"
	"testing"
)

func TestDatabaseBuilder(t *testing.T) {
	db := NewDatabaseBuilder().
		Host("db.example.com").
		Port(5432).
		Username("admin").
		Password("secret123").
		Build()

	if db == nil {
		t.Fatal("Build returned nil")
	}

	if db.host != "db.example.com" {
		t.Errorf("Expected host db.example.com, got %s", db.host)
	}

	if db.port != 5432 {
		t.Errorf("Expected port 5432, got %d", db.port)
	}

	if db.username != "admin" {
		t.Errorf("Expected username admin, got %s", db.username)
	}

	if db.password != "secret123" {
		t.Errorf("Expected password secret123, got %s", db.password)
	}
}

func TestDatabaseGetters(t *testing.T) {
	db := NewDatabaseBuilder().
		Host("localhost").
		Port(3306).
		Username("root").
		Password("rootpass").
		Build()

	// Test getters that should exist
	if db.GetHost() != "localhost" {
		t.Errorf("GetHost() = %s, want localhost", db.GetHost())
	}

	if db.GetPort() != 3306 {
		t.Errorf("GetPort() = %d, want 3306", db.GetPort())
	}

	if db.GetUsername() != "root" {
		t.Errorf("GetUsername() = %s, want root", db.GetUsername())
	}

	// Verify GetPassword does NOT exist (getter:false)
	dbType := reflect.TypeOf(db)
	_, hasGetPassword := dbType.MethodByName("GetPassword")
	if hasGetPassword {
		t.Error("GetPassword() should not exist due to constructor:\"getter:false\" tag")
	}

	// Verify GetPoolSize DOES exist (setter:false, but getter should exist)
	_, hasGetPoolSize := dbType.MethodByName("GetPoolSize")
	if !hasGetPoolSize {
		t.Error("GetPoolSize() should exist even with constructor:\"setter:false\" tag")
	}
}

func TestDatabaseSetterSkipping(t *testing.T) {
	// Verify that PoolSize method does NOT exist (setter:false)
	builderType := reflect.TypeOf(&DatabaseBuilder{})
	_, hasPoolSize := builderType.MethodByName("PoolSize")
	if hasPoolSize {
		t.Error("PoolSize() setter should not exist due to constructor:\"setter:false\" tag")
	}
}
