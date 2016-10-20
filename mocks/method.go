// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package mocks

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
	"unicode"
)

const (
	inputFmt     = "arg%d"
	outputFmt    = "ret%d"
	receiverName = "m"
)

type Method struct {
	receiver   Mock
	name       string
	implements *ast.FuncType
}

func MethodFor(receiver Mock, name string, typ *ast.FuncType) Method {
	return Method{
		receiver:   receiver,
		name:       name,
		implements: typ,
	}
}

func (m Method) Ast() *ast.FuncDecl {
	f := &ast.FuncDecl{}
	f.Name = &ast.Ident{Name: m.name}
	f.Type = m.implements
	f.Recv = m.recv()
	f.Body = m.body()
	return f
}

func (m Method) recv() *ast.FieldList {
	return &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: receiverName}},
				Type: &ast.StarExpr{
					X: &ast.Ident{Name: m.receiver.Name()},
				},
			},
		},
	}
}

func (m Method) sendOn(receiver string, fields ...string) *ast.SendStmt {
	return &ast.SendStmt{Chan: selectors(receiver, fields...)}
}

func (m Method) called() ast.Stmt {
	stmt := m.sendOn(receiverName, m.name+"Called")
	stmt.Value = &ast.Ident{Name: "true"}
	return stmt
}

func (m Method) params() []*ast.Field {
	for idx, f := range m.implements.Params.List {
		if f.Names == nil {
			// altering the field directly is okay here, since it's needed anyway
			f.Names = []*ast.Ident{{Name: fmt.Sprintf(inputFmt, idx)}}
		}
		if f.Names[0].Name == receiverName {
			f.Names[0].Name += "_"
		}
	}
	return m.implements.Params.List
}

func (m Method) results() []*ast.Field {
	if m.implements.Results == nil {
		if !*m.receiver.blockingReturn {
			return nil
		}
		return []*ast.Field{
			{
				Names: []*ast.Ident{
					{Name: "blockReturn"},
				},
				Type: &ast.Ident{Name: "bool"},
			},
		}
	}
	fields := make([]*ast.Field, 0, len(m.implements.Results.List))
	for idx, f := range m.implements.Results.List {
		if f.Names == nil {
			// to avoid changing the method definition, make a copy
			copy := *f
			f = &copy
			f.Names = []*ast.Ident{{Name: fmt.Sprintf(outputFmt, idx)}}
		}
		fields = append(fields, f)
	}
	return fields
}

func (m Method) inputs() (stmts []ast.Stmt) {
	for _, input := range m.params() {
		for _, n := range input.Names {
			// Undo our hack to avoid name collisions with the receiver.
			name := n.Name
			if name == receiverName+"_" {
				name = receiverName
			}
			stmt := m.sendOn(receiverName, m.name+"Input", strings.Title(name))
			stmt.Value = &ast.Ident{Name: n.Name}
			stmts = append(stmts, stmt)
		}
	}
	return stmts
}

func (m Method) PrependLocalPackage(name string) {
	m.prependPackage(name, m.implements.Results)
	m.prependPackage(name, m.implements.Params)
}

func (m Method) prependPackage(name string, fields *ast.FieldList) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		field.Type = m.prependTypePackage(name, field.Type)
	}
}

func (m Method) prependTypePackage(name string, typ ast.Expr) ast.Expr {
	switch src := typ.(type) {
	case *ast.Ident:
		if !unicode.IsUpper(rune(src.String()[0])) {
			// Assume a built-in type, at least for now
			return src
		}
		return selectors(name, src.String())
	case *ast.FuncType:
		m.prependPackage(name, src.Params)
		m.prependPackage(name, src.Results)
		return src
	case *ast.ArrayType:
		src.Elt = m.prependTypePackage(name, src.Elt)
		return src
	case *ast.MapType:
		src.Key = m.prependTypePackage(name, src.Key)
		src.Value = m.prependTypePackage(name, src.Value)
		return src
	default:
		return typ
	}
}

func (m Method) recvFrom(receiver string, fields ...string) *ast.UnaryExpr {
	return &ast.UnaryExpr{Op: token.ARROW, X: selectors(receiver, fields...)}
}

func (m Method) returnsExprs() (exprs []ast.Expr) {
	for _, output := range m.results() {
		for _, name := range output.Names {
			exprs = append(exprs, m.recvFrom(receiverName, m.name+"Output", strings.Title(name.String())))
		}
	}
	return exprs
}

func (m Method) returns() ast.Stmt {
	if m.implements.Results == nil {
		if !*m.receiver.blockingReturn {
			return nil
		}
		return &ast.ExprStmt{X: m.returnsExprs()[0]}
	}
	return &ast.ReturnStmt{Results: m.returnsExprs()}
}

func (m Method) body() *ast.BlockStmt {
	stmts := []ast.Stmt{m.called()}
	stmts = append(stmts, m.inputs()...)
	if returnStmt := m.returns(); returnStmt != nil {
		stmts = append(stmts, m.returns())
	}
	return &ast.BlockStmt{
		List: stmts,
	}
}
