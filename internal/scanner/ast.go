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

// Symbol represents a rich profile of an exported Go symbol, capturing all
// fields needed to populate symbols.json in Phase 10.
type Symbol struct {
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`      // "func" | "method" | "struct" | "interface"
	Signature string   `json:"signature"` // human-readable, e.g. "func Foo(x int) string"
	Start     int      `json:"start"`     // 1-indexed start line
	End       int      `json:"end"`       // 1-indexed end line
	Receiver  string   `json:"receiver"`  // empty for non-methods
	StableID  string   `json:"stableID"`  // "<pkg>.<Name>" or "<pkg>.<Recv>.<Name>"
	Imports   []string `json:"imports"`   // file-level import paths (same value for all symbols in a file)
	CallEdges []string `json:"callEdges"` // populated by Phase 10 body-walking pass
	TypeRefs  []string `json:"typeRefs"`  // populated by Phase 10 body-walking pass
	FilePath  string   `json:"filePath"`  // relative path to the source file
}

// extractImportPaths returns the import paths declared in f, including blank
// and dot imports (they appear in Imports but are excluded from the alias map).
func extractImportPaths(f *ast.File) []string {
	paths := make([]string, 0, len(f.Imports))
	for _, imp := range f.Imports {
		paths = append(paths, strings.Trim(imp.Path.Value, `"`))
	}
	return paths
}

// buildImportAliasMap maps local identifier -> import path.
// Blank imports (_) and dot imports (.) are skipped — they cannot be used as identifiers.
func buildImportAliasMap(f *ast.File) map[string]string {
	m := make(map[string]string)
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		local := path[strings.LastIndex(path, "/")+1:]
		if imp.Name != nil {
			switch imp.Name.Name {
			case "_", ".":
				continue
			default:
				local = imp.Name.Name
			}
		}
		m[local] = path
	}
	return m
}

// receiverTypeName unwraps the receiver type name, stripping pointer and generic
// decorators. Returns "" for non-method functions.
func receiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	field := recv.List[0]
	switch t := field.Type.(type) {
	case *ast.StarExpr:
		switch x := t.X.(type) {
		case *ast.Ident:
			return x.Name
		case *ast.IndexExpr:
			if ident, ok := x.X.(*ast.Ident); ok {
				return ident.Name
			}
		}
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

// buildStructSig renders exported struct fields as "Name type; Name type" format.
func buildStructSig(st *ast.StructType, fset *token.FileSet) string {
	var parts []string
	for _, field := range st.Fields.List {
		typeStr := nodeToString(field.Type, fset)
		if len(field.Names) == 0 {
			parts = append(parts, typeStr) // embedded field
		} else {
			for _, name := range field.Names {
				parts = append(parts, name.Name+" "+typeStr)
			}
		}
	}
	return strings.Join(parts, "; ")
}

// buildInterfaceSig renders interface method signatures joined by "; ".
// Embedded interfaces are rendered as their type name.
func buildInterfaceSig(it *ast.InterfaceType, fset *token.FileSet) string {
	var parts []string
	for _, method := range it.Methods.List {
		if len(method.Names) == 0 {
			// embedded interface — render type name
			parts = append(parts, nodeToString(method.Type, fset))
			continue
		}
		for _, name := range method.Names {
			if ft, ok := method.Type.(*ast.FuncType); ok {
				sig := buildMethodSigFromFuncType(name.Name, ft, fset)
				parts = append(parts, sig)
			}
		}
	}
	return strings.Join(parts, "; ")
}

// buildMethodSigFromFuncType renders "Name(params) results" for an interface
// method entry, reusing nodeToString for type rendering.
func buildMethodSigFromFuncType(name string, ft *ast.FuncType, fset *token.FileSet) string {
	var sb strings.Builder
	sb.WriteString(name)
	sb.WriteString("(")
	if ft.Params != nil {
		parts := make([]string, 0, len(ft.Params.List))
		for _, field := range ft.Params.List {
			typeStr := nodeToString(field.Type, fset)
			if len(field.Names) == 0 {
				parts = append(parts, typeStr)
			} else {
				for _, n := range field.Names {
					parts = append(parts, n.Name+" "+typeStr)
				}
			}
		}
		sb.WriteString(strings.Join(parts, ", "))
	}
	sb.WriteString(")")
	if ft.Results != nil && len(ft.Results.List) > 0 {
		if len(ft.Results.List) == 1 && len(ft.Results.List[0].Names) == 0 {
			sb.WriteString(" ")
			sb.WriteString(nodeToString(ft.Results.List[0].Type, fset))
		} else {
			parts := make([]string, 0, len(ft.Results.List))
			for _, field := range ft.Results.List {
				typeStr := nodeToString(field.Type, fset)
				if len(field.Names) == 0 {
					parts = append(parts, typeStr)
				} else {
					for _, n := range field.Names {
						parts = append(parts, n.Name+" "+typeStr)
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

// extractCallEdges walks the function body and returns a deduplicated list of
// call targets. Import-qualified calls (e.g. fmt.Println) are resolved to
// "<pkgname>.<symbol>" using importAliasMap. Unresolved selector calls and
// direct calls are stored as-is. Returns nil if body is nil.
func extractCallEdges(body *ast.BlockStmt, importAliasMap map[string]string) []string {
	if body == nil {
		return nil
	}
	var edges []string
	seen := make(map[string]bool)
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		var target string
		switch fun := call.Fun.(type) {
		case *ast.Ident:
			// direct call: foo()
			target = fun.Name
		case *ast.SelectorExpr:
			// selector call: pkg.Func() or obj.Method()
			if ident, ok := fun.X.(*ast.Ident); ok {
				if fullPkg, found := importAliasMap[ident.Name]; found {
					// Resolve to <pkgname>.<symbol> where pkgname = last segment of import path
					pkgName := fullPkg[strings.LastIndex(fullPkg, "/")+1:]
					target = pkgName + "." + fun.Sel.Name
				} else {
					// obj.Method — not a known import alias; store raw
					target = ident.Name + "." + fun.Sel.Name
				}
			}
		}
		if target != "" && !seen[target] {
			seen[target] = true
			edges = append(edges, target)
		}
		return true
	})
	return edges
}

// extractTypeRefs walks the function body and returns a deduplicated list of
// exported type references. Collects:
//   - ast.SelectorExpr where X.Name is a known import alias: "<pkgname>.<Type>"
//   - ast.Ident where Name is in knownTypes (exported type names from same file)
//
// Primitives (int, string, bool, etc.) are excluded because they are not exported
// (not capitalized) or not in knownTypes. Returns nil if body is nil.
func extractTypeRefs(body *ast.BlockStmt, importAliasMap map[string]string, knownTypes map[string]bool) []string {
	if body == nil {
		return nil
	}
	var refs []string
	seen := make(map[string]bool)
	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.SelectorExpr:
			// Cross-package type reference: pkg.Type
			if ident, ok := node.X.(*ast.Ident); ok {
				if fullPkg, found := importAliasMap[ident.Name]; found {
					pkgName := fullPkg[strings.LastIndex(fullPkg, "/")+1:]
					// Only collect if the selector looks like a type (exported = capitalized)
					if ast.IsExported(node.Sel.Name) {
						ref := pkgName + "." + node.Sel.Name
						if !seen[ref] {
							seen[ref] = true
							refs = append(refs, ref)
						}
					}
				}
			}
			// Return false to avoid descending into the SelectorExpr's children
			// independently (we already handled it at this level).
			return false
		case *ast.Ident:
			// File-local exported type reference
			if knownTypes[node.Name] && !seen[node.Name] {
				seen[node.Name] = true
				refs = append(refs, node.Name)
			}
		}
		return true
	})
	return refs
}

// ExtractSymbolsFull parses the Go source file at path and returns a rich
// Symbol profile for every exported symbol. Unexported symbols and methods
// on unexported receiver types are excluded (per D-09, D-10).
//
// CallEdges and TypeRefs are populated by walking exported function bodies.
//
// ExtractSymbols, ExtractSymbolMap, and their callers are not modified.
func ExtractSymbolsFull(path string) ([]Symbol, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("ExtractSymbolsFull: parse %s: %w", path, err)
	}

	pkg := f.Name.Name
	imports := extractImportPaths(f) // file-level, shared across all symbols
	if len(imports) == 0 {
		imports = nil
	}

	importAliasMap := buildImportAliasMap(f)

	// Pre-pass: collect exported type names declared in this file for TypeRefs matching
	knownTypes := make(map[string]bool)
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || !ts.Name.IsExported() {
				continue
			}
			knownTypes[ts.Name.Name] = true
		}
	}

	var symbols []Symbol

	for _, decl := range f.Decls {
		switch d := decl.(type) {

		case *ast.FuncDecl:
			if !d.Name.IsExported() {
				continue
			}
			recv := receiverTypeName(d.Recv)
			// D-10: exclude methods on unexported receiver types
			if recv != "" && !ast.IsExported(recv) {
				continue
			}

			kind := "func"
			if recv != "" {
				kind = "method"
			}

			sig := buildFuncSig(d, fset) // reuses existing helper

			stableID := pkg + "." + d.Name.Name
			if recv != "" {
				stableID = pkg + "." + recv + "." + d.Name.Name
			}

			sym := Symbol{
				Name:      d.Name.Name,
				Kind:      kind,
				Signature: sig,
				Start:     fset.Position(d.Pos()).Line,
				End:       fset.Position(d.End()).Line,
				Receiver:  recv,
				StableID:  stableID,
				Imports:   imports,
			}
			sym.CallEdges = extractCallEdges(d.Body, importAliasMap)
			sym.TypeRefs = extractTypeRefs(d.Body, importAliasMap, knownTypes)
			symbols = append(symbols, sym)

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !ts.Name.IsExported() {
					continue
				}

				var kind, sig string
				switch t := ts.Type.(type) {
				case *ast.StructType:
					kind = "struct"
					sig = buildStructSig(t, fset)
				case *ast.InterfaceType:
					kind = "interface"
					sig = buildInterfaceSig(t, fset)
				default:
					// type aliases and other type forms — skip per RESEARCH.md open question 3
					continue
				}

				symbols = append(symbols, Symbol{
					Name:      ts.Name.Name,
					Kind:      kind,
					Signature: sig,
					Start:     fset.Position(ts.Pos()).Line,
					End:       fset.Position(ts.End()).Line,
					StableID:  pkg + "." + ts.Name.Name,
					Imports:   imports,
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
