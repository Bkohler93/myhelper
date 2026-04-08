package scanner

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

// ExtractSymbols parses the Go source file at path and returns:
//   - pkg: the package name declared in the file
//   - symbols: exported func signatures and exported type/interface names
//   - err: non-nil if the file cannot be parsed (syntax error, missing file, etc.)
//
// Exported consts and vars are excluded per D-04.
func ExtractSymbols(path string) (pkg string, symbols []string, err error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return "", nil, fmt.Errorf("parse %s: %w", path, err)
	}

	pkg = f.Name.Name

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if !d.Name.IsExported() {
				continue
			}
			sig := buildFuncSig(d, fset)
			symbols = append(symbols, sig)

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					// const or var spec — skip
					continue
				}
				if !ts.Name.IsExported() {
					continue
				}
				switch ts.Type.(type) {
				case *ast.StructType:
					symbols = append(symbols, "type "+ts.Name.Name+" struct")
				case *ast.InterfaceType:
					symbols = append(symbols, "type "+ts.Name.Name+" interface")
				// other type aliases skipped
				}
			}
		}
	}

	return pkg, symbols, nil
}

// SymbolLine represents a named symbol (function or exported type) in a Go
// source file, with its 1-indexed start and end line numbers.
type SymbolLine struct {
	Name  string
	Start int
	End   int
}

// ExtractSymbolMap parses the Go source file at path and returns a slice of
// SymbolLine for:
//   - all function declarations (exported and unexported, including methods)
//   - exported struct and interface type declarations
//
// Name format:
//   - Functions: "func Name" (no signature, just name with "func " prefix)
//   - Types: "type Name struct" or "type Name interface"
//
// Line numbers are 1-indexed from go/ast fset.Position().
// Returns an error if the file cannot be parsed; never panics.
// ExtractSymbols is NOT called or modified by this function.
func ExtractSymbolMap(path string) ([]SymbolLine, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("ExtractSymbolMap: parse %s: %w", path, err)
	}

	var symbols []SymbolLine
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			symbols = append(symbols, SymbolLine{
				Name:  "func " + d.Name.Name,
				Start: fset.Position(d.Pos()).Line,
				End:   fset.Position(d.End()).Line,
			})

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !ts.Name.IsExported() {
					continue
				}
				var label string
				switch ts.Type.(type) {
				case *ast.StructType:
					label = "type " + ts.Name.Name + " struct"
				case *ast.InterfaceType:
					label = "type " + ts.Name.Name + " interface"
				default:
					continue
				}
				symbols = append(symbols, SymbolLine{
					Name:  label,
					Start: fset.Position(ts.Pos()).Line,
					End:   fset.Position(ts.End()).Line,
				})
			}
		}
	}
	return symbols, nil
}

// buildFuncSig builds a human-readable function signature string for an
// exported function declaration. Format: "func Name(params) results"
func buildFuncSig(decl *ast.FuncDecl, fset *token.FileSet) string {
	var sb strings.Builder
	sb.WriteString("func ")
	sb.WriteString(decl.Name.Name)
	sb.WriteString("(")

	params := decl.Type.Params
	if params != nil {
		parts := make([]string, 0, len(params.List))
		for _, field := range params.List {
			typeStr := nodeToString(field.Type, fset)
			if len(field.Names) == 0 {
				parts = append(parts, typeStr)
			} else {
				for _, name := range field.Names {
					parts = append(parts, name.Name+" "+typeStr)
				}
			}
		}
		sb.WriteString(strings.Join(parts, ", "))
	}
	sb.WriteString(")")

	results := decl.Type.Results
	if results != nil && len(results.List) > 0 {
		if len(results.List) == 1 && len(results.List[0].Names) == 0 {
			sb.WriteString(" ")
			sb.WriteString(nodeToString(results.List[0].Type, fset))
		} else {
			parts := make([]string, 0, len(results.List))
			for _, field := range results.List {
				typeStr := nodeToString(field.Type, fset)
				if len(field.Names) == 0 {
					parts = append(parts, typeStr)
				} else {
					for _, name := range field.Names {
						parts = append(parts, name.Name+" "+typeStr)
					}
				}
			}
			sb.WriteString(" (")
			sb.WriteString(strings.Join(parts, ", "))
			sb.WriteString(")")
		}
	}

	return sb.String()
}

// nodeToString renders an AST expression as source text using go/format.
func nodeToString(expr ast.Expr, fset *token.FileSet) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, expr); err != nil {
		return fmt.Sprintf("%v", expr)
	}
	return buf.String()
}
