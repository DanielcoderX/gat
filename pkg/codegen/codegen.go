package codegen

import (
	"encoding/binary"
	"fmt"

	"gat/pkg/ir"
	"gat/pkg/types"
)

type RelocKind int

const (
	RelocRipRelative RelocKind = iota // 32-bit RIP-relative displacement
	RelocAbs64                        // 64-bit absolute address
	RelocIATRelative                  // 32-bit RIP-relative to IAT entry
)

type Relocation struct {
	Offset int       // byte offset in code/data section
	Symbol string    // symbol name
	Kind   RelocKind // relocation type
	Addend int64
}

type CodeGenerator struct {
	prog           *ir.Program
	code           []byte
	relocs         []Relocation
	symOffsets     map[string]int
	strOffsets     map[string]int
	localStack     map[*ir.Value]int // stack offset from RBP (negative)
	structBuffers  map[*ir.Value]int // buffer stack offset for value structs
	stackFrameSize int
	labelOffsets   map[string]int
	pendingJumps   []pendingJump
}

type pendingJump struct {
	codeOffset int
	label      string
}

func New(prog *ir.Program) *CodeGenerator {
	return &CodeGenerator{
		prog:          prog,
		symOffsets:    make(map[string]int),
		strOffsets:    make(map[string]int),
		localStack:    make(map[*ir.Value]int),
		structBuffers: make(map[*ir.Value]int),
		labelOffsets:  make(map[string]int),
	}
}

func (cg *CodeGenerator) Generate() ([]byte, []Relocation, map[string]int) {
	// 1. Generate runtime helpers (__gat_alloc_heap, __gat_retain, __gat_release, __gat_print_str, __gat_print_i64, __gat_deinit_dispatch)
	cg.emitRuntimeHelpers()

	// 2. Generate user functions
	for _, fn := range cg.prog.Functions {
		cg.emitFunction(fn)
	}

	// 3. Resolve all pending label jumps
	for _, pj := range cg.pendingJumps {
		targetOffset, ok := cg.labelOffsets[pj.label]
		if !ok {
			panic(fmt.Sprintf("unresolved label: %s", pj.label))
		}
		disp := int32(targetOffset - (pj.codeOffset + 4))
		binary.LittleEndian.PutUint32(cg.code[pj.codeOffset:pj.codeOffset+4], uint32(disp))
	}

	return cg.code, cg.relocs, cg.symOffsets
}

func (cg *CodeGenerator) emitBytes(b ...byte) {
	cg.code = append(cg.code, b...)
}

func (cg *CodeGenerator) emitU32(val uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], val)
	cg.code = append(cg.code, b[:]...)
}

func (cg *CodeGenerator) emitU64(val uint64) {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], val)
	cg.code = append(cg.code, b[:]...)
}

func (cg *CodeGenerator) currentOffset() int {
	return len(cg.code)
}

func (cg *CodeGenerator) defineSymbol(name string) {
	cg.symOffsets[name] = len(cg.code)
}

func (cg *CodeGenerator) defineLabel(name string) {
	cg.labelOffsets[name] = len(cg.code)
}

// -------------------------------------------------------------
// Runtime Helpers
// -------------------------------------------------------------

func (cg *CodeGenerator) emitRuntimeHelpers() {
	cg.emitAllocHeapHelper()
	cg.emitRetainHelper()
	cg.emitReleaseHelper()
	cg.emitPrintStrHelper()
	cg.emitPrintI64Helper()
	cg.emitPrintBoolHelper()

	cg.emitAllocMemHelper()
	cg.emitFreeMemHelper()
	cg.emitStrLenHelper()
	cg.emitStrCharHelper()
	cg.emitStrEqHelper()
	cg.emitStrSubHelper()
	cg.emitStrConcatHelper()
	cg.emitStrFromIntHelper()
	cg.emitReadFileHelper()
	cg.emitWriteFileHelper()
	cg.emitGetCmdArgHelper()
}

// __gat_alloc_heap(payloadSize: RCX, typeId: RDX) -> RAX (user ptr)
func (cg *CodeGenerator) emitAllocHeapHelper() {
	cg.defineSymbol("__gat_alloc_heap")
	// push rbp; mov rbp, rsp; sub rsp, 64
	cg.emitBytes(0x55, 0x48, 0x89, 0xE5, 0x48, 0x83, 0xEC, 0x40)
	// mov [rbp-8], rcx (payloadSize)
	cg.emitBytes(0x48, 0x89, 0x4D, 0xF8)
	// mov [rbp-16], rdx (typeId)
	cg.emitBytes(0x48, 0x89, 0x55, 0xF0)

	// Call GetProcessHeap() -> RAX
	cg.emitCallIAT("GetProcessHeap")

	// mov rcx, rax (hHeap)
	cg.emitBytes(0x48, 0x89, 0xC1)
	// mov rdx, 8 (HEAP_ZERO_MEMORY = 0x08)
	cg.emitBytes(0x48, 0xC7, 0xC2, 0x08, 0x00, 0x00, 0x00)
	// mov r8, [rbp-8]; add r8, 16 (total size = payload + 16)
	cg.emitBytes(0x4C, 0x8B, 0x45, 0xF8, 0x49, 0x83, 0xC0, 0x10)

	// call qword ptr [__iat_HeapAlloc] -> RAX (raw ptr)
	cg.emitCallIAT("HeapAlloc")

	// [rax] = 1 (refcount = 1)
	cg.emitBytes(0x48, 0xC7, 0x00, 0x01, 0x00, 0x00, 0x00)
	// mov rdx, [rbp-16]; mov [rax+8], rdx (typeId)
	cg.emitBytes(0x48, 0x8B, 0x55, 0xF0, 0x48, 0x89, 0x50, 0x08)
	// add rax, 16 (return user pointer)
	cg.emitBytes(0x48, 0x83, 0xC0, 0x10)

	// mov rsp, rbp; pop rbp; ret
	cg.emitBytes(0x48, 0x89, 0xEC, 0x5D, 0xC3)
}

// __gat_retain(userPtr: RCX)
func (cg *CodeGenerator) emitRetainHelper() {
	cg.defineSymbol("__gat_retain")
	// test rcx, rcx; jz .L_ret (4 bytes ahead)
	cg.emitBytes(0x48, 0x85, 0xC9, 0x74, 0x04)
	// inc qword ptr [rcx - 16] (4 bytes: 48 FF 41 F0)
	cg.emitBytes(0x48, 0xFF, 0x41, 0xF0)
	// .L_ret: ret (1 byte: C3)
	cg.emitBytes(0xC3)
}

// __gat_release(userPtr: RCX)
func (cg *CodeGenerator) emitReleaseHelper() {
	cg.defineSymbol("__gat_release")
	// push rbp; mov rbp, rsp; sub rsp, 64
	cg.emitBytes(0x55, 0x48, 0x89, 0xE5, 0x48, 0x83, 0xEC, 0x40)
	// test rcx, rcx; jz .L_rel_done
	cg.emitBytes(0x48, 0x85, 0xC9)
	// jz forward 50 bytes (we jump to cleanup)
	// let's do a conditional branch with pending label:
	cg.emitBranchCond(0x0F, 0x84, ".L_rel_done")

	// dec qword ptr [rcx - 16]
	cg.emitBytes(0x48, 0xFF, 0x49, 0xF0)
	// jnz .L_rel_done
	cg.emitBranchCond(0x0F, 0x85, ".L_rel_done")

	// Refcount hit 0!
	// Save userPtr at [rbp-8]
	cg.emitBytes(0x48, 0x89, 0x4D, 0xF8)

	// Read type_id = [rcx - 8]
	cg.emitBytes(0x48, 0x8B, 0x51, 0xF8) // mov rdx, [rcx-8]
	// Save type_id at [rbp-16]
	cg.emitBytes(0x48, 0x89, 0x55, 0xF0)

	// Call user deinit dispatch if present
	for _, cls := range cg.prog.Classes {
		if cls.HasDeinit {
			// cmp rdx, cls.TypeId; jne skip
			cg.emitBytes(0x48, 0x81, 0xFA)
			cg.emitU32(uint32(cls.TypeId))
			skipLbl := fmt.Sprintf(".L_skip_deinit_%d", cls.TypeId)
			cg.emitBranchCond(0x0F, 0x85, skipLbl)

			// mov rcx, [rbp-8] (userPtr)
			cg.emitBytes(0x48, 0x8B, 0x4D, 0xF8)
			// call __deinit_user_<ClassName>
			cg.emitCallSym(fmt.Sprintf("__deinit_user_%s", cls.Name))

			cg.defineLabel(skipLbl)
		}
	}

	// Free memory via HeapFree(GetProcessHeap(), 0, rawPtr = userPtr - 16)
	cg.emitCallIAT("GetProcessHeap")
	// mov rcx, rax (hHeap)
	cg.emitBytes(0x48, 0x89, 0xC1)
	// xor rdx, rdx (flags = 0)
	cg.emitBytes(0x48, 0x31, 0xD2)
	// mov r8, [rbp-8]; sub r8, 16 (rawPtr)
	cg.emitBytes(0x4C, 0x8B, 0x45, 0xF8, 0x49, 0x83, 0xE8, 0x10)
	cg.emitCallIAT("HeapFree")

	cg.defineLabel(".L_rel_done")
	// mov rsp, rbp; pop rbp; ret
	cg.emitBytes(0x48, 0x89, 0xEC, 0x5D, 0xC3)
}

// __gat_print_str(strPtr: RCX)
func (cg *CodeGenerator) emitPrintStrHelper() {
	cg.defineSymbol("__gat_print_str")
	// push rbp; mov rbp, rsp; sub rsp, 96
	cg.emitBytes(0x55, 0x48, 0x89, 0xE5, 0x48, 0x83, 0xEC, 0x60)
	// mov [rbp-8], rcx (strPtr)
	cg.emitBytes(0x48, 0x89, 0x4D, 0xF8)

	// test rcx, rcx; jz done
	doneLbl := ".L_print_str_len_done"
	nullLbl := ".L_print_str_null"
	cg.emitBytes(0x48, 0x85, 0xC9)
	cg.emitBranchCond(0x0F, 0x84, nullLbl)

	// strlen loop:
	cg.emitBytes(0x48, 0x89, 0xC8) // mov rax, rcx
	loopLbl := ".L_print_str_len_loop"

	cg.defineLabel(loopLbl)
	cg.emitBytes(0x80, 0x38, 0x00) // cmp byte ptr [rax], 0
	cg.emitBranchCond(0x0F, 0x84, doneLbl) // jz doneLbl
	cg.emitBytes(0x48, 0xFF, 0xC0) // inc rax
	cg.emitJump(loopLbl)

	cg.defineLabel(doneLbl)
	cg.emitBytes(0x48, 0x2B, 0x45, 0xF8) // sub rax, [rbp-8] (len)
	cg.emitBytes(0x48, 0x89, 0x45, 0xF0) // mov [rbp-16], rax (len)

	// GetStdHandle(-11)
	cg.emitBytes(0x48, 0xC7, 0xC1, 0xF5, 0xFF, 0xFF, 0xFF)
	cg.emitCallIAT("GetStdHandle")
	cg.emitBytes(0x48, 0x89, 0xC1)       // mov rcx, rax (hFile)
	cg.emitBytes(0x48, 0x8B, 0x55, 0xF8) // mov rdx, [rbp-8] (lpBuffer)
	cg.emitBytes(0x4C, 0x8B, 0x45, 0xF0) // mov r8, [rbp-16] (len)
	cg.emitBytes(0x4C, 0x8D, 0x4D, 0xE8) // lea r9, [rbp-24] (lpBytesWritten)
	cg.emitBytes(0x48, 0xC7, 0x44, 0x24, 0x20, 0x00, 0x00, 0x00, 0x00) // lpOverlapped = 0
	cg.emitCallIAT("WriteFile")

	cg.defineLabel(nullLbl)
	// mov rsp, rbp; pop rbp; ret
	cg.emitBytes(0x48, 0x89, 0xEC, 0x5D, 0xC3)
}

// __gat_print_i64(n: RCX)
func (cg *CodeGenerator) emitPrintI64Helper() {
	cg.defineSymbol("__gat_print_i64")
	// push rbp; mov rbp, rsp; sub rsp, 96
	cg.emitBytes(0x55, 0x48, 0x89, 0xE5, 0x48, 0x83, 0xEC, 0x60)

	// Buffer at [rbp-32] (holds 32 chars)
	cg.emitBytes(0x48, 0x89, 0xC8)                         // mov rax, rcx
	cg.emitBytes(0x49, 0xC7, 0xC2, 0x0A, 0x00, 0x00, 0x00) // mov r10, 10
	cg.emitBytes(0x4C, 0x8D, 0x45, 0xE0)                   // lea r8, [rbp-32] (buf start)
	cg.emitBytes(0x4D, 0x31, 0xC9)                         // xor r9, r9 (is_neg)

	posLbl := ".L_i64_pos"
	cg.emitBytes(0x48, 0x85, 0xC0)               // test rax, rax
	cg.emitBranchCond(0x0F, 0x89, posLbl)       // jns posLbl
	cg.emitBytes(0x48, 0xF7, 0xD8)               // neg rax
	cg.emitBytes(0x41, 0xB1, 0x01)               // mov r9b, 1

	cg.defineLabel(posLbl)
	divLoopLbl := ".L_i64_div_loop"
	doneDigitsLbl := ".L_i64_done_digits"

	cg.defineLabel(divLoopLbl)
	cg.emitBytes(0x48, 0x31, 0xD2)             // xor rdx, rdx
	cg.emitBytes(0x49, 0xF7, 0xF2)             // div r10
	cg.emitBytes(0x80, 0xC2, 0x30)             // add dl, '0'
	cg.emitBytes(0x41, 0x88, 0x10)             // mov [r8], dl
	cg.emitBytes(0x49, 0xFF, 0xC0)             // inc r8
	cg.emitBytes(0x48, 0x85, 0xC0)             // test rax, rax
	cg.emitBranchCond(0x0F, 0x85, divLoopLbl) // jnz divLoopLbl

	// Check if negative
	cg.emitBytes(0x45, 0x84, 0xC9)                 // test r9b, r9b
	cg.emitBranchCond(0x0F, 0x84, doneDigitsLbl) // jz doneDigitsLbl
	cg.emitBytes(0x41, 0xC6, 0x00, 0x2D)         // mov byte ptr [r8], '-'
	cg.emitBytes(0x49, 0xFF, 0xC0)               // inc r8

	cg.defineLabel(doneDigitsLbl)
	// Reverse buffer: left = r10 (rbp-32), right = r11 (r8-1)
	cg.emitBytes(0x4C, 0x8D, 0x55, 0xE0) // lea r10, [rbp-32]
	cg.emitBytes(0x4D, 0x89, 0xC3)       // mov r11, r8
	cg.emitBytes(0x49, 0xFF, 0xCB)       // dec r11

	revLoopLbl := ".L_i64_rev_loop"
	printOutLbl := ".L_i64_print_out"

	cg.defineLabel(revLoopLbl)
	cg.emitBytes(0x4D, 0x39, 0xDA)             // cmp r10, r11
	cg.emitBranchCond(0x0F, 0x83, printOutLbl) // jae printOutLbl
	cg.emitBytes(0x41, 0x8A, 0x02)             // mov al, [r10]
	cg.emitBytes(0x41, 0x8A, 0x0B)             // mov cl, [r11]
	cg.emitBytes(0x41, 0x88, 0x0A)             // mov [r10], cl
	cg.emitBytes(0x41, 0x88, 0x03)             // mov [r11], al
	cg.emitBytes(0x49, 0xFF, 0xC2)             // inc r10
	cg.emitBytes(0x49, 0xFF, 0xCB)             // dec r11
	cg.emitJump(revLoopLbl)

	cg.defineLabel(printOutLbl)
	// Length in r8: r8 - (rbp-32)
	cg.emitBytes(0x4C, 0x8D, 0x55, 0xE0) // lea r10, [rbp-32]
	cg.emitBytes(0x4D, 0x29, 0xD0)       // sub r8, r10 (len)
	cg.emitBytes(0x4C, 0x89, 0x45, 0xD8) // save len at [rbp-40]

	// GetStdHandle(-11)
	cg.emitBytes(0x48, 0xC7, 0xC1, 0xF5, 0xFF, 0xFF, 0xFF)
	cg.emitCallIAT("GetStdHandle")
	cg.emitBytes(0x48, 0x89, 0xC1)       // mov rcx, rax (hFile)
	cg.emitBytes(0x48, 0x8D, 0x55, 0xE0) // lea rdx, [rbp-32] (lpBuffer)
	cg.emitBytes(0x4C, 0x8B, 0x45, 0xD8) // mov r8, [rbp-40] (len)
	cg.emitBytes(0x4C, 0x8D, 0x4D, 0xD0) // lea r9, [rbp-48] (lpBytesWritten)
	cg.emitBytes(0x48, 0xC7, 0x44, 0x24, 0x20, 0x00, 0x00, 0x00, 0x00) // lpOverlapped = 0
	cg.emitCallIAT("WriteFile")

	// mov rsp, rbp; pop rbp; ret
	cg.emitBytes(0x48, 0x89, 0xEC, 0x5D, 0xC3)
}

// __gat_print_bool(b: RCX)
func (cg *CodeGenerator) emitPrintBoolHelper() {
	cg.defineSymbol("__gat_print_bool")
	// push rbp; mov rbp, rsp; sub rsp, 64
	cg.emitBytes(0x55, 0x48, 0x89, 0xE5, 0x48, 0x83, 0xEC, 0x40)

	trueLbl := ".L_pbool_true"
	cg.emitBytes(0x48, 0x85, 0xC9) // test rcx, rcx
	cg.emitBranchCond(0x0F, 0x85, trueLbl)

	// false branch
	cg.emitBytes(0x48, 0x8D, 0x0D) // lea rcx, [rip + str_false_const]
	cg.emitReloc("str_false_const", RelocRipRelative, -4)
	cg.emitCallSym("__gat_print_str")
	cg.emitBytes(0x48, 0x89, 0xEC, 0x5D, 0xC3) // mov rsp, rbp; pop rbp; ret

	cg.defineLabel(trueLbl)
	// true branch
	cg.emitBytes(0x48, 0x8D, 0x0D) // lea rcx, [rip + str_true_const]
	cg.emitReloc("str_true_const", RelocRipRelative, -4)
	cg.emitCallSym("__gat_print_str")
	cg.emitBytes(0x48, 0x89, 0xEC, 0x5D, 0xC3) // mov rsp, rbp; pop rbp; ret
}

func (cg *CodeGenerator) emitAllocMemHelper() {
	cg.defineSymbol("__gat_alloc_mem")
	// push rbp; mov rbp, rsp; sub rsp, 64
	cg.emitBytes(0x55, 0x48, 0x89, 0xE5, 0x48, 0x83, 0xEC, 0x40)
	// mov [rbp-8], rcx (size)
	cg.emitBytes(0x48, 0x89, 0x4D, 0xF8)
	cg.emitCallIAT("GetProcessHeap")
	cg.emitBytes(0x48, 0x89, 0xC1) // mov rcx, rax (hHeap)
	cg.emitBytes(0x48, 0xC7, 0xC2, 0x08, 0x00, 0x00, 0x00) // mov rdx, 8 (HEAP_ZERO_MEMORY)
	cg.emitBytes(0x4C, 0x8B, 0x45, 0xF8) // mov r8, [rbp-8] (size)
	cg.emitCallIAT("HeapAlloc")
	// mov rsp, rbp; pop rbp; ret
	cg.emitBytes(0x48, 0x89, 0xEC, 0x5D, 0xC3)
}

func (cg *CodeGenerator) emitFreeMemHelper() {
	cg.defineSymbol("__gat_free_mem")
	// push rbp; mov rbp, rsp; sub rsp, 64
	cg.emitBytes(0x55, 0x48, 0x89, 0xE5, 0x48, 0x83, 0xEC, 0x40)
	// test rcx, rcx; jz .L_free_done
	doneLbl := ".L_free_done"
	cg.emitBytes(0x48, 0x85, 0xC9)
	cg.emitBranchCond(0x0F, 0x84, doneLbl)
	// mov [rbp-8], rcx (ptr)
	cg.emitBytes(0x48, 0x89, 0x4D, 0xF8)
	cg.emitCallIAT("GetProcessHeap")
	cg.emitBytes(0x48, 0x89, 0xC1) // mov rcx, rax (hHeap)
	cg.emitBytes(0x48, 0x31, 0xD2) // xor rdx, rdx (flags = 0)
	cg.emitBytes(0x4C, 0x8B, 0x45, 0xF8) // mov r8, [rbp-8] (ptr)
	cg.emitCallIAT("HeapFree")
	cg.defineLabel(doneLbl)
	// mov rsp, rbp; pop rbp; ret
	cg.emitBytes(0x48, 0x89, 0xEC, 0x5D, 0xC3)
}

func (cg *CodeGenerator) emitStrLenHelper() {
	cg.defineSymbol("__gat_str_len")
	loopLbl := ".L_strlen_loop"
	doneLbl := ".L_strlen_done"
	nullLbl := ".L_strlen_null"
	cg.emitBytes(0x48, 0x85, 0xC9) // test rcx, rcx
	cg.emitBranchCond(0x0F, 0x84, nullLbl)
	cg.emitBytes(0x48, 0x89, 0xC8) // mov rax, rcx
	cg.defineLabel(loopLbl)
	cg.emitBytes(0x80, 0x38, 0x00) // cmp byte ptr [rax], 0
	cg.emitBranchCond(0x0F, 0x84, doneLbl)
	cg.emitBytes(0x48, 0xFF, 0xC0) // inc rax
	cg.emitJump(loopLbl)
	cg.defineLabel(doneLbl)
	cg.emitBytes(0x48, 0x29, 0xC8) // sub rax, rcx
	cg.emitBytes(0xC3)
	cg.defineLabel(nullLbl)
	cg.emitBytes(0x48, 0x31, 0xC0, 0xC3) // xor rax, rax; ret
}

func (cg *CodeGenerator) emitStrCharHelper() {
	cg.defineSymbol("__gat_str_char")
	nullLbl := ".L_strchar_null"
	cg.emitBytes(0x48, 0x85, 0xC9) // test rcx, rcx
	cg.emitBranchCond(0x0F, 0x84, nullLbl)
	// movzx rax, byte ptr [rcx + rdx] -> 48 0F B6 04 11
	cg.emitBytes(0x48, 0x0F, 0xB6, 0x04, 0x11, 0xC3)
	cg.defineLabel(nullLbl)
	cg.emitBytes(0x48, 0x31, 0xC0, 0xC3) // xor rax, rax; ret
}

func (cg *CodeGenerator) emitStrEqHelper() {
	cg.defineSymbol("__gat_str_eq")
	eqLbl := ".L_streq_equal"
	diffLbl := ".L_streq_diff"
	loopLbl := ".L_streq_loop"
	cg.emitBytes(0x48, 0x39, 0xD1) // cmp rcx, rdx
	cg.emitBranchCond(0x0F, 0x84, eqLbl)
	cg.emitBytes(0x48, 0x85, 0xC9) // test rcx, rcx
	cg.emitBranchCond(0x0F, 0x84, diffLbl)
	cg.emitBytes(0x48, 0x85, 0xD2) // test rdx, rdx
	cg.emitBranchCond(0x0F, 0x84, diffLbl)

	cg.defineLabel(loopLbl)
	cg.emitBytes(0x8A, 0x01)       // mov al, [rcx]
	cg.emitBytes(0x44, 0x8A, 0x02) // mov r8b, [rdx]
	cg.emitBytes(0x41, 0x38, 0xC0) // cmp r8b, al
	cg.emitBranchCond(0x0F, 0x85, diffLbl)
	cg.emitBytes(0x84, 0xC0) // test al, al
	cg.emitBranchCond(0x0F, 0x84, eqLbl)
	cg.emitBytes(0x48, 0xFF, 0xC1) // inc rcx
	cg.emitBytes(0x48, 0xFF, 0xC2) // inc rdx
	cg.emitJump(loopLbl)

	cg.defineLabel(diffLbl)
	cg.emitBytes(0x48, 0x31, 0xC0, 0xC3) // xor rax, rax; ret

	cg.defineLabel(eqLbl)
	cg.emitBytes(0x48, 0xC7, 0xC0, 0x01, 0x00, 0x00, 0x00, 0xC3) // mov rax, 1; ret
}

func (cg *CodeGenerator) emitStrSubHelper() {
	cg.defineSymbol("__gat_str_sub")
	// push rbp; mov rbp, rsp; sub rsp, 64
	cg.emitBytes(0x55, 0x48, 0x89, 0xE5, 0x48, 0x83, 0xEC, 0x40)
	// mov [rbp-8], rcx (str), mov [rbp-16], rdx (start), mov [rbp-24], r8 (len)
	cg.emitBytes(0x48, 0x89, 0x4D, 0xF8, 0x48, 0x89, 0x55, 0xF0, 0x4C, 0x89, 0x45, 0xE8)
	// rcx = len + 1
	cg.emitBytes(0x48, 0x8B, 0x4D, 0xE8, 0x48, 0xFF, 0xC1)
	cg.emitCallSym("__gat_alloc_mem")
	// rax has newBuf. mov [rbp-32], rax
	cg.emitBytes(0x48, 0x89, 0x45, 0xE0)

	// copy loop: rcx (i) = 0
	cg.emitBytes(0x48, 0x31, 0xC9)
	loopLbl := ".L_strsub_loop"
	doneLbl := ".L_strsub_done"
	cg.defineLabel(loopLbl)
	cg.emitBytes(0x48, 0x3B, 0x4D, 0xE8) // cmp rcx, [rbp-24]
	cg.emitBranchCond(0x0F, 0x8D, doneLbl)

	// rdx = [rbp-8] (str) + [rbp-16] (start)
	cg.emitBytes(0x48, 0x8B, 0x55, 0xF8, 0x48, 0x03, 0x55, 0xF0)
	// al = byte ptr [rdx + rcx]
	cg.emitBytes(0x8A, 0x04, 0x0A)
	// rbx = [rbp-32] (newBuf)
	cg.emitBytes(0x48, 0x8B, 0x5D, 0xE0)
	// byte ptr [rbx + rcx] = al
	cg.emitBytes(0x88, 0x04, 0x0B)
	cg.emitBytes(0x48, 0xFF, 0xC1) // inc rcx
	cg.emitJump(loopLbl)

	cg.defineLabel(doneLbl)
	// newBuf[len] = 0
	cg.emitBytes(0x48, 0x8B, 0x5D, 0xE0, 0x48, 0x03, 0x5D, 0xE8, 0xC6, 0x03, 0x00)
	cg.emitBytes(0x48, 0x8B, 0x45, 0xE0) // return newBuf
	// mov rsp, rbp; pop rbp; ret
	cg.emitBytes(0x48, 0x89, 0xEC, 0x5D, 0xC3)
}

func (cg *CodeGenerator) emitStrConcatHelper() {
	cg.defineSymbol("__gat_str_concat")
	// push rbp; mov rbp, rsp; sub rsp, 64
	cg.emitBytes(0x55, 0x48, 0x89, 0xE5, 0x48, 0x83, 0xEC, 0x40)
	// mov [rbp-8], rcx (a), mov [rbp-16], rdx (b)
	cg.emitBytes(0x48, 0x89, 0x4D, 0xF8, 0x48, 0x89, 0x55, 0xF0)
	// strlen(a) -> [rbp-24]
	cg.emitBytes(0x48, 0x8B, 0x4D, 0xF8)
	cg.emitCallSym("__gat_str_len")
	cg.emitBytes(0x48, 0x89, 0x45, 0xE8)
	// strlen(b) -> [rbp-32]
	cg.emitBytes(0x48, 0x8B, 0x4D, 0xF0)
	cg.emitCallSym("__gat_str_len")
	cg.emitBytes(0x48, 0x89, 0x45, 0xE0)

	// total = lenA + lenB + 1
	cg.emitBytes(0x48, 0x8B, 0x4D, 0xE8, 0x48, 0x03, 0x4D, 0xE0, 0x48, 0xFF, 0xC1)
	cg.emitCallSym("__gat_alloc_mem")
	// mov [rbp-40], rax (newBuf)
	cg.emitBytes(0x48, 0x89, 0x45, 0xD8)

	// Copy a: rcx (i) = 0
	cg.emitBytes(0x48, 0x31, 0xC9)
	loopALbl := ".L_concat_a_loop"
	doneALbl := ".L_concat_a_done"
	cg.defineLabel(loopALbl)
	cg.emitBytes(0x48, 0x3B, 0x4D, 0xE8) // cmp rcx, lenA
	cg.emitBranchCond(0x0F, 0x8D, doneALbl)
	cg.emitBytes(0x48, 0x8B, 0x55, 0xF8) // rdx = a
	cg.emitBytes(0x8A, 0x04, 0x0A)       // mov al, [rdx + rcx]
	cg.emitBytes(0x48, 0x8B, 0x5D, 0xD8) // rbx = newBuf
	cg.emitBytes(0x88, 0x04, 0x0B)       // mov [rbx + rcx], al
	cg.emitBytes(0x48, 0xFF, 0xC1)       // inc rcx
	cg.emitJump(loopALbl)

	cg.defineLabel(doneALbl)
	// Copy b: rcx (i) = 0
	cg.emitBytes(0x48, 0x31, 0xC9)
	loopBLbl := ".L_concat_b_loop"
	doneBLbl := ".L_concat_b_done"
	cg.defineLabel(loopBLbl)
	cg.emitBytes(0x48, 0x3B, 0x4D, 0xE0) // cmp rcx, lenB
	cg.emitBranchCond(0x0F, 0x8D, doneBLbl)
	cg.emitBytes(0x48, 0x8B, 0x55, 0xF0) // rdx = b
	cg.emitBytes(0x8A, 0x04, 0x0A)       // mov al, [rdx + rcx]
	cg.emitBytes(0x48, 0x8B, 0x5D, 0xD8, 0x48, 0x03, 0x5D, 0xE8) // rbx = newBuf + lenA
	cg.emitBytes(0x88, 0x04, 0x0B)                               // mov [rbx + rcx], al
	cg.emitBytes(0x48, 0xFF, 0xC1)                               // inc rcx
	cg.emitJump(loopBLbl)

	cg.defineLabel(doneBLbl)
	// newBuf[lenA + lenB] = 0
	cg.emitBytes(0x48, 0x8B, 0x5D, 0xD8, 0x48, 0x03, 0x5D, 0xE8, 0x48, 0x03, 0x5D, 0xE0, 0xC6, 0x03, 0x00)
	cg.emitBytes(0x48, 0x8B, 0x45, 0xD8) // return newBuf
	// mov rsp, rbp; pop rbp; ret
	cg.emitBytes(0x48, 0x89, 0xEC, 0x5D, 0xC3)
}

func (cg *CodeGenerator) emitStrFromIntHelper() {
	cg.defineSymbol("__gat_str_from_int")
	// push rbp; mov rbp, rsp; sub rsp, 96
	cg.emitBytes(0x55, 0x48, 0x89, 0xE5, 0x48, 0x83, 0xEC, 0x60)

	// Format int into buffer at [rbp-32]
	cg.emitBytes(0x48, 0x89, 0xC8)                         // mov rax, rcx
	cg.emitBytes(0x49, 0xC7, 0xC2, 0x0A, 0x00, 0x00, 0x00) // mov r10, 10
	cg.emitBytes(0x4C, 0x8D, 0x45, 0xE0)                   // lea r8, [rbp-32] (buf start)
	cg.emitBytes(0x4D, 0x31, 0xC9)                         // xor r9, r9 (is_neg)

	posLbl := ".L_sfi_pos"
	cg.emitBytes(0x48, 0x85, 0xC0)
	cg.emitBranchCond(0x0F, 0x89, posLbl)
	cg.emitBytes(0x48, 0xF7, 0xD8)
	cg.emitBytes(0x41, 0xB1, 0x01)

	cg.defineLabel(posLbl)
	divLoopLbl := ".L_sfi_div_loop"
	doneDigitsLbl := ".L_sfi_done_digits"

	cg.defineLabel(divLoopLbl)
	cg.emitBytes(0x48, 0x31, 0xD2)
	cg.emitBytes(0x49, 0xF7, 0xF2)
	cg.emitBytes(0x80, 0xC2, 0x30)
	cg.emitBytes(0x41, 0x88, 0x10)
	cg.emitBytes(0x49, 0xFF, 0xC0)
	cg.emitBytes(0x48, 0x85, 0xC0)
	cg.emitBranchCond(0x0F, 0x85, divLoopLbl)

	cg.emitBytes(0x45, 0x84, 0xC9)
	cg.emitBranchCond(0x0F, 0x84, doneDigitsLbl)
	cg.emitBytes(0x41, 0xC6, 0x00, 0x2D)
	cg.emitBytes(0x49, 0xFF, 0xC0)

	cg.defineLabel(doneDigitsLbl)
	// Reverse buffer: left = r10 (rbp-32), right = r11 (r8-1)
	cg.emitBytes(0x4C, 0x8D, 0x55, 0xE0)
	cg.emitBytes(0x4D, 0x89, 0xC3)
	cg.emitBytes(0x49, 0xFF, 0xCB)

	revLoopLbl := ".L_sfi_rev_loop"
	doneRevLbl := ".L_sfi_done_rev"

	cg.defineLabel(revLoopLbl)
	cg.emitBytes(0x4D, 0x39, 0xDA)
	cg.emitBranchCond(0x0F, 0x83, doneRevLbl)
	cg.emitBytes(0x41, 0x8A, 0x02)
	cg.emitBytes(0x41, 0x8A, 0x0B)
	cg.emitBytes(0x41, 0x88, 0x0A)
	cg.emitBytes(0x41, 0x88, 0x03)
	cg.emitBytes(0x49, 0xFF, 0xC2)
	cg.emitBytes(0x49, 0xFF, 0xCB)
	cg.emitJump(revLoopLbl)

	cg.defineLabel(doneRevLbl)
	// len in r8: r8 - (rbp-32)
	cg.emitBytes(0x4C, 0x8D, 0x55, 0xE0)
	cg.emitBytes(0x4D, 0x29, 0xD0)
	cg.emitBytes(0x4C, 0x89, 0x45, 0xD8) // [rbp-40] = len

	// alloc_mem(len + 1)
	cg.emitBytes(0x48, 0x8B, 0x4D, 0xD8, 0x48, 0xFF, 0xC1)
	cg.emitCallSym("__gat_alloc_mem")
	cg.emitBytes(0x48, 0x89, 0x45, 0xD0) // [rbp-48] = newBuf

	// copy digits to newBuf: r8 (i) = 0
	cg.emitBytes(0x4D, 0x31, 0xC0)
	copyLoopLbl := ".L_sfi_copy_loop"
	doneCopyLbl := ".L_sfi_done_copy"
	cg.defineLabel(copyLoopLbl)
	cg.emitBytes(0x4C, 0x3B, 0x45, 0xD8) // cmp r8, len
	cg.emitBranchCond(0x0F, 0x8D, doneCopyLbl)
	cg.emitBytes(0x48, 0x8D, 0x55, 0xE0) // rdx = lea [rbp-32]
	cg.emitBytes(0x42, 0x8A, 0x04, 0x02) // mov al, [rdx + r8]
	cg.emitBytes(0x48, 0x8B, 0x5D, 0xD0) // rbx = newBuf
	cg.emitBytes(0x42, 0x88, 0x04, 0x03) // mov [rbx + r8], al
	cg.emitBytes(0x49, 0xFF, 0xC0)       // inc r8
	cg.emitJump(copyLoopLbl)

	cg.defineLabel(doneCopyLbl)
	// newBuf[len] = 0
	cg.emitBytes(0x48, 0x8B, 0x55, 0xD0, 0x48, 0x03, 0x55, 0xD8, 0xC6, 0x02, 0x00)
	cg.emitBytes(0x48, 0x8B, 0x45, 0xD0) // return newBuf
	// mov rsp, rbp; pop rbp; ret
	cg.emitBytes(0x48, 0x89, 0xEC, 0x5D, 0xC3)
}

func (cg *CodeGenerator) emitReadFileHelper() {
	cg.defineSymbol("__gat_read_file")
	// push rbp; mov rbp, rsp; sub rsp, 96
	cg.emitBytes(0x55, 0x48, 0x89, 0xE5, 0x48, 0x83, 0xEC, 0x60)
	// mov [rbp-8], rcx (path)
	cg.emitBytes(0x48, 0x89, 0x4D, 0xF8)

	// CreateFileA(path: rcx, GENERIC_READ (0x80000000): rdx, FILE_SHARE_READ (1): r8, NULL: r9, OPEN_EXISTING (3): [rsp+0x20], FILE_ATTRIBUTE_NORMAL (0x80): [rsp+0x28], NULL: [rsp+0x30])
	cg.emitBytes(0x48, 0x8B, 0x4D, 0xF8)
	cg.emitBytes(0x48, 0xC7, 0xC2, 0x00, 0x00, 0x00, 0x80)
	cg.emitBytes(0x49, 0xC7, 0xC0, 0x01, 0x00, 0x00, 0x00)
	cg.emitBytes(0x4D, 0x31, 0xC9)
	cg.emitBytes(0x48, 0xC7, 0x44, 0x24, 0x20, 0x03, 0x00, 0x00, 0x00)
	cg.emitBytes(0x48, 0xC7, 0x44, 0x24, 0x28, 0x80, 0x00, 0x00, 0x00)
	cg.emitBytes(0x48, 0xC7, 0x44, 0x24, 0x30, 0x00, 0x00, 0x00, 0x00)
	cg.emitCallIAT("CreateFileA")

	errLbl := ".L_readfile_err"
	cg.emitBytes(0x48, 0x83, 0xF8, 0xFF)
	cg.emitBranchCond(0x0F, 0x84, errLbl)
	cg.emitBytes(0x48, 0x89, 0x45, 0xF0) // [rbp-16] = hFile

	// GetFileSizeEx(hFile: rcx, &size: rdx)
	cg.emitBytes(0x48, 0x8B, 0x4D, 0xF0)
	cg.emitBytes(0x48, 0x8D, 0x55, 0xE8)
	cg.emitCallIAT("GetFileSizeEx")

	// size in [rbp-24]. Alloc size + 1
	cg.emitBytes(0x48, 0x8B, 0x4D, 0xE8, 0x48, 0xFF, 0xC1)
	cg.emitCallSym("__gat_alloc_mem")
	cg.emitBytes(0x48, 0x89, 0x45, 0xE0) // [rbp-32] = buf

	// ReadFile(hFile: rcx, buf: rdx, size: r8, &bytesRead: r9, lpOverlapped: [rsp+0x20] = 0)
	cg.emitBytes(0x48, 0x8B, 0x4D, 0xF0)
	cg.emitBytes(0x48, 0x8B, 0x55, 0xE0)
	cg.emitBytes(0x4C, 0x8B, 0x45, 0xE8)
	cg.emitBytes(0x4C, 0x8D, 0x4D, 0xD8)
	cg.emitBytes(0x48, 0xC7, 0x44, 0x24, 0x20, 0x00, 0x00, 0x00, 0x00)
	cg.emitCallIAT("ReadFile")

	// CloseHandle(hFile)
	cg.emitBytes(0x48, 0x8B, 0x4D, 0xF0)
	cg.emitCallIAT("CloseHandle")

	// buf[size] = 0
	cg.emitBytes(0x48, 0x8B, 0x55, 0xE0, 0x48, 0x03, 0x55, 0xE8, 0xC6, 0x02, 0x00)
	cg.emitBytes(0x48, 0x8B, 0x45, 0xE0) // return buf
	cg.emitBytes(0x48, 0x89, 0xEC, 0x5D, 0xC3)

	cg.defineLabel(errLbl)
	// Return empty string (1 byte)
	cg.emitBytes(0x48, 0xC7, 0xC1, 0x01, 0x00, 0x00, 0x00)
	cg.emitCallSym("__gat_alloc_mem")
	cg.emitBytes(0x48, 0x89, 0xEC, 0x5D, 0xC3)
}

func (cg *CodeGenerator) emitWriteFileHelper() {
	cg.defineSymbol("__gat_write_file")
	// push rbp; mov rbp, rsp; sub rsp, 96
	cg.emitBytes(0x55, 0x48, 0x89, 0xE5, 0x48, 0x83, 0xEC, 0x60)
	// mov [rbp-8], rcx (path), mov [rbp-16], rdx (data), mov [rbp-24], r8 (len)
	cg.emitBytes(0x48, 0x89, 0x4D, 0xF8, 0x48, 0x89, 0x55, 0xF0, 0x4C, 0x89, 0x45, 0xE8)

	// If len < 0, calculate strlen(data)
	lenOkLbl := ".L_wf_len_ok"
	cg.emitBytes(0x48, 0x83, 0x7D, 0xE8, 0x00)
	cg.emitBranchCond(0x0F, 0x8D, lenOkLbl)
	cg.emitBytes(0x48, 0x8B, 0x4D, 0xF0)
	cg.emitCallSym("__gat_str_len")
	cg.emitBytes(0x48, 0x89, 0x45, 0xE8)
	cg.defineLabel(lenOkLbl)

	// CreateFileA(path: rcx, GENERIC_WRITE (0x40000000): rdx, 0: r8, NULL: r9, CREATE_ALWAYS (2): [rsp+0x20], FILE_ATTRIBUTE_NORMAL (0x80): [rsp+0x28], NULL: [rsp+0x30])
	cg.emitBytes(0x48, 0x8B, 0x4D, 0xF8)
	cg.emitBytes(0x48, 0xC7, 0xC2, 0x00, 0x00, 0x00, 0x40)
	cg.emitBytes(0x4D, 0x31, 0xC0)
	cg.emitBytes(0x4D, 0x31, 0xC9)
	cg.emitBytes(0x48, 0xC7, 0x44, 0x24, 0x20, 0x02, 0x00, 0x00, 0x00)
	cg.emitBytes(0x48, 0xC7, 0x44, 0x24, 0x28, 0x80, 0x00, 0x00, 0x00)
	cg.emitBytes(0x48, 0xC7, 0x44, 0x24, 0x30, 0x00, 0x00, 0x00, 0x00)
	cg.emitCallIAT("CreateFileA")

	errLbl := ".L_wf_err"
	cg.emitBytes(0x48, 0x83, 0xF8, 0xFF)
	cg.emitBranchCond(0x0F, 0x84, errLbl)
	cg.emitBytes(0x48, 0x89, 0x45, 0xD0) // [rbp-48] = hFile

	// WriteFile(hFile: rcx, data: rdx, len: r8, &written: r9, lpOverlapped: [rsp+0x20] = 0)
	cg.emitBytes(0x48, 0x8B, 0x4D, 0xD0) // rcx = hFile
	cg.emitBytes(0x48, 0x8B, 0x55, 0xF0) // rdx = data
	cg.emitBytes(0x4C, 0x8B, 0x45, 0xE8) // r8 = len
	cg.emitBytes(0x4C, 0x8D, 0x4D, 0xD8) // r9 = lea [rbp-40] (&written)
	cg.emitBytes(0x48, 0xC7, 0x44, 0x24, 0x20, 0x00, 0x00, 0x00, 0x00)
	cg.emitCallIAT("WriteFile")

	// CloseHandle(hFile)
	cg.emitBytes(0x48, 0x8B, 0x4D, 0xD0)
	cg.emitCallIAT("CloseHandle")

	cg.emitBytes(0x48, 0x8B, 0x45, 0xD8) // return written
	cg.emitBytes(0x48, 0x89, 0xEC, 0x5D, 0xC3)

	cg.defineLabel(errLbl)
	cg.emitBytes(0x48, 0xC7, 0xC0, 0xFF, 0xFF, 0xFF, 0xFF) // return -1
	cg.emitBytes(0x48, 0x89, 0xEC, 0x5D, 0xC3)
}

func (cg *CodeGenerator) emitGetCmdArgHelper() {
	cg.defineSymbol("__gat_get_cmd_arg")
	// push rbp; mov rbp, rsp; sub rsp, 96
	cg.emitBytes(0x55, 0x48, 0x89, 0xE5, 0x48, 0x83, 0xEC, 0x60)
	// mov [rbp-8], rcx (targetIdx)
	cg.emitBytes(0x48, 0x89, 0x4D, 0xF8)
	cg.emitCallIAT("GetCommandLineA")
	// rax has cmdline ptr. mov [rbp-16], rax
	cg.emitBytes(0x48, 0x89, 0x45, 0xF0)

	// currentArgIdx = 0 ([rbp-24])
	cg.emitBytes(0x48, 0xC7, 0x45, 0xE8, 0x00, 0x00, 0x00, 0x00)
	// ptr = [rbp-16] ([rbp-32])
	cg.emitBytes(0x48, 0x8B, 0x45, 0xF0, 0x48, 0x89, 0x45, 0xE0)

	tokenLoopLbl := ".L_arg_token_loop"
	doneNotFoundLbl := ".L_arg_done_not_found"
	cg.defineLabel(tokenLoopLbl)

	// skip spaces:
	skipSpaceLbl := ".L_arg_skip_space"
	cg.defineLabel(skipSpaceLbl)
	cg.emitBytes(0x48, 0x8B, 0x55, 0xE0, 0x8A, 0x02) // al = *ptr
	cg.emitBytes(0x84, 0xC0)                         // test al, al
	cg.emitBranchCond(0x0F, 0x84, doneNotFoundLbl)
	cg.emitBytes(0x3C, 0x20) // cmp al, ' '
	skipNextLbl := ".L_arg_start_token"
	cg.emitBranchCond(0x0F, 0x85, skipNextLbl)
	cg.emitBytes(0x48, 0xFF, 0x45, 0xE0) // inc ptr
	cg.emitJump(skipSpaceLbl)

	cg.defineLabel(skipNextLbl)
	// check if quoted:
	// startPtr = ptr ([rbp-40])
	cg.emitBytes(0x48, 0x8B, 0x55, 0xE0, 0x48, 0x89, 0x55, 0xD8)
	cg.emitBytes(0x8A, 0x02, 0x3C, 0x22) // cmp al, '"'
	isQuotedLbl := ".L_arg_quoted"
	isUnquotedLbl := ".L_arg_unquoted"
	cg.emitBranchCond(0x0F, 0x84, isQuotedLbl)

	// Unquoted scan until space or 0
	cg.defineLabel(isUnquotedLbl)
	cg.emitBytes(0x48, 0x8B, 0x55, 0xE0, 0x8A, 0x02)
	cg.emitBytes(0x84, 0xC0)
	endTokLbl := ".L_arg_end_token"
	cg.emitBranchCond(0x0F, 0x84, endTokLbl)
	cg.emitBytes(0x3C, 0x20)
	cg.emitBranchCond(0x0F, 0x84, endTokLbl)
	cg.emitBytes(0x48, 0xFF, 0x45, 0xE0)
	cg.emitJump(isUnquotedLbl)

	// Quoted scan until '"' or 0
	cg.defineLabel(isQuotedLbl)
	cg.emitBytes(0x48, 0xFF, 0x45, 0xD8) // skip initial '"' in startPtr
	cg.emitBytes(0x48, 0xFF, 0x45, 0xE0) // skip initial '"' in ptr
	quotedLoopLbl := ".L_arg_quoted_loop"
	cg.defineLabel(quotedLoopLbl)
	cg.emitBytes(0x48, 0x8B, 0x55, 0xE0, 0x8A, 0x02)
	cg.emitBytes(0x84, 0xC0)
	cg.emitBranchCond(0x0F, 0x84, endTokLbl)
	cg.emitBytes(0x3C, 0x22)
	cg.emitBranchCond(0x0F, 0x84, endTokLbl)
	cg.emitBytes(0x48, 0xFF, 0x45, 0xE0)
	cg.emitJump(quotedLoopLbl)

	cg.defineLabel(endTokLbl)
	// Check if currentArgIdx == targetIdx
	cg.emitBytes(0x48, 0x8B, 0x55, 0xE8, 0x48, 0x3B, 0x55, 0xF8)
	isTargetLbl := ".L_arg_is_target"
	cg.emitBranchCond(0x0F, 0x84, isTargetLbl)

	// Not target: if at '"', skip it
	cg.emitBytes(0x48, 0x8B, 0x55, 0xE0, 0x8A, 0x02, 0x3C, 0x22)
	skipQuoteLbl := ".L_arg_skip_q"
	cg.emitBranchCond(0x0F, 0x85, skipQuoteLbl)
	cg.emitBytes(0x48, 0xFF, 0x45, 0xE0)
	cg.defineLabel(skipQuoteLbl)
	cg.emitBytes(0x48, 0xFF, 0x45, 0xE8) // inc currentArgIdx
	cg.emitJump(tokenLoopLbl)

	cg.defineLabel(isTargetLbl)
	// len = ptr - startPtr -> r8
	// Call str_sub(startPtr: rcx, 0: rdx, len: r8)
	cg.emitBytes(0x48, 0x8B, 0x4D, 0xD8)                          // rcx = startPtr
	cg.emitBytes(0x48, 0x31, 0xD2)                                // rdx = 0
	cg.emitBytes(0x4C, 0x8B, 0x45, 0xE0, 0x4C, 0x2B, 0x45, 0xD8) // r8 = ptr - startPtr
	cg.emitCallSym("__gat_str_sub")
	// mov rsp, rbp; pop rbp; ret
	cg.emitBytes(0x48, 0x89, 0xEC, 0x5D, 0xC3)

	cg.defineLabel(doneNotFoundLbl)
	// Return empty string
	cg.emitBytes(0x48, 0xC7, 0xC1, 0x01, 0x00, 0x00, 0x00)
	cg.emitCallSym("__gat_alloc_mem")
	cg.emitBytes(0x48, 0x89, 0xEC, 0x5D, 0xC3)
}

// -------------------------------------------------------------
// User Function Codegen
// -------------------------------------------------------------

func (cg *CodeGenerator) emitFunction(fn *ir.Function) {
	cg.defineSymbol(fn.Name)
	cg.localStack = make(map[*ir.Value]int)

	// Layout stack frame:
	// RBP + 16: Arg 4+ (on stack)
	// RBP - 8: Local 0 ...
	// Windows Fastcall incoming params in RCX, RDX, R8, R9.
	// We save all params to their stack slots at prologue.

	offset := 8
	for _, p := range fn.Params {
		cg.localStack[p] = -offset
		offset += 8
	}
	for _, l := range fn.Locals {
		if _, exists := cg.localStack[l]; !exists {
			cg.localStack[l] = -offset
			offset += 8
		}
	}
	for _, inst := range fn.Instructions {
		if inst.Op == ir.OpAllocStack && inst.Dest != nil {
			sz := inst.Type.Size()
			if sz%8 != 0 {
				sz += 8 - (sz % 8)
			}
			offset += sz
			cg.structBuffers[inst.Dest] = -offset
		}
	}

	maxArgs := 4
	for _, inst := range fn.Instructions {
		if inst.Op == ir.OpCall && len(inst.Args) > maxArgs {
			maxArgs = len(inst.Args)
		}
	}
	outgoingSpace := maxArgs * 8
	if outgoingSpace < 32 {
		outgoingSpace = 32
	}

	frameSize := offset + outgoingSpace
	if frameSize%16 != 0 {
		frameSize += 16 - (frameSize % 16)
	}
	cg.stackFrameSize = frameSize

	// Prologue: push rbp; mov rbp, rsp; sub rsp, frameSize
	cg.emitBytes(0x55, 0x48, 0x89, 0xE5)
	cg.emitBytes(0x48, 0x81, 0xEC)
	cg.emitU32(uint32(frameSize))

	// Save incoming parameter registers/stack to stack slots FIRST
	for i, p := range fn.Params {
		disp := cg.localStack[p]
		switch i {
		case 0: // mov [rbp+disp], rcx
			cg.emitMovStackReg(disp, "rcx")
		case 1: // mov [rbp+disp], rdx
			cg.emitMovStackReg(disp, "rdx")
		case 2: // mov [rbp+disp], r8
			cg.emitMovStackReg(disp, "r8")
		case 3: // mov [rbp+disp], r9
			cg.emitMovStackReg(disp, "r9")
		default: // mov rax, [rbp + 0x30 + (i-4)*8]; mov [rbp+disp], rax
			cg.emitBytes(0x48, 0x8B, 0x45, byte(0x30+(i-4)*8))
			cg.emitMovStackReg(disp, "rax")
		}
	}

	// Zero-initialize all local variable slots (ensures clean initial state for ARC)
	for _, l := range fn.Locals {
		disp := cg.localStack[l]
		cg.emitBytes(0x48, 0xC7, 0x85)
		cg.emitU32(uint32(int32(disp)))
		cg.emitU32(0)
	}

	// Emit instructions
	for _, inst := range fn.Instructions {
		cg.emitInstruction(inst)
	}

	// Epilogue (fallback if missing ret): mov rsp, rbp; pop rbp; ret
	cg.emitBytes(0x48, 0x89, 0xEC, 0x5D, 0xC3)
}

func (cg *CodeGenerator) emitInstruction(inst *ir.Instruction) {
	switch inst.Op {
	case ir.OpConstInt:
		disp := cg.localStack[inst.Dest]
		if inst.IntVal >= -2147483648 && inst.IntVal <= 2147483647 {
			// mov qword ptr [rbp+disp], imm32 (sign extended)
			cg.emitBytes(0x48, 0xC7, 0x85)
			cg.emitU32(uint32(int32(disp)))
			cg.emitU32(uint32(int32(inst.IntVal)))
		} else {
			// movabs rax, imm64 -> 48 B8 [u64]
			cg.emitBytes(0x48, 0xB8)
			cg.emitU64(uint64(inst.IntVal))
			// mov [rbp+disp], rax -> 48 89 85 [disp32]
			cg.emitBytes(0x48, 0x89, 0x85)
			cg.emitU32(uint32(int32(disp)))
		}

	case ir.OpConstBool:
		disp := cg.localStack[inst.Dest]
		val := uint32(0)
		if inst.BoolVal {
			val = 1
		}
		cg.emitBytes(0x48, 0xC7, 0x85)
		cg.emitU32(uint32(int32(disp)))
		cg.emitU32(val)

	case ir.OpConstNil:
		disp := cg.localStack[inst.Dest]
		cg.emitBytes(0x48, 0xC7, 0x85)
		cg.emitU32(uint32(int32(disp)))
		cg.emitU32(0)

	case ir.OpConstString:
		// lea rax, [rip + StrVal]; mov [rbp+dest], rax
		disp := cg.localStack[inst.Dest]
		cg.emitBytes(0x48, 0x8D, 0x05)
		cg.emitReloc(inst.StrVal, RelocRipRelative, -4)
		cg.emitMovStackReg(disp, "rax")

	case ir.OpAllocHeap:
		// RCX = payloadSize, RDX = typeId
		ct := inst.Type.(*types.ClassType)
		// mov rcx, payloadSize
		cg.emitBytes(0x48, 0xC7, 0xC1)
		cg.emitU32(uint32(ct.PayloadSize))
		// mov rdx, typeId
		cg.emitBytes(0x48, 0xC7, 0xC2)
		cg.emitU32(uint32(ct.TypeId))
		cg.emitCallSym("__gat_alloc_heap")
		// mov [rbp+dest], rax
		disp := cg.localStack[inst.Dest]
		cg.emitMovStackReg(disp, "rax")

	case ir.OpAllocStack:
		// Stack allocated struct: lea rax, [rbp + structBufferDisp]
		// mov [rbp + destDisp], rax
		bufDisp := cg.structBuffers[inst.Dest]
		cg.emitBytes(0x48, 0x8D, 0x85)
		cg.emitU32(uint32(int32(bufDisp)))
		disp := cg.localStack[inst.Dest]
		cg.emitMovStackReg(disp, "rax")

	case ir.OpCopy:
		// mov rax, [rbp+src]; mov [rbp+dest], rax
		cg.emitMovRegStack("rax", cg.localStack[inst.Src1])
		cg.emitMovStackReg(cg.localStack[inst.Dest], "rax")

	case ir.OpLoad:
		// Load field: mov rbx, [rbp+src1]; mov rax, [rbx + offset]; mov [rbp+dest], rax
		cg.emitMovRegStack("rbx", cg.localStack[inst.Src1])
		cg.emitBytes(0x48, 0x8B, 0x83)
		cg.emitU32(uint32(int32(inst.Offset)))
		cg.emitMovStackReg(cg.localStack[inst.Dest], "rax")

	case ir.OpStore:
		// Store field: mov rbx, [rbp+dest]; mov rax, [rbp+src1]; mov [rbx + offset], rax
		cg.emitMovRegStack("rbx", cg.localStack[inst.Dest])
		cg.emitMovRegStack("rax", cg.localStack[inst.Src1])
		cg.emitBytes(0x48, 0x89, 0x83)
		cg.emitU32(uint32(int32(inst.Offset)))

	case ir.OpLoadIndex:
		// mov rbx, [rbp+src1] (base ptr)
		// mov rcx, [rbp+src2] (index)
		cg.emitMovRegStack("rbx", cg.localStack[inst.Src1])
		cg.emitMovRegStack("rcx", cg.localStack[inst.Src2])
		if isByteType(inst.Src1.Type) || inst.Type.Size() == 1 {
			// movzx rax, byte ptr [rbx + rcx] -> 48 0F B6 04 0B
			cg.emitBytes(0x48, 0x0F, 0xB6, 0x04, 0x0B)
		} else {
			// mov rax, qword ptr [rbx + rcx*8] -> 48 8B 04 CB
			cg.emitBytes(0x48, 0x8B, 0x04, 0xCB)
		}
		cg.emitMovStackReg(cg.localStack[inst.Dest], "rax")

	case ir.OpStoreIndex:
		// mov rbx, [rbp+dest] (base ptr)
		// mov rcx, [rbp+src2] (index)
		// mov rax, [rbp+src1] (value)
		cg.emitMovRegStack("rbx", cg.localStack[inst.Dest])
		cg.emitMovRegStack("rcx", cg.localStack[inst.Src2])
		cg.emitMovRegStack("rax", cg.localStack[inst.Src1])
		if isByteType(inst.Dest.Type) || inst.Type.Size() == 1 {
			// mov byte ptr [rbx + rcx], al -> 88 04 0B
			cg.emitBytes(0x88, 0x04, 0x0B)
		} else {
			// mov qword ptr [rbx + rcx*8], rax -> 48 89 04 CB
			cg.emitBytes(0x48, 0x89, 0x04, 0xCB)
		}

	case ir.OpBinOp:
		cg.emitMovRegStack("rax", cg.localStack[inst.Src1])
		cg.emitMovRegStack("rbx", cg.localStack[inst.Src2])
		switch inst.StrVal {
		case "+":
			cg.emitBytes(0x48, 0x01, 0xD8) // add rax, rbx
		case "-":
			cg.emitBytes(0x48, 0x29, 0xD8) // sub rax, rbx
		case "*":
			cg.emitBytes(0x48, 0x0F, 0xAF, 0xC3) // imul rax, rbx
		case "/":
			cg.emitBytes(0x48, 0x99, 0x48, 0xF7, 0xFB) // cqo; idiv rbx
		case "%":
			cg.emitBytes(0x48, 0x99, 0x48, 0xF7, 0xFB, 0x48, 0x89, 0xD0) // cqo; idiv rbx; mov rax, rdx
		case "==":
			cg.emitBytes(0x48, 0x39, 0xD8, 0x0F, 0x94, 0xC0, 0x48, 0x0F, 0xB6, 0xC0) // cmp rax, rbx; sete al; movzx rax, al
		case "!=":
			cg.emitBytes(0x48, 0x39, 0xD8, 0x0F, 0x95, 0xC0, 0x48, 0x0F, 0xB6, 0xC0) // cmp rax, rbx; setne al; movzx rax, al
		case "<":
			cg.emitBytes(0x48, 0x39, 0xD8, 0x0F, 0x9C, 0xC0, 0x48, 0x0F, 0xB6, 0xC0) // cmp rax, rbx; setl al; movzx rax, al
		case "<=":
			cg.emitBytes(0x48, 0x39, 0xD8, 0x0F, 0x9E, 0xC0, 0x48, 0x0F, 0xB6, 0xC0) // cmp rax, rbx; setle al; movzx rax, al
		case ">":
			cg.emitBytes(0x48, 0x39, 0xD8, 0x0F, 0x9F, 0xC0, 0x48, 0x0F, 0xB6, 0xC0) // cmp rax, rbx; setg al; movzx rax, al
		case ">=":
			cg.emitBytes(0x48, 0x39, 0xD8, 0x0F, 0x9D, 0xC0, 0x48, 0x0F, 0xB6, 0xC0) // cmp rax, rbx; setge al; movzx rax, al
		case "&&":
			cg.emitBytes(0x48, 0x21, 0xD8) // and rax, rbx
		case "||":
			cg.emitBytes(0x48, 0x09, 0xD8) // or rax, rbx
		}
		cg.emitMovStackReg(cg.localStack[inst.Dest], "rax")

	case ir.OpUnOp:
		cg.emitMovRegStack("rax", cg.localStack[inst.Src1])
		switch inst.StrVal {
		case "-":
			cg.emitBytes(0x48, 0xF7, 0xD8) // neg rax
		case "!":
			cg.emitBytes(0x48, 0x85, 0xC0, 0x0F, 0x94, 0xC0, 0x48, 0x0F, 0xB6, 0xC0) // test rax, rax; sete al; movzx rax, al
		case "raw":
			// rax already holds pointer [rbp + src1]
		}
		cg.emitMovStackReg(cg.localStack[inst.Dest], "rax")

	case ir.OpCall:
		// Setup arguments in RCX, RDX, R8, R9
		for i, arg := range inst.Args {
			disp := cg.localStack[arg]
			switch i {
			case 0:
				cg.emitMovRegStack("rcx", disp)
			case 1:
				cg.emitMovRegStack("rdx", disp)
			case 2:
				cg.emitMovRegStack("r8", disp)
			case 3:
				cg.emitMovRegStack("r9", disp)
			default:
				// push onto stack at [rsp + 0x20 + (i-4)*8]
				cg.emitMovRegStack("rax", disp)
				cg.emitBytes(0x48, 0x89, 0x44, 0x24, byte(0x20+(i-4)*8))
			}
		}
		callee := inst.StrVal
		switch callee {
		case "read_file", "write_file", "alloc_mem", "free_mem", "str_len", "str_eq", "str_char", "str_sub", "str_concat", "str_from_int", "get_cmd_arg":
			callee = "__gat_" + callee
		}
		cg.emitCallSym(callee)
		if inst.Dest != nil {
			cg.emitMovStackReg(cg.localStack[inst.Dest], "rax")
		}

	case ir.OpPrint:
		for _, arg := range inst.Args {
			disp := cg.localStack[arg]
			switch arg.Type.Kind() {
			case types.KindString:
				cg.emitMovRegStack("rcx", disp)
				cg.emitCallSym("__gat_print_str")
			case types.KindInt64:
				cg.emitMovRegStack("rcx", disp)
				cg.emitCallSym("__gat_print_i64")
			case types.KindBool:
				cg.emitMovRegStack("rcx", disp)
				cg.emitCallSym("__gat_print_bool")
			default:
				cg.emitMovRegStack("rcx", disp)
				cg.emitCallSym("__gat_print_i64")
			}
		}

	case ir.OpRetain:
		cg.emitMovRegStack("rcx", cg.localStack[inst.Src1])
		cg.emitCallSym("__gat_retain")

	case ir.OpRelease:
		cg.emitMovRegStack("rcx", cg.localStack[inst.Src1])
		cg.emitCallSym("__gat_release")

	case ir.OpReturn:
		if inst.Src1 != nil {
			cg.emitMovRegStack("rax", cg.localStack[inst.Src1])
		}
		// Epilogue: mov rsp, rbp; pop rbp; ret
		cg.emitBytes(0x48, 0x89, 0xEC, 0x5D, 0xC3)

	case ir.OpBranch:
		cg.emitMovRegStack("rax", cg.localStack[inst.Src1])
		// test rax, rax
		cg.emitBytes(0x48, 0x85, 0xC0)
		// jnz trueLabel; jmp falseLabel
		cg.emitBranchCond(0x0F, 0x85, inst.TrueLabel)
		cg.emitJump(inst.FalseLabel)

	case ir.OpJump:
		cg.emitJump(inst.Label)

	case ir.OpLabel:
		cg.defineLabel(inst.Label)
	}
}

// -------------------------------------------------------------
// Assembly Encoding Utilities
// -------------------------------------------------------------

func (cg *CodeGenerator) emitMovStackReg(disp int, reg string) {
	// mov [rbp + disp32], reg
	var regCode byte
	switch reg {
	case "rax":
		regCode = 0x85
	case "rcx":
		regCode = 0x8D
	case "rdx":
		regCode = 0x95
	case "rbx":
		regCode = 0x9D
	case "r8":
		cg.emitBytes(0x4C, 0x89, 0x85)
		cg.emitU32(uint32(int32(disp)))
		return
	case "r9":
		cg.emitBytes(0x4C, 0x89, 0x8D)
		cg.emitU32(uint32(int32(disp)))
		return
	}
	cg.emitBytes(0x48, 0x89, regCode)
	cg.emitU32(uint32(int32(disp)))
}

func (cg *CodeGenerator) emitMovRegStack(reg string, disp int) {
	// mov reg, [rbp + disp32]
	var regCode byte
	switch reg {
	case "rax":
		regCode = 0x85
	case "rcx":
		regCode = 0x8D
	case "rdx":
		regCode = 0x95
	case "rbx":
		regCode = 0x9D
	case "r8":
		cg.emitBytes(0x4C, 0x8B, 0x85)
		cg.emitU32(uint32(int32(disp)))
		return
	case "r9":
		cg.emitBytes(0x4C, 0x8B, 0x8D)
		cg.emitU32(uint32(int32(disp)))
		return
	}
	cg.emitBytes(0x48, 0x8B, regCode)
	cg.emitU32(uint32(int32(disp)))
}

func (cg *CodeGenerator) emitCallSym(sym string) {
	// call rel32
	cg.emitBytes(0xE8)
	cg.emitReloc(sym, RelocRipRelative, -4)
}

func (cg *CodeGenerator) emitCallIAT(fnName string) {
	// call qword ptr [rip + disp32] -> FF 15 disp32
	cg.emitBytes(0xFF, 0x15)
	cg.emitReloc("__iat_"+fnName, RelocIATRelative, -4)
}

func (cg *CodeGenerator) emitReloc(symbol string, kind RelocKind, addend int64) {
	offset := cg.currentOffset()
	cg.relocs = append(cg.relocs, Relocation{
		Offset: offset,
		Symbol: symbol,
		Kind:   kind,
		Addend: addend,
	})
	cg.emitU32(0)
}

func (cg *CodeGenerator) emitBranchCond(b1, b2 byte, label string) {
	cg.emitBytes(b1, b2)
	offset := cg.currentOffset()
	cg.pendingJumps = append(cg.pendingJumps, pendingJump{
		codeOffset: offset,
		label:      label,
	})
	cg.emitU32(0)
}

func (cg *CodeGenerator) emitJump(label string) {
	cg.emitBytes(0xE9)
	offset := cg.currentOffset()
	cg.pendingJumps = append(cg.pendingJumps, pendingJump{
		codeOffset: offset,
		label:      label,
	})
	cg.emitU32(0)
}

func isByteType(t types.Type) bool {
	if t == nil {
		return false
	}
	if t.Size() == 1 || t.Kind() == types.KindString {
		return true
	}
	if raw, ok := t.(*types.RawType); ok {
		return raw.BaseType.Size() == 1
	}
	return false
}
