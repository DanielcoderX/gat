---
layout: default
title: Dual Backend Architecture
nav_order: 7
description: "How Gat generates native code for Windows PE32+ and Linux ELF64."
permalink: /dual_backend.html
---

# Dual Backend Architecture

Gat features a data-driven backend design (`src/target.gat` and `src/codegen.gat`) allowing the compiler to emit standalone native binaries for two distinct OS platforms from a single unified AST and IR representation.

---

## 1. Windows Backend (PE32+)

### Target Selection
Default behavior when running on Windows, or explicitly via:
```powershell
gat build main.gat -o main.exe --target=windows
```

### Architecture & Format
- **Format**: Portable Executable (PE32+) 64-bit console subsystem.
- **Import Address Table (IAT)**: Direct linking against `kernel32.dll` APIs:
  - `GetStdHandle`, `WriteFile`, `ReadFile`, `CreateFileA`, `CloseHandle`
  - `VirtualAlloc`, `VirtualFree` (for dynamic heap sizing)
  - `CreateThread`, `WaitForSingleObject` (for concurrency)
  - `GetCommandLineA`, `ExitProcess`, `GetProcessId`, `Sleep`
- **Zero C Runtime Dependency**: Emits no references to `msvcrt.dll` or `ucrtbase.dll`.

---

## 2. Linux Backend (ELF64 Direct Syscalls)

### Target Selection
```bash
gat build main.gat -o main --target=linux
```

### Architecture & Format
- **Format**: Executable and Linkable Format 64-bit (ELF64), static executable (`ET_EXEC`).
- **Zero Libc Dependency**: Bypasses `glibc`, `musl`, and dynamic linkers completely. Emits direct x86-64 `syscall` instructions:

| Syscall Name | Syscall Number (`rax`) | Purpose in Gat Runtime |
|---|---|---|
| `sys_read` | 0 | File and stdin reading (`read_file`, `fs_read_all`) |
| `sys_write` | 1 | Console output and file writing (`print`, `write_file`) |
| `sys_open` | 2 | File descriptor acquisition |
| `sys_close` | 3 | Resource cleanup |
| `sys_stat` | 4 | File metadata and size queries (`file_size`) |
| `sys_mmap` | 9 | Heap page allocation (`alloc_mem`) |
| `sys_munmap` | 11 | Heap page reclamation (`free_mem`) |
| `sys_nanosleep` | 35 | Process sleep (`proc_sleep`) |
| `sys_getpid` | 39 | Process ID query (`proc_get_pid`) |
| `sys_clone` | 56 | Native kernel thread spawning (`thread_spawn`) |
| `sys_exit_group` | 231 | Clean multi-threaded process termination |

---

## 3. Bitwise Reproducible Self-Hosting

The compiler is written in Gat itself and can self-host on both platforms:
- On Windows: `fc /b gatc-stage2.exe gatc-stage3.exe` produces `0 bytes difference` (100% bitwise identity).
- On Linux: `cmp gatc-gen2 gatc-gen3` produces an exact bitwise match.
