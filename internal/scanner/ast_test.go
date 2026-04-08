package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractSymbols(t *testing.T) {
	writeGoFile := func(t *testing.T, content string) string {
		t.Helper()
		dir := t.TempDir()
		path := filepath.Join(dir, "test.go")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	t.Run("exported func signature included", func(t *testing.T) {
		src := `package foo

func Foo(x int) string {
	return ""
}
`
		path := writeGoFile(t, src)
		pkg, symbols, err := ExtractSymbols(path)
		if err != nil {
			t.Fatalf("ExtractSymbols() error: %v", err)
		}
		if pkg != "foo" {
			t.Errorf("expected package 'foo', got %q", pkg)
		}
		if len(symbols) != 1 {
			t.Fatalf("expected 1 symbol, got %d: %v", len(symbols), symbols)
		}
		if symbols[0] != "func Foo(x int) string" {
			t.Errorf("expected 'func Foo(x int) string', got %q", symbols[0])
		}
	})

	t.Run("unexported func excluded", func(t *testing.T) {
		src := `package foo

func foo(x int) string {
	return ""
}
`
		path := writeGoFile(t, src)
		_, symbols, err := ExtractSymbols(path)
		if err != nil {
			t.Fatalf("ExtractSymbols() error: %v", err)
		}
		if len(symbols) != 0 {
			t.Errorf("expected 0 symbols (unexported func), got %d: %v", len(symbols), symbols)
		}
	})

	t.Run("exported struct type included", func(t *testing.T) {
		src := `package foo

type MyStruct struct {
	Field string
}
`
		path := writeGoFile(t, src)
		_, symbols, err := ExtractSymbols(path)
		if err != nil {
			t.Fatalf("ExtractSymbols() error: %v", err)
		}
		if len(symbols) != 1 {
			t.Fatalf("expected 1 symbol, got %d: %v", len(symbols), symbols)
		}
		if symbols[0] != "type MyStruct struct" {
			t.Errorf("expected 'type MyStruct struct', got %q", symbols[0])
		}
	})

	t.Run("exported interface type included", func(t *testing.T) {
		src := `package foo

type MyInterface interface {
	DoThing() error
}
`
		path := writeGoFile(t, src)
		_, symbols, err := ExtractSymbols(path)
		if err != nil {
			t.Fatalf("ExtractSymbols() error: %v", err)
		}
		if len(symbols) != 1 {
			t.Fatalf("expected 1 symbol, got %d: %v", len(symbols), symbols)
		}
		if symbols[0] != "type MyInterface interface" {
			t.Errorf("expected 'type MyInterface interface', got %q", symbols[0])
		}
	})

	t.Run("exported const excluded", func(t *testing.T) {
		src := `package foo

const ExportedConst = 1
`
		path := writeGoFile(t, src)
		_, symbols, err := ExtractSymbols(path)
		if err != nil {
			t.Fatalf("ExtractSymbols() error: %v", err)
		}
		if len(symbols) != 0 {
			t.Errorf("expected 0 symbols (consts excluded), got %d: %v", len(symbols), symbols)
		}
	})

	t.Run("exported var excluded", func(t *testing.T) {
		src := `package foo

var ExportedVar = "x"
`
		path := writeGoFile(t, src)
		_, symbols, err := ExtractSymbols(path)
		if err != nil {
			t.Fatalf("ExtractSymbols() error: %v", err)
		}
		if len(symbols) != 0 {
			t.Errorf("expected 0 symbols (vars excluded), got %d: %v", len(symbols), symbols)
		}
	})

	t.Run("package name extracted correctly", func(t *testing.T) {
		src := `package scanner

type Index struct{}
`
		path := writeGoFile(t, src)
		pkg, _, err := ExtractSymbols(path)
		if err != nil {
			t.Fatalf("ExtractSymbols() error: %v", err)
		}
		if pkg != "scanner" {
			t.Errorf("expected package 'scanner', got %q", pkg)
		}
	})

	t.Run("syntax error returns error not panic", func(t *testing.T) {
		src := `package foo

func Broken( {
	this is not valid Go
`
		path := writeGoFile(t, src)
		_, _, err := ExtractSymbols(path)
		if err == nil {
			t.Error("expected error for syntax error file, got nil")
		}
	})
}
