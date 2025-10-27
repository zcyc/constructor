package allargs

import (
	"reflect"
	"testing"
)

func TestNewProduct(t *testing.T) {
	product := NewProduct(1, "Laptop", 999.99, "High-end laptop")

	if product == nil {
		t.Fatal("NewProduct returned nil")
	}

	if product.id != 1 {
		t.Errorf("Expected id 1, got %d", product.id)
	}

	if product.name != "Laptop" {
		t.Errorf("Expected name Laptop, got %s", product.name)
	}

	if product.price != 999.99 {
		t.Errorf("Expected price 999.99, got %f", product.price)
	}

	if product.description != "High-end laptop" {
		t.Errorf("Expected description 'High-end laptop', got %s", product.description)
	}
}

func TestProductGetters(t *testing.T) {
	product := NewProduct(2, "Mouse", 29.99, "Wireless mouse")

	// Test getters exist and work
	if product.GetId() != 2 {
		t.Errorf("GetId() = %d, want 2", product.GetId())
	}

	if product.GetName() != "Mouse" {
		t.Errorf("GetName() = %s, want Mouse", product.GetName())
	}

	if product.GetPrice() != 29.99 {
		t.Errorf("GetPrice() = %f, want 29.99", product.GetPrice())
	}

	// Verify that GetDescription does NOT exist (getter:false)
	productType := reflect.TypeOf(product)
	_, hasGetDescription := productType.MethodByName("GetDescription")
	if hasGetDescription {
		t.Error("GetDescription() should not exist due to constructor:\"getter:false\" tag")
	}
}
