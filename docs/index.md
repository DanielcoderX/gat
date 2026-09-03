---
layout: default
title: Home
nav_order: 1
description: "Gat is a fast, self-hosted, statically-typed systems programming language."
permalink: /
---

# The Gat Programming Language

**Gat** is a self-hosted, low-level systems programming language designed for mechanical sympathy, deterministic memory safety, and uncompromising native performance.

It compiles directly into standalone **x86-64 machine code** with zero external dependencies:
- **Windows**: Native PE32+ executables linked directly against `kernel32.dll`.
- **Linux**: Standalone ELF64 binaries using raw Linux kernel syscalls—**zero libc or dynamic linker required**.

```rust
// A complete, runnable Gat program
fn main() -> i64 {
    print("Hello, world! Welcome to Gat.\n");
    return 0;
}
```

---

## What Makes Gat Different?

### 1. Dual Memory Architecture (Value Structs vs. ARC Classes)
Gat provides precise control over data placement:
- **`struct` (Value Types)**: Reside on the stack or inline within enclosing types. Copied by value with **zero heap allocation** and **zero reference counting overhead**.
- **`class` (Reference Types)**: Managed via deterministic **Automatic Reference Counting (ARC)**. When the reference count reaches zero, memory is reclaimed immediately without unpredictable garbage-collection pauses.
- **Weak References (`weak T`)**: First-class non-owning references break cyclic structures safely.

### 2. Zero-Dependency Direct-Syscall Linux Binaries
Unlike languages that require `glibc`, `musl`, or dynamic linkers, Gat's Linux backend emits raw `syscall` instructions directly for:
- Memory allocation (`sys_mmap` / `sys_munmap`)
- File and console I/O (`sys_read`, `sys_write`, `sys_open`, `sys_close`, `sys_stat`)
- Multithreading (`sys_clone`) and process lifecycle (`sys_exit_group`, `sys_getpid`, `sys_nanosleep`)
- Futex-backed or atomic CAS synchronization primitives
- Sockets & TCP Networking (`sys_socket`, `sys_connect`, `sys_bind`, `sys_listen`, `sys_accept`, `sys_sendto`, `sys_recvfrom`)

A binary compiled with `gat build app.gat -o app --target=linux` runs on **any 64-bit Linux kernel** with zero shared library dependencies.

### 3. Fully Self-Hosted with Bitwise-Identical Stage Verification
Gat is 100% written in Gat (`src/compiler.gat`). Every build is validated via a multi-stage bootstrap pipeline:
1. `gatc` compiles `src/compiler.gat` &rarr; `gatc-stage2`
2. `gatc-stage2` compiles `src/compiler.gat` &rarr; `gatc-stage3`
3. A bitwise comparison (`fc /b` or `cmp`) proves that `gatc-stage2` and `gatc-stage3` are **100% bitwise identical**, proving compiler determinism and self-hosting correctness on both Windows and Linux.

### 4. Zero-Friction Native Toolchain
- **Built-in Package Manager**: `gat init`, `gat add`, `gat install` with lockfile verification (`gat.mod` & `gat.lock`).
- **Language Server Protocol (LSP)**: Complete editor support with diagnostics, hover inspection, go-to-definition, and autocomplete.
- **Cross-Platform Networking**: `std/net.gat` for TCP client/server streaming, high-level listeners, and automatic RAII socket lifecycles.
- **Modern Optimizing Pipeline**: AST type inference, generic monomorphization, dead-code elimination (DCE), SSA-inspired IR, constant folding, and linear-scan register allocation.

---

## Try Gat in 60 Seconds

### Download Pre-Built Binaries
Grab the latest release archive from [GitHub Releases](https://github.com/DanielcoderX/gat/releases/latest):
- **Windows**: `gat-v0.2.0-windows-x64.zip`
- **Linux**: `gat-v0.2.0-linux-x64.tar.gz`

### Run Your First Program
```powershell
# Windows
.\bin\gat.exe run examples\showcase\01_hello_world.gat

# Linux
./bin/gat run examples/showcase/01_hello_world.gat
```

### Compile to Standalone Native Binary
```powershell
# Build Windows PE32+
.\bin\gat.exe build app.gat -o app.exe

# Build Linux ELF64 (Cross-compile or Native)
.\bin\gat.exe build app.gat -o app --target=linux
```

---

## Documentation Roadmap

- [**Getting Started**](getting-started.html): Installation, compiler usage, project setup.
- [**Curated Example Gallery**](examples.html): Step-by-step tutorial programs from Hello World to mini CLI tools.
- [**Language Specification**](LANGUAGE_SPEC.html): Syntax, keywords, types, control flow, memory model, and EBNF grammar.
- [**Standard Library**](STDLIB.html): Built-in modules (`std/str.gat`, `std/fs.gat`, `std/math.gat`, `std/process.gat`, etc.).
- [**Module System**](MODULES.html): Namespaced imports, aliases, collision avoidance, and project layout.
- [**Dual Backend Guide**](dual_backend.html): Deep dive into PE32+ kernel32 IAT and direct Linux syscall emission.
- [**Contributing Guide**](CONTRIBUTING.html): Development workflow, running `test.ps1`, and PR conventions.
