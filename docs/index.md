---
layout: default
title: Home
nav_order: 1
description: "Gat is a fast, self-hosted, statically-typed systems programming language."
permalink: /
---

# The Gat Programming Language

**Gat** is a self-hosted, low-level systems programming language designed for mechanical sympathy, deterministic memory safety, and uncompromising native performance.

It compiles directly into standalone native machine code with zero external toolchain requirements:
- **macOS Apple Silicon**: Native Mach-O 64-bit ARM64 objects with direct Darwin syscalls (`sys_mmap`, `sys_write`, `sys_open`, `sys_close`).
- **Linux (x86-64 & ARM64)**: Standalone ELF64 binaries using raw Linux kernel syscalls—**zero libc or dynamic linker required**.
- **Windows (x86-64)**: Native PE32+ executables linked directly against `kernel32.dll`.

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

### 2. Zero-Dependency Direct-Syscall Linux & macOS Binaries
Unlike languages that require `glibc`, `musl`, or dynamic linkers, Gat's Linux backend emits raw `syscall` instructions directly for:
- Memory allocation (`sys_mmap` / `sys_munmap`)
- File and console I/O (`sys_read`, `sys_write`, `sys_open`, `sys_close`, `sys_stat`)
- Multithreading (`sys_clone` on x86-64; ARM64 concurrency is experimental) and process lifecycle (`sys_exit_group`, `sys_getpid`, `sys_nanosleep`)
- Futex-backed or atomic CAS synchronization primitives
- Sockets & TCP Networking (`sys_socket`, `sys_connect`, `sys_bind`, `sys_listen`, `sys_accept`, `sys_sendto`, `sys_recvfrom`)

On macOS Apple Silicon (ARM64), Gat emits native Mach-O object files utilizing Darwin `SVC #0x80` direct kernel syscalls, ensuring maximum efficiency without runtime baggage.

### 3. Fully Self-Hosted with Bitwise-Identical Stage Verification
Gat is 100% written in Gat (`src/compiler.gat`). Every build is validated via a multi-stage bootstrap pipeline:
1. `gatc` compiles `src/compiler.gat` &rarr; `gatc-stage2`
2. `gatc-stage2` compiles `src/compiler.gat` &rarr; `gatc-stage3`
3. A bitwise comparison (`fc /b` or `cmp`) proves that `gatc-stage2` and `gatc-stage3` are **100% bitwise identical**, proving compiler determinism and self-hosting correctness on Windows, Linux, and macOS.

### 4. Zero-Friction Native Toolchain
- **Built-in Package Manager**: `gat init`, `gat add`, `gat install` with lockfile verification (`gat.mod` & `gat.lock`).
- **Language Server Protocol (LSP)**: Complete editor support with diagnostics, hover inspection, go-to-definition, and autocomplete.
- **Cross-Platform Networking**: `std/net.gat` for TCP client/server streaming, high-level listeners, and automatic RAII socket lifecycles.
- **Modern Optimizing Pipeline**: AST type inference, uniform word-sized generics erasure, dead-code elimination (DCE), SSA-inspired IR, constant folding, and linear-scan register allocation.

---

## Try Gat in 60 Seconds

### Download Pre-Built Binaries
Grab the latest release archive from [GitHub Releases](https://github.com/DanielcoderX/gat/releases/latest):
- **Windows x86-64**: `gat-v1.0.0-windows-x64.zip`
- **Linux x86-64**: `gat-v1.0.0-linux-x64.tar.gz`
- **Linux ARM64**: `gat-v1.0.0-linux-arm64.tar.gz`
- **macOS ARM64**: `gat-v1.0.0-macos-arm64.tar.gz`

### Run Your First Program
```powershell
# Windows
.\bin\gat.exe run examples\showcase\01_hello_world.gat

# Linux & macOS
./bin/gat run examples/showcase/01_hello_world.gat
```

### Compile to Standalone Native Binary
```powershell
# Build Windows PE32+
.\bin\gat.exe build app.gat -o app.exe

# Build Linux ELF64 (Cross-compile or Native)
./bin/gat build app.gat -o app --target=linux

# Build macOS Apple Silicon (Native Mach-O)
./bin/gat build app.gat -o app --target=macos-arm64
```

---

## Documentation Roadmap

- [**Stability Promise**](STABILITY.html): Formal v1.0 stability guarantee, platform support matrix, and versioning policy.
- [**Diagnostics Catalog**](DIAGNOSTICS.html): Error codes (`E0001`–`E0010`), explanation engine, and code fix examples.
- [**Getting Started**](getting-started.html): Installation, compiler usage, project setup.
- [**Curated Example Gallery**](examples.html): Step-by-step tutorial programs from Hello World to mini CLI tools.
- [**Language Specification**](LANGUAGE_SPEC.html): Syntax, keywords, types, control flow, memory model, and EBNF grammar.
- [**Standard Library**](STDLIB.html): Built-in modules (`std/str.gat`, `std/fs.gat`, `std/math.gat`, `std/process.gat`, etc.).
- [**Module System**](MODULES.html): Namespaced imports, aliases, collision avoidance, and project layout.
- [**Multi-Backend Architecture**](dual_backend.html): Deep dive into PE32+ kernel32 IAT, direct Linux syscalls, and Darwin Mach-O ARM64 emission.
- [**Performance Benchmarks**](BENCHMARKS.html): Reproducible load benchmarks for concurrent HTTP serving.
- [**Contributing Guide**](CONTRIBUTING.html): Development workflow, running test suites, and PR conventions.
