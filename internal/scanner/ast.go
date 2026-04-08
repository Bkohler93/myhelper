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
