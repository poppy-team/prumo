package repomap

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"regexp"
	"sort"
	"strings"
)

var (
	rustPubFnRe     = regexp.MustCompile(`(?m)^\s*pub\s+(?:async\s+)?fn\s+([a-zA-Z_]\w*)\s*(<[^>]*>)?\s*(\([^)]*\))(?:\s*->\s*([^{;]+))?`)
	rustPubStructRe = regexp.MustCompile(`(?m)^\s*pub\s+struct\s+([A-Z]\w*)`)
	rustPubEnumRe   = regexp.MustCompile(`(?m)^\s*pub\s+enum\s+([A-Z]\w*)`)
	rustPubTraitRe  = regexp.MustCompile(`(?m)^\s*pub\s+trait\s+([A-Z]\w*)`)
	rustPubTypeRe   = regexp.MustCompile(`(?m)^\s*pub\s+type\s+([A-Z]\w*)`)

	tsExportFnRe    = regexp.MustCompile(`(?m)^\s*export\s+(?:default\s+)?(?:async\s+)?function\s+([a-zA-Z_]\w*)\s*(\([^)]*\))`)
	tsExportClassRe = regexp.MustCompile(`(?m)^\s*export\s+(?:default\s+)?class\s+([A-Z]\w*)`)
	tsExportInterRe = regexp.MustCompile(`(?m)^\s*export\s+interface\s+([A-Z]\w*)`)
	tsExportTypeRe  = regexp.MustCompile(`(?m)^\s*export\s+type\s+([A-Z]\w*)`)
	tsExportConstRe = regexp.MustCompile(`(?m)^\s*export\s+(?:const|let|var)\s+([a-zA-Z_]\w*)`)

	pyDefRe   = regexp.MustCompile(`(?m)^\s*def\s+([a-zA-Z_]\w*)\s*(\([^)]*\))`)
	pyClassRe = regexp.MustCompile(`(?m)^\s*class\s+([A-Z]\w*)`)

	identRe = regexp.MustCompile(`\b[a-zA-Z_]\w*\b`)
)

// ExtractGo extracts exported declarations and referenced identifiers from Go source using go/parser AST.
func ExtractGo(file string, data []byte) (pkg string, symbols []Symbol, references []string) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, data, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		// Fallback to lightweight regex scanning on syntax error
		return fallbackGo(file, string(data))
	}

	pkg = f.Name.Name
	refSet := make(map[string]struct{})

	// 1. Extract package imports as references
	for _, imp := range f.Imports {
		if imp.Path != nil {
			val := strings.Trim(imp.Path.Value, `"`)
			parts := strings.Split(val, "/")
			if len(parts) > 0 {
				refSet[parts[len(parts)-1]] = struct{}{}
			}
		}
	}

	// 2. Extract declarations
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if ast.IsExported(d.Name.Name) {
				kind := "func"
				sig := formatFuncSignature(d)
				if d.Recv != nil && len(d.Recv.List) > 0 {
					kind = "method"
				}
				symbols = append(symbols, Symbol{
					Name:      d.Name.Name,
					Kind:      kind,
					Signature: sig,
					File:      file,
					Line:      fset.Position(d.Pos()).Line,
				})
			}
		case *ast.GenDecl:
			switch d.Tok {
			case token.TYPE:
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok && ast.IsExported(ts.Name.Name) {
						kind := "type"
						switch ts.Type.(type) {
						case *ast.StructType:
							kind = "struct"
						case *ast.InterfaceType:
							kind = "interface"
						}
						symbols = append(symbols, Symbol{
							Name: ts.Name.Name,
							Kind: kind,
							File: file,
							Line: fset.Position(ts.Pos()).Line,
						})
					}
				}
			case token.CONST, token.VAR:
				kind := "const"
				if d.Tok == token.VAR {
					kind = "var"
				}
				for _, spec := range d.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok {
						for _, name := range vs.Names {
							if ast.IsExported(name.Name) {
								symbols = append(symbols, Symbol{
									Name: name.Name,
									Kind: kind,
									File: file,
									Line: fset.Position(name.Pos()).Line,
								})
							}
						}
					}
				}
			}
		}
	}

	// 3. Extract identifier references (calls, types, fields)
	defNames := make(map[string]struct{}, len(symbols))
	for _, s := range symbols {
		defNames[s.Name] = struct{}{}
	}

	ast.Inspect(f, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			if _, isDef := defNames[id.Name]; !isDef && len(id.Name) > 1 {
				refSet[id.Name] = struct{}{}
			}
		}
		return true
	})

	for r := range refSet {
		references = append(references, r)
	}

	sort.Slice(symbols, func(i, j int) bool {
		if symbols[i].Line != symbols[j].Line {
			return symbols[i].Line < symbols[j].Line
		}
		return symbols[i].Name < symbols[j].Name
	})

	return pkg, symbols, references
}

func formatFuncSignature(d *ast.FuncDecl) string {
	fset := token.NewFileSet()
	clone := &ast.FuncDecl{
		Name: d.Name,
		Type: d.Type,
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, clone); err == nil {
		s := buf.String()
		s = strings.TrimPrefix(s, "func ")
		s = strings.TrimPrefix(s, d.Name.Name)
		return strings.TrimSpace(s)
	}
	return ""
}

func fallbackGo(file, src string) (string, []Symbol, []string) {
	pkg := firstMatch(goPackageRe, src)
	syms := goSymbols(file, src)
	var refs []string
	matches := identRe.FindAllString(src, -1)
	for _, m := range matches {
		if len(m) > 2 {
			refs = append(refs, m)
		}
	}
	return pkg, syms, refs
}

// ExtractRust parses Rust definitions and references.
func ExtractRust(file, src string) ([]Symbol, []string) {
	var symbols []Symbol
	lines := strings.Split(src, "\n")
	lineOf := func(idx int) int {
		c := 0
		for i, line := range lines {
			c += len(line) + 1
			if c > idx {
				return i + 1
			}
		}
		return len(lines)
	}

	for _, m := range rustPubFnRe.FindAllStringSubmatchIndex(src, -1) {
		name := src[m[2]:m[3]]
		sig := ""
		if len(m) >= 8 && m[6] >= 0 {
			sig = src[m[6]:m[7]]
		}
		symbols = append(symbols, Symbol{Name: name, Kind: "func", Signature: sig, File: file, Line: lineOf(m[0])})
	}
	for _, m := range rustPubStructRe.FindAllStringSubmatchIndex(src, -1) {
		symbols = append(symbols, Symbol{Name: src[m[2]:m[3]], Kind: "struct", File: file, Line: lineOf(m[0])})
	}
	for _, m := range rustPubEnumRe.FindAllStringSubmatchIndex(src, -1) {
		symbols = append(symbols, Symbol{Name: src[m[2]:m[3]], Kind: "enum", File: file, Line: lineOf(m[0])})
	}
	for _, m := range rustPubTraitRe.FindAllStringSubmatchIndex(src, -1) {
		symbols = append(symbols, Symbol{Name: src[m[2]:m[3]], Kind: "trait", File: file, Line: lineOf(m[0])})
	}
	for _, m := range rustPubTypeRe.FindAllStringSubmatchIndex(src, -1) {
		symbols = append(symbols, Symbol{Name: src[m[2]:m[3]], Kind: "type", File: file, Line: lineOf(m[0])})
	}

	var refs []string
	matches := identRe.FindAllString(src, -1)
	for _, m := range matches {
		if len(m) > 2 {
			refs = append(refs, m)
		}
	}
	return symbols, refs
}

// ExtractTypeScript parses TypeScript / JavaScript definitions and references.
func ExtractTypeScript(file, src string) ([]Symbol, []string) {
	var symbols []Symbol
	lines := strings.Split(src, "\n")
	lineOf := func(idx int) int {
		c := 0
		for i, line := range lines {
			c += len(line) + 1
			if c > idx {
				return i + 1
			}
		}
		return len(lines)
	}

	for _, m := range tsExportFnRe.FindAllStringSubmatchIndex(src, -1) {
		name := src[m[2]:m[3]]
		sig := ""
		if len(m) >= 6 && m[4] >= 0 {
			sig = src[m[4]:m[5]]
		}
		symbols = append(symbols, Symbol{Name: name, Kind: "func", Signature: sig, File: file, Line: lineOf(m[0])})
	}
	for _, m := range tsExportClassRe.FindAllStringSubmatchIndex(src, -1) {
		symbols = append(symbols, Symbol{Name: src[m[2]:m[3]], Kind: "class", File: file, Line: lineOf(m[0])})
	}
	for _, m := range tsExportInterRe.FindAllStringSubmatchIndex(src, -1) {
		symbols = append(symbols, Symbol{Name: src[m[2]:m[3]], Kind: "interface", File: file, Line: lineOf(m[0])})
	}
	for _, m := range tsExportTypeRe.FindAllStringSubmatchIndex(src, -1) {
		symbols = append(symbols, Symbol{Name: src[m[2]:m[3]], Kind: "type", File: file, Line: lineOf(m[0])})
	}
	for _, m := range tsExportConstRe.FindAllStringSubmatchIndex(src, -1) {
		symbols = append(symbols, Symbol{Name: src[m[2]:m[3]], Kind: "const", File: file, Line: lineOf(m[0])})
	}

	var refs []string
	matches := identRe.FindAllString(src, -1)
	for _, m := range matches {
		if len(m) > 2 {
			refs = append(refs, m)
		}
	}
	return symbols, refs
}

// ExtractPython parses Python definitions and references.
func ExtractPython(file, src string) ([]Symbol, []string) {
	var symbols []Symbol
	lines := strings.Split(src, "\n")
	lineOf := func(idx int) int {
		c := 0
		for i, line := range lines {
			c += len(line) + 1
			if c > idx {
				return i + 1
			}
		}
		return len(lines)
	}

	for _, m := range pyDefRe.FindAllStringSubmatchIndex(src, -1) {
		name := src[m[2]:m[3]]
		if !strings.HasPrefix(name, "_") {
			sig := ""
			if len(m) >= 6 && m[4] >= 0 {
				sig = src[m[4]:m[5]]
			}
			symbols = append(symbols, Symbol{Name: name, Kind: "func", Signature: sig, File: file, Line: lineOf(m[0])})
		}
	}
	for _, m := range pyClassRe.FindAllStringSubmatchIndex(src, -1) {
		name := src[m[2]:m[3]]
		symbols = append(symbols, Symbol{Name: name, Kind: "class", File: file, Line: lineOf(m[0])})
	}

	var refs []string
	matches := identRe.FindAllString(src, -1)
	for _, m := range matches {
		if len(m) > 2 {
			refs = append(refs, m)
		}
	}
	return symbols, refs
}
