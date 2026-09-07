---
layout: default
title: Stability Promise & Versioning Policy
nav_order: 9
description: "Gat v1.0 Language Stability Guarantee, Backend Support Matrix, and Semantic Versioning Policy."
permalink: /STABILITY.html
---

# Gat v1.0 Stability Promise & Versioning Policy

**Version:** 1.0.0  
**Effective Date:** September 2026  
**Status:** Active  

---

## 1. Executive Summary

The `gat` v1.0 release establishes a formal stability guarantee for language users, library authors, and tooling integrators. As a self-hosting systems programming language, `gat` adheres to a strict contract: **code that compiles and runs correctly under v1.0 will continue to compile and run with identical semantics on all minor and patch releases within the v1.x series.**

To earn this promise honestly, this document outlines the exact boundaries of what is **Frozen (Stable)** and guaranteed, and what remains explicitly **Experimental** or backend-constrained.

---

## 2. Language Surface Stability Matrix

### 2.1 Core Syntax & Type System (FROZEN)

| Feature | Stability Status | Guarantee |
|---|---|---|
| **Primitive Types** (`i64`, `bool`, `string`, `void`, `array`) | **Stable** | Layout, size, and semantics are permanently fixed. |
| **Pointers & Buffers** (`raw T`, `alloc_mem`, `free_mem`) | **Stable** | Word-sized raw pointer semantics and manual memory primitives remain unchanged. |
| **Composite Types** (`struct`, `class`) | **Stable** | 8-byte aligned field layouts, heap allocations via `new`, and class header layouts are frozen. |
| **Algebraic Enums** (`enum Name { Variant(Payload) }`) | **Stable** | Tagged sum types with 64-byte payload representation and pattern matching (`match`) are frozen. |
| **First-Class Functions** (`fn(T) -> U`) | **Stable** | 64-bit function address representation, indirect calls, and signature checking are frozen. |
| **Generics Model** (`<T, U>`) | **Stable** | Uniform 64-bit word-sized type erasure model is frozen for v1.x. Multi-word value types require `class` references or pointers. |
| **Control Flow** (`if`/`else`, `while`, `for ... in`, `return`) | **Stable** | Syntax and semantics are frozen. |
| **Pattern Matching** (`match target { Enum.Variant(var) => ... }`) | **Stable** | Exhaustive variant extraction syntax is frozen. |

### 2.2 Memory Model & Automatic Reference Counting (FROZEN)

| Feature | Stability Status | Guarantee |
|---|---|---|
| **Object Header Layout** | **Stable** | 24-byte header (`[-24: strong_count]`, `[-16: weak_count]`, `[-8: type_tag]`) is frozen for ABI compatibility. |
| **Deterministic RAII Destructors** (`deinit`) | **Stable** | Immediate synchronous invocation upon `strong_count == 0` is guaranteed. |
| **Cycle Breaking** (`weak T`, `weak_from`, `weak_upgrade`) | **Stable** | Non-owning weak reference semantics and two-tier deallocation are frozen. |
| **Thread-Boundary Isolation** | **Stable** | Compile-time prohibition against passing reference-counted types (`class`, `string`, `weak T`) across thread boundaries is frozen. |

### 2.3 Module System & Toolchain (FROZEN)

| Feature | Stability Status | Guarantee |
|---|---|---|
| **Flat Imports** (`import "path.gat";`) | **Stable** | Transitive file resolution and deduplication are frozen. |
| **Namespaced Imports** (`import "path.gat" as alias;`) | **Stable** | Symbol prefixing and collision isolation are frozen. |
| **Package Dependency System** (`.gat/deps/`, `gat.mod`, `gat.lock`) | **Stable** | Manifest schema and dependency directory layout are frozen. |
| **CLI Interface** (`gat run`, `gat build`, `gat check`, `gat explain`) | **Stable** | Core CLI flags (`-o`, `--target`, `--check`, `--explain`) are frozen. |
| **LSP Protocol Support** (LSP 3.17) | **Stable** | Hover, definition, symbols, autocomplete, and diagnostics protocol interfaces are frozen. |

---

## 3. Backend Platform Support Matrix

`gat` provides direct machine code generation across four target environments. Their stability tiers are defined below:

| Platform / Target | Architecture | Object / Binary Format | Runtime Model | Concurrency Support | Stability Tier |
|---|---|---|---|---|---|
| **Windows** | x86-64 | PE32+ Standalone `.exe` | Direct `kernel32.dll` / `ws2_32.dll` | Win32 Threads & Critical Sections | **Tier 1 (Stable)** |
| **Linux** | x86-64 | Static ELF64 Executable | Zero libc direct `syscall` instructions | Direct `sys_clone` & Futex Mutexes | **Tier 1 (Stable)** |
| **Linux ARM64** | AArch64 | Static ELF64 Executable | Zero libc direct `svc #0` instructions | **Experimental** (missing `sys_clone`) | **Tier 2 (Supported)** |
| **macOS Apple Silicon** | ARM64 | Mach-O 64-bit Object (`.o`) | Direct Darwin `svc #0x80` syscalls linked with clang | **Experimental** (missing Darwin thread runtime) | **Tier 2 (Supported)** |

### Explicit Exclusions & Backend Exceptions

> [!WARNING]
> **ARM64 Threading & Concurrency Limitation**:
> `std/thread.gat` (`thread_spawn`, `thread_join`) and `std/sync.gat` (`Mutex`) are **only supported on Tier 1 x86-64 backends (Windows and Linux)**. The ARM64 backend (`src/codegen_arm64.gat`) does not currently emit thread spawning instructions. ARM64 concurrency is explicitly excluded from the v1.0 stability guarantee and remains **Experimental**.

> [!NOTE]
> **macOS Mach-O Direct Executable Linking**:
> On macOS ARM64, `gatc` emits relocatable Mach-O 64-bit object files (`.o`) which are linked to final executables via host `clang` or `ld64`. Direct standalone Mach-O executable emission without external linker invocation is slated for a future minor release and is not frozen in v1.0.

---

## 4. Standard Library Surface (v1.0 Frozen vs Experimental)

### Frozen Modules (100% Stability Guarantee)
- **`std/str.gat`**: String utilities, search, slice, character extraction, concatenation, number formatting.
- **`std/io.gat`**: Basic standard input, output, and buffered printing.
- **`std/fs.gat`**: File reading, writing, size queries, existence checks, file deletion, temporary directory queries.
- **`std/math.gat`**: Arithmetic utilities, min/max, absolute value, power functions.
- **`std/process.gat`**: Process exit, PID inspection, command execution, process sleep.
- **`std/option.gat`**: `Option<T>` enum (`Some`, `None`), unwrapping, mapping, and default value extraction.
- **`std/result.gat`**: `Result<T, E>` enum (`Ok`, `Err`), status inspection, and unwrapping.
- **`std/vec.gat`**: Dynamic resizable array container.
- **`std/map.gat`**: Key-value hash map associative container.
- **`std/weak.gat`**: Weak reference lifecycle management (`weak_from`, `weak_upgrade`, count inspection).
- **`std/net.gat`**: Low-level TCP socket primitives (`socket`, `bind`, `listen`, `accept`, `connect`, `send`, `recv`, `close`) and `TcpStream`/`TcpListener` abstractions.
- **`std/json.gat`**: Streaming JSON tokenizer, AST parser, and serializer.
- **`std/http.gat`**: HTTP/1.1 client request parser, router, sequential server (`http_server_handle_one`), and response serializers.

### Conditionally Stable Modules
- **`std/thread.gat`** and **`std/sync.gat`**: **Frozen on Windows x86-64 and Linux x86-64**. Marked **Experimental** on ARM64 targets.
- **`http_server_handle_one_concurrent`**: Frozen on Tier 1 backends.

---

## 5. Semantic Versioning Policy (Post-1.0)

Following v1.0.0, `gat` strictly adheres to **Semantic Versioning 2.0.0** (`MAJOR.MINOR.PATCH`):

### 5.1 Patch Releases (`1.0.x`)
- Bug fixes in the compiler, code generators, and standard library.
- Performance enhancements, optimizer passes, and register allocation tuning.
- Documentation clarifications and error message quality improvements.
- **No changes to syntax, type rules, ABI, or standard library API signatures.**

### 5.2 Minor Releases (`1.x.0`)
- Backwards-compatible language additions (e.g. new keywords that do not collide with identifiers, new built-in intrinsics).
- New standard library functions, modules, or non-breaking methods.
- Upgrades of Tier 2 backends (e.g. stabilizing ARM64 concurrency).
- Implementation of compiler diagnostics explanation tools or IDE capabilities.
- Deprecation warnings for language features or stdlib APIs planned for removal in v2.0.

### 5.3 Major Releases (`2.0.0`)
- Any backward-incompatible syntax modification.
- Breaking modifications to standard library function signatures.
- Changes to object header layouts or calling convention ABIs.
- Removal of previously deprecated language features.

---

## 6. Deprecation Process

To ensure predictable maintenance for library authors, `gat` guarantees a transparent deprecation pipeline:

1. **Deprecation Notice**: A feature or API marked for deprecation will be flagged in official release notes and documentation with a clear migration path.
2. **Compiler Warning**: The compiler will emit a diagnostic warning when the deprecated syntax or symbol is encountered, referencing the recommended replacement.
3. **Minimum Window**: A deprecated feature must remain functional for a minimum of **two minor versions** (or 6 months, whichever is longer) before removal in the next major version release.
4. **Zero Silent Removals**: No feature or API will be removed or altered in breaking fashion within the v1.x series.
