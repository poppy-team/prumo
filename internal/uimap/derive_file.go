package uimap

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
)

// readGoFile returns what a source file declares, qualified by directory as
// `dir.Name`. Qualifying by directory rather than by declared package name keeps
// a nested package unambiguous in a map's trace.
//
// Declarations are split by what each name can be used for, because the merge
// step treats them differently: a type or a method can *be* an interface element
// and is worth offering to the author, while a constant is a value the interface
// uses and is not.
func readGoFile(path, dir string) (fileDecls, error) {
	var decls fileDecls
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return fileDecls{}, fmt.Errorf("uimap: parse %s: %w", path, err)
	}
	prefix := filepath.Base(dir)
	if dir == "." || dir == "" {
		prefix = file.Name.Name
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if !ast.IsExported(d.Name.Name) {
				continue
			}
			if d.Recv == nil {
				decls.Symbols = append(decls.Symbols, prefix+"."+d.Name.Name)
				decls.Elements = append(decls.Elements, prefix+"."+d.Name.Name)
				continue
			}
			// Methods matter too: `(*Model).View` is how a UI element actually
			// renders, and a map that could only point at free functions would
			// force authors to invent one.
			if name := receiverName(d); name != "" {
				qualified := prefix + "." + name + "." + d.Name.Name
				decls.Symbols = append(decls.Symbols, qualified)
				decls.Elements = append(decls.Elements, qualified)
			}
		case *ast.GenDecl:
			decls.Symbols, decls.Elements, decls.Constants = readGenDecl(d, prefix, decls.Symbols, decls.Elements, decls.Constants)
		}
	}
	return decls, nil
}

func readGenDecl(d *ast.GenDecl, prefix string, symbols, elements, constants []string) ([]string, []string, []string) {
	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			if !ast.IsExported(s.Name.Name) {
				continue
			}
			symbols = append(symbols, prefix+"."+s.Name.Name)
			elements = append(elements, prefix+"."+s.Name.Name)
			// Exported struct fields are addresses too. A map often needs to
			// point at the field a label renders from, and forcing authors to
			// name the whole struct instead would make the trace less precise
			// than the interface it describes.
			structType, ok := s.Type.(*ast.StructType)
			if !ok {
				continue
			}
			for _, field := range structType.Fields.List {
				for _, name := range field.Names {
					if !ast.IsExported(name.Name) {
						continue
					}
					qualified := prefix + "." + s.Name.Name + "." + name.Name
					symbols = append(symbols, qualified)
					elements = append(elements, qualified)
				}
			}
		case *ast.ValueSpec:
			if d.Tok != token.CONST {
				continue
			}
			for _, name := range s.Names {
				if !ast.IsExported(name.Name) || name.Name == "_" {
					continue
				}
				constants = append(constants, prefix+"."+name.Name)
				symbols = append(symbols, prefix+"."+name.Name)
			}
		}
	}
	return symbols, elements, constants
}

// receiverName is the receiver's type name, pointer or not.
func receiverName(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return ""
	}
	switch t := d.Recv.List[0].Type.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}
