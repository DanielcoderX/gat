package typecheck

import (
	"fmt"
	"strings"

	"gat/pkg/ast"
	"gat/pkg/types"
)

type Symbol struct {
	Name       string
	Type       types.Type
	IsParam    bool
	IsBorrowed bool
}

type Scope struct {
	parent  *Scope
	symbols map[string]*Symbol
}

func NewScope(parent *Scope) *Scope {
	return &Scope{
		parent:  parent,
		symbols: make(map[string]*Symbol),
	}
}

func (s *Scope) Define(name string, t types.Type, isParam, isBorrowed bool) *Symbol {
	sym := &Symbol{
		Name:       name,
		Type:       t,
		IsParam:    isParam,
		IsBorrowed: isBorrowed,
	}
	s.symbols[name] = sym
	return sym
}

func (s *Scope) Lookup(name string) (*Symbol, bool) {
	if sym, ok := s.symbols[name]; ok {
		return sym, true
	}
	if s.parent != nil {
		return s.parent.Lookup(name)
	}
	return nil, false
}

type TypeChecker struct {
	typesTable   map[string]types.Type
	fnTable      map[string]*types.FnType
	classDecls   map[string]*ast.ClassDecl
	structDecls  map[string]*ast.StructDecl
	fnDecls      map[string]*ast.FnDecl
	exprTypes    map[ast.Expr]types.Type
	globalScope  *Scope
	currentScope *Scope
	currentFn    *ast.FnDecl
	errors       []string
	typeIdSeq    int64
}

func New() *TypeChecker {
	tc := &TypeChecker{
		typesTable:  make(map[string]types.Type),
		fnTable:     make(map[string]*types.FnType),
		classDecls:  make(map[string]*ast.ClassDecl),
		structDecls: make(map[string]*ast.StructDecl),
		fnDecls:     make(map[string]*ast.FnDecl),
		exprTypes:   make(map[ast.Expr]types.Type),
		globalScope: NewScope(nil),
		errors:      []string{},
		typeIdSeq:   1,
	}
	tc.currentScope = tc.globalScope

	// Builtin types
	tc.typesTable["i8"] = types.Int8
	tc.typesTable["i64"] = types.Int64
	tc.typesTable["bool"] = types.Bool
	tc.typesTable["void"] = types.Void
	tc.typesTable["string"] = types.String

	// Builtin functions
	tc.fnTable["read_file"] = &types.FnType{Name: "read_file", ParamTypes: []types.Type{types.String}, ReturnType: types.String}
	tc.fnTable["write_file"] = &types.FnType{Name: "write_file", ParamTypes: []types.Type{types.String, types.String, types.Int64}, ReturnType: types.Int64}
	tc.fnTable["alloc_mem"] = &types.FnType{Name: "alloc_mem", ParamTypes: []types.Type{types.Int64}, ReturnType: &types.RawType{BaseType: types.Int8}}
	tc.fnTable["free_mem"] = &types.FnType{Name: "free_mem", ParamTypes: []types.Type{&types.RawType{BaseType: types.Int8}}, ReturnType: types.Void}
	tc.fnTable["str_len"] = &types.FnType{Name: "str_len", ParamTypes: []types.Type{types.String}, ReturnType: types.Int64}
	tc.fnTable["str_eq"] = &types.FnType{Name: "str_eq", ParamTypes: []types.Type{types.String, types.String}, ReturnType: types.Bool}
	tc.fnTable["str_char"] = &types.FnType{Name: "str_char", ParamTypes: []types.Type{types.String, types.Int64}, ReturnType: types.Int64}
	tc.fnTable["str_sub"] = &types.FnType{Name: "str_sub", ParamTypes: []types.Type{types.String, types.Int64, types.Int64}, ReturnType: types.String}
	tc.fnTable["str_concat"] = &types.FnType{Name: "str_concat", ParamTypes: []types.Type{types.String, types.String}, ReturnType: types.String}
	tc.fnTable["str_from_int"] = &types.FnType{Name: "str_from_int", ParamTypes: []types.Type{types.Int64}, ReturnType: types.String}
	tc.fnTable["get_cmd_arg"] = &types.FnType{Name: "get_cmd_arg", ParamTypes: []types.Type{types.Int64}, ReturnType: types.String}

	return tc
}

func (tc *TypeChecker) Errors() []string {
	return tc.errors
}

func (tc *TypeChecker) ExprType(e ast.Expr) types.Type {
	return tc.exprTypes[e]
}

func (tc *TypeChecker) GetType(name string) (types.Type, bool) {
	t, ok := tc.typesTable[name]
	return t, ok
}

func (tc *TypeChecker) GetFn(name string) (*types.FnType, bool) {
	t, ok := tc.fnTable[name]
	return t, ok
}

func (tc *TypeChecker) CheckProgram(prog *ast.Program) bool {
	// Pass 1: Register all struct and class type stubs and global let
	for _, decl := range prog.Decls {
		switch d := decl.(type) {
		case *ast.StructDecl:
			tc.structDecls[d.Name] = d
			tc.typesTable[d.Name] = &types.StructType{Name: d.Name}
		case *ast.ClassDecl:
			tc.classDecls[d.Name] = d
			tc.typesTable[d.Name] = &types.ClassType{
				Name:      d.Name,
				TypeId:    tc.typeIdSeq,
				HasDeinit: d.Deinit != nil,
			}
			tc.typeIdSeq++
		case *ast.EnumDecl:
			tc.typesTable[d.Name] = &types.EnumType{Name: d.Name}
		case *ast.FnDecl:
			tc.fnDecls[d.Name] = d
		case *ast.LetStmt:
			var typ types.Type
			if d.Type != nil {
				typ = tc.resolveTypeNode(d.Type)
			}
			if d.Value != nil {
				valType := tc.checkExpr(d.Value)
				if typ == nil || typ == types.Void {
					typ = valType
				}
			}
			tc.globalScope.Define(d.Name, typ, false, false)
		}
	}

	// Pass 2: Resolve struct & class fields and layouts
	for _, decl := range prog.Decls {
		switch d := decl.(type) {
		case *ast.StructDecl:
			tc.resolveStructLayout(d)
		case *ast.ClassDecl:
			tc.resolveClassLayout(d)
		case *ast.EnumDecl:
			tc.resolveEnumLayout(d)
		}
	}

	// Pass 3: Register function signatures
	for _, decl := range prog.Decls {
		if fn, ok := decl.(*ast.FnDecl); ok {
			tc.registerFnSignature(fn)
		}
	}

	// Pass 4: Check function bodies and class deinit bodies
	for _, decl := range prog.Decls {
		switch d := decl.(type) {
		case *ast.FnDecl:
			tc.checkFn(d)
		case *ast.ClassDecl:
			if d.Deinit != nil {
				tc.checkClassDeinit(d)
			}
		}
	}

	return len(tc.errors) == 0
}

func (tc *TypeChecker) ResolveTypeNode(tn ast.TypeNode) types.Type {
	return tc.resolveTypeNode(tn)
}

func (tc *TypeChecker) resolveTypeNode(tn ast.TypeNode) types.Type {
	if tn == nil {
		return types.Void
	}
	switch t := tn.(type) {
	case *ast.NamedType:
		if typ, ok := tc.typesTable[t.Name]; ok {
			return typ
		}
		tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] unknown type %q", t.Pos().Line, t.Pos().Col, t.Name))
		return types.Void
	case *ast.RawType:
		base := tc.resolveTypeNode(t.BaseType)
		return &types.RawType{BaseType: base}
	case *ast.ArrayType:
		elem := tc.resolveTypeNode(t.ElemType)
		return &types.ArrayType{ElemType: elem, Length: t.Length}
	case *ast.SliceType:
		elem := tc.resolveTypeNode(t.ElemType)
		return &types.SliceType{ElemType: elem}
	default:
		return types.Void
	}
}

func (tc *TypeChecker) resolveEnumLayout(d *ast.EnumDecl) {
	et := tc.typesTable[d.Name].(*types.EnumType)
	maxPayload := 0
	for _, v := range d.Variants {
		var payloadTypes []types.Type
		payloadSize := 0
		for _, vt := range v.Types {
			t := tc.resolveTypeNode(vt)
			payloadTypes = append(payloadTypes, t)
			payloadSize += t.Size()
			if payloadSize%8 != 0 {
				payloadSize += 8 - (payloadSize % 8)
			}
		}
		if payloadSize > maxPayload {
			maxPayload = payloadSize
		}
		et.Variants = append(et.Variants, types.EnumVariantInfo{
			Name:        v.Name,
			Tag:         v.Tag,
			PayloadType: payloadTypes,
		})
	}
	et.PayloadSize = maxPayload
}

func (tc *TypeChecker) resolveStructLayout(d *ast.StructDecl) {
	st := tc.typesTable[d.Name].(*types.StructType)
	offset := 0
	for _, f := range d.Fields {
		ft := tc.resolveTypeNode(f.Type)
		st.Fields = append(st.Fields, types.FieldInfo{
			Name:   f.Name,
			Type:   ft,
			Offset: offset,
		})
		offset += ft.Size()
		// align to 8 bytes
		if offset%8 != 0 {
			offset += 8 - (offset % 8)
		}
	}
	st.TotalSize = offset
}

func (tc *TypeChecker) resolveClassLayout(d *ast.ClassDecl) {
	ct := tc.typesTable[d.Name].(*types.ClassType)
	offset := 0 // offset from user pointer
	for _, f := range d.Fields {
		ft := tc.resolveTypeNode(f.Type)
		ct.Fields = append(ct.Fields, types.FieldInfo{
			Name:   f.Name,
			Type:   ft,
			Offset: offset,
		})
		offset += ft.Size()
		// align to 8 bytes
		if offset%8 != 0 {
			offset += 8 - (offset % 8)
		}
	}
	ct.PayloadSize = offset
}

func (tc *TypeChecker) registerFnSignature(fn *ast.FnDecl) {
	retType := tc.resolveTypeNode(fn.ReturnType)
	var pNames []string
	var pTypes []types.Type
	for _, p := range fn.Params {
		pt := tc.resolveTypeNode(p.Type)
		pNames = append(pNames, p.Name)
		pTypes = append(pTypes, pt)
	}

	fnType := &types.FnType{
		Name:       fn.Name,
		ParamNames: pNames,
		ParamTypes: pTypes,
		ReturnType: retType,
	}
	tc.fnTable[fn.Name] = fnType
}

func (tc *TypeChecker) checkFn(fn *ast.FnDecl) {
	tc.currentFn = fn
	fnType := tc.fnTable[fn.Name]
	tc.currentScope = NewScope(tc.globalScope)

	// In Gat, parameters are borrowed by default
	for i, p := range fn.Params {
		pType := fnType.ParamTypes[i]
		isBorrowed := pType.IsRef()
		tc.currentScope.Define(p.Name, pType, true, isBorrowed)
	}

	tc.checkBlock(fn.Body)
	tc.currentFn = nil
}

func (tc *TypeChecker) checkClassDeinit(cd *ast.ClassDecl) {
	ct := tc.typesTable[cd.Name].(*types.ClassType)
	tc.currentScope = NewScope(tc.globalScope)
	// inside deinit, `self` is available as borrowed reference
	tc.currentScope.Define("self", ct, true, true)
	tc.checkBlock(cd.Deinit)
}

func (tc *TypeChecker) checkBlock(block *ast.BlockStmt) {
	parentScope := tc.currentScope
	tc.currentScope = NewScope(parentScope)
	for _, stmt := range block.Stmts {
		tc.checkStmt(stmt)
	}
	tc.currentScope = parentScope
}

func (tc *TypeChecker) checkStmt(stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.LetStmt:
		valType := tc.checkExpr(s.Value)
		var declType types.Type
		if s.Type != nil {
			declType = tc.resolveTypeNode(s.Type)
			if !types.Equal(declType, valType) {
				tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] cannot assign %s to variable of type %s",
					s.Pos().Line, s.Pos().Col, valType, declType))
			}
		} else {
			declType = valType
		}
		tc.currentScope.Define(s.Name, declType, false, false)

	case *ast.AssignStmt:
		targetType := tc.checkExpr(s.Target)
		valType := tc.checkExpr(s.Value)
		if !types.Equal(targetType, valType) {
			tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] assignment type mismatch: %s vs %s",
				s.Pos().Line, s.Pos().Col, targetType, valType))
		}

	case *ast.ReturnStmt:
		var retType types.Type = types.Void
		if s.Value != nil {
			retType = tc.checkExpr(s.Value)
		}
		if tc.currentFn != nil {
			expected := tc.fnTable[tc.currentFn.Name].ReturnType
			if !types.Equal(expected, retType) {
				tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] return type mismatch: expected %s, got %s",
					s.Pos().Line, s.Pos().Col, expected, retType))
			}
		}

	case *ast.IfStmt:
		condType := tc.checkExpr(s.Condition)
		if condType != types.Bool && condType != types.Int64 {
			tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] if condition must be bool or int, got %s",
				s.Pos().Line, s.Pos().Col, condType))
		}
		tc.checkBlock(s.ThenBranch)
		if s.ElseBranch != nil {
			switch eb := s.ElseBranch.(type) {
			case *ast.BlockStmt:
				tc.checkBlock(eb)
			case *ast.IfStmt:
				tc.checkStmt(eb)
			}
		}

	case *ast.WhileStmt:
		condType := tc.checkExpr(s.Condition)
		if condType != types.Bool && condType != types.Int64 {
			tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] while condition must be bool or int, got %s",
				s.Pos().Line, s.Pos().Col, condType))
		}
		tc.checkBlock(s.Body)

	case *ast.MatchStmt:
		targetType := tc.checkExpr(s.Expr)
		enumType, isEnum := targetType.(*types.EnumType)
		if !isEnum {
			tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] match target must be an enum, got %s", s.Pos().Line, s.Pos().Col, targetType))
		}
		for _, arm := range s.Arms {
			if arm.IsWildcard {
				tc.checkBlock(arm.Body)
				continue
			}
			if isEnum {
				v, _, ok := enumType.GetVariant(arm.Variant)
				if !ok {
					tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] enum %s has no variant %s", arm.Position.Line, arm.Position.Col, enumType.Name, arm.Variant))
				} else {
					prevScope := tc.currentScope
					tc.currentScope = NewScope(prevScope)
					for i, bName := range arm.Bindings {
						if i < len(v.PayloadType) {
							tc.currentScope.Define(bName, v.PayloadType[i], false, false)
						}
					}
					for _, stmt := range arm.Body.Stmts {
						tc.checkStmt(stmt)
					}
					tc.currentScope = prevScope
				}
			}
		}

	case *ast.ExprStmt:
		tc.checkExpr(s.Expr)
	}
}

func (tc *TypeChecker) checkExpr(expr ast.Expr) types.Type {
	if expr == nil {
		return types.Void
	}

	var typ types.Type

	switch e := expr.(type) {
	case *ast.IntLitExpr:
		typ = types.Int64
	case *ast.BoolLitExpr:
		typ = types.Bool
	case *ast.StringLitExpr:
		typ = types.String
	case *ast.NilLitExpr:
		typ = types.Nil

	case *ast.IdentExpr:
		if sym, ok := tc.currentScope.Lookup(e.Name); ok {
			typ = sym.Type
		} else {
			tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] undefined identifier %q", e.Pos().Line, e.Pos().Col, e.Name))
			typ = types.Void
		}

	case *ast.UnaryExpr:
		rightType := tc.checkExpr(e.Right)
		if e.Op == "raw" {
			typ = &types.RawType{BaseType: rightType}
		} else if e.Op == "!" {
			typ = types.Bool
		} else if e.Op == "-" {
			typ = types.Int64
		} else {
			typ = rightType
		}

	case *ast.BinaryExpr:
		lt := tc.checkExpr(e.Left)
		rt := tc.checkExpr(e.Right)
		switch e.Op {
		case "+", "-", "*", "/", "%":
			if !isIntType(lt) || !isIntType(rt) {
				tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] arithmetic operator %q requires integer type", e.Pos().Line, e.Pos().Col, e.Op))
			}
			typ = types.Int64
		case "==", "!=":
			if !types.Equal(lt, rt) && !(isIntType(lt) && isIntType(rt)) {
				tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] comparison type mismatch %s %s %s", e.Pos().Line, e.Pos().Col, lt, e.Op, rt))
			}
			typ = types.Bool
		case "<", "<=", ">", ">=":
			typ = types.Bool
		case "&&", "||":
			typ = types.Bool
		default:
			typ = types.Void
		}

	case *ast.CallExpr:
		if strings.Contains(e.Callee, ".") {
			parts := strings.Split(e.Callee, ".")
			if len(parts) == 2 {
				if et, ok := tc.typesTable[parts[0]].(*types.EnumType); ok {
					if v, _, ok := et.GetVariant(parts[1]); ok {
						for i, arg := range e.Args {
							at := tc.checkExpr(arg)
							if i < len(v.PayloadType) {
								pt := v.PayloadType[i]
								if !types.Equal(pt, at) {
									tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] variant %s arg %d expected %s, got %s",
										arg.Pos().Line, arg.Pos().Col, parts[1], i+1, pt, at))
								}
							}
						}
						return et
					}
				}
			}
		}
		if fnType, ok := tc.fnTable[e.Callee]; ok {
			if len(e.Args) != len(fnType.ParamTypes) {
				tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] call to %s expects %d args, got %d",
					e.Pos().Line, e.Pos().Col, e.Callee, len(fnType.ParamTypes), len(e.Args)))
			} else {
				for i, arg := range e.Args {
					at := tc.checkExpr(arg)
					pt := fnType.ParamTypes[i]
					if !types.Equal(pt, at) {
						if (pt.Kind() == types.KindString && isRawI8(at)) || (isRawI8(pt) && at.Kind() == types.KindString) {
							// compatible
						} else {
							tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] arg %d of %s: expected %s, got %s",
								arg.Pos().Line, arg.Pos().Col, i+1, e.Callee, pt, at))
						}
					}
				}
			}
			typ = fnType.ReturnType
		} else {
			tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] undefined function %q", e.Pos().Line, e.Pos().Col, e.Callee))
			typ = types.Void
		}

	case *ast.MemberExpr:
		if ident, ok := e.Target.(*ast.IdentExpr); ok {
			if et, ok := tc.typesTable[ident.Name].(*types.EnumType); ok {
				if _, _, ok := et.GetVariant(e.Member); ok {
					return et
				}
			}
		}
		targetType := tc.checkExpr(e.Target)
		switch t := targetType.(type) {
		case *types.StructType:
			if f, _, ok := t.GetField(e.Member); ok {
				typ = f.Type
			} else {
				tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] struct %s has no field %s", e.Pos().Line, e.Pos().Col, t.Name, e.Member))
				typ = types.Void
			}
		case *types.ClassType:
			if f, _, ok := t.GetField(e.Member); ok {
				typ = f.Type
			} else {
				tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] class %s has no field %s", e.Pos().Line, e.Pos().Col, t.Name, e.Member))
				typ = types.Void
			}
		case *types.RawType:
			if st, ok := t.BaseType.(*types.StructType); ok {
				if f, _, ok := st.GetField(e.Member); ok {
					typ = f.Type
				} else {
					tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] raw struct %s has no field %s", e.Pos().Line, e.Pos().Col, st.Name, e.Member))
					typ = types.Void
				}
			} else {
				tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] member access on invalid raw type %s", e.Pos().Line, e.Pos().Col, t))
				typ = types.Void
			}
		default:
			tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] member access on non-aggregate type %s", e.Pos().Line, e.Pos().Col, targetType))
			typ = types.Void
		}

	case *ast.ArrayLitExpr:
		if len(e.Elements) == 0 {
			typ = &types.ArrayType{ElemType: types.Int64, Length: 0}
		} else {
			elemType := tc.checkExpr(e.Elements[0])
			for i := 1; i < len(e.Elements); i++ {
				it := tc.checkExpr(e.Elements[i])
				if !types.Equal(elemType, it) {
					tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] array element type mismatch: expected %s, got %s",
						e.Elements[i].Pos().Line, e.Elements[i].Pos().Col, elemType, it))
				}
			}
			typ = &types.ArrayType{ElemType: elemType, Length: int64(len(e.Elements))}
		}

	case *ast.IndexExpr:
		targetType := tc.checkExpr(e.Target)
		indexType := tc.checkExpr(e.Index)
		if !isIntType(indexType) {
			tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] index must be integer, got %s", e.Index.Pos().Line, e.Index.Pos().Col, indexType))
		}
		if arr, ok := targetType.(*types.ArrayType); ok {
			typ = arr.ElemType
		} else if sl, ok := targetType.(*types.SliceType); ok {
			typ = sl.ElemType
		} else if raw, ok := targetType.(*types.RawType); ok {
			typ = raw.BaseType
		} else if targetType == types.String {
			typ = types.Int64
		} else {
			tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] index operator on non-indexable type %s", e.Target.Pos().Line, e.Target.Pos().Col, targetType))
			typ = types.Void
		}

	case *ast.NewExpr:
		t, ok := tc.typesTable[e.TypeName]
		if !ok {
			tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] unknown type in new expr %q", e.Pos().Line, e.Pos().Col, e.TypeName))
			typ = types.Void
		} else {
			typ = t
			// check field inits
			for _, init := range e.FieldInits {
				it := tc.checkExpr(init.Value)
				if ct, ok := t.(*types.ClassType); ok {
					if f, _, found := ct.GetField(init.Name); found {
						if !types.Equal(f.Type, it) {
							tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] field %s expected %s, got %s", init.Value.Pos().Line, init.Value.Pos().Col, init.Name, f.Type, it))
						}
					}
				} else if st, ok := t.(*types.StructType); ok {
					if f, _, found := st.GetField(init.Name); found {
						if !types.Equal(f.Type, it) {
							tc.errors = append(tc.errors, fmt.Sprintf("[%d:%d] field %s expected %s, got %s", init.Value.Pos().Line, init.Value.Pos().Col, init.Name, f.Type, it))
						}
					}
				}
			}
		}

	case *ast.PrintExpr:
		for _, a := range e.Args {
			tc.checkExpr(a)
		}
		typ = types.Void
	}

	tc.exprTypes[expr] = typ
	return typ
}

func isIntType(t types.Type) bool {
	if t == nil {
		return false
	}
	return t == types.Int64 || t == types.Int8
}

func isRawI8(t types.Type) bool {
	if t == nil {
		return false
	}
	if raw, ok := t.(*types.RawType); ok {
		return raw.BaseType == types.Int8
	}
	return false
}
