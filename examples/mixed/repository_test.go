package mixed

import (
	"reflect"
	"testing"
	"time"
)

func TestNewRepository(t *testing.T) {
	repo := NewRepository(
		"postgres://localhost:5432/db",
		100,
		5*time.Minute,
		"password123",
	)

	if repo == nil {
		t.Fatal("NewRepository returned nil")
	}

	if repo.dsn != "postgres://localhost:5432/db" {
		t.Errorf("Expected dsn postgres://localhost:5432/db, got %s", repo.dsn)
	}

	if repo.maxConns != 100 {
		t.Errorf("Expected maxConns 100, got %d", repo.maxConns)
	}

	if repo.idleTimeout != 5*time.Minute {
		t.Errorf("Expected idleTimeout 5m, got %v", repo.idleTimeout)
	}

	if repo.password != "password123" {
		t.Errorf("Expected password password123, got %s", repo.password)
	}
}

func TestRepositoryBuilder(t *testing.T) {
	repo := NewRepositoryBuilder().
		Dsn("mysql://localhost:3306/db").
		MaxConns(50).
		IdleTimeout(10 * time.Minute).
		Password("secret").
		Build()

	if repo == nil {
		t.Fatal("Build returned nil")
	}

	if repo.dsn != "mysql://localhost:3306/db" {
		t.Errorf("Expected dsn mysql://localhost:3306/db, got %s", repo.dsn)
	}

	if repo.maxConns != 50 {
		t.Errorf("Expected maxConns 50, got %d", repo.maxConns)
	}

	if repo.idleTimeout != 10*time.Minute {
		t.Errorf("Expected idleTimeout 10m, got %v", repo.idleTimeout)
	}

	if repo.password != "secret" {
		t.Errorf("Expected password secret, got %s", repo.password)
	}
}

func TestRepositoryWithOptions(t *testing.T) {
	repo := NewRepositoryWithOptions(
		WithDsn("sqlite://data.db"),
		WithMaxConns(20),
		WithIdleTimeout(2*time.Minute),
		WithPassword("pass"),
	)

	if repo == nil {
		t.Fatal("NewRepositoryWithOptions returned nil")
	}

	if repo.dsn != "sqlite://data.db" {
		t.Errorf("Expected dsn sqlite://data.db, got %s", repo.dsn)
	}

	if repo.maxConns != 20 {
		t.Errorf("Expected maxConns 20, got %d", repo.maxConns)
	}

	if repo.idleTimeout != 2*time.Minute {
		t.Errorf("Expected idleTimeout 2m, got %v", repo.idleTimeout)
	}

	if repo.password != "pass" {
		t.Errorf("Expected password pass, got %s", repo.password)
	}
}

func TestRepositoryGetters(t *testing.T) {
	repo := NewRepository("dsn", 10, time.Minute, "pwd")

	// Test getters that should exist
	if repo.GetDsn() != "dsn" {
		t.Errorf("GetDsn() = %s, want dsn", repo.GetDsn())
	}

	if repo.GetMaxConns() != 10 {
		t.Errorf("GetMaxConns() = %d, want 10", repo.GetMaxConns())
	}

	if repo.GetIdleTimeout() != time.Minute {
		t.Errorf("GetIdleTimeout() = %v, want 1m", repo.GetIdleTimeout())
	}

	// Verify GetPassword does NOT exist (getter:false)
	repoType := reflect.TypeOf(repo)
	_, hasGetPassword := repoType.MethodByName("GetPassword")
	if hasGetPassword {
		t.Error("GetPassword() should not exist due to constructor:\"getter:false\" tag")
	}

	// Verify GetConnCount DOES exist (setter:false, but getter should exist)
	_, hasGetConnCount := repoType.MethodByName("GetConnCount")
	if !hasGetConnCount {
		t.Error("GetConnCount() should exist even with constructor:\"setter:false\" tag")
	}

	// Test the getter
	repo.connCount = 42
	if repo.GetConnCount() != 42 {
		t.Errorf("GetConnCount() = %d, want 42", repo.GetConnCount())
	}
}

func TestRepositoryFieldSkipping(t *testing.T) {
	// Verify internal field is not in any constructor
	repo := NewRepository("dsn", 10, time.Minute, "pwd")

	// We can still set internal directly
	repo.internal = "test-internal"
	if repo.internal != "test-internal" {
		t.Error("Internal field should be settable directly")
	}

	// Verify connCount is not in constructors (verified by compilation)
	// but can be set directly
	repo.connCount = 5
	if repo.connCount != 5 {
		t.Error("ConnCount should be settable directly")
	}

	// Verify builder doesn't have ConnCount setter
	builderType := reflect.TypeOf(&RepositoryBuilder{})
	_, hasConnCount := builderType.MethodByName("ConnCount")
	if hasConnCount {
		t.Error("ConnCount() setter should not exist due to constructor:\"setter:false\" tag")
	}
}
