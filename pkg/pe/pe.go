package pe

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"

	"gat/pkg/codegen"
	"gat/pkg/ir"
)

type ImportSymbol struct {
	Name string
	Hint uint16
}

var kernel32Imports = []ImportSymbol{
	{Name: "ExitProcess", Hint: 0},
	{Name: "GetStdHandle", Hint: 0},
	{Name: "WriteFile", Hint: 0},
	{Name: "GetProcessHeap", Hint: 0},
	{Name: "HeapAlloc", Hint: 0},
	{Name: "HeapFree", Hint: 0},
	{Name: "CreateFileA", Hint: 0},
	{Name: "ReadFile", Hint: 0},
	{Name: "CloseHandle", Hint: 0},
	{Name: "GetFileSizeEx", Hint: 0},
	{Name: "GetCommandLineA", Hint: 0},
}

type PEBuilder struct {
	irProg     *ir.Program
	code       []byte
	relocs     []codegen.Relocation
	symOffsets map[string]int

	imageBytes []byte
}

func NewBuilder(irProg *ir.Program, code []byte, relocs []codegen.Relocation, symOffsets map[string]int) *PEBuilder {
	return &PEBuilder{
		irProg:     irProg,
		code:       code,
		relocs:     relocs,
		symOffsets: symOffsets,
	}
}

func alignUp(val, align uint32) uint32 {
	if val%align == 0 {
		return val
	}
	return val + (align - (val % align))
}

func (pe *PEBuilder) BuildExecutable(outputPath string) error {
	const (
		sectionAlign uint32 = 0x1000
		fileAlign    uint32 = 0x200
		imageBase    uint64 = 0x0000000140000000
	)

	// -------------------------------------------------------------
	// 1. Prepare Entrypoint Trampoline in .text
	// -------------------------------------------------------------
	// Entry trampoline:
	// sub rsp, 40 (shadow space + align)
	// call main
	// mov rcx, rax (exit code)
	// call ExitProcess
	var entryCode []byte
	entryCode = append(entryCode,
		0x48, 0x83, 0xEC, 0x28, // sub rsp, 40
	)
	entryMainCallOffset := len(entryCode)
	entryCode = append(entryCode, 0xE8, 0x00, 0x00, 0x00, 0x00) // call rel32 (main)
	entryCode = append(entryCode, 0x48, 0x89, 0xC1)             // mov rcx, rax
	entryExitCallOffset := len(entryCode)
	entryCode = append(entryCode, 0xFF, 0x15, 0x00, 0x00, 0x00, 0x00) // call qword ptr [rip + __iat_ExitProcess]

	// Full .text code = entryCode + generated code
	fullText := make([]byte, 0, len(entryCode)+len(pe.code))
	fullText = append(fullText, entryCode...)
	codeBaseOffset := len(entryCode)
	fullText = append(fullText, pe.code...)

	// Adjust symbol offsets by entryCode length
	textSymOffsets := make(map[string]int)
	for k, v := range pe.symOffsets {
		textSymOffsets[k] = v + codeBaseOffset
	}

	// -------------------------------------------------------------
	// 2. Prepare .rdata Section: Strings + Imports + IAT
	// -------------------------------------------------------------
	var rdataBuf bytes.Buffer

	// (A) String literals
	rdataStringOffsets := make(map[string]uint32)

	// Boolean constants
	rdataStringOffsets["str_true_const"] = uint32(rdataBuf.Len())
	rdataBuf.WriteString("true\x00")
	rdataStringOffsets["str_false_const"] = uint32(rdataBuf.Len())
	rdataBuf.WriteString("false\x00")

	// User strings from IR
	for _, sc := range pe.irProg.Strings {
		rdataStringOffsets[sc.Label] = uint32(rdataBuf.Len())
		rdataBuf.WriteString(sc.Value)
		rdataBuf.WriteByte(0)
	}

	// Align to 8 bytes
	for rdataBuf.Len()%8 != 0 {
		rdataBuf.WriteByte(0)
	}

	// (B) IAT (Import Address Table / FirstThunk)
	iatStartOffset := uint32(rdataBuf.Len())
	iatSymbolOffsets := make(map[string]uint32)
	for _, imp := range kernel32Imports {
		iatSymbolOffsets["__iat_"+imp.Name] = uint32(rdataBuf.Len())
		// Placeholder for 64-bit thunk address
		var zero [8]byte
		rdataBuf.Write(zero[:])
	}
	// Null terminator entry for IAT
	var zeroThunk [8]byte
	rdataBuf.Write(zeroThunk[:])
	iatEndOffset := uint32(rdataBuf.Len())

	// (C) ILT (Import Lookup Table / OriginalFirstThunk)
	iltStartOffset := uint32(rdataBuf.Len())
	for range kernel32Imports {
		var zero [8]byte
		rdataBuf.Write(zero[:])
	}
	rdataBuf.Write(zeroThunk[:])

	// (D) Import Directory Table (20 bytes per descriptor, ends with 20 zero bytes)
	importDirOffset := uint32(rdataBuf.Len())
	// IMAGE_IMPORT_DESCRIPTOR placeholder (20 bytes)
	var importDesc [20]byte
	rdataBuf.Write(importDesc[:])
	// Null descriptor (20 bytes)
	rdataBuf.Write(importDesc[:])
	importDirEndOffset := uint32(rdataBuf.Len())

	// (E) DLL Name string: "KERNEL32.dll\0"
	dllNameOffset := uint32(rdataBuf.Len())
	rdataBuf.WriteString("KERNEL32.dll\x00")
	for rdataBuf.Len()%2 != 0 {
		rdataBuf.WriteByte(0)
	}

	// (F) Hint/Name entries for each import symbol
	hintNameOffsets := make([]uint32, len(kernel32Imports))
	for i, imp := range kernel32Imports {
		hintNameOffsets[i] = uint32(rdataBuf.Len())
		var hint [2]byte
		binary.LittleEndian.PutUint16(hint[:], imp.Hint)
		rdataBuf.Write(hint[:])
		rdataBuf.WriteString(imp.Name)
		rdataBuf.WriteByte(0)
		for rdataBuf.Len()%2 != 0 {
			rdataBuf.WriteByte(0)
		}
	}

	rdataBytes := rdataBuf.Bytes()

	// -------------------------------------------------------------
	// 3. Calculate Layout & RVAs
	// -------------------------------------------------------------
	headerSize := alignUp(0x400, fileAlign) // DOS + PE + Section headers
	textRVA := sectionAlign
	textRawSize := alignUp(uint32(len(fullText)), fileAlign)
	textSize := alignUp(uint32(len(fullText)), sectionAlign)

	rdataRVA := textRVA + textSize
	rdataRawSize := alignUp(uint32(len(rdataBytes)), fileAlign)
	rdataSize := alignUp(uint32(len(rdataBytes)), sectionAlign)

	imageSize := rdataRVA + rdataSize

	// Now patch .rdata internal RVAs:
	// Fill ILT and IAT entries with RVA of Hint/Name table
	for i := range kernel32Imports {
		hnRVA := uint64(rdataRVA + hintNameOffsets[i])
		// Patch ILT
		iltEntryOffset := iltStartOffset + uint32(i*8)
		binary.LittleEndian.PutUint64(rdataBytes[iltEntryOffset:iltEntryOffset+8], hnRVA)
		// Patch IAT
		iatEntryOffset := iatStartOffset + uint32(i*8)
		binary.LittleEndian.PutUint64(rdataBytes[iatEntryOffset:iatEntryOffset+8], hnRVA)
	}

	// Patch Import Directory Descriptor:
	// OriginalFirstThunk = ILT RVA
	binary.LittleEndian.PutUint32(rdataBytes[importDirOffset+0:], rdataRVA+iltStartOffset)
	// TimeDateStamp = 0
	binary.LittleEndian.PutUint32(rdataBytes[importDirOffset+4:], 0)
	// ForwarderChain = 0
	binary.LittleEndian.PutUint32(rdataBytes[importDirOffset+8:], 0)
	// Name = DLL name RVA
	binary.LittleEndian.PutUint32(rdataBytes[importDirOffset+12:], rdataRVA+dllNameOffset)
	// FirstThunk = IAT RVA
	binary.LittleEndian.PutUint32(rdataBytes[importDirOffset+16:], rdataRVA+iatStartOffset)

	// -------------------------------------------------------------
	// 4. Resolve Relocations in .text
	// -------------------------------------------------------------
	// (A) Patch entry trampoline call to main
	mainOffset, hasMain := textSymOffsets["main"]
	if !hasMain {
		return fmt.Errorf("no 'main' function found in program")
	}
	mainDisp := int32(mainOffset - (entryMainCallOffset + 5))
	binary.LittleEndian.PutUint32(fullText[entryMainCallOffset+1:entryMainCallOffset+5], uint32(mainDisp))

	// (B) Patch entry trampoline call to ExitProcess
	exitProcessIatOffset := iatSymbolOffsets["__iat_ExitProcess"]
	exitTargetRVA := rdataRVA + exitProcessIatOffset
	exitCallRipRVA := textRVA + uint32(entryExitCallOffset+6) // RIP after FF 15 disp32
	exitDisp := int32(exitTargetRVA - exitCallRipRVA)
	binary.LittleEndian.PutUint32(fullText[entryExitCallOffset+2:entryExitCallOffset+6], uint32(exitDisp))

	// (C) Patch code generator relocations
	for _, rel := range pe.relocs {
		targetInCodeOffset := rel.Offset + codeBaseOffset
		ripRVA := textRVA + uint32(targetInCodeOffset+4)

		switch rel.Kind {
		case codegen.RelocRipRelative:
			// Target could be a text symbol or an rdata string
			if symOff, ok := textSymOffsets[rel.Symbol]; ok {
				targetRVA := textRVA + uint32(symOff)
				disp := int32(targetRVA - ripRVA)
				binary.LittleEndian.PutUint32(fullText[targetInCodeOffset:targetInCodeOffset+4], uint32(disp))
			} else if strOff, ok := rdataStringOffsets[rel.Symbol]; ok {
				targetRVA := rdataRVA + strOff
				disp := int32(targetRVA - ripRVA)
				binary.LittleEndian.PutUint32(fullText[targetInCodeOffset:targetInCodeOffset+4], uint32(disp))
			} else {
				return fmt.Errorf("unresolved symbol for RIP relocation: %s", rel.Symbol)
			}

		case codegen.RelocIATRelative:
			if iatOff, ok := iatSymbolOffsets[rel.Symbol]; ok {
				targetRVA := rdataRVA + iatOff
				disp := int32(targetRVA - ripRVA)
				binary.LittleEndian.PutUint32(fullText[targetInCodeOffset:targetInCodeOffset+4], uint32(disp))
			} else {
				return fmt.Errorf("unresolved IAT symbol: %s", rel.Symbol)
			}
		}
	}

	// -------------------------------------------------------------
	// 5. Build PE Headers & Assemble Final File
	// -------------------------------------------------------------
	out := &bytes.Buffer{}

	// (A) DOS Header (64 bytes)
	dosHeader := make([]byte, 64)
	dosHeader[0] = 'M'
	dosHeader[1] = 'Z'
	binary.LittleEndian.PutUint32(dosHeader[0x3C:], 0x80) // e_lfanew = 128
	out.Write(dosHeader)

	// (B) DOS Stub (64 bytes)
	dosStub := make([]byte, 64)
	out.Write(dosStub)

	// (C) PE Signature (4 bytes)
	out.WriteString("PE\x00\x00")

	// (D) COFF File Header (20 bytes)
	var coff [20]byte
	binary.LittleEndian.PutUint16(coff[0:], 0x8664) // Machine AMD64
	binary.LittleEndian.PutUint16(coff[2:], 2)      // Number of sections (.text, .rdata)
	binary.LittleEndian.PutUint32(coff[4:], 0)      // TimeDateStamp
	binary.LittleEndian.PutUint32(coff[8:], 0)      // PointerToSymbolTable
	binary.LittleEndian.PutUint32(coff[12:], 0)     // NumberOfSymbols
	binary.LittleEndian.PutUint16(coff[16:], 0xF0)  // SizeOfOptionalHeader (240 bytes)
	binary.LittleEndian.PutUint16(coff[18:], 0x0022) // Characteristics: EXECUTABLE_IMAGE | LARGE_ADDRESS_AWARE
	out.Write(coff[:])

	// (E) Optional Header Standard Fields + Windows Specific (240 bytes)
	var opt [240]byte
	binary.LittleEndian.PutUint16(opt[0:], 0x020B)              // Magic: PE32+ (64-bit)
	opt[2] = 1                                                 // MajorLinkerVersion
	opt[3] = 0                                                 // MinorLinkerVersion
	binary.LittleEndian.PutUint32(opt[4:], textRawSize)        // SizeOfCode
	binary.LittleEndian.PutUint32(opt[8:], rdataRawSize)       // SizeOfInitializedData
	binary.LittleEndian.PutUint32(opt[12:], 0)                 // SizeOfUninitializedData
	binary.LittleEndian.PutUint32(opt[16:], textRVA)           // AddressOfEntryPoint (trampoline at start of .text)
	binary.LittleEndian.PutUint32(opt[20:], textRVA)           // BaseOfCode
	binary.LittleEndian.PutUint64(opt[24:], imageBase)         // ImageBase
	binary.LittleEndian.PutUint32(opt[32:], sectionAlign)      // SectionAlignment
	binary.LittleEndian.PutUint32(opt[36:], fileAlign)         // FileAlignment
	binary.LittleEndian.PutUint16(opt[40:], 6)                 // MajorOperatingSystemVersion
	binary.LittleEndian.PutUint16(opt[42:], 0)                 // MinorOperatingSystemVersion
	binary.LittleEndian.PutUint16(opt[48:], 6)                 // MajorSubsystemVersion
	binary.LittleEndian.PutUint16(opt[50:], 0)                 // MinorSubsystemVersion
	binary.LittleEndian.PutUint32(opt[56:], imageSize)         // SizeOfImage
	binary.LittleEndian.PutUint32(opt[60:], headerSize)        // SizeOfHeaders
	binary.LittleEndian.PutUint16(opt[68:], 3)                 // Subsystem: IMAGE_SUBSYSTEM_WINDOWS_CUI (Console)
	binary.LittleEndian.PutUint16(opt[70:], 0x8160)            // DllCharacteristics: DYNAMIC_BASE | NX_COMPAT | TERMINAL_SERVER_AWARE
	binary.LittleEndian.PutUint64(opt[72:], 0x800000)          // SizeOfStackReserve (8MB)
	binary.LittleEndian.PutUint64(opt[80:], 0x800000)          // SizeOfStackCommit (8MB)
	binary.LittleEndian.PutUint64(opt[88:], 0x100000)          // SizeOfHeapReserve (1MB)
	binary.LittleEndian.PutUint64(opt[96:], 0x1000)            // SizeOfHeapCommit (4KB)
	binary.LittleEndian.PutUint32(opt[108:], 16)               // NumberOfRvaAndSizes

	// Data Directories
	// Index 1: Import Table
	importDirRVA := rdataRVA + importDirOffset
	importDirSize := importDirEndOffset - importDirOffset
	binary.LittleEndian.PutUint32(opt[120:], importDirRVA)
	binary.LittleEndian.PutUint32(opt[124:], importDirSize)

	// Index 12: IAT Table
	iatRVA := rdataRVA + iatStartOffset
	iatSize := iatEndOffset - iatStartOffset
	binary.LittleEndian.PutUint32(opt[208:], iatRVA)
	binary.LittleEndian.PutUint32(opt[212:], iatSize)

	out.Write(opt[:])

	// (F) Section Headers (40 bytes each)
	// 1. .text Section Header
	var textSec [40]byte
	copy(textSec[0:8], []byte(".text\x00\x00\x00"))
	binary.LittleEndian.PutUint32(textSec[8:], uint32(len(fullText))) // VirtualSize
	binary.LittleEndian.PutUint32(textSec[12:], textRVA)              // VirtualAddress
	binary.LittleEndian.PutUint32(textSec[16:], textRawSize)          // SizeOfRawData
	binary.LittleEndian.PutUint32(textSec[20:], headerSize)           // PointerToRawData
	binary.LittleEndian.PutUint32(textSec[36:], 0x60000020)          // Characteristics: CODE | EXECUTE | READ
	out.Write(textSec[:])

	// 2. .rdata Section Header
	var rdataSec [40]byte
	copy(rdataSec[0:8], []byte(".rdata\x00\x00"))
	binary.LittleEndian.PutUint32(rdataSec[8:], uint32(len(rdataBytes))) // VirtualSize
	binary.LittleEndian.PutUint32(rdataSec[12:], rdataRVA)               // VirtualAddress
	binary.LittleEndian.PutUint32(rdataSec[16:], rdataRawSize)           // SizeOfRawData
	binary.LittleEndian.PutUint32(rdataSec[20:], headerSize+textRawSize) // PointerToRawData
	binary.LittleEndian.PutUint32(rdataSec[36:], 0xC0000040)           // Characteristics: INITIALIZED_DATA | READ | WRITE
	out.Write(rdataSec[:])

	// Pad headers to headerSize (0x400)
	for out.Len() < int(headerSize) {
		out.WriteByte(0)
	}

	// (G) Write .text section data + padding
	out.Write(fullText)
	for out.Len() < int(headerSize+textRawSize) {
		out.WriteByte(0)
	}

	// (H) Write .rdata section data + padding
	out.Write(rdataBytes)
	for out.Len() < int(headerSize+textRawSize+rdataRawSize) {
		out.WriteByte(0)
	}

	// Write output binary
	return os.WriteFile(outputPath, out.Bytes(), 0755)
}
