// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package mocks

import (
	"go/ast"
	"go/format"
	"go/token"
	"io"
)

const commentHeader = `// This file was generated by github.com/nelsam/hel.  Do not
// edit this code by hand unless you *really* know what you're
// doing.  Expect any changes made manually to be overwritten
// the next time hel regenerates this file.

`

//go:generate hel --type TypeFinder --output mock_type_finder_test.go

type TypeFinder interface {
	ExportedTypes() (types []*ast.TypeSpec)
	Dependencies(inter *ast.InterfaceType) (dependencies []*ast.TypeSpec)
}

type Mocks []Mock

func (m Mocks) Output(pkg string, chanSize int, dest io.Writer) error {
	if _, err := dest.Write([]byte(commentHeader)); err != nil {
		return err
	}
	f := &ast.File{
		Name:  &ast.Ident{Name: pkg},
		Decls: m.decls(chanSize),
	}
	return format.Node(dest, token.NewFileSet(), f)
}

func (m Mocks) PrependLocalPackage(name string) {
	for _, m := range m {
		m.PrependLocalPackage(name)
	}
}

func (m Mocks) SetBlockingReturn(blockingReturn bool) {
	for _, m := range m {
		m.SetBlockingReturn(blockingReturn)
	}
}

func (m Mocks) decls(chanSize int) (decls []ast.Decl) {
	for _, mock := range m {
		decls = append(decls, mock.Ast(chanSize)...)
	}
	return decls
}

func Generate(finder TypeFinder) (Mocks, error) {
	base := finder.ExportedTypes()
	var types []*ast.TypeSpec
	for _, typ := range base {
		types = append(types, typ)
		if inter, ok := typ.Type.(*ast.InterfaceType); ok {
			types = append(types, finder.Dependencies(inter)...)
		}
	}
	m := make(Mocks, 0, len(types))
	for _, typ := range types {
		newMock, err := For(typ)
		if err != nil {
			return nil, err
		}
		m = append(m, newMock)
	}
	return m, nil
}
