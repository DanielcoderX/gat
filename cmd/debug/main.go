package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procCreateProcess  = kernel32.NewProc("CreateProcessW")
	procWaitForDebug   = kernel32.NewProc("WaitForDebugEvent")
	procContinueDebug  = kernel32.NewProc("ContinueDebugEvent")
	procGetThreadContext = kernel32.NewProc("GetThreadContext")
)

const (
	DEBUG_ONLY_THIS_PROCESS = 0x00000002
	EXCEPTION_DEBUG_EVENT   = 1
	CREATE_PROCESS_DEBUG_EVENT = 3
	EXIT_PROCESS_DEBUG_EVENT = 5
	DBG_CONTINUE            = 0x00010002
	DBG_EXCEPTION_NOT_HANDLED = 0x80010001
	CONTEXT_ALL             = 0x0010001F
)

type CONTEXT64 struct {
	P1Home               uint64
	P2Home               uint64
	P3Home               uint64
	P4Home               uint64
	P5Home               uint64
	P6Home               uint64
	ContextFlags         uint32
	MxCsr                uint32
	SegCs                uint16
	SegDs                uint16
	SegEs                uint16
	SegFs                uint16
	SegGs                uint16
	SegSs                uint16
	EFlags               uint32
	Dr0, Dr1, Dr2, Dr3   uint64
	Dr6, Dr7             uint64
	Rax, Rcx, Rdx, Rbx   uint64
	Rsp, Rbp, Rsi, Rdi   uint64
	R8, R9, R10, R11     uint64
	R12, R13, R14, R15   uint64
	Rip                  uint64
	// FPU / XMM follows
	_dummy               [512]byte
}

type EXCEPTION_RECORD64 struct {
	ExceptionCode        uint32
	ExceptionFlags       uint32
	ExceptionRecord      uint64
	ExceptionAddress     uint64
	NumberParameters     uint32
	_padding             uint32
	ExceptionInformation [15]uint64
}

type EXCEPTION_DEBUG_INFO64 struct {
	ExceptionRecord EXCEPTION_RECORD64
	FirstChance     uint32
}

type CREATE_PROCESS_DEBUG_INFO struct {
	HFile                 syscall.Handle
	HProcess               syscall.Handle
	HThread                syscall.Handle
	LpBaseOfImage          uint64
	DwDebugInfoFileOffset uint32
	NNumberOfBytesToRead  uint32
	LpThreadLocalBase     uint64
	LpStartAddress        uint64
	LpImageName           uint64
	FUnicode              uint16
}

type DEBUG_EVENT struct {
	DebugEventCode uint32
	ProcessId      uint32
	ThreadId       uint32
	_padding       uint32
	_u             [160]byte
}

func main() {
	cmd := "bin\\gatc-v2.exe src/compiler.gat -o bin/gatc-v3.exe"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	cmdPtr, _ := syscall.UTF16PtrFromString(cmd)
	var si syscall.StartupInfo
	var pi syscall.ProcessInformation
	si.Cb = uint32(unsafe.Sizeof(si))

	r, _, err := procCreateProcess.Call(
		0,
		uintptr(unsafe.Pointer(cmdPtr)),
		0, 0, 0,
		DEBUG_ONLY_THIS_PROCESS,
		0, 0,
		uintptr(unsafe.Pointer(&si)),
		uintptr(unsafe.Pointer(&pi)),
	)
	if r == 0 {
		fmt.Printf("CreateProcess failed: %v\n", err)
		return
	}

	var de DEBUG_EVENT
	var imageBase uint64
	var hThread syscall.Handle
	var cpHProcess syscall.Handle

	for {
		r, _, err := procWaitForDebug.Call(uintptr(unsafe.Pointer(&de)), 0xFFFFFFFF)
		if r == 0 {
			fmt.Printf("WaitForDebugEvent failed: %v\n", err)
			break
		}

		contStatus := uint32(DBG_CONTINUE)

		switch de.DebugEventCode {
		case CREATE_PROCESS_DEBUG_EVENT:
			cp := (*CREATE_PROCESS_DEBUG_INFO)(unsafe.Pointer(&de._u[0]))
			imageBase = cp.LpBaseOfImage
			hThread = cp.HThread
			cpHProcess = cp.HProcess
			fmt.Printf("Process created, ImageBase = 0x%X, hThread = %v\n", imageBase, hThread)

		case EXCEPTION_DEBUG_EVENT:
			exc := (*EXCEPTION_DEBUG_INFO64)(unsafe.Pointer(&de._u[0]))
			code := exc.ExceptionRecord.ExceptionCode
			addr := exc.ExceptionRecord.ExceptionAddress
			fmt.Printf("Exception: Code 0x%X at 0x%X (RVA 0x%X), FirstChance=%d\n",
				code, addr, addr-imageBase, exc.FirstChance)

			if code != 0x80000003 {
				var ctx CONTEXT64
				ctx.ContextFlags = CONTEXT_ALL
				procGetThreadContext.Call(uintptr(hThread), uintptr(unsafe.Pointer(&ctx)))
				fmt.Printf("Registers:\n")
				fmt.Printf("  RIP = 0x%X (RVA 0x%X)\n", ctx.Rip, ctx.Rip-imageBase)
				fmt.Printf("  RSP = 0x%X (rsp%%16 = %d)\n", ctx.Rsp, ctx.Rsp%16)
				fmt.Printf("  RBP = 0x%X\n", ctx.Rbp)
				fmt.Printf("  RAX = 0x%X\n", ctx.Rax)
				fmt.Printf("  RCX = 0x%X\n", ctx.Rcx)
				fmt.Printf("  RDX = 0x%X\n", ctx.Rdx)
				fmt.Printf("  R8  = 0x%X\n", ctx.R8)
				fmt.Printf("  R9  = 0x%X\n", ctx.R9)
				fmt.Printf("  R10 = 0x%X\n", ctx.R10)

				fmt.Printf("Stack frames (RSP=0x%X):\n", ctx.Rsp)
				for i := 0; i < 50; i++ {
					var val uint64
					kernel32.NewProc("ReadProcessMemory").Call(
						uintptr(cpHProcess),
						uintptr(ctx.Rsp+uint64(i*8)),
						uintptr(unsafe.Pointer(&val)),
						8,
						0,
					)
					if val >= imageBase && val < imageBase+0x100000 {
						fmt.Printf("  [RSP+%02X] = 0x%X (RVA 0x%X, .text off 0x%X)\n",
							i*8, val, val-imageBase, val-imageBase-0x1000)
					} else {
						fmt.Printf("  [RSP+%02X] = 0x%X\n", i*8, val)
					}
				}
				return
			}

		case EXIT_PROCESS_DEBUG_EVENT:
			exitCode := *(*uint32)(unsafe.Pointer(&de._u[0]))
			fmt.Printf("Process exited with code 0x%X (%d)\n", exitCode, int32(exitCode))
			procContinueDebug.Call(uintptr(de.ProcessId), uintptr(de.ThreadId), uintptr(contStatus))
			return
		}

		procContinueDebug.Call(uintptr(de.ProcessId), uintptr(de.ThreadId), uintptr(contStatus))
	}
}
