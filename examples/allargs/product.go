package allargs

//go:generate go run ../../. -type=Product -constructorTypes=allArgs -withGetter

// Product represents a product with getters
// This example demonstrates:
// 1. All-args constructor with automatic getter generation
// 2. Skip getter for specific fields with constructor:"getter:false"
type Product struct {
	id          int     // Will have constructor parameter and getter
	name        string  // Will have constructor parameter and getter
	price       float64 // Will have constructor parameter and getter
	description string  `constructor:"getter:false"` // Will have constructor parameter but NO getter
}
