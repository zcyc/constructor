package main

import "testing"

func TestSameFile(t *testing.T) {
	if !sameFile("./user.go", "user.go") {
		t.Error("equivalent relative paths should match")
	}
	if sameFile("user.go", "user_gen.go") {
		t.Error("different output path should not match")
	}
}
