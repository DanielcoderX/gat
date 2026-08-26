package ir

import (
	"fmt"
	"strings"

	"gat/pkg/ast"
	"gat/pkg/typecheck"
	"gat/pkg/types"
)

type Builder struct {
	tc          *typecheck.TypeChecker
	prog        *ast.Program
	irProg      *Program
	curFn       *Function
	valSeq      int
	labelSeq    int
	stringMap   map[string]string
	varMap      map[string]*Value
	constants   map[string]ast.Expr
	scopeStack  []map[string]*Value
}

func NewBuilder(tc *typecheck.TypeChecker, prog *ast.Program) *Builder {
	return &Builder{
		tc:        tc,
		prog:      prog,
		irProg:    &Program{},
		stringMap: make(map[string]string),
		varMap:    make(map[string]*Value),
		constants: make(map[string]ast.Expr),
	}
}

func (b *Builder) Build() *Program {
	// Collect structs, classes and constants
	for _, decl := range b.prog.Decls {
		switch d := decl.(type) {
		case *ast.StructDecl:
			t, _ := b.tc.GetType(d.Name)
			b.irProg.Structs = append(b.irProg.Structs, t.(*types.StructType))
		case *ast.ClassDecl:
			t, _ := b.tc.GetType(d.Name)
			b.irProg.Classes = append(b.irProg.Classes, t.(*types.ClassType))
		case *ast.LetStmt:
			b.constants[d.Name] = d.Value
		}
	}

	// Lower functions
	for _, decl := range b.prog.Decls {
		switch d := decl.(type) {
		case *ast.FnDecl:
			b.buildFn(d)
		case *ast.ClassDecl:
			if d.Deinit != nil {
				b.buildClassDeinit(d)
			}
		}
	}

	return b.irProg
}

func (b *Builder) newValue(t types.Type, name string) *Value {
	b.valSeq++
	v := &Value{
		ID:   ValueID(b.valSeq),
		Name: name,
		Type: t,
	}
	if b.curFn != nil {
		b.curFn.Locals = append(b.curFn.Locals, v)
	}
	return v
}

func (b *Builder) newLabel(prefix string) string {
	b.labelSeq++
	return fmt.Sprintf(".L_%s_%d", prefix, b.labelSeq)
}

func (b *Builder) emit(inst *Instruction) *Instruction {
	b.curFn.Instructions = append(b.curFn.Instructions, inst)
	return inst
}

func (b *Builder) getStringConstant(str string) string {
	if lbl, ok := b.stringMap[str]; ok {
		return lbl
	}
	lbl := fmt.Sprintf("str_%d", len(b.irProg.Strings))
	b.stringMap[str] = lbl
	b.irProg.Strings = append(b.irProg.Strings, StringConstant{
		Label: lbl,
		Value: str,
	})
	return lbl
}

func (b *Builder) pushScope() {
	b.scopeStack = append(b.scopeStack, make(map[string]*Value))
}

func (b *Builder) popScope() {
	if len(b.scopeStack) > 0 {
		b.scopeStack = b.scopeStack[:len(b.scopeStack)-1]
	}
}

func (b *Builder) setScopedVar(name string, v *Value) {
	if len(b.scopeStack) > 0 {
		b.scopeStack[len(b.scopeStack)-1][name] = v
	}
	b.varMap[name] = v
}

func (b *Builder) getVar(name string) *Value {
	for i := len(b.scopeStack) - 1; i >= 0; i-- {
		if v, ok := b.scopeStack[i][name]; ok {
			return v
		}
	}
	return b.varMap[name]
}

func (b *Builder) buildFn(fn *ast.FnDecl) {
	fnType, _ := b.tc.GetFn(fn.Name)
	b.curFn = &Function{
		Name:   fn.Name,
		FnType: fnType,
	}
	b.irProg.Functions = append(b.irProg.Functions, b.curFn)
	b.scopeStack = nil
	b.varMap = make(map[string]*Value)
	b.pushScope()

	for i, p := range fn.Params {
		pt := fnType.ParamTypes[i]
		paramVal := &Value{
			ID:         ValueID(b.valSeq + 1),
			Name:       p.Name,
			Type:       pt,
			IsParam:    true,
			IsBorrowed: pt.IsRef(),
		}
		b.valSeq++
		b.curFn.Params = append(b.curFn.Params, paramVal)
		b.setScopedVar(p.Name, paramVal)
	}

	b.buildBlock(fn.Body)

	// Ensure trailing return for void functions if missing
	if len(b.curFn.Instructions) == 0 || b.curFn.Instructions[len(b.curFn.Instructions)-1].Op != OpReturn {
		if fnType.ReturnType == types.Void {
			b.emit(&Instruction{Op: OpReturn})
		}
	}

	b.popScope()
	b.curFn = nil
}

func (b *Builder) buildClassDeinit(cd *ast.ClassDecl) {
	ct, _ := b.tc.GetType(cd.Name)
	classType := ct.(*types.ClassType)
	deinitFnName := fmt.Sprintf("__deinit_user_%s", cd.Name)

	fnType := &types.FnType{
		Name:       deinitFnName,
		ParamNames: []string{"self"},
		ParamTypes: []types.Type{classType},
		ReturnType: types.Void,
	}

	b.curFn = &Function{
		Name:      deinitFnName,
		FnType:    fnType,
		IsDeinit:  true,
		ClassName: cd.Name,
	}
	b.irProg.Functions = append(b.irProg.Functions, b.curFn)
	b.scopeStack = nil
	b.varMap = make(map[string]*Value)
	b.pushScope()

	selfVal := &Value{
		ID:         ValueID(b.valSeq + 1),
		Name:       "self",
		Type:       classType,
		IsParam:    true,
		IsBorrowed: true,
	}
	b.valSeq++
	b.curFn.Params = append(b.curFn.Params, selfVal)
	b.setScopedVar("self", selfVal)

	b.buildBlock(cd.Deinit)

	if len(b.curFn.Instructions) == 0 || b.curFn.Instructions[len(b.curFn.Instructions)-1].Op != OpReturn {
		b.emit(&Instruction{Op: OpReturn})
	}

	b.popScope()
	b.curFn = nil
}

func (b *Builder) buildBlock(block *ast.BlockStmt) {
	if block == nil {
		return
	}
	b.pushScope()
	for _, stmt := range block.Stmts {
		b.buildStmt(stmt)
	}
	b.popScope()
}

func (b *Builder) buildStmt(stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.LetStmt:
		val := b.buildExpr(s.Value)
		varType := val.Type
		if s.Type != nil {
			declType := b.tc.ResolveTypeNode(s.Type)
			if declType != nil && declType != types.Void {
				varType = declType
			}
		}
		varLocal := b.newValue(varType, s.Name)
		b.emit(&Instruction{
			Op:   OpCopy,
			Dest: varLocal,
			Src1: val,
			Type: varType,
		})
		b.setScopedVar(s.Name, varLocal)

	case *ast.AssignStmt:
		val := b.buildExpr(s.Value)
		switch target := s.Target.(type) {
		case *ast.IdentExpr:
			dest := b.getVar(target.Name)
			b.emit(&Instruction{
				Op:   OpCopy,
				Dest: dest,
				Src1: val,
				Type: dest.Type,
			})
		case *ast.MemberExpr:
			obj := b.buildExpr(target.Target)
			offset := b.getFieldOffset(obj.Type, target.Member)
			b.emit(&Instruction{
				Op:     OpStore,
				Dest:   obj,
				Offset: offset,
				Src1:   val,
				Type:   val.Type,
			})
		case *ast.IndexExpr:
			targetObj := b.buildExpr(target.Target)
			indexVal := b.buildExpr(target.Index)
			b.emit(&Instruction{
				Op:   OpStoreIndex,
				Dest: targetObj,
				Src1: val,
				Src2: indexVal,
				Type: val.Type,
			})
		}

	case *ast.ReturnStmt:
		if s.Value != nil {
			val := b.buildExpr(s.Value)
			b.emit(&Instruction{
				Op:   OpReturn,
				Src1: val,
				Type: val.Type,
			})
		} else {
			b.emit(&Instruction{
				Op: OpReturn,
			})
		}

	case *ast.IfStmt:
		cond := b.buildExpr(s.Condition)
		thenLbl := b.newLabel("then")
		elseLbl := b.newLabel("else")
		endLbl := b.newLabel("endif")

		if s.ElseBranch != nil {
			b.emit(&Instruction{
				Op:         OpBranch,
				Src1:       cond,
				TrueLabel:  thenLbl,
				FalseLabel: elseLbl,
			})
			b.emit(&Instruction{Op: OpLabel, Label: thenLbl})
			b.buildBlock(s.ThenBranch)
			b.emit(&Instruction{Op: OpJump, Label: endLbl})

			b.emit(&Instruction{Op: OpLabel, Label: elseLbl})
			switch eb := s.ElseBranch.(type) {
			case *ast.BlockStmt:
				b.buildBlock(eb)
			case *ast.IfStmt:
				b.buildStmt(eb)
			}
			b.emit(&Instruction{Op: OpLabel, Label: endLbl})
		} else {
			b.emit(&Instruction{
				Op:         OpBranch,
				Src1:       cond,
				TrueLabel:  thenLbl,
				FalseLabel: endLbl,
			})
			b.emit(&Instruction{Op: OpLabel, Label: thenLbl})
			b.buildBlock(s.ThenBranch)
			b.emit(&Instruction{Op: OpLabel, Label: endLbl})
		}

	case *ast.WhileStmt:
		condLbl := b.newLabel("while_cond")
		bodyLbl := b.newLabel("while_body")
		endLbl := b.newLabel("while_end")

		b.emit(&Instruction{Op: OpLabel, Label: condLbl})
		cond := b.buildExpr(s.Condition)
		b.emit(&Instruction{
			Op:         OpBranch,
			Src1:       cond,
			TrueLabel:  bodyLbl,
			FalseLabel: endLbl,
		})

		b.emit(&Instruction{Op: OpLabel, Label: bodyLbl})
		b.buildBlock(s.Body)
		b.emit(&Instruction{Op: OpJump, Label: condLbl})
		b.emit(&Instruction{Op: OpLabel, Label: endLbl})

	case *ast.MatchStmt:
		targetVal := b.buildExpr(s.Expr)
		enumType, _ := targetVal.Type.(*types.EnumType)
		endLbl := b.newLabel("match_end")
		tagVal := b.newValue(types.Int64, "")
		b.emit(&Instruction{Op: OpLoad, Dest: tagVal, Src1: targetVal, Offset: 0, Type: types.Int64})
		for _, arm := range s.Arms {
			if arm.IsWildcard {
				b.buildBlock(arm.Body)
				b.emit(&Instruction{Op: OpJump, Label: endLbl})
				break
			}
			if enumType != nil {
				variant, _, _ := enumType.GetVariant(arm.Variant)
				armLbl := b.newLabel("arm_" + arm.Variant)
				nextLbl := b.newLabel("next_arm_" + arm.Variant)
				expectedTag := b.newValue(types.Int64, "")
				b.emit(&Instruction{Op: OpConstInt, Dest: expectedTag, IntVal: variant.Tag, Type: types.Int64})
				eqVal := b.newValue(types.Bool, "")
				b.emit(&Instruction{Op: OpBinOp, Dest: eqVal, Src1: tagVal, Src2: expectedTag, StrVal: "==", Type: types.Bool})
				b.emit(&Instruction{Op: OpBranch, Src1: eqVal, TrueLabel: armLbl, FalseLabel: nextLbl})
				b.emit(&Instruction{Op: OpLabel, Label: armLbl})
				b.pushScope()
				for i, bName := range arm.Bindings {
					if i < len(variant.PayloadType) {
						pType := variant.PayloadType[i]
						bVal := b.newValue(pType, bName)
						b.emit(&Instruction{Op: OpLoad, Dest: bVal, Src1: targetVal, Offset: 8 + i*8, Type: pType})
						b.setScopedVar(bName, bVal)
					}
				}
				for _, stmt := range arm.Body.Stmts {
					b.buildStmt(stmt)
				}
				b.popScope()
				b.emit(&Instruction{Op: OpJump, Label: endLbl})
				b.emit(&Instruction{Op: OpLabel, Label: nextLbl})
			}
		}
		b.emit(&Instruction{Op: OpLabel, Label: endLbl})

	case *ast.ExprStmt:
		b.buildExpr(s.Expr)
	}
}

func (b *Builder) buildExpr(expr ast.Expr) *Value {
	if expr == nil {
		return nil
	}

	t := b.tc.ExprType(expr)

	switch e := expr.(type) {
	case *ast.IntLitExpr:
		v := b.newValue(types.Int64, "")
		b.emit(&Instruction{
			Op:     OpConstInt,
			Dest:   v,
			IntVal: e.Value,
		})
		return v

	case *ast.BoolLitExpr:
		v := b.newValue(types.Bool, "")
		b.emit(&Instruction{
			Op:      OpConstBool,
			Dest:    v,
			BoolVal: e.Value,
		})
		return v

	case *ast.StringLitExpr:
		lbl := b.getStringConstant(e.Value)
		v := b.newValue(types.String, "")
		b.emit(&Instruction{
			Op:     OpConstString,
			Dest:   v,
			StrVal: lbl,
		})
		return v

	case *ast.NilLitExpr:
		v := b.newValue(types.Nil, "")
		b.emit(&Instruction{
			Op:   OpConstNil,
			Dest: v,
		})
		return v

	case *ast.IdentExpr:
		if c, ok := b.constants[e.Name]; ok {
			return b.buildExpr(c)
		}
		return b.getVar(e.Name)

	case *ast.UnaryExpr:
		right := b.buildExpr(e.Right)
		if e.Op == "raw" {
			// Unsafe address-of / raw pointer conversion
			v := b.newValue(&types.RawType{BaseType: right.Type}, "")
			b.emit(&Instruction{
				Op:     OpUnOp,
				Dest:   v,
				Src1:   right,
				StrVal: "raw",
			})
			return v
		}
		v := b.newValue(t, "")
		b.emit(&Instruction{
			Op:     OpUnOp,
			Dest:   v,
			Src1:   right,
			StrVal: e.Op,
		})
		return v

	case *ast.BinaryExpr:
		left := b.buildExpr(e.Left)
		right := b.buildExpr(e.Right)
		v := b.newValue(t, "")
		b.emit(&Instruction{
			Op:     OpBinOp,
			Dest:   v,
			Src1:   left,
			Src2:   right,
			StrVal: e.Op,
		})
		return v

	case *ast.CallExpr:
		if strings.Contains(e.Callee, ".") {
			parts := strings.Split(e.Callee, ".")
			if len(parts) == 2 {
				if typ, ok := b.tc.GetType(parts[0]); ok {
					if et, ok := typ.(*types.EnumType); ok {
						if variant, _, ok := et.GetVariant(parts[1]); ok {
							v := b.newValue(et, "")
							b.emit(&Instruction{Op: OpAllocStack, Dest: v, Type: et, Offset: et.Size()})
							tagVal := b.newValue(types.Int64, "")
							b.emit(&Instruction{Op: OpConstInt, Dest: tagVal, IntVal: variant.Tag, Type: types.Int64})
							b.emit(&Instruction{Op: OpStore, Dest: v, Offset: 0, Src1: tagVal, Type: types.Int64})
							for i, arg := range e.Args {
								argVal := b.buildExpr(arg)
								b.emit(&Instruction{Op: OpStore, Dest: v, Offset: 8 + i*8, Src1: argVal, Type: argVal.Type})
							}
							return v
						}
					}
				}
			}
		}
		var args []*Value
		for _, arg := range e.Args {
			args = append(args, b.buildExpr(arg))
		}
		v := b.newValue(t, "")
		b.emit(&Instruction{
			Op:     OpCall,
			Dest:   v,
			StrVal: e.Callee,
			Args:   args,
			Type:   t,
		})
		return v

	case *ast.MemberExpr:
		if ident, ok := e.Target.(*ast.IdentExpr); ok {
			if typ, ok := b.tc.GetType(ident.Name); ok {
				if et, ok := typ.(*types.EnumType); ok {
					if variant, _, ok := et.GetVariant(e.Member); ok {
						v := b.newValue(et, "")
						b.emit(&Instruction{Op: OpAllocStack, Dest: v, Type: et, Offset: et.Size()})
						tagVal := b.newValue(types.Int64, "")
						b.emit(&Instruction{Op: OpConstInt, Dest: tagVal, IntVal: variant.Tag, Type: types.Int64})
						b.emit(&Instruction{Op: OpStore, Dest: v, Offset: 0, Src1: tagVal, Type: types.Int64})
						return v
					}
				}
			}
		}
		obj := b.buildExpr(e.Target)
		offset := b.getFieldOffset(obj.Type, e.Member)
		v := b.newValue(t, "")
		b.emit(&Instruction{
			Op:     OpLoad,
			Dest:   v,
			Src1:   obj,
			Offset: offset,
			Type:   t,
		})
		return v

	case *ast.ArrayLitExpr:
		arrType := t.(*types.ArrayType)
		v := b.newValue(arrType, "")
		b.emit(&Instruction{Op: OpAllocStack, Dest: v, Type: arrType, Offset: arrType.Size()})
		elemSize := arrType.ElemType.Size()
		if elemSize < 8 {
			elemSize = 8
		}
		for i, elem := range e.Elements {
			elemVal := b.buildExpr(elem)
			b.emit(&Instruction{Op: OpStore, Dest: v, Offset: i * elemSize, Src1: elemVal, Type: elemVal.Type})
		}
		return v

	case *ast.NewExpr:
		typ, _ := b.tc.GetType(e.TypeName)
		if ct, ok := typ.(*types.ClassType); ok {
			v := b.newValue(ct, "")
			// Alloc heap object: payload size + 16 header
			b.emit(&Instruction{
				Op:     OpAllocHeap,
				Dest:   v,
				Type:   ct,
				Offset: ct.PayloadSize,
			})
			// Initialize fields
			for _, init := range e.FieldInits {
				val := b.buildExpr(init.Value)
				offset := b.getFieldOffset(ct, init.Name)
				b.emit(&Instruction{
					Op:     OpStore,
					Dest:   v,
					Offset: offset,
					Src1:   val,
					Type:   val.Type,
				})
			}
			return v
		} else if st, ok := typ.(*types.StructType); ok {
			v := b.newValue(st, "")
			b.emit(&Instruction{
				Op:     OpAllocStack,
				Dest:   v,
				Type:   st,
				Offset: st.TotalSize,
			})
			for _, init := range e.FieldInits {
				val := b.buildExpr(init.Value)
				offset := b.getFieldOffset(st, init.Name)
				b.emit(&Instruction{
					Op:     OpStore,
					Dest:   v,
					Offset: offset,
					Src1:   val,
					Type:   val.Type,
				})
			}
			return v
		}

	case *ast.PrintExpr:
		var args []*Value
		for _, arg := range e.Args {
			args = append(args, b.buildExpr(arg))
		}
		b.emit(&Instruction{
			Op:   OpPrint,
			Args: args,
		})
		return nil

	case *ast.IndexExpr:
		target := b.buildExpr(e.Target)
		index := b.buildExpr(e.Index)
		v := b.newValue(t, "")
		b.emit(&Instruction{
			Op:   OpLoadIndex,
			Dest: v,
			Src1: target,
			Src2: index,
			Type: t,
		})
		return v
	}

	return nil
}

func (b *Builder) getFieldOffset(t types.Type, fieldName string) int {
	switch typ := t.(type) {
	case *types.ClassType:
		if f, _, ok := typ.GetField(fieldName); ok {
			return f.Offset
		}
	case *types.StructType:
		if f, _, ok := typ.GetField(fieldName); ok {
			return f.Offset
		}
	case *types.RawType:
		if st, ok := typ.BaseType.(*types.StructType); ok {
			if f, _, ok := st.GetField(fieldName); ok {
				return f.Offset
			}
		}
	}
	return 0
}
