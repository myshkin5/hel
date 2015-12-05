package mocks

import (
	"go/ast"
	"go/format"
	"go/token"
	"io"
)

type TypeFinder interface {
	ExportedTypes() []*ast.TypeSpec
}

type Mocks []Mock

func (m Mocks) Output(pkg string, chanSize int, dest io.Writer) error {
	f := &ast.File{
		Name:  &ast.Ident{Name: pkg},
		Decls: m.decls(chanSize),
	}
	return format.Node(dest, token.NewFileSet(), f)
}

func (m Mocks) decls(chanSize int) (decls []ast.Decl) {
	for _, mock := range m {
		decls = append(decls, mock.Ast(chanSize)...)
	}
	return decls
}

func Generate(finder TypeFinder) (Mocks, error) {
	types := finder.ExportedTypes()
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