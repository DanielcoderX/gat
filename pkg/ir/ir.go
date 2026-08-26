package ir

import (
	"fmt"
	"gat/pkg/types"
)

type ValueID int

type Value struct {
	ID         ValueID
	Name       string
	Type       types.Type
	IsParam    bool
	IsBorrowed bool
}

func (v *Value) String() string {
	if v.Name != "" {
		return fmt.Sprintf("%%%s(%s)", v.Name, v.Type)
	}
	return fmt.Sprintf("%%v%d(%s)", v.ID, v.Type)
}

type OpKind int

const (
	OpAllocHeap OpKind = iota
	OpAllocStack
	OpLoad
	OpStore
	OpLoadIndex
	OpStoreIndex
	OpCopy
	OpConstInt
	OpConstString
	OpConstBool
	OpConstNil
	OpBinOp
	OpUnOp
	OpCall
	OpPrint
	OpRetain
	OpRelease
	OpReturn
	OpBranch
	OpJump
	OpLabel
)

type Instruction struct {
	Op        OpKind
	Dest      *Value
	Src1      *Value
	Src2      *Value
	Args      []*Value
	IntVal    int64
	StrVal    string
	BoolVal   bool
	Offset    int
	Type      types.Type
	Label     string
	TrueLabel string
	FalseLabel string
}

func (inst *Instruction) String() string {
	switch inst.Op {
	case OpAllocHeap:
		return fmt.Sprintf("  %s = alloc_heap %s (%d bytes)", inst.Dest, inst.Type, inst.Offset)
	case OpAllocStack:
		return fmt.Sprintf("  %s = alloc_stack %s (%d bytes)", inst.Dest, inst.Type, inst.Offset)
	case OpLoad:
		return fmt.Sprintf("  %s = load %s[%d]", inst.Dest, inst.Src1, inst.Offset)
	case OpStore:
		return fmt.Sprintf("  store %s[%d] = %s", inst.Dest, inst.Offset, inst.Src1)
	case OpLoadIndex:
		return fmt.Sprintf("  %s = load_index %s[%s]", inst.Dest, inst.Src1, inst.Src2)
	case OpStoreIndex:
		return fmt.Sprintf("  store_index %s[%s] = %s", inst.Dest, inst.Src2, inst.Src1)
	case OpCopy:
		return fmt.Sprintf("  %s = copy %s", inst.Dest, inst.Src1)
	case OpConstInt:
		return fmt.Sprintf("  %s = const_i64 %d", inst.Dest, inst.IntVal)
	case OpConstString:
		return fmt.Sprintf("  %s = const_str %q", inst.Dest, inst.StrVal)
	case OpConstBool:
		return fmt.Sprintf("  %s = const_bool %v", inst.Dest, inst.BoolVal)
	case OpConstNil:
		return fmt.Sprintf("  %s = const_nil", inst.Dest)
	case OpBinOp:
		return fmt.Sprintf("  %s = %s %s, %s", inst.Dest, inst.StrVal, inst.Src1, inst.Src2)
	case OpUnOp:
		return fmt.Sprintf("  %s = %s %s", inst.Dest, inst.StrVal, inst.Src1)
	case OpCall:
		return fmt.Sprintf("  %s = call %s(%v)", inst.Dest, inst.StrVal, inst.Args)
	case OpPrint:
		return fmt.Sprintf("  print(%v)", inst.Args)
	case OpRetain:
		return fmt.Sprintf("  retain %s", inst.Src1)
	case OpRelease:
		return fmt.Sprintf("  release %s", inst.Src1)
	case OpReturn:
		if inst.Src1 != nil {
			return fmt.Sprintf("  return %s", inst.Src1)
		}
		return "  return"
	case OpBranch:
		return fmt.Sprintf("  br %s ? %s : %s", inst.Src1, inst.TrueLabel, inst.FalseLabel)
	case OpJump:
		return fmt.Sprintf("  jmp %s", inst.Label)
	case OpLabel:
		return fmt.Sprintf("%s:", inst.Label)
	default:
		return "  unknown"
	}
}

type Function struct {
	Name         string
	FnType       *types.FnType
	Params       []*Value
	Locals       []*Value
	Instructions []*Instruction
	IsDeinit     bool
	ClassName    string
}

type StringConstant struct {
	Label string
	Value string
}

type Program struct {
	Classes   []*types.ClassType
	Structs   []*types.StructType
	Functions []*Function
	Strings   []StringConstant
}
