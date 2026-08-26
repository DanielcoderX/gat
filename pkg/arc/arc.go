package arc

import (
	"gat/pkg/ir"
)

type Pass struct {
	prog *ir.Program
}

func NewPass(prog *ir.Program) *Pass {
	return &Pass{prog: prog}
}

func (p *Pass) Run() {
	for _, fn := range p.prog.Functions {
		p.processFunction(fn)
	}
}

func (p *Pass) processFunction(fn *ir.Function) {
	var newInsts []*ir.Instruction

	// Track variables that hold owned class references (need release at scope/function exit)
	ownedLocals := make(map[*ir.Value]bool)

	// Note: Parameters marked with IsBorrowed = true do NOT get added to ownedLocals
	for _, param := range fn.Params {
		if !param.IsBorrowed && param.Type.IsRef() {
			ownedLocals[param] = true
		}
	}

	for i := 0; i < len(fn.Instructions); i++ {
		inst := fn.Instructions[i]

		switch inst.Op {
		case ir.OpAllocHeap:
			// Newly allocated heap object is owned (refcount starts at 1)
			ownedLocals[inst.Dest] = true
			newInsts = append(newInsts, inst)

		case ir.OpCopy:
			if inst.Type != nil && inst.Type.IsRef() {
				// Rebinding: if dest already held an owned reference, release old value before copying
				if ownedLocals[inst.Dest] {
					newInsts = append(newInsts, &ir.Instruction{
						Op:   ir.OpRelease,
						Src1: inst.Dest,
						Type: inst.Type,
					})
				}
				// If Src1 is a fresh temporary from alloc/call, move ownership without retain
				if inst.Src1 != nil && inst.Src1.Name == "" && ownedLocals[inst.Src1] {
					delete(ownedLocals, inst.Src1)
					ownedLocals[inst.Dest] = true
				} else {
					// Otherwise, retain RHS
					newInsts = append(newInsts, &ir.Instruction{
						Op:   ir.OpRetain,
						Src1: inst.Src1,
						Type: inst.Type,
					})
					ownedLocals[inst.Dest] = true
				}
			}
			newInsts = append(newInsts, inst)

		case ir.OpStore:
			// Storing class reference into field: retain stored value
			if inst.Type != nil && inst.Type.IsRef() {
				newInsts = append(newInsts, &ir.Instruction{
					Op:   ir.OpRetain,
					Src1: inst.Src1,
					Type: inst.Type,
				})
			}
			newInsts = append(newInsts, inst)

		case ir.OpCall:
			// Result of function call returning a class is owned by caller
			if inst.Dest != nil && inst.Type != nil && inst.Type.IsRef() {
				ownedLocals[inst.Dest] = true
			}
			newInsts = append(newInsts, inst)

		case ir.OpReturn:
			// Release all owned locals except the one being returned
			retVal := inst.Src1
			for local := range ownedLocals {
				if local != retVal && local.Type != nil && local.Type.IsRef() {
					newInsts = append(newInsts, &ir.Instruction{
						Op:   ir.OpRelease,
						Src1: local,
						Type: local.Type,
					})
				}
			}
			newInsts = append(newInsts, inst)

		default:
			newInsts = append(newInsts, inst)
		}
	}

	fn.Instructions = newInsts
}
