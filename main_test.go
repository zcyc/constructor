package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSameFile(t *testing.T) {
	if !sameFile("./user.go", "user.go") {
		t.Error("equivalent relative paths should match")
	}
	if sameFile("user.go", "user_gen.go") {
		t.Error("different output path should not match")
	}
}

func TestSameFileResolvesSymlinks(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.go")
	output := filepath.Join(dir, "output.go")
	if err := os.WriteFile(source, []byte("package test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(source, output); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}
	if !sameFile(source, output) {
		t.Fatal("a symlink to the source file must be rejected")
	}
}

func TestWriteGeneratedFileReplacesAtomically(t *testing.T) {
	output := filepath.Join(t.TempDir(), "generated.go")
	if err := os.WriteFile(output, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeGeneratedFile(output, "new"); err != nil {
		t.Fatalf("writeGeneratedFile failed: %v", err)
	}
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new" {
		t.Fatalf("output = %q, want %q", content, "new")
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("permissions = %o, want 600", info.Mode().Perm())
	}
}
