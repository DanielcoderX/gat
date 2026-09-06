---
layout: default
title: Multi-Backend Architecture
nav_order: 7
description: "How Gat generates native code for Windows PE32+, Linux ELF64, and macOS Mach-O ARM64."
permalink: /dual_backend.html
---

# Multi-Backend Architecture

Gat features a data-driven backend design (`src/target.gat`, `src/codegen.gat`, and `src/codegen_arm64.gat`) allowing the compiler to emit standalone native binaries across Windows PE32+, Linux ELF64 (x86-64 and AArch64), and macOS Apple Silicon Mach-O from a single unified AST and IR representation.

---

## 1. macOS Apple Silicon Backend (Mach-O ARM64)

### Target Selection
Default behavior when running on macOS Apple Silicon, or explicitly via:
```bash
gat build main.gat -o main --target=macos-arm64
```

### Architecture & Format
- **Format**: Mach-O 64-bit Relocatable Object (`MH_OBJECT`, `CPU_TYPE_ARM64`).
- **ABI & Syscalls**: Darwin AArch64 ABI calling convention:
  - Register argument passing: `X0` through `X7`.
  - Direct kernel syscall invocation via `SVC #0x80`, syscall number in `X16`:
    - `sys_mmap` (197) and `sys_munmap` (73)
    - `sys_write` (4), `sys_read` (3), `sys_open` (5), `sys_close` (6), `sys_lseek` (199)
    - `sys_getpid` (20)
  - Stack management using 64-bit extended register operations conforming to Apple's 16-byte alignment requirements.
- **Linker Integration**: The compiler driver invokes the host linker (`clang <obj.o> -o <out>`) to sign and produce executable binaries, or outputs raw `.o` files when `-o file.o` is specified.

---

## 2. Linux Backend (ELF64 Direct Syscalls: x86-64 & ARM64)

### Target Selection
```bash
# x86-64 Linux
gat build main.gat -o main --target=linux

# ARM64 / AArch64 Linux
gat build main.gat -o main --target=linux-arm64
```

### Architecture & Format
- **Format**: Executable and Linkable Format 64-bit (ELF64), static executable (`ET_EXEC`).
- **Zero Libc Dependency**: Emits direct `syscall` (x86-64) or `svc #0` (ARM64) kernel instructions:

| Syscall Name | x86-64 (`rax`) | ARM64 (`x8`) | Purpose in Gat Runtime |
|---|---|---|---|
| `sys_read` | 0 | 63 | File and stdin reading (`read_file`, `fs_read_all`) |
| `sys_write` | 1 | 64 | Console output and file writing (`print`, `write_file`) |
| `sys_open` | 2 | 56 (openat) | File descriptor acquisition |
| `sys_close` | 3 | 57 | Resource cleanup |
| `sys_stat` | 4 | 80 (fstat) | File metadata and size queries (`file_size`) |
| `sys_mmap` | 9 | 222 | Heap page allocation (`alloc_mem`) |
| `sys_munmap` | 11 | 215 | Heap page reclamation (`free_mem`) |
| `sys_nanosleep` | 35 | 101 | Process sleep (`proc_sleep`) |
| `sys_getpid` | 39 | 172 | Process ID query (`proc_get_pid`) |
| `sys_clone` | 56 | 220 | Native kernel thread spawning (`thread_spawn`) |
| `sys_exit_group` | 231 | 94 | Clean multi-threaded process termination |

---

## 3. Windows Backend (PE32+)

### Target Selection
Default behavior when running on Windows, or explicitly via:
```powershell
gat build main.gat -o main.exe --target=windows
```

### Architecture & Format
- **Format**: Portable Executable (PE32+) 64-bit console subsystem.
- **Import Address Table (IAT)**: Direct linking against `kernel32.dll` APIs:
  - `GetStdHandle`, `WriteFile`, `ReadFile`, `CreateFileA`, `CloseHandle`
  - `VirtualAlloc`, `VirtualFree` (dynamic heap sizing)
  - `CreateThread`, `WaitForSingleObject` (concurrency)
  - `GetCommandLineA`, `ExitProcess`, `GetProcessId`, `Sleep`
- **Zero C Runtime Dependency**: Emits no references to `msvcrt.dll` or `ucrtbase.dll`.

---

## 4. Bitwise Reproducible Self-Hosting

The compiler is written in Gat itself (`src/compiler.gat`) and self-hosts across all supported platforms:
- **Windows**: `fc /b gatc-stage2.exe gatc-stage3.exe` produces 0 byte differences (100% bitwise identity).
- **Linux**: `cmp gatc-gen2 gatc-gen3` produces 0 byte differences.
- **macOS ARM64**: `cmp bin/gatc_stage2.o bin/gatc_stage3.o` produces 0 byte differences.
