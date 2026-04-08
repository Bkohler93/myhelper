package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractSymbolMap(t *testing.T) {
	writeGoFile := func(t *testing.T, content string) string {
		t.Helper()
		dir := t.TempDir()
		path := filepath.Join(dir, "test.go")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	t.Run("exported func has correct name and line range", func(t *testing.T) {
		src := "package foo\n\nfunc Exported() {}\n"
		path := writeGoFile(t, src)
		symbols, err := ExtractSymbolMap(path)
		if err != nil {
			t.Fatalf("ExtractSymbolMap() error: %v", err)
		}
		if len(symbols) != 1 {
			t.Fatalf("expected 1 symbol, got %d: %v", len(symbols), symbols)
		}
		if symbols[0].Name != "func Exported" {
			t.Errorf("expected Name 'func Exported', got %q", symbols[0].Name)
		}
		if symbols[0].Start != 3 {
			t.Errorf("expected Start 3, got %d", symbols[0].Start)
		}
		if symbols[0].End != 3 {
			t.Errorf("expected End 3, got %d", symbols[0].End)
		}
	})

	t.Run("unexported func included with correct name", func(t *testing.T) {
		src := "package foo\n\nfunc unexported() {}\n"
		path := writeGoFile(t, src)
		symbols, err := ExtractSymbolMap(path)
		if err != nil {
			t.Fatalf("ExtractSymbolMap() error: %v", err)
		}
		if len(symbols) != 1 {
			t.Fatalf("expected 1 symbol, got %d: %v", len(symbols), symbols)
		}
		if symbols[0].Name != "func unexported" {
			t.Errorf("expected Name 'func unexported', got %q", symbols[0].Name)
		}
		if symbols[0].Start != 3 {
			t.Errorf("expected Start 3, got %d", symbols[0].Start)
		}
		if symbols[0].End != 3 {
			t.Errorf("expected End 3, got %d", symbols[0].End)
		}
	})

	t.Run("exported struct type included", func(t *testing.T) {
		src := "package foo\n\ntype MyStruct struct {\n\tField string\n}\n"
		path := writeGoFile(t, src)
		symbols, err := ExtractSymbolMap(path)
		if err != nil {
			t.Fatalf("ExtractSymbolMap() error: %v", err)
		}
		if len(symbols) != 1 {
			t.Fatalf("expected 1 symbol, got %d: %v", len(symbols), symbols)
		}
		if symbols[0].Name != "type MyStruct struct" {
			t.Errorf("expected Name 'type MyStruct struct', got %q", symbols[0].Name)
		}
		if symbols[0].Start != 3 {
			t.Errorf("expected Start 3, got %d", symbols[0].Start)
		}
	})

	t.Run("exported interface type included", func(t *testing.T) {
		src := "package foo\n\ntype MyInterface interface {\n\tDoThing() error\n}\n"
		path := writeGoFile(t, src)
		symbols, err := ExtractSymbolMap(path)
		if err != nil {
			t.Fatalf("ExtractSymbolMap() error: %v", err)
		}
		if len(symbols) != 1 {
			t.Fatalf("expected 1 symbol, got %d: %v", len(symbols), symbols)
		}
		if symbols[0].Name != "type MyInterface interface" {
			t.Errorf("expected Name 'type MyInterface interface', got %q", symbols[0].Name)
		}
		if symbols[0].Start != 3 {
			t.Errorf("expected Start 3, got %d", symbols[0].Start)
		}
	})

	t.Run("unexported struct NOT included", func(t *testing.T) {
		src := "package foo\n\ntype internal struct {\n\tx int\n}\n"
		path := writeGoFile(t, src)
		symbols, err := ExtractSymbolMap(path)
		if err != nil {
			t.Fatalf("ExtractSymbolMap() error: %v", err)
		}
		if len(symbols) != 0 {
			t.Errorf("expected 0 symbols, got %d: %v", len(symbols), symbols)
		}
	})

	t.Run("multi-line func body - End line > Start line", func(t *testing.T) {
		src := "package foo\n\nfunc BigFunc() {\n\tx := 1\n\t_ = x\n}\n"
		path := writeGoFile(t, src)
		symbols, err := ExtractSymbolMap(path)
		if err != nil {
			t.Fatalf("ExtractSymbolMap() error: %v", err)
		}
		if len(symbols) != 1 {
			t.Fatalf("expected 1 symbol, got %d: %v", len(symbols), symbols)
		}
		if symbols[0].Start != 3 {
			t.Errorf("expected Start 3, got %d", symbols[0].Start)
		}
		if symbols[0].End != 6 {
			t.Errorf("expected End 6, got %d", symbols[0].End)
		}
	})

	t.Run("syntax error returns error not panic", func(t *testing.T) {
		src := "package foo\nfunc Broken( {"
		path := writeGoFile(t, src)
		_, err := ExtractSymbolMap(path)
		if err == nil {
			t.Error("expected error for syntax error file, got nil")
		}
	})

	t.Run("exported const and var excluded", func(t *testing.T) {
		src := "package foo\n\nconst ExportedConst = 1\nvar ExportedVar = \"x\"\n"
		path := writeGoFile(t, src)
		symbols, err := ExtractSymbolMap(path)
		if err != nil {
			t.Fatalf("ExtractSymbolMap() error: %v", err)
		}
		if len(symbols) != 0 {
			t.Errorf("expected 0 symbols, got %d: %v", len(symbols), symbols)
		}
	})
}

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
