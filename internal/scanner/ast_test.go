package scanner

import (
	"os"
	"path/filepath"
	"strings"
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

func TestExtractSymbolsFull(t *testing.T) {
	writeGoFile := func(t *testing.T, content string) string {
		t.Helper()
		dir := t.TempDir()
		path := filepath.Join(dir, "test.go")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	t.Run("kind", func(t *testing.T) {
		src := `package foo

type Server struct{}

func Foo() {}
func (s *Server) Handle() {}

type Config struct{ Name string }
type Doer interface{ Do() error }
`
		path := writeGoFile(t, src)
		symbols, err := ExtractSymbolsFull(path)
		if err != nil {
			t.Fatalf("ExtractSymbolsFull() error: %v", err)
		}
		kindMap := make(map[string]string)
		for _, sym := range symbols {
			kindMap[sym.Name] = sym.Kind
		}
		cases := []struct {
			name string
			want string
		}{
			{"Foo", "func"},
			{"Handle", "method"},
			{"Config", "struct"},
			{"Doer", "interface"},
			{"Server", "struct"},
		}
		for _, tc := range cases {
			got, ok := kindMap[tc.name]
			if !ok {
				t.Errorf("symbol %q not found in results", tc.name)
				continue
			}
			if got != tc.want {
				t.Errorf("symbol %q: Kind=%q, want %q", tc.name, got, tc.want)
			}
		}
	})

	t.Run("signature", func(t *testing.T) {
		t.Run("func", func(t *testing.T) {
			src := `package foo

func Foo(x int, y string) (bool, error) {
	return false, nil
}
`
			path := writeGoFile(t, src)
			symbols, err := ExtractSymbolsFull(path)
			if err != nil {
				t.Fatalf("ExtractSymbolsFull() error: %v", err)
			}
			if len(symbols) != 1 {
				t.Fatalf("expected 1 symbol, got %d: %v", len(symbols), symbols)
			}
			want := "func Foo(x int, y string) (bool, error)"
			if symbols[0].Signature != want {
				t.Errorf("Signature=%q, want %q", symbols[0].Signature, want)
			}
		})

		t.Run("method", func(t *testing.T) {
			src := `package foo

type Server struct{}

func (s *Server) Handle(n int) bool {
	return true
}
`
			path := writeGoFile(t, src)
			symbols, err := ExtractSymbolsFull(path)
			if err != nil {
				t.Fatalf("ExtractSymbolsFull() error: %v", err)
			}
			var handleSym *Symbol
			for i := range symbols {
				if symbols[i].Name == "Handle" {
					handleSym = &symbols[i]
				}
			}
			if handleSym == nil {
				t.Fatal("symbol Handle not found")
			}
			want := "func Handle(n int) bool"
			if handleSym.Signature != want {
				t.Errorf("Signature=%q, want %q", handleSym.Signature, want)
			}
			if handleSym.Receiver != "Server" {
				t.Errorf("Receiver=%q, want %q", handleSym.Receiver, "Server")
			}
		})

		t.Run("struct", func(t *testing.T) {
			src := `package foo

type Config struct {
	Name string
	Age  int
}
`
			path := writeGoFile(t, src)
			symbols, err := ExtractSymbolsFull(path)
			if err != nil {
				t.Fatalf("ExtractSymbolsFull() error: %v", err)
			}
			if len(symbols) != 1 {
				t.Fatalf("expected 1 symbol, got %d: %v", len(symbols), symbols)
			}
			want := "Name string; Age int"
			if symbols[0].Signature != want {
				t.Errorf("Signature=%q, want %q", symbols[0].Signature, want)
			}
		})

		t.Run("interface", func(t *testing.T) {
			src := `package foo

type Doer interface {
	Do() error
}
`
			path := writeGoFile(t, src)
			symbols, err := ExtractSymbolsFull(path)
			if err != nil {
				t.Fatalf("ExtractSymbolsFull() error: %v", err)
			}
			if len(symbols) != 1 {
				t.Fatalf("expected 1 symbol, got %d: %v", len(symbols), symbols)
			}
			if symbols[0].Signature == "" {
				t.Error("interface Signature is empty, want non-empty")
			}
			// should contain the method signature
			if !strings.Contains(symbols[0].Signature, "Do") {
				t.Errorf("interface Signature=%q, expected to contain 'Do'", symbols[0].Signature)
			}
		})
	})

	t.Run("lines", func(t *testing.T) {
		t.Run("one-liner func at line 3", func(t *testing.T) {
			src := "package foo\n\nfunc Foo() {}\n"
			path := writeGoFile(t, src)
			symbols, err := ExtractSymbolsFull(path)
			if err != nil {
				t.Fatalf("ExtractSymbolsFull() error: %v", err)
			}
			if len(symbols) != 1 {
				t.Fatalf("expected 1 symbol, got %d", len(symbols))
			}
			if symbols[0].Start != 3 {
				t.Errorf("Start=%d, want 3", symbols[0].Start)
			}
			if symbols[0].End != 3 {
				t.Errorf("End=%d, want 3", symbols[0].End)
			}
		})

		t.Run("multi-line func End > Start", func(t *testing.T) {
			src := "package foo\n\nfunc BigFunc() {\n\tx := 1\n\t_ = x\n}\n"
			path := writeGoFile(t, src)
			symbols, err := ExtractSymbolsFull(path)
			if err != nil {
				t.Fatalf("ExtractSymbolsFull() error: %v", err)
			}
			if len(symbols) != 1 {
				t.Fatalf("expected 1 symbol, got %d", len(symbols))
			}
			if symbols[0].End <= symbols[0].Start {
				t.Errorf("multi-line func: End=%d should be > Start=%d", symbols[0].End, symbols[0].Start)
			}
		})
	})

	t.Run("imports", func(t *testing.T) {
		t.Run("file with imports - every symbol carries all import paths", func(t *testing.T) {
			src := `package foo

import (
	"fmt"
	"os"
)

func Foo() string {
	_ = fmt.Sprintf
	_ = os.Stderr
	return ""
}
`
			path := writeGoFile(t, src)
			symbols, err := ExtractSymbolsFull(path)
			if err != nil {
				t.Fatalf("ExtractSymbolsFull() error: %v", err)
			}
			if len(symbols) != 1 {
				t.Fatalf("expected 1 symbol, got %d", len(symbols))
			}
			imports := symbols[0].Imports
			if len(imports) != 2 {
				t.Errorf("Imports=%v, want [fmt os]", imports)
			}
			importSet := make(map[string]bool)
			for _, imp := range imports {
				importSet[imp] = true
			}
			for _, want := range []string{"fmt", "os"} {
				if !importSet[want] {
					t.Errorf("Imports missing %q, got %v", want, imports)
				}
			}
		})

		t.Run("file with no imports - symbol has nil or empty Imports", func(t *testing.T) {
			src := "package foo\n\nfunc Foo() {}\n"
			path := writeGoFile(t, src)
			symbols, err := ExtractSymbolsFull(path)
			if err != nil {
				t.Fatalf("ExtractSymbolsFull() error: %v", err)
			}
			if len(symbols) != 1 {
				t.Fatalf("expected 1 symbol, got %d", len(symbols))
			}
			if len(symbols[0].Imports) != 0 {
				t.Errorf("expected empty Imports, got %v", symbols[0].Imports)
			}
		})
	})

	t.Run("stable_id", func(t *testing.T) {
		src := `package foo

type Server struct{}

func Foo() {}
func (s *Server) Handle() {}
type Config struct{ Name string }
`
		path := writeGoFile(t, src)
		symbols, err := ExtractSymbolsFull(path)
		if err != nil {
			t.Fatalf("ExtractSymbolsFull() error: %v", err)
		}
		idMap := make(map[string]string)
		for _, sym := range symbols {
			idMap[sym.Name] = sym.StableID
		}
		cases := []struct {
			name string
			want string
		}{
			{"Foo", "foo.Foo"},
			{"Config", "foo.Config"},
			{"Handle", "foo.Server.Handle"},
		}
		for _, tc := range cases {
			got, ok := idMap[tc.name]
			if !ok {
				t.Errorf("symbol %q not found in results", tc.name)
				continue
			}
			if got != tc.want {
				t.Errorf("symbol %q: StableID=%q, want %q", tc.name, got, tc.want)
			}
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
