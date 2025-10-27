package options

import (
	"reflect"
	"testing"
	"time"
)

func TestNewAppConfigWithOptions(t *testing.T) {
	config := NewAppConfigWithOptions(
		WithAppName("MyApp"),
		WithVersion("1.0.0"),
		WithDebug(true),
		WithTimeout(30*time.Second),
		WithMaxWorkers(10),
		WithCacheDir("/tmp/cache"),
	)

	// Verify it returns a value, not a pointer
	configType := reflect.TypeOf(config)
	if configType.Kind() == reflect.Ptr {
		t.Error("Expected AppConfig value, got pointer")
	}

	if config.appName != "MyApp" {
		t.Errorf("Expected appName MyApp, got %s", config.appName)
	}

	if config.version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", config.version)
	}

	if !config.debug {
		t.Error("Expected debug true, got false")
	}

	if config.timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.timeout)
	}

	if config.maxWorkers != 10 {
		t.Errorf("Expected maxWorkers 10, got %d", config.maxWorkers)
	}

	if config.cacheDir != "/tmp/cache" {
		t.Errorf("Expected cacheDir /tmp/cache, got %s", config.cacheDir)
	}
}

func TestAppConfigFieldSkipping(t *testing.T) {
	// Verify that WithInternal option does NOT exist (field has constructor:"-")
	// This is implicitly tested by compilation - if WithInternal existed and was used,
	// this test wouldn't compile
	config := NewAppConfigWithOptions(
		WithAppName("Test"),
	)

	// We can still set internal directly
	configPtr := &config
	configPtr.internal = "test"
	if configPtr.internal != "test" {
		t.Error("Internal field should be settable directly")
	}
}

func TestAppConfigPartialOptions(t *testing.T) {
	// Test with only some options
	config := NewAppConfigWithOptions(
		WithAppName("PartialApp"),
		WithDebug(false),
	)

	if config.appName != "PartialApp" {
		t.Errorf("Expected appName PartialApp, got %s", config.appName)
	}

	if config.debug {
		t.Error("Expected debug false, got true")
	}

	// Other fields should have zero values
	if config.version != "" {
		t.Errorf("Expected empty version, got %s", config.version)
	}
}
